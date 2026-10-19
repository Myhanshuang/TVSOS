package task

// ============================================================================
// MOEA/D 求解器插件（Zhang & Li 2007，分解式多目标进化算法）。
//
// 把 4 目标问题分解为 N 个标量子问题（每子问题一个权重向量 λ），每个体
// 对应一个子问题；进化只在其权重邻域内进行（邻域内解互相替换）。
// 聚合函数可选 Tchebycheff / 加权和 / PBI（decomposition 参数）。
// 与主要目标法的结合：子问题聚合值在归一化目标上计算，另叠加
// 软约束相对违反量惩罚与硬可行性惩罚——分解探索多目标结构，
// ε-约束语义约束可行域，终选仍走统一 ε 分数裁决。
// ============================================================================

import (
	"math"
	"sort"

	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

type moeadSolver struct{}

func init() { RegisterSolver(moeadSolver{}) }

func (moeadSolver) Name() string        { return sandboxpkg.SolverMOEAD }
func (moeadSolver) DisplayName() string { return "MOEA/D（分解 + 邻域进化）" }

// moeadDecomposition 聚合函数代号解析（非法值回落 tchebycheff）。
func moeadDecomposition(name string) string {
	switch name {
	case "weighted_sum", "pbi":
		return name
	default:
		return "tchebycheff"
	}
}

// pbiPenaltyTheta PBI 惩罚系数 θ（常规取 5）。
const pbiPenaltyTheta = 5.0

func (moeadSolver) Solve(sc *SolveContext) SolveOutput {
	p := sc.Params()
	H := p.RefDivisions
	if H <= 0 {
		H = 4
	}
	weights := dasDennisPoints(H, moObjectiveCount)
	decomp := moeadDecomposition(p.Decomposition)

	// 子问题数=种群规模：未指定时取权重向量数；显式指定时权重循环复用
	N := p.Population
	if N <= 0 {
		N = len(weights)
	}
	d := resolveMoeaDefaults(p, N)
	N = d.Population
	neighborhood := p.Neighborhood
	if neighborhood <= 0 {
		neighborhood = 10
	}

	// 子问题权重（循环复用）与邻域（权重欧氏距离近邻）
	lambdas := make([][]float64, N)
	for i := range lambdas {
		lambdas[i] = weights[i%len(weights)]
	}
	B := moeadNeighborhoods(lambdas, neighborhood)

	// 初始种群：一子问题一个体；理想/最差点（归一化用）
	pop := initPopulation(sc, N)
	ideal, nadir := moeadIdealNadir(pop)

	best := SolverCandidate{}
	hasBest := false
	for i := range pop {
		updateRunningBest(sc, &best, &hasBest, pop[i].assigned, pop[i].eval)
	}
	updateRunningBest(sc, &best, &hasBest, sc.Initial, sc.InitialEval)

	g := func(ind *moeaIndividual, lambda []float64) float64 {
		return moeadAggregate(ind, ideal, nadir, lambda, decomp)
	}

	for gen := 1; gen <= d.Generations; gen++ {
		for i := 0; i < N; i++ {
			// 邻域内取双亲繁殖一个子代（父代切片拷贝防别名污染）
			p1 := pop[B[i][sc.RandInt(len(B[i]))]].assigned
			p2 := pop[B[i][sc.RandInt(len(B[i]))]].assigned
			child := append([]int(nil), p1...)
			if sc.Rand() < d.CrossoverProb {
				c1, _ := uniformCrossover(sc, p1, p2)
				child = c1
			}
			perGene := d.MutationProb / float64(maxInt(1, len(child)))
			mutateAssignment(sc, child, perGene)

			y := newMoeaIndividual(sc, child)

			// 更新理想/最差点（归一化锚点）
			for j := 0; j < 4; j++ {
				if y.obj[j] < ideal[j] {
					ideal[j] = y.obj[j]
				}
				if y.obj[j] > nadir[j] {
					nadir[j] = y.obj[j]
				}
			}

			// 邻域内解替换：子问题聚合值更优则替换
			for _, j := range B[i] {
				if g(&y, lambdas[j]) <= g(&pop[j], lambdas[j]) {
					pop[j] = y
					updateRunningBest(sc, &best, &hasBest, y.assigned, y.eval)
				}
			}
		}
		sc.RecordPoint(float64(gen), best.Eval, nondominatedCount(pop))
	}

	// 终代非支配子集作为候选集
	fronts := fastNondominatedSort(pop)
	var nd []int
	if len(fronts) > 0 {
		nd = fronts[0]
	}
	return SolveOutput{
		Candidates: individualsToCandidates(pop, nd),
		Best:       best,
		HasBest:    hasBest,
	}
}

// moeadIdealNadir 取种群当前理想点（各维最小）与最差点（各维最大）。
func moeadIdealNadir(pop []moeaIndividual) ([4]float64, [4]float64) {
	var ideal, nadir [4]float64
	if len(pop) == 0 {
		return ideal, nadir
	}
	ideal, nadir = pop[0].obj, pop[0].obj
	for _, p := range pop[1:] {
		for j := 0; j < 4; j++ {
			if p.obj[j] < ideal[j] {
				ideal[j] = p.obj[j]
			}
			if p.obj[j] > nadir[j] {
				nadir[j] = p.obj[j]
			}
		}
	}
	return ideal, nadir
}

// moeadAggregate 子问题聚合值：目标先按理想/最差点归一（量纲无关），
// 再按聚合函数计算；叠加软约束违反惩罚（同 ε-约束权重）与硬可行性惩罚。
func moeadAggregate(ind *moeaIndividual, ideal, nadir [4]float64, lambda []float64, decomp string) float64 {
	if ind.eval.Infeasible > 0 {
		return SAPenaltyInf + float64(ind.eval.Infeasible) + ind.eval.Violation
	}

	// 归一化目标 ∈ [0,1]（宽度退化维置 0.5 中性值）
	var f [4]float64
	for j := 0; j < 4; j++ {
		width := nadir[j] - ideal[j]
		if width <= 0 {
			f[j] = 0.5
		} else {
			f[j] = (ind.obj[j] - ideal[j]) / width
		}
	}

	var scalar float64
	switch decomp {
	case "weighted_sum":
		for j := 0; j < 4; j++ {
			scalar += lambda[j] * f[j]
		}
	case "pbi":
		l2 := 0.0
		dot := 0.0
		for j := 0; j < 4; j++ {
			l2 += lambda[j] * lambda[j]
			dot += f[j] * lambda[j]
		}
		if l2 <= 1e-12 {
			l2 = 1
		}
		d1 := dot / math.Sqrt(l2)
		d2sq := 0.0
		for j := 0; j < 4; j++ {
			proj := d1 * lambda[j] / math.Sqrt(l2)
			d := f[j] - proj
			d2sq += d * d
		}
		scalar = d1 + pbiPenaltyTheta*math.Sqrt(d2sq)
	default: // tchebycheff
		scalar = -math.Inf(1)
		for j := 0; j < 4; j++ {
			w := lambda[j]
			if w < 1e-6 {
				w = 1e-6
			}
			v := f[j] / w
			if v > scalar {
				scalar = v
			}
		}
	}
	return scalar + softPenaltyWeight*ind.eval.Violation
}

// moeadNeighborhoods 按权重欧氏距离构造每子问题的近邻下标集合。
func moeadNeighborhoods(lambdas [][]float64, T int) [][]int {
	n := len(lambdas)
	if T > n {
		T = n
	}
	B := make([][]int, n)
	for i := range B {
		idx := make([]int, n)
		for k := range idx {
			idx[k] = k
		}
		sort.SliceStable(idx, func(a, b int) bool {
			return weightDist(lambdas[i], lambdas[idx[a]]) < weightDist(lambdas[i], lambdas[idx[b]])
		})
		B[i] = idx[:T]
	}
	return B
}

func weightDist(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return math.Sqrt(s)
}
