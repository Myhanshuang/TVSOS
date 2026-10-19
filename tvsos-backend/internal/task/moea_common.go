package task

// ============================================================================
// 种群多目标算法共享基础设施（NSGA-II/III、MOEA/D、SPEA2、MOPSO 共用）。
//
// 个体编码 []int（运单下标→车辆下标）；评价统一走 SolveContext.Evaluate
// （多目标向量 + ε-约束分数）。内部目标统一转成"全最小化"四维向量
// {C2, -U2, U4, T2}，供支配排序/拥挤度/参考点关联/分解聚合复用。
//
// ε-约束支配（Deb 约束支配的主要目标法形态，种群算法的选择压力）：
//   硬可行（超载/类型/LIFO 无解）优于不可行 → 软约束违反量低者优 →
//   4 主目标帕累托支配。可行域内即经典支配，跨越可行边界时可行优先，
//   与 ε-终选（可行优先+主目标最优）语义一致。
// ============================================================================

import (
	"math"
	"sort"

	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

// moeaIndividual 种群个体。
type moeaIndividual struct {
	assigned []int
	eval     stateEvaluation
	obj      [4]float64 // 最小化方向原始目标（C2, -U2, U4, T2P95）
	norm     [4]float64 // 归一化目标（当代种群 min/max 归一，供拥挤度/关联/聚合）
	rank     int        // 非支配层级（1 起）
	crowd    float64    // 拥挤度（NSGA-II 截断用）
	fitness  float64    // SPEA2 适应度（越小越好）
}

// moObjectiveCount 参与种群选择的目标维数（4 主目标；F1/G2 走 ε-约束软约束路径）。
const moObjectiveCount = 4

// minimizeObjectives 目标向量转全最小化四维（U2 取负）。
func minimizeObjectives(v ObjectiveVector) [4]float64 {
	return [4]float64{v.C2UnitCost, -v.U2LoadingRate, v.U4EmptyRate, v.T2ResponseP95}
}

// moDominance ε-约束支配：+1 a 支配 b；-1 a 被 b 支配；0 互不支配。
func moDominance(a, b stateEvaluation) int {
	if a.Infeasible != b.Infeasible {
		if a.Infeasible < b.Infeasible {
			return 1
		}
		return -1
	}
	if a.Infeasible > 0 {
		// 双方均不可行：比较违反量（均为软约束口径的次级排序）
		if a.Violation != b.Violation {
			if a.Violation < b.Violation {
				return 1
			}
			return -1
		}
		return 0
	}
	if a.Violation != b.Violation {
		if a.Violation < b.Violation {
			return 1
		}
		return -1
	}
	return paretoDominance(a.Vector, b.Vector)
}

// moeaNormalize 当代种群 min/max 归一化（写入 ind.norm；宽度为 0 的维全置 0.5）。
func moeaNormalize(pop []moeaIndividual) {
	if len(pop) == 0 {
		return
	}
	var lo, hi [4]float64
	lo, hi = pop[0].obj, pop[0].obj
	for _, p := range pop[1:] {
		for j := 0; j < 4; j++ {
			if p.obj[j] < lo[j] {
				lo[j] = p.obj[j]
			}
			if p.obj[j] > hi[j] {
				hi[j] = p.obj[j]
			}
		}
	}
	for i := range pop {
		for j := 0; j < 4; j++ {
			width := hi[j] - lo[j]
			if width <= 0 {
				pop[i].norm[j] = 0.5
			} else {
				pop[i].norm[j] = (pop[i].obj[j] - lo[j]) / width
			}
		}
	}
}

// fastNondominatedSort 快速非支配排序（moDominance 口径），返回逐层下标集合（层内序稳定）。
func fastNondominatedSort(pop []moeaIndividual) [][]int {
	n := len(pop)
	dominatedBy := make([]int, n)          // 支配 i 的个体数
	dominates := make([][]int, 0, n)      // i 支配的个体列表
	fronts := make([][]int, 0, 8)
	current := make([]int, 0, n)
	for i := 0; i < n; i++ {
		dominates = append(dominates, nil)
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			if moDominance(pop[i].eval, pop[j].eval) > 0 {
				dominates[i] = append(dominates[i], j)
			} else if moDominance(pop[j].eval, pop[i].eval) > 0 {
				dominatedBy[i]++
			}
		}
		if dominatedBy[i] == 0 {
			current = append(current, i)
		}
	}
	rank := 1
	for len(current) > 0 {
		for _, i := range current {
			pop[i].rank = rank
		}
		fronts = append(fronts, current)
		next := make([]int, 0, len(current))
		for _, i := range current {
			for _, j := range dominates[i] {
				dominatedBy[j]--
				if dominatedBy[j] == 0 {
					next = append(next, j)
				}
			}
		}
		current = next
		rank++
	}
	return fronts
}

