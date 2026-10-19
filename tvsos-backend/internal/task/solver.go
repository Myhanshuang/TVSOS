package task

// ============================================================================
// 求解器插件体系（可更换的调度优化算法）——指标体系文档 §2.6。
//
// 所有派单求解算法（SA/NSGA-II/NSGA-III/MOEA-D/SPEA2/MOPSO/贪心NF）实现统一的
// Solver 接口并注册进 solverRegistry；实验参数 [algorithm].solver 选择任意算法，
// 算法独有超参经 TOML 平铺键（population/generations/ref_divisions/...）传入。
//
// 统一约定（对所有插件一致，保证跨算法可比）：
//  1. 个体编码：[]int（运单下标 → 车辆下标），与既有 SA/评估器共用；
//  2. 评价口径：evaluateState（多目标向量 + ε-约束标量化分数），所有算法同一打分；
//  3. 终选口径：ε-约束标量化分数（主要目标法语义，可行优先+主目标最优），
//     种群/档案算法最终候选集交由派发器统一 ε-终选，不各自为政；
//  4. 随机源：实验种子 RNG（rt.Draw/DrawIntn），单线程确定性 ⇒ 同种子逐 tick 复现；
//  5. 收敛记录：RecordPoint（SA=温度，种群算法=代数），进同一 ConvergencePoint 曲线；
//  6. weighted 旧标量化模式仅 SA/NF 支持（种群多目标算法依赖目标向量），
//     派发器自动回落 SA 并记日志。
// ============================================================================

import (
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/model"
	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
	"go.uber.org/zap"
)

// SolveContext 一次批次求解的完整输入快照（求解期只读）。
type SolveContext struct {
	RT             *ExperimentRuntime
	ObjCtx         *objectiveContext
	Shipments      []*model.Shipment
	Vehicles       []*model.Vehicle
	Stats          *Statistics
	CargoCache     map[uint]*model.Cargo
	ProjectedLoads map[uint]int
	Pois           map[uint]*model.Poi
	Now            time.Time

	Initial    []int          // 共享初始解（随机可行化，所有算法同一起点）
	InitialEval stateEvaluation
}

// Evaluate 评价一个候选指派（统一走 evaluateState，目标口径与 M/C1/A3 一致）。
func (sc *SolveContext) Evaluate(assigned []int) stateEvaluation {
	return evaluateState(sc.ObjCtx, assigned, sc.Shipments, sc.Vehicles, sc.Stats,
		sc.CargoCache, sc.ProjectedLoads, sc.Pois, sc.Now)
}

// RecordPoint 记录一个收敛采样点：step 为 SA 温度或种群算法代数；
// best 为当前 ε 分数最优解；frontSize 为当前非支配解规模（档案/前沿）。
func (sc *SolveContext) RecordPoint(step float64, best stateEvaluation, frontSize int) {
	sc.RT.RecordConvergence(ConvergencePoint{
		Batch:         sc.RT.Clock.Tick(),
		Temp:          step,
		BestCost:      best.scoreOrRawCost(sc.ObjCtx.weighted),
		BestPrimary:   primaryRawValue(sc.ObjCtx, best.Vector),
		BestViolation: best.Violation,
		ParetoSize:    frontSize,
	})
}

// Rand 种子随机源（严格复现）。
func (sc *SolveContext) Rand() float64 { return sc.RT.Draw() }

// RandInt [0,n) 种子随机整数。
func (sc *SolveContext) RandInt(n int) int { return sc.RT.DrawIntn(n) }

// RandomVehicleFor 可行偏好取样：为运单 i 随机选车，优先可行车（至多尝试 tries 次），
// 不可行兜底强指（由评价的硬可行性惩罚淘汰）。种群算法初始化/变异共用，
// 保证初始种群大部分可行而不需要复杂修复算子。
func (sc *SolveContext) RandomVehicleFor(i int, tries int) int {
	ship := sc.Shipments[i]
	cargo := getCargoByCache(sc.CargoCache, ship.CargoId)
	vIdx := sc.RandInt(len(sc.Vehicles))
	for t := 0; t < tries; t++ {
		cand := sc.RandInt(len(sc.Vehicles))
		if isVehicleFeasible(sc.Vehicles[cand], ship, cargo, sc.ProjectedLoads[sc.Vehicles[cand].Id]) {
			return cand
		}
		if t == 0 {
			vIdx = cand // 保底记一个
		}
	}
	return vIdx
}

// RandomAssignment 随机可行化个体（逐单取样，同 SA 初始解语义）。
func (sc *SolveContext) RandomAssignment() []int {
	assigned := make([]int, len(sc.Shipments))
	for i := range sc.Shipments {
		assigned[i] = sc.RandomVehicleFor(i, 10)
	}
	return assigned
}

// Params 求解器超参（实验参数原样透传，各算法自行回落默认）。
func (sc *SolveContext) Params() sandboxpkg.AlgorithmParams { return sc.RT.Params.Algorithm }

// SolverCandidate 求解器产出的候选解。
type SolverCandidate struct {
	Assigned []int
	Eval     stateEvaluation
}

// SolveOutput 求解器输出：
// Candidates = 终代非支配/档案候选集（交派发器统一 ε-终选与帕累托档案合并）；
// Best = 运行全程 ε 分数最优解（派发器与档案最优比较取终解）。
type SolveOutput struct {
	Candidates []SolverCandidate
	Best       SolverCandidate
	HasBest    bool
}

// Solver 求解器插件接口。实现方只关心"从 sc 初始解出发搜索"，不负责：
// 初始解生成/尺度标定/收敛统计/ε-终选/结果映射（均为派发器统一职责）。
type Solver interface {
	Name() string      // 代号（与 [algorithm].solver 匹配）
	DisplayName() string
	Solve(sc *SolveContext) SolveOutput
}

