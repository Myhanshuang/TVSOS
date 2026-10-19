package task

// ============================================================================
// SPEA2 求解器插件（Zitzler et al. 2001，改进强度帕累托进化算法）。
//
// 适应度 F(i) = R(i) + D(i)：
//   R(i)（原始适应度）= Σ_{j 支配 i} S(j)，S(j)=j 支配的个体数 →
//     被强支配者（支配了许多个体的解的后代）的 R 大，F(i) ≥ 1 即被支配；
//   D(i)（密度）= 1/(σ_k(i)+2)，σ_k = 到第 k 近邻的距离，k=√N →
//     密度高者 D 大，天然低强度地惩罚拥挤。
// 环境选择：非支配解全部进外部档案；超额按 k-近邻距离迭代截断
// （每次移除离剩余群体最近者，保持档案多样性）；不足从支配解按 F 补齐。
// 配对锦标赛仅从档案中选取。支配口径为 ε-约束支配。
// ============================================================================

import (
	"math"
	"sort"

	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

type spea2Solver struct{}

func init() { RegisterSolver(spea2Solver{}) }

func (spea2Solver) Name() string        { return sandboxpkg.SolverSPEA2 }
func (spea2Solver) DisplayName() string { return "SPEA2（强度适应度 + k-近邻密度档案）" }

// spea2Entry 档案/合并集条目（个体 + 归一化坐标缓存）。
type spea2Entry struct {
	ind      moeaIndividual
	dist     []float64 // 到合并集内其余成员的欧氏距离（norm 空间）
	strength int
	raw      float64
	density  float64
	fitness  float64
}

func (spea2Solver) Solve(sc *SolveContext) SolveOutput {
	p := sc.Params()
	archiveSize := p.ArchiveSize
	if archiveSize <= 0 {
		archiveSize = 30
	}
	if archiveSize > paretoArchiveCap*4 {
		archiveSize = paretoArchiveCap * 4 // 上限防极端配置内存膨胀
	}
	d := resolveMoeaDefaults(p, nsgaDefaultPop)

	// 初始种群与初始档案（种群非支配子集）
	pop := initPopulation(sc, d.Population)
	archive := spea2Nondominated(pop)

	best := SolverCandidate{}
	hasBest := false
	for i := range pop {
		updateRunningBest(sc, &best, &hasBest, pop[i].assigned, pop[i].eval)
	}
	updateRunningBest(sc, &best, &hasBest, sc.Initial, sc.InitialEval)

	for gen := 1; gen <= d.Generations; gen++ {
		// 合并种群与档案 → 计算强度/原始适应度/密度/适应度
		combined := append(append([]moeaIndividual(nil), pop...), archive...)
		moeaNormalize(combined)
		entries := spea2Fitness(combined)

		// 环境选择构建新档案
		archive = spea2Environmental(entries, archiveSize)

		// 繁殖：档案内二元锦标赛（适应度小者胜）+ 交叉 + 变异
		offspring := make([]moeaIndividual, 0, d.Population)
		for len(offspring) < d.Population {
			p1 := archive[spea2Tournament(archive, sc)].assigned
			p2 := archive[spea2Tournament(archive, sc)].assigned
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
		pop = offspring

		for i := range archive {
			updateRunningBest(sc, &best, &hasBest, archive[i].assigned, archive[i].eval)
		}
		sc.RecordPoint(float64(gen), best.Eval, len(archive))
	}

	// 档案即候选集（ε-终选交派发器统一裁决）
	candidates := make([]SolverCandidate, 0, len(archive))
	for i := range archive {
		candidates = append(candidates, SolverCandidate{Assigned: archive[i].assigned, Eval: archive[i].eval})
	}
	return SolveOutput{
		Candidates: candidates,
		Best:       best,
		HasBest:    hasBest,
	}
}

// spea2Fitness 计算 SPEA2 适应度：strength → raw → density → fitness。
func spea2Fitness(combined []moeaIndividual) []spea2Entry {
	n := len(combined)
	entries := make([]spea2Entry, n)
	for i := range entries {
		entries[i].ind = combined[i]
		entries[i].dist = make([]float64, n)
	}

	// 强度 S(i) 与距离矩阵（norm 欧氏）
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				entries[i].dist[j] = math.Inf(1)
				continue
			}
			s := 0.0
			for k := 0; k < 4; k++ {
				d := combined[i].norm[k] - combined[j].norm[k]
				s += d * d
			}
			entries[i].dist[j] = math.Sqrt(s)
			if moDominance(combined[i].eval, combined[j].eval) > 0 {
				entries[i].strength++
			}
		}
	}
	// 原始适应度 R(i) = Σ_{j 支配 i} S(j)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i != j && moDominance(combined[j].eval, combined[i].eval) > 0 {
				entries[i].raw += float64(entries[j].strength)
			}
		}
	}
	// 密度 D(i) = 1/(σ_k + 2)，k = √N
	k := int(math.Sqrt(float64(n)))
	if k < 1 {
		k = 1
	}
	for i := range entries {
		ds := append([]float64(nil), entries[i].dist...)
		sort.Float64s(ds)
		sigmaK := ds[k-1]
		entries[i].density = 1 / (sigmaK + 2)
		entries[i].fitness = entries[i].raw + entries[i].density
	}
	return entries
}