// crowdingDistance 计算某层内个体的拥挤度（基于 norm 四维；边界个体置 +Inf）。
func crowdingDistance(pop []moeaIndividual, front []int) {
	for _, i := range front {
		pop[i].crowd = 0
	}
	if len(front) <= 2 {
		for _, i := range front {
			pop[i].crowd = math.Inf(1)
		}
		return
	}
	for j := 0; j < moObjectiveCount; j++ {
		order := append([]int(nil), front...)
		sort.SliceStable(order, func(a, b int) bool { return pop[order[a]].norm[j] < pop[order[b]].norm[j] })
		first, last := order[0], order[len(order)-1]
		pop[first].crowd = math.Inf(1)
		pop[last].crowd = math.Inf(1)
		width := pop[last].norm[j] - pop[first].norm[j]
		if width <= 0 {
			continue
		}
		for k := 1; k < len(order)-1; k++ {
			i := order[k]
			if math.IsInf(pop[i].crowd, 1) {
				continue
			}
			pop[i].crowd += (pop[order[k+1]].norm[j] - pop[order[k-1]].norm[j]) / width
		}
	}
}

// moeaTournament 二元锦标赛（rank 小者、同级拥挤度大者胜出）。
func moeaTournament(pop []moeaIndividual, sc *SolveContext) int {
	a := sc.RandInt(len(pop))
	b := sc.RandInt(len(pop))
	if pop[a].rank < pop[b].rank {
		return a
	}
	if pop[b].rank < pop[a].rank {
		return b
	}
	if pop[a].crowd > pop[b].crowd {
		return a
	}
	return b
}

// uniformCrossover 均匀交叉：逐基因按硬币从父代取样（适合指派编码）。
func uniformCrossover(sc *SolveContext, p1, p2 []int) ([]int, []int) {
	c1 := make([]int, len(p1))
	c2 := make([]int, len(p1))
	for i := range p1 {
		if sc.Rand() < 0.5 {
			c1[i], c2[i] = p1[i], p2[i]
		} else {
			c1[i], c2[i] = p2[i], p1[i]
		}
	}
	return c1, c2
}

// mutateAssignment 变异：以 perGene 概率对基因重派随机（可行偏好）车辆。
func mutateAssignment(sc *SolveContext, x []int, perGeneProb float64) {
	for i := range x {
		if sc.Rand() < perGeneProb {
			x[i] = sc.RandomVehicleFor(i, 5)
		}
	}
}

// newMoeaIndividual 评价并构造个体。
func newMoeaIndividual(sc *SolveContext, assigned []int) moeaIndividual {
	ev := sc.Evaluate(assigned)
	return moeaIndividual{assigned: assigned, eval: ev, obj: minimizeObjectives(ev.Vector)}
}

// initPopulation 随机可行化初始种群（含共享初始解为第 0 个个体）。
func initPopulation(sc *SolveContext, size int) []moeaIndividual {
	pop := make([]moeaIndividual, 0, size)
	pop = append(pop, newMoeaIndividual(sc, append([]int(nil), sc.Initial...)))
	for len(pop) < size {
		pop = append(pop, newMoeaIndividual(sc, sc.RandomAssignment()))
	}
	return pop
}