// solverRegistry 已注册求解器（init 注册，实验参数按代号选取）。
var solverRegistry = map[string]Solver{}

// RegisterSolver 注册求解器插件（重复注册后者覆盖）。
func RegisterSolver(s Solver) { solverRegistry[s.Name()] = s }

// GetSolver 按代号取求解器；未知代号回落 SA。
func GetSolver(name string) Solver {
	if s, ok := solverRegistry[name]; ok {
		return s
	}
	return solverRegistry[sandboxpkg.SolverSA]
}

// SolverDisplayNames 全部已注册求解器的代号→名称映射（API/文档用）。
func SolverDisplayNames() map[string]string {
	out := make(map[string]string, len(solverRegistry))
	for k, v := range solverRegistry {
		out[k] = v.DisplayName()
	}
	return out
}

// solveDispatch 派单求解统一入口（插件派发器）：
// 构建 SolveContext（共享初始解+尺度标定）→ 按参数选插件求解 →
// 帕累托档案合并候选集 → ε-约束终选 → 多目标运行统计 → 运单→车辆映射。
// objCtx 为批次多目标上下文（nil 时按实验参数现建；M/C1/A3 评估传同一实例）。
func solveDispatch(
	rt *ExperimentRuntime,
	objCtx *objectiveContext,
	shipments []*model.Shipment,
	vehicles []*model.Vehicle,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	projectedLoads map[uint]int,
	pois map[uint]*model.Poi,
) map[uint]*model.Vehicle {
	if len(shipments) == 0 || len(vehicles) == 0 {
		return make(map[uint]*model.Vehicle)
	}
	if objCtx == nil {
		objCtx = newObjectiveContext(rt.Params.Objectives, stats)
	}

	solverName := rt.Params.Algorithm.EffectiveSolver()
	// weighted 旧标量化模式无目标向量，种群多目标算法不支持 → 回落 SA
	if objCtx.weighted && solverName != sandboxpkg.SolverSA && solverName != sandboxpkg.SolverNF {
		if Logger.Logger != nil {
			Logger.Logger.Warn("weighted 目标模式仅支持 sa/nf 求解器，已回落 SA",
				zap.String("solver", solverName))
		}
		solverName = sandboxpkg.SolverSA
	}
	solver := GetSolver(solverName)

	// 共享初始解 + 主目标归一化尺度标定（所有算法同一起点与口径）
	initial := make([]int, len(shipments))
	for i, ship := range shipments {
		cargo := getCargoByCache(cargoCache, ship.CargoId)
		vIdx := rt.DrawIntn(len(vehicles))
		for j := 0; j < 10; j++ {
			if isVehicleFeasible(vehicles[vIdx], ship, cargo, projectedLoads[vehicles[vIdx].Id]) {
				break
			}
			vIdx = rt.DrawIntn(len(vehicles))
		}
		initial[i] = vIdx
	}
	initialEval := evaluateState(objCtx, initial, shipments, vehicles, stats, cargoCache, projectedLoads, pois, rt.Clock.Now())
	objCtx.calibratePrimaryScale(initialEval.Vector)

	sc := &SolveContext{
		RT: rt, ObjCtx: objCtx,
		Shipments: shipments, Vehicles: vehicles, Stats: stats,
		CargoCache: cargoCache, ProjectedLoads: projectedLoads, Pois: pois,
		Now: rt.Clock.Now(),
		Initial: initial, InitialEval: initialEval,
	}

	out := solver.Solve(sc)

	// 帕累托档案合并候选集（不可行/重复由 offer 自行剔除）
	archive := newParetoArchive(objCtx.primary)
	archive.offer(initial, initialEval)
	for _, c := range out.Candidates {
		archive.offer(c.Assigned, c.Eval)
	}

	// ε-约束终选：档案分数最优成员与求解器 Best（及初始兜底）比较取优
	finalAssigned := initial
	finalEval := initialEval
	if out.HasBest {
		if out.Best.Eval.scoreOrRawCost(objCtx.weighted) < finalEval.scoreOrRawCost(objCtx.weighted) {
			finalAssigned, finalEval = out.Best.Assigned, out.Best.Eval
		}
	}
	if !objCtx.weighted {
		if entry, ok := archive.bestByScore(); ok && entry.score < finalEval.scoreOrRawCost(false) {
			finalAssigned = entry.assigned
			finalEval = stateEvaluation{Score: entry.score, Vector: entry.vector, Violation: entry.violation}
		}
	}

	// 实验级多目标运行统计（实时看板/终态结果数据源）
	recordObjectiveBatch(objCtx, finalEval, archive.size(), archive.frontVectors())

	if Logger.Logger != nil {
		Logger.Logger.Debug("求解器完成批次",
			zap.String("solver", solverName), zap.Int("shipments", len(shipments)),
			zap.Int("vehicles", len(vehicles)), zap.Float64("finalScore", finalEval.scoreOrRawCost(objCtx.weighted)),
			zap.Int("paretoFront", archive.size()))
	}

	result := make(map[uint]*model.Vehicle)
	for i, vIdx := range finalAssigned {
		result[shipments[i].Id] = vehicles[vIdx]
	}
	return result
}

// assignBySimulatedAnnealing 兼容包装：旧入口转插件派发器（默认/显式 sa 均走原路径）。
func assignBySimulatedAnnealing(
	rt *ExperimentRuntime,
	objCtx *objectiveContext,
	shipments []*model.Shipment,
	vehicles []*model.Vehicle,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	projectedLoads map[uint]int,
	pois map[uint]*model.Poi,
) map[uint]*model.Vehicle {
	return solveDispatch(rt, objCtx, shipments, vehicles, stats, cargoCache, projectedLoads, pois)
}
