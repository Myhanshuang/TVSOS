package task

import (
	"math"
	"time"

	"github.com/kiritosuki/mover/internal/model"
	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

const (
	SAInitialTemp         = 1000.0
	SACoolingRate         = 0.95
	SAMinTemp             = 0.1
	SAItersPerTemp        = 50
	SADetourKmPerTonLimit = 50.0 // 如果绕路距离(km) / 重量(ton) 大于此阈值，施加重罚
	SAPenaltyInf          = 1e9
)

// saSolver 模拟退火求解器插件（sa-v3.0-mo）：
// 基于模拟退火完成运单→车辆分配，Metropolis 接受准则跳出局部最优。
// 多目标模式（主要目标法+帕累托考量）：接受准则叠加帕累托支配判定（候选解在
// 4 主目标上支配当前解时无条件接受），全程维护有界帕累托档案；终代档案与
// 全程最优交派发器统一 ε-终选（指标体系文档 §2.5/§2.6）。
type saSolver struct{}

func init() { RegisterSolver(saSolver{}) }

func (saSolver) Name() string        { return sandboxpkg.SolverSA }
func (saSolver) DisplayName() string { return "模拟退火 SA（多目标 ε-约束 + 帕累托）" }

// Solve 执行 SA 搜索：从共享初始解出发按温度调度迭代邻域扰动。
func (saSolver) Solve(sc *SolveContext) SolveOutput {
	rt := sc.RT
	objCtx := sc.ObjCtx

	// SA 超参：实验参数优先，零值回落常量默认
	saCfg := rt.Params.Algorithm
	initialTemp := saCfg.SAInitialTemp
	if initialTemp <= 0 {
		initialTemp = SAInitialTemp
	}
	coolingRate := saCfg.SACoolingRate
	if coolingRate <= 0 || coolingRate >= 1 {
		coolingRate = SACoolingRate
	}
	minTemp := saCfg.SAMinTemp
	if minTemp <= 0 {
		minTemp = SAMinTemp
	}
	itersPerTemp := saCfg.SAItersPerTemp
	if itersPerTemp <= 0 {
		itersPerTemp = SAItersPerTemp
	}

	currentAssigned := append([]int(nil), sc.Initial...)
	currentEval := sc.InitialEval

	bestAssigned := append([]int(nil), sc.Initial...)
	bestEval := sc.InitialEval

	// 帕累托档案（primary 模式生效；weighted 模式不启用）
	archive := newParetoArchive(objCtx.primary)
	if !objCtx.weighted {
		archive.offer(currentAssigned, currentEval)
	}

	temp := initialTemp
	for temp > minTemp {
		for i := 0; i < itersPerTemp; i++ {
			_, nextAssigned := generateSANeighbor(rt, currentAssigned, len(sc.Vehicles))
			nextEval := sc.Evaluate(nextAssigned)

			// 多目标接受准则（帕累托考量）：
			//  1) 候选解在 4 主目标上支配当前解 → 无条件接受（前沿方向自由移动）；
			//  2) 否则按 ε-约束标量化分数走经典 Metropolis（劣解按温度概率探索）。
			dom := 0
			if !objCtx.weighted {
				dom = paretoDominance(nextEval.Vector, currentEval.Vector)
			}
			if dom > 0 || nextEval.scoreOrRawCost(objCtx.weighted) < currentEval.scoreOrRawCost(objCtx.weighted) ||
				math.Exp((currentEval.scoreOrRawCost(objCtx.weighted)-nextEval.scoreOrRawCost(objCtx.weighted))/temp) > rt.Draw() {
				currentAssigned = nextAssigned
				currentEval = nextEval
				if !objCtx.weighted {
					archive.offer(currentAssigned, currentEval)
				}
				if currentEval.scoreOrRawCost(objCtx.weighted) < bestEval.scoreOrRawCost(objCtx.weighted) {
					copy(bestAssigned, currentAssigned)
					bestEval = currentEval
				}
			}
		}
		// 收敛曲线采样（温度步进；primary 模式下 BestCost=ε-约束标量化分数）
		sc.RecordPoint(temp, bestEval, archive.size())
		temp *= coolingRate
	}

	return SolveOutput{
		Candidates: solverCandidatesFromArchive(archive),
		Best:       SolverCandidate{Assigned: bestAssigned, Eval: bestEval},
		HasBest:    true,
	}
}

// solverCandidatesFromArchive 档案成员转求解器候选集（供派发器 ε-终选）。
func solverCandidatesFromArchive(a *paretoArchive) []SolverCandidate {
	if a == nil || len(a.entries) == 0 {
		return nil
	}
	out := make([]SolverCandidate, 0, len(a.entries))
	for _, e := range a.entries {
		out = append(out, SolverCandidate{
			Assigned: e.assigned,
			Eval:     stateEvaluation{Score: e.score, Vector: e.vector, Violation: e.violation},
		})
	}
	return out
}

// primaryRawValue 取主目标原始值（weighted 模式或异常时返回 0）。
func primaryRawValue(ctx *objectiveContext, v ObjectiveVector) float64 {
	if ctx == nil || ctx.weighted {
		return 0
	}
	raw, _ := v.metricValue(ctx.primary)
	if math.IsNaN(raw) || math.IsInf(raw, 0) || raw < 0 {
		return 0
	}
	return raw
}

// generateSANeighbor 邻域动作生成函数：生成模拟退火算法的下一个状态(邻居解)
// 主要通过交换两运单的指派车辆(Swap)或直接修改某一运单的指派车辆(Move)实现状态扰动
func generateSANeighbor(rt *ExperimentRuntime, current []int, numVehicles int) (string, []int) {
	next := make([]int, len(current))
	copy(next, current)

	if len(current) > 1 && rt.Draw() < 0.5 {
		// Swap
		idx1 := rt.DrawIntn(len(current))
		idx2 := rt.DrawIntn(len(current))
		if next[idx1] != next[idx2] {
			next[idx1], next[idx2] = next[idx2], next[idx1]
			return "swap", next
		}
	}
	// Move
	idx := rt.DrawIntn(len(current))
	vIdx := rt.DrawIntn(numVehicles)
	next[idx] = vIdx
	return "move", next
}

// evaluateStateCost 代价评估函数：计算当前全局状态在当前优化目标口径下的标量分数。
// weighted 模式（objCtx 为 nil 或模式为 weighted）= 原始综合成本（含超载/绕路/积压惩罚）；
// primary 模式 = 主要目标法 ε-约束标量化分数（主目标归一项 + 软约束惩罚 + 硬可行性惩罚）。
// M1-M4/C1/A3 评估与本函数共用同一 objCtx，保证参考答案与调度器目标口径一致。
func evaluateStateCost(
	objCtx *objectiveContext,
	assigned []int,
	shipments []*model.Shipment,
	vehicles []*model.Vehicle,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	projectedLoads map[uint]int,
	pois map[uint]*model.Poi,
	now time.Time,
) float64 {
	return evaluateState(objCtx, assigned, shipments, vehicles, stats, cargoCache, projectedLoads, pois, now).scoreOrRawCost(objCtx == nil || objCtx.weighted)
}

// evaluateState 完整评估：一次遍历同时产出原始成本口径、多目标标量化分数与目标向量。
// 在派车场景中，系统Cost包含了超载惩罚、绕跑惩罚、空闲惩罚、未完成运单积压惩罚等多维指标的加权总和。
func evaluateState(
	objCtx *objectiveContext,
	assigned []int,
	shipments []*model.Shipment,
	vehicles []*model.Vehicle,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	projectedLoads map[uint]int,
	pois map[uint]*model.Poi,
	now time.Time,
) stateEvaluation {
	weightedMode := objCtx == nil || objCtx.weighted
	stats = normalizeStats(stats)

	// 按车辆汇总其分配的任务
	vTasks := make([][]*model.Shipment, len(vehicles))
	for i, vIdx := range assigned {
		vTasks[vIdx] = append(vTasks[vIdx], shipments[i])
	}

	totalRawCost := 0.0
	totalCleanCost := 0.0
	infeasible := 0
	detourHits := 0
	projections := make(map[int]*vehicleObjectiveProjection, len(vehicles))

	for vIdx, tasks := range vTasks {
		if len(tasks) == 0 {
			continue
		}
		v := vehicles[vIdx]
		vCost, feasible, hits, proj := evaluateVehicleTasks(v, tasks, stats, cargoCache, projectedLoads[v.Id], pois, now)
		if !feasible {
			infeasible++
			totalRawCost += SAPenaltyInf
		}
		totalRawCost += vCost
		totalCleanCost += vCost - float64(hits)*(SAPenaltyInf/100.0)
		detourHits += hits
		projections[vIdx] = &proj
	}

	eval := stateEvaluation{
		RawCost:     totalRawCost,
		CleanCost:   totalCleanCost,
		Vector:      ObjectiveVector{},
		Violation:   0,
		Infeasible:  infeasible,
		DetourHits:  detourHits,
	}
	if weightedMode {
		// 旧口径：仅综合成本（跳过 O(车队) 的向量构建，行为与旧版完全一致）
		eval.Score = totalRawCost
		return eval
	}

	// 指派后全车队单车装载率（容量>0 车辆，与全局 U2/F1 同口径）
	loadingRates := make([]float64, 0, len(vehicles))
	for vIdx, v := range vehicles {
		if v == nil || v.Capacity <= 0 {
			continue
		}
		load := float64(v.Size + v.ReservedSize)
		if p, ok := projections[vIdx]; ok {
			load = float64(p.LoadKg)
		}
		loadingRates = append(loadingRates, load/float64(v.Capacity))
	}

	vector := buildObjectiveVector(vehicles, projections, loadingRates, totalCleanCost)
	eval.Vector = vector
	eval.Violation = objCtx.violationOf(vector)
	eval.Score = objCtx.score(totalCleanCost, vector, infeasible, detourHits)
	return eval
}

func evaluateVehicleTasks(
	v *model.Vehicle,
	tasks []*model.Shipment,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	initialProjectedLoad int,
	pois map[uint]*model.Poi,
	now time.Time,
) (float64, bool, int, vehicleObjectiveProjection) {
	vCost := 0.0
	currentProjected := initialProjectedLoad
	proj := vehicleObjectiveProjection{}
	detourHits := 0

	speed := 10.0
	if v != nil && v.Speed > 0 {
		speed = v.Speed
	}

	for _, task := range tasks {
		cargo := getCargoByCache(cargoCache, task.CargoId)
		if !isVehicleFeasible(v, task, cargo, currentProjected) {
			return 0, false, 0, proj
		}
		currentProjected += (cargo.Weight * task.Count)

		// 累加基础贪婪Cost分值
		startPoi := pois[task.StartPoiId]
		endPoi := pois[task.EndPoiId]
		cost := CalculateCost(v, task, stats, cargo, startPoi, endPoi, now)
		vCost += cost

		// 多目标投影：响应时间（现有等待+接驾）与载重段/吨公里
		if startPoi != nil && endPoi != nil {
			loadedLeg := haversine(startPoi.Lon, startPoi.Lat, endPoi.Lon, endPoi.Lat)
			proj.LoadedDist += loadedLeg
			proj.TonDistance += float64(cargo.Weight*task.Count) / 1000.0 * loadedLeg
		}
		waitSecs := stats.cargoWaitByShipment[task.Id]
		if waitSecs == 0 && task != nil {
			waitSecs = secondsSince(task.CreateTime, now)
		}
		if v != nil && startPoi != nil {
			timeToStart := haversine(v.Lon, v.Lat, startPoi.Lon, startPoi.Lat) / speed
			proj.ResponseSecs = append(proj.ResponseSecs, waitSecs+timeToStart)
		} else {
			proj.ResponseSecs = append(proj.ResponseSecs, waitSecs)
		}
	}
	proj.LoadKg = currentProjected

	// 计算 LIFO 最优拼载路线距离
	minDist, valid := findMinLIFORouteDistance(v.Lon, v.Lat, tasks, pois)
	if !valid {
		return 0, false, 0, proj // 没有任何合法的LIFO序列组合
	}
	proj.RouteDist = minDist

	// 增量绕路代价分析 (Detour check)
	for i, task := range tasks {
		cargo := getCargoByCache(cargoCache, task.CargoId)
		ton := float64(cargo.Weight*task.Count) / 1000.0
		if ton <= 0 {
			ton = 0.01 // 最小质量防除零
		}

		// 估算不包含该 task 时，该车辆的最优 LIFO 里程
		subTasks := make([]*model.Shipment, 0, len(tasks)-1)
		for j, t := range tasks {
			if i != j {
				subTasks = append(subTasks, t)
			}
		}
		subDist, subValid := findMinLIFORouteDistance(v.Lon, v.Lat, subTasks, pois)
		if !subValid {
			subDist = 0
		}

		detourDistance := (minDist - subDist) / 1000.0 // km
		if detourDistance > 0 && (detourDistance/ton) > SADetourKmPerTonLimit {
			// 惩罚Cost极大增加，代表防止为了这个小单绕远路
			vCost += SAPenaltyInf / 100.0
			detourHits++
		}
	}

	// 把 LIFO 实际找出的最短作业总距离的影响加进去
	vCost += (minDist / 1000.0) * 10.0 // 将真实合并距离作为影响因子

	return vCost, true, detourHits, proj
}

func findMinLIFORouteDistance(startLon float64, startLat float64, tasks []*model.Shipment, pois map[uint]*model.Poi) (float64, bool) {
	if len(tasks) == 0 {
		return 0, true
	}

	// 为了简单快速遍历所有LIFO操作排列，只支持不超过一定数量的任务(防阶乘暴雷)
	if len(tasks) > 8 { // 这里防止递归过深，当大量单时妥协
		return defaultDistanceEstimate(startLon, startLat, tasks, pois), true
	}

	minDist := math.MaxFloat64
	foundValid := false

	// 使用DFS穷举接送货状态机（栈）
	stack := make([]*model.Shipment, 0, len(tasks))
	visitedPickup := make(map[uint]bool)
	visitedDrop := make(map[uint]bool)

	var dfs func(currLon float64, currLat float64, currDist float64, completed int)
	dfs = func(currLon float64, currLat float64, currDist float64, completed int) {
		if currDist >= minDist {
			return
		}
		if completed == len(tasks) {
			if currDist < minDist {
				minDist = currDist
				foundValid = true
			}
			return
		}

		// Option 1: 尝试Pickup所有未Pick的单 (入栈)
		for _, task := range tasks {
			if !visitedPickup[task.Id] {
				poi := pois[task.StartPoiId]
				if poi == nil {
					continue
				}

				visitedPickup[task.Id] = true
				stack = append(stack, task)
				dist := haversine(currLon, currLat, poi.Lon, poi.Lat)

				dfs(poi.Lon, poi.Lat, currDist+dist, completed)

				stack = stack[:len(stack)-1]
				visitedPickup[task.Id] = false
			}
		}

		// Option 2: 尝试Dropoff栈顶的单 (L出栈，即LIFO)
		if len(stack) > 0 {
			topTask := stack[len(stack)-1]
			poi := pois[topTask.EndPoiId]
			if poi != nil {
				visitedDrop[topTask.Id] = true
				stack = stack[:len(stack)-1] // 弹栈
				dist := haversine(currLon, currLat, poi.Lon, poi.Lat)

				dfs(poi.Lon, poi.Lat, currDist+dist, completed+1)

				stack = append(stack, topTask) // 恢复栈
				visitedDrop[topTask.Id] = false
			}
		}
	}

	dfs(startLon, startLat, 0, 0)

	return minDist, foundValid
}

func defaultDistanceEstimate(lon float64, lat float64, tasks []*model.Shipment, pois map[uint]*model.Poi) float64 {
	dist := 0.0
	currLon, currLat := lon, lat
	for _, t := range tasks {
		sp := pois[t.StartPoiId]
		ep := pois[t.EndPoiId]
		if sp != nil && ep != nil {
			dist += haversine(currLon, currLat, sp.Lon, sp.Lat)
			dist += haversine(sp.Lon, sp.Lat, ep.Lon, ep.Lat)
			currLon, currLat = ep.Lon, ep.Lat
		}
	}
	return dist
}
