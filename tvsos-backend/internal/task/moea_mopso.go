package task

// ============================================================================
// MOPSO 求解器插件（Coello et al. 2004 多目标粒子群，离散指派编码）。
//
// 粒子位置 = 指派向量 []int；速度 = 每运单对全体车辆的偏好向量（连续值）。
// 速度更新（对基因 i、车辆 j）：
//   v[i][j] ← w·v[i][j] + c1·r1·(1[j=pbest_i] − 1[j=x_i]) + c2·r2·(1[j=gbest_i] − 1[j=x_i])
// 位置更新：对每运单按 softmax(v[i]) 采样车辆（探索性离散化），
// 不可行采样做有限重试（可行偏好）。全局档案（外部非支配集，容量淘汰
// 按 ε 分数最差者出局）作为 leader 池，每代每粒子随机取一档案成员为
// gbest；pbest 按 ε-约束支配更新（支配或同代不可比但分数更优时替换）。
// ============================================================================

import (
	"math"
	"sort"

	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

type mopsoSolver struct{}

func init() { RegisterSolver(mopsoSolver{}) }

func (mopsoSolver) Name() string        { return sandboxpkg.SolverMOPSO }
func (mopsoSolver) DisplayName() string { return "MOPSO（多目标粒子群 + 外部档案）" }

// mopsoParticle 粒子：位置/速度/个体最优。
type mopsoParticle struct {
	x     []int
	v     [][]float64
	pbest []int
	pEval stateEvaluation
	eval  stateEvaluation
}

func (mopsoSolver) Solve(sc *SolveContext) SolveOutput {
	p := sc.Params()
	archiveSize := p.ArchiveSize
	if archiveSize <= 0 {
		archiveSize = 30
	}
	d := resolveMoeaDefaults(p, nsgaDefaultPop)
	w := p.InertiaWeight
	if w < 0 || w >= 1 {
		w = 0.4
	}
	c1 := p.CognitiveCoef
	if c1 <= 0 {
		c1 = 1.5
	}
	c2 := p.SocialCoef
	if c2 <= 0 {
		c2 = 1.5
	}

	nGenes := len(sc.Shipments)
	nVehicles := len(sc.Vehicles)

	// 初始粒子群（第 0 粒子=共享初始解）与外部档案
	particles := make([]mopsoParticle, 0, d.Population)
	first := &mopsoParticle{
		x:     append([]int(nil), sc.Initial...),
		pbest: append([]int(nil), sc.Initial...),
	}
	first.eval = sc.InitialEval
	first.pEval = sc.InitialEval
	first.v = mopsoInitVelocity(nGenes, nVehicles, sc)
	particles = append(particles, *first)
	for len(particles) < d.Population {
		x := sc.RandomAssignment()
		ev := sc.Evaluate(x)
		particles = append(particles, mopsoParticle{
			x: x, pbest: append([]int(nil), x...), pEval: ev, eval: ev,
			v: mopsoInitVelocity(nGenes, nVehicles, sc),
		})
	}

	archive := []moeaIndividual{}
	archive = mopsoArchiveUpdate(archive, append([]int(nil), sc.Initial...), sc.InitialEval, archiveSize)
	for _, pt := range particles {
		archive = mopsoArchiveUpdate(archive, append([]int(nil), pt.x...), pt.eval, archiveSize)
	}

	best := SolverCandidate{}
	hasBest := false
	updateRunningBest(sc, &best, &hasBest, sc.Initial, sc.InitialEval)
	for _, pt := range particles {
		updateRunningBest(sc, &best, &hasBest, pt.x, pt.eval)
	}

	for gen := 1; gen <= d.Generations; gen++ {
		for pi := range particles {
			pt := &particles[pi]
			// leader：档案随机成员（经典 MOPSO 从非支配档案取 gbest）
			leader := archive[sc.RandInt(len(archive))].assigned

			// 速度 + 位置更新（逐基因）
			nx := make([]int, nGenes)
			r1, r2 := sc.Rand(), sc.Rand()
			for i := 0; i < nGenes; i++ {
				for j := 0; j < nVehicles; j++ {
					own := 0.0
					if j == pt.x[i] {
						own = 1
					}
					pb := 0.0
					if j == pt.pbest[i] {
						pb = 1
					}
					gb := 0.0
					if leader[i] == j {
						gb = 1
					}
					pt.v[i][j] = w*pt.v[i][j] + c1*r1*(pb-own) + c2*r2*(gb-own)
				}
				nx[i] = mopsoSampleGene(sc, pt.v[i], i)
			}
			pt.x = nx
			pt.eval = sc.Evaluate(nx)

			// pbest 更新：ε-约束支配；互不支配但 ε 分数更优亦替换（历史最优化）
			if moDominance(pt.eval, pt.pEval) > 0 ||
				(moDominance(pt.eval, pt.pEval) == 0 && pt.eval.scoreOrRawCost(sc.ObjCtx.weighted) < pt.pEval.scoreOrRawCost(sc.ObjCtx.weighted)) {
				pt.pbest = append([]int(nil), pt.x...)
				pt.pEval = pt.eval
			}
			updateRunningBest(sc, &best, &hasBest, pt.x, pt.eval)
			archive = mopsoArchiveUpdate(archive, append([]int(nil), pt.x...), pt.eval, archiveSize)
		}
		sc.RecordPoint(float64(gen), best.Eval, len(archive))
	}

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

// mopsoInitVelocity 初始速度：均匀小幅随机值 ∈ [-0.1, 0.1]。
func mopsoInitVelocity(nGenes, nVehicles int, sc *SolveContext) [][]float64 {
	v := make([][]float64, nGenes)
	for i := range v {
		v[i] = make([]float64, nVehicles)
		for j := range v[i] {
			v[i][j] = (sc.Rand() - 0.5) * 0.2
		}
	}
	return v
}

// mopsoSampleGene softmax 采样一基因的车辆（数值稳定：减最大值），
// 不可行采样最多重试 3 次后保底（硬可行性由评价惩罚淘汰）。
func mopsoSampleGene(sc *SolveContext, v []float64, gene int) int {
	pick := func() int {
		maxV := v[0]
		for _, x := range v[1:] {
			if x > maxV {
				maxV = x
			}
		}
		var exps []float64
		sum := 0.0
		for _, x := range v {
			e := math.Exp(x - maxV)
			exps = append(exps, e)
			sum += e
		}
		r := sc.Rand() * sum
		acc := 0.0
		for j, e := range exps {
			acc += e
			if r <= acc {
				return j
			}
		}
		return len(exps) - 1
	}
	for t := 0; t < 3; t++ {
		j := pick()
		ship := sc.Shipments[gene]
		cargo := getCargoByCache(sc.CargoCache, ship.CargoId)
		if isVehicleFeasible(sc.Vehicles[j], ship, cargo, sc.ProjectedLoads[sc.Vehicles[j].Id]) {
			return j
		}
	}
	return pick()
}

// mopsoArchiveUpdate 外部档案更新：非支配插入 + 支配剔除 + 容量淘汰（ε 分数最差出局）。
func mopsoArchiveUpdate(archive []moeaIndividual, assigned []int, ev stateEvaluation, cap int) []moeaIndividual {
	// 被档案任一成员支配 → 不入档
	for _, a := range archive {
		if moDominance(a.eval, ev) > 0 {
			return archive
		}
	}
	// 剔除被新解支配的成员
	kept := archive[:0]
	for _, a := range archive {
		if moDominance(ev, a.eval) > 0 {
			continue
		}
		kept = append(kept, a)
	}
	kept = append(kept, moeaIndividual{assigned: assigned, eval: ev, obj: minimizeObjectives(ev.Vector)})
	archive = kept

	// 容量淘汰：ε 约束分数最差者出局（保留对终选最有价值的成员）
	if len(archive) > cap {
		moeaNormalize(archive)
		sort.SliceStable(archive, func(a, b int) bool {
			return archive[a].eval.Score < archive[b].eval.Score
		})
		archive = archive[:cap]
	}
	return archive
}