// moeaDefaults 种群算法超参回落（零值→内置默认）。
type moeaDefaults struct {
	Population, Generations int
	CrossoverProb, MutationProb float64
}

func resolveMoeaDefaults(p sandboxpkg.AlgorithmParams, autoPopulation int) moeaDefaults {
	d := moeaDefaults{
		Population: p.Population, Generations: p.Generations,
		CrossoverProb: p.CrossoverProb, MutationProb: p.MutationProb,
	}
	if d.Population <= 0 {
		d.Population = autoPopulation
		if d.Population <= 0 {
			d.Population = 40
		}
	}
	if d.Generations <= 0 {
		d.Generations = 60
	}
	if d.CrossoverProb <= 0 {
		d.CrossoverProb = 0.9
	}
	if d.MutationProb <= 0 {
		d.MutationProb = 0.25
	}
	return d
}

// moeaBestByScore 种群中 ε 分数最优个体下标。
func moeaBestByScore(pop []moeaIndividual, weighted bool) (int, stateEvaluation, bool) {
	best := -1
	for i, p := range pop {
		if best < 0 || p.eval.scoreOrRawCost(weighted) < pop[best].eval.scoreOrRawCost(weighted) {
			best = i
		}
	}
	if best < 0 {
		return 0, stateEvaluation{}, false
	}
	return best, pop[best].eval, true
}

// updateRunningBest 更新（含拷贝指派）跨代运行最优解。
func updateRunningBest(sc *SolveContext, best *SolverCandidate, has *bool, cand []int, ev stateEvaluation) {
	if *has && best.Eval.scoreOrRawCost(sc.ObjCtx.weighted) <= ev.scoreOrRawCost(sc.ObjCtx.weighted) {
		return
	}
	best.Assigned = append([]int(nil), cand...)
	best.Eval = ev
	*has = true
}

// individualsToCandidates 个体转求解器候选集。
func individualsToCandidates(pop []moeaIndividual, idx []int) []SolverCandidate {
	if len(idx) == 0 {
		return nil
	}
	out := make([]SolverCandidate, 0, len(idx))
	for _, i := range idx {
		out = append(out, SolverCandidate{Assigned: pop[i].assigned, Eval: pop[i].eval})
	}
	return out
}

// ---- 贪心首可行基线（NF 求解器，C1 基线同口径） ----

// nfSolver 贪心首可行求解器：按运单顺序取车辆列表中第一辆可行车
// （不可行强指派首车，由评价的硬可行性惩罚体现）。无迭代、无随机源，
// 与 C1 基线策略完全同口径，作为"算法集"中的确定性参照。
type nfSolver struct{}

func init() { RegisterSolver(nfSolver{}) }

func (nfSolver) Name() string        { return sandboxpkg.SolverNF }
func (nfSolver) DisplayName() string { return "贪心首可行 NF（确定性基线）" }

func (nfSolver) Solve(sc *SolveContext) SolveOutput {
	assigned := make([]int, len(sc.Shipments))
	loads := make(map[uint]int, len(sc.Vehicles))
	for _, v := range sc.Vehicles {
		if v != nil {
			loads[v.Id] = sc.ProjectedLoads[v.Id]
		}
	}
	for i, ship := range sc.Shipments {
		cargo := getCargoByCache(sc.CargoCache, ship.CargoId)
		chosen := 0
		for vi, v := range sc.Vehicles {
			if isVehicleFeasible(v, ship, cargo, loads[v.Id]) {
				chosen = vi
				break
			}
		}
		assigned[i] = chosen
		if cargo != nil && sc.Vehicles[chosen] != nil {
			loads[sc.Vehicles[chosen].Id] += cargo.Weight * ship.Count
		}
	}
	ev := sc.Evaluate(assigned)
	sc.RecordPoint(0, ev, 1)
	return SolveOutput{
		Candidates: []SolverCandidate{{Assigned: assigned, Eval: ev}},
		Best:       SolverCandidate{Assigned: assigned, Eval: ev},
		HasBest:    true,
	}
}