// spea2Nondominated 取个体集合的非支配子集（moDominance 口径）。
func spea2Nondominated(pop []moeaIndividual) []moeaIndividual {
	var out []moeaIndividual
	for i, p := range pop {
		dominated := false
		for j, q := range pop {
			if i != j && moDominance(q.eval, p.eval) > 0 {
				dominated = true
				break
			}
		}
		if !dominated {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = append(out, pop[0]) // 空集防御（理论不可达：自不支配自）
	}
	return out
}

// spea2Environmental 环境选择：非支配（F<1）入档案；超额截断；不足补齐。
func spea2Environmental(entries []spea2Entry, archiveSize int) []moeaIndividual {
	var nd []spea2Entry
	var dominated []spea2Entry
	for _, e := range entries {
		if e.raw < 1 {
			nd = append(nd, e)
		} else {
			dominated = append(dominated, e)
		}
	}

	if len(nd) > archiveSize {
		// 截断：迭代移除"离剩余成员最近"者（SPEA2 原义），用其 dist 矩阵近似
		alive := make([]bool, len(nd))
		for i := range alive {
			alive[i] = true
		}
		for count := len(nd); count > archiveSize; count-- {
			worst := -1
			worstDist := math.Inf(1)
			for i := range nd {
				if !alive[i] {
					continue
				}
				nearest := math.Inf(1)
				for j := range nd {
					if i == j || !alive[j] {
						continue
					}
					// dist 是对合并集（含种群+旧档案）的；截断时按档案内距离近似
					d := distNorm(nd[i].ind, nd[j].ind)
					if d < nearest {
						nearest = d
					}
				}
				if nearest < worstDist {
					worstDist, worst = nearest, i
				}
			}
			if worst < 0 {
				break
			}
			alive[worst] = false
		}
		var trimmed []moeaIndividual
		for i := range nd {
			if alive[i] {
				trimmed = append(trimmed, nd[i].ind)
			}
		}
		return trimmed
	}

	out := make([]moeaIndividual, 0, archiveSize)
	for _, e := range nd {
		out = append(out, e.ind)
	}
	// 不足额：按适应度从小到大（最优在前）补支配解
	sort.SliceStable(dominated, func(a, b int) bool { return dominated[a].fitness < dominated[b].fitness })
	for _, e := range dominated {
		if len(out) >= archiveSize {
			break
		}
		out = append(out, e.ind)
	}
	return out
}

// distNorm 两个体归一化坐标的欧氏距离（截断用；归一化取 [0,1] 域内极值近似）。
func distNorm(a, b moeaIndividual) float64 {
	// norm 已在 spea2Fitness 前由 moeaNormalize(combined) 填好
	s := 0.0
	for k := 0; k < 4; k++ {
		d := a.norm[k] - b.norm[k]
		s += d * d
	}
	return math.Sqrt(s)
}

// spea2Tournament 档案内二元锦标赛（SPEA2 适应度小者胜）。
func spea2Tournament(archive []moeaIndividual, sc *SolveContext) int {
	a := sc.RandInt(len(archive))
	b := sc.RandInt(len(archive))
	// 档案无 fitness 字段：按 ε 分数替代锦标赛判据（档案内多为非支配解，
	// 分数为主要目标法视角的次级排序；与 SPEA2 从档案选种语义兼容）
	if archive[a].eval.scoreOrRawCost(sc.ObjCtx.weighted) <= archive[b].eval.scoreOrRawCost(sc.ObjCtx.weighted) {
		return a
	}
	return b
}
