package task

// ============================================================================
// NSGA-II / NSGA-III 求解器插件。
//
// NSGA-II（Deb 2002）：快速非支配排序 + 拥挤度距离的环境选择，
// 二元锦标赛配对 + 均匀交叉 + 变异生成子代。支配口径为 ε-约束支配
// （硬可行 → 软约束违反量 → 4 主目标向量），使进化压力指向
// "可满足软约束的帕累托前沿"，与主要目标法语义一致。
//
// NSGA-III（Deb & Jain 2014）：以 Das-Dennis 参考点替代拥挤度做
// 多目标（≥4 维）下的多样性保持：极值点 ASF 归一化 → 参考点垂直
// 距离关联 → 小生境计数（niche count）填充最后前沿层。
// ============================================================================

import (
	"math"
	"sort"

	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

// nsgaBaseNSGA2Pop 默认种群规模。
const nsgaDefaultPop = 40

// ---- NSGA-II ----

type nsga2Solver struct{}

func init() { RegisterSolver(nsga2Solver{}) }

func (nsga2Solver) Name() string        { return sandboxpkg.SolverNSGA2 }
func (nsga2Solver) DisplayName() string { return "NSGA-II（快速非支配排序 + 拥挤度）" }

func (nsga2Solver) Solve(sc *SolveContext) SolveOutput {
	d := resolveMoeaDefaults(sc.Params(), nsgaDefaultPop)
	pop := initPopulation(sc, d.Population)
	nsga2PrepareSelection(pop) // 排位+拥挤度（锦标赛判据）

	best := SolverCandidate{}
	hasBest := false
	for i := range pop {
		updateRunningBest(sc, &best, &hasBest, pop[i].assigned, pop[i].eval)
	}
	updateRunningBest(sc, &best, &hasBest, sc.Initial, sc.InitialEval)

	for gen := 1; gen <= d.Generations; gen++ {
		// 配对繁殖：锦标赛选择 + 均匀交叉 + 变异（父代切片拷贝防别名污染）
		offspring := make([]moeaIndividual, 0, d.Population)
		for len(offspring) < d.Population {
			p1 := pop[moeaTournament(pop, sc)].assigned
			p2 := pop[moeaTournament(pop, sc)].assigned
			c1, c2 := append([]int(nil), p1...), append([]int(nil), p2...)
			if sc.Rand() < d.CrossoverProb {
				c1, c2 = uniformCrossover(sc, p1, p2)
			}
			perGene := d.MutationProb / float64(maxInt(1, len(c1)))
			mutateAssignment(sc, c1, perGene)
			mutateAssignment(sc, c2, perGene)
			offspring = append(offspring, newMoeaIndividual(sc, c1))
			if len(offspring) < d.Population {
				offspring = append(offspring, newMoeaIndividual(sc, c2))
			}
		}

		// 合并父子代 → 非支配排序 + 拥挤度截断的环境选择（μ+λ）
		merged := append(append([]moeaIndividual(nil), pop...), offspring...)
		moeaNormalize(merged)
		fronts := fastNondominatedSort(merged)
		next := make([]moeaIndividual, 0, d.Population)
		var lastFront []int
		for _, f := range fronts {
			if len(next)+len(f) <= d.Population {
				for _, i := range f {
					next = append(next, merged[i])
				}
				continue
			}
			lastFront = f
			break
		}
		if len(lastFront) > 0 && len(next) < d.Population {
			crowdingDistance(merged, lastFront)
			order := append([]int(nil), lastFront...)
			sort.SliceStable(order, func(a, b int) bool { return merged[order[a]].crowd > merged[order[b]].crowd })
			for _, i := range order {
				if len(next) >= d.Population {
					break
				}
				next = append(next, merged[i])
			}
		}
		pop = next
		nsga2PrepareSelection(pop) // 更新排位+拥挤度（下代锦标赛判据）

		for i := range pop {
			updateRunningBest(sc, &best, &hasBest, pop[i].assigned, pop[i].eval)
		}
		sc.RecordPoint(float64(gen), best.Eval, nondominatedCount(pop))
	}

	// 终代第一前沿为候选集（ε-终选交派发器统一裁决）
	moeaNormalize(pop)
	fronts := fastNondominatedSort(pop)
	front1 := []int(nil)
	if len(fronts) > 0 {
		front1 = fronts[0]
	}
	return SolveOutput{
		Candidates: individualsToCandidates(pop, front1),
		Best:       best,
		HasBest:    hasBest,
	}
}

// nondominatedCount 种群中不被种群内任一个体支配的数量（前沿规模近似）。
func nondominatedCount(pop []moeaIndividual) int {
	count := 0
	for i, p := range pop {
		dominated := false
		for j, q := range pop {
			if i == j {
				continue
			}
			if moDominance(q.eval, p.eval) > 0 {
				dominated = true
				break
			}
		}
		if !dominated {
			count++
		}
	}
	return count
}

// nsga2PrepareSelection 交配选择前的种群预处理：归一化 + 非支配排位 + 逐层拥挤度
// （锦标赛判据 rank 小者优、同级 crowd 大者优；标准 NSGA-II 对全部前沿层算拥挤度）。
func nsga2PrepareSelection(pop []moeaIndividual) {
	if len(pop) == 0 {
		return
	}
	moeaNormalize(pop)
	fronts := fastNondominatedSort(pop)
	for _, f := range fronts {
		crowdingDistance(pop, f)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---- NSGA-III ----

type nsga3Solver struct{}

func init() { RegisterSolver(nsga3Solver{}) }

func (nsga3Solver) Name() string        { return sandboxpkg.SolverNSGA3 }
func (nsga3Solver) DisplayName() string { return "NSGA-III（参考点小生境保留）" }

func (nsga3Solver) Solve(sc *SolveContext) SolveOutput {
	p := sc.Params()
	H := p.RefDivisions
	if H <= 0 {
		H = 4
	}
	refs := dasDennisPoints(H, moObjectiveCount)
	d := resolveMoeaDefaults(p, len(refs))
	// NSGA-III 种群规模与参考点数对齐（取 ≥ 参考点数的最小倍数保证覆盖）
	if p.Population <= 0 {
		d.Population = len(refs)
	}

	pop := initPopulation(sc, d.Population)

	best := SolverCandidate{}
	hasBest := false
	for i := range pop {
		updateRunningBest(sc, &best, &hasBest, pop[i].assigned, pop[i].eval)
	}
	updateRunningBest(sc, &best, &hasBest, sc.Initial, sc.InitialEval)

	for gen := 1; gen <= d.Generations; gen++ {
		// 繁殖（与 NSGA-II 同款算子；父代切片拷贝防别名污染）
		offspring := make([]moeaIndividual, 0, d.Population)
		for len(offspring) < d.Population {
			p1 := pop[moeaTournament(pop, sc)].assigned
			p2 := pop[moeaTournament(pop, sc)].assigned
			c1, c2 := append([]int(nil), p1...), append([]int(nil), p2...)
			if sc.Rand() < d.CrossoverProb {
				c1, c2 = uniformCrossover(sc, p1, p2)
			}
			perGene := d.MutationProb / float64(maxInt(1, len(c1)))
			mutateAssignment(sc, c1, perGene)
			mutateAssignment(sc, c2, perGene)
			offspring = append(offspring, newMoeaIndividual(sc, c1))
			if len(offspring) < d.Population {
				offspring = append(offspring, newMoeaIndividual(sc, c2))
			}
		}

		merged := append(append([]moeaIndividual(nil), pop...), offspring...)
		fronts := fastNondominatedSort(merged)
		nsga3Normalize(merged)

		// 环境选择：整层填充 + 最后层小生境保留
		next := make([]moeaIndividual, 0, d.Population)
		var lastFront []int
		selected := make([]int, 0, d.Population)
		for _, f := range fronts {
			if len(next)+len(f) <= d.Population {
				for _, i := range f {
					next = append(next, merged[i])
					selected = append(selected, i)
				}
				continue
			}
			lastFront = f
			break
		}
		if len(lastFront) > 0 && len(next) < d.Population {
			chosen := nsga3Niching(merged, selected, lastFront, refs, sc, d.Population-len(next))
			for _, i := range chosen {
				next = append(next, merged[i])
			}
		}
		pop = next

		for i := range pop {
			updateRunningBest(sc, &best, &hasBest, pop[i].assigned, pop[i].eval)
		}
		sc.RecordPoint(float64(gen), best.Eval, nondominatedCount(pop))
	}

	fronts := fastNondominatedSort(pop)
	front1 := []int(nil)
	if len(fronts) > 0 {
		front1 = fronts[0]
	}
	return SolveOutput{
		Candidates: individualsToCandidates(pop, front1),
		Best:       best,
		HasBest:    hasBest,
	}
}

// nsga3Normalize NSGA-III 归一化：理想点 + 极值点(ASF)截距（失败回落当代极差）。
func nsga3Normalize(pop []moeaIndividual) {
	if len(pop) == 0 {
		return
	}
	var ideal [4]float64
	ideal = pop[0].obj
	for _, p := range pop[1:] {
		for j := 0; j < 4; j++ {
			if p.obj[j] < ideal[j] {
				ideal[j] = p.obj[j]
			}
		}
	}

	// 极值点：各轴 ASF 最小者（w 轴=1，其余取小 eps 防零）
	extremes := make([][4]float64, 4)
	for axis := 0; axis < 4; axis++ {
		bestIdx := 0
		bestASF := math.Inf(1)
		for i, p := range pop {
			asf := -math.Inf(1)
			for j := 0; j < 4; j++ {
				w := 1e-6
				if j == axis {
					w = 1
				}
				v := (p.obj[j] - ideal[j]) / w
				if v > asf {
					asf = v
				}
			}
			if asf < bestASF {
				bestASF = asf
				bestIdx = i
			}
		}
		extremes[axis] = pop[bestIdx].obj
	}

	// 截距：过 4 个极值点的超平面 Σ f_j/a_j = 1 → 解 A·x=1（x_j = 1/a_j）
	var intercepts [4]float64
	if xs, ok := solveLinear4(extremes); ok {
		for j := 0; j < 4; j++ {
			if xs[j] > 1e-9 {
				intercepts[j] = 1 / xs[j]
				continue
			}
			intercepts[j] = 0
		}
	}
	// 回落：截距非法（奇异/非正）时用当代极差
	var nadir [4]float64
	nadir = pop[0].obj
	for _, p := range pop[1:] {
		for j := 0; j < 4; j++ {
			if p.obj[j] > nadir[j] {
				nadir[j] = p.obj[j]
			}
		}
	}
	for j := 0; j < 4; j++ {
		if intercepts[j] <= 1e-9 {
			intercepts[j] = nadir[j] - ideal[j]
		}
	}
	for i := range pop {
		for j := 0; j < 4; j++ {
			w := intercepts[j]
			if w <= 0 {
				pop[i].norm[j] = 0.5
			} else {
				pop[i].norm[j] = (pop[i].obj[j] - ideal[j]) / w
			}
		}
	}
}

// solveLinear4 高斯消元解 4×4 线性方程 A·x = 1（A 行向量=极值点坐标）。
func solveLinear4(A [][4]float64) ([4]float64, bool) {
	m := make([][]float64, 4)
	for i := 0; i < 4; i++ {
		m[i] = make([]float64, 5)
		for j := 0; j < 4; j++ {
			m[i][j] = A[i][j]
		}
		m[i][4] = 1
	}
	for col := 0; col < 4; col++ {
		pivot := col
		for r := col + 1; r < 4; r++ {
			if math.Abs(m[r][col]) > math.Abs(m[pivot][col]) {
				pivot = r
			}
		}
		if math.Abs(m[pivot][col]) < 1e-12 {
			return [4]float64{}, false
		}
		m[col], m[pivot] = m[pivot], m[col]
		for r := 0; r < 4; r++ {
			if r == col {
				continue
			}
			factor := m[r][col] / m[col][col]
			for c := col; c < 5; c++ {
				m[r][c] -= factor * m[col][c]
			}
		}
	}
	var x [4]float64
	for i := 0; i < 4; i++ {
		x[i] = m[i][4] / m[i][i]
	}
	return x, true
}

// dasDennisPoints Das-Dennis 单纯形格点：M 维 H 划分 → C(H+M-1, M-1) 个权重/参考点。
func dasDennisPoints(H, M int) [][]float64 {
	if H <= 0 {
		H = 4
	}
	var out [][]float64
	var build func(curr []int, remaining int)
	build = func(curr []int, remaining int) {
		if len(curr) == M-1 {
			w := make([]float64, M)
			for i, c := range curr {
				w[i] = float64(c) / float64(H)
			}
			w[M-1] = float64(remaining) / float64(H)
			out = append(out, w)
			return
		}
		for x := 0; x <= remaining; x++ {
			build(append(curr, x), remaining-x)
		}
	}
	build([]int{}, H)
	return out
}

// nsga3Niching NSGA-III 小生境保留选择：返回 lastFront 中入选个体下标。
// 流程：全体个体按垂直距离关联到最近参考点 → 已入选层构成各参考点的 niche 计数 →
// 按计数升序（平局用预洗牌的随机序）逐参考点取其关联的最后前沿成员（距离近者优先）。
func nsga3Niching(pop []moeaIndividual, selected, lastFront []int, refs [][]float64, sc *SolveContext, slots int) []int {
	if slots <= 0 || len(lastFront) == 0 {
		return nil
	}

	// 关联：个体 → 最近参考点（参考射线垂直距离）
	assocOf := func(norm [4]float64) (int, float64) {
		best, bestDist := 0, math.Inf(1)
		for ri, r := range refs {
			rn := 0.0
			for j := 0; j < 4; j++ {
				rn += r[j] * r[j]
			}
			if rn <= 1e-12 {
				continue
			}
			dot := 0.0
			for j := 0; j < 4; j++ {
				dot += norm[j] * r[j]
			}
			projScale := dot / rn
			dist := 0.0
			for j := 0; j < 4; j++ {
				d := norm[j] - projScale*r[j]
				dist += d * d
			}
			if dist < bestDist {
				bestDist, best = dist, ri
			}
		}
		return best, math.Sqrt(bestDist)
	}

	// niche 计数（已入选层）
	niche := make([]int, len(refs))
	for _, i := range selected {
		r, _ := assocOf(pop[i].norm)
		if r >= 0 && r < len(niche) {
			niche[r]++
		}
	}
	// 最后前沿成员按参考点分桶（距离升序）
	type member struct{ idx, ref int; dist float64 }
	members := make([]member, 0, len(lastFront))
	for _, i := range lastFront {
		r, dist := assocOf(pop[i].norm)
		members = append(members, member{idx: i, ref: r, dist: dist})
	}

	// 参考点序：niche 升序 + 预洗牌随机平局
	order := make([]int, len(refs))
	for i := range order {
		order[i] = i
	}
	for i := len(order) - 1; i > 0; i-- {
		j := sc.RandInt(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	sort.SliceStable(order, func(a, b int) bool { return niche[order[a]] < niche[order[b]] })

	chosen := make([]int, 0, slots)
	for _, ri := range order {
		if len(chosen) >= slots {
			break
		}
		bucket := make([]member, 0, 4)
		for _, m := range members {
			if m.ref == ri {
				bucket = append(bucket, m)
			}
		}
		if len(bucket) == 0 {
			continue
		}
		sort.SliceStable(bucket, func(a, b int) bool { return bucket[a].dist < bucket[b].dist })
		take := 1
		if niche[ri] == 0 {
			take = 1 // 标准取一：ρ=0 时取距离最近者；一轮结束后若仍有空位再循环
		}
		for k := 0; k < take && k < len(bucket) && len(chosen) < slots; k++ {
			chosen = append(chosen, bucket[k].idx)
			niche[ri]++
			// 移除已选成员
			for mi := range members {
				if members[mi].idx == bucket[k].idx {
					members = append(members[:mi], members[mi+1:]...)
					break
				}
			}
		}
	}
	// 仍有空位：第二轮不再考虑 niche 计数，按剩余距离补齐
	if len(chosen) < slots {
		sort.SliceStable(members, func(a, b int) bool { return members[a].dist < members[b].dist })
		for _, m := range members {
			if len(chosen) >= slots {
				break
			}
			chosen = append(chosen, m.idx)
		}
	}
	return chosen
}
