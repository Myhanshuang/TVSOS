package task

import (
	"math"
	"math/rand"
	"time"

	"github.com/kiritosuki/mover/internal/model"
)

const (
	SAInitialTemp         = 1000.0
	SACoolingRate         = 0.95
	SAMinTemp             = 0.1
	SAItersPerTemp        = 50
	SADetourKmPerTonLimit = 50.0 // 如果绕路距离(km) / 重量(ton) 大于此阈值，施加重罚
	SAPenaltyInf          = 1e9
)

func assignBySimulatedAnnealing(
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

	rand.Seed(time.Now().UnixNano())

	// 初始解: 使用类似贪心的随机初始分配
	currentAssigned := make([]int, len(shipments)) // shipment_index -> vehicle_index
	for i, ship := range shipments {
		cargo := getCargoByCache(cargoCache, ship.CargoId)
		// 随机找一个可行车辆
		for j := 0; j < 10; j++ {
			vIdx := rand.Intn(len(vehicles))
			if isVehicleFeasible(vehicles[vIdx], ship, cargo, projectedLoads[vehicles[vIdx].Id]) {
				currentAssigned[i] = vIdx
				break
			}
			currentAssigned[i] = vIdx // 如果都不满足，先强行放进去，靠Cost惩罚剔除
		}
	}

	currentCost := evaluateStateCost(currentAssigned, shipments, vehicles, stats, cargoCache, projectedLoads, pois)
	bestAssigned := make([]int, len(shipments))
	copy(bestAssigned, currentAssigned)
	bestCost := currentCost

	temp := SAInitialTemp
	for temp > SAMinTemp {
		for i := 0; i < SAItersPerTemp; i++ {
			neighborDesc, nextAssigned := generateSANeighbor(currentAssigned, len(vehicles))
			nextCost := evaluateStateCost(nextAssigned, shipments, vehicles, stats, cargoCache, projectedLoads, pois)

			if nextCost < currentCost || math.Exp((currentCost-nextCost)/temp) > rand.Float64() {
				currentAssigned = nextAssigned
				currentCost = nextCost
				if currentCost < bestCost {
					copy(bestAssigned, currentAssigned)
					bestCost = currentCost
				}
			}
			_ = neighborDesc // hold
		}
		temp *= SACoolingRate
	}

	result := make(map[uint]*model.Vehicle)
	for i, vIdx := range bestAssigned {
		result[shipments[i].Id] = vehicles[vIdx]
	}
	return result
}

func generateSANeighbor(current []int, numVehicles int) (string, []int) {
	next := make([]int, len(current))
	copy(next, current)

	if len(current) > 1 && rand.Float32() < 0.5 {
		// Swap
		idx1 := rand.Intn(len(current))
		idx2 := rand.Intn(len(current))
		if next[idx1] != next[idx2] {
			next[idx1], next[idx2] = next[idx2], next[idx1]
			return "swap", next
		}
	}
	// Move
	idx := rand.Intn(len(current))
	vIdx := rand.Intn(numVehicles)
	next[idx] = vIdx
	return "move", next
}

func evaluateStateCost(
	assigned []int,
	shipments []*model.Shipment,
	vehicles []*model.Vehicle,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	projectedLoads map[uint]int,
	pois map[uint]*model.Poi,
) float64 {
	totalCost := 0.0

	// 按车辆汇总其分配的任务
	vTasks := make([][]*model.Shipment, len(vehicles))
	for i, vIdx := range assigned {
		vTasks[vIdx] = append(vTasks[vIdx], shipments[i])
	}

	for vIdx, tasks := range vTasks {
		if len(tasks) == 0 {
			continue
		}
		v := vehicles[vIdx]
		vCost, feasible := evaluateVehicleTasks(v, tasks, stats, cargoCache, projectedLoads[v.Id], pois)
		if !feasible {
			totalCost += SAPenaltyInf
		}
		totalCost += vCost
	}

	return totalCost
}

func evaluateVehicleTasks(
	v *model.Vehicle,
	tasks []*model.Shipment,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	initialProjectedLoad int,
	pois map[uint]*model.Poi,
) (float64, bool) {
	vCost := 0.0
	currentProjected := initialProjectedLoad

	for _, task := range tasks {
		cargo := getCargoByCache(cargoCache, task.CargoId)
		if !isVehicleFeasible(v, task, cargo, currentProjected) {
			return 0, false
		}
		currentProjected += (cargo.Weight * task.Count)

		// 累加基础贪婪Cost分值
		startPoi := pois[task.StartPoiId]
		endPoi := pois[task.EndPoiId]
		cost := CalculateCost(v, task, stats, cargo, startPoi, endPoi)
		vCost += cost
	}

	// 计算 LIFO 最优拼载路线距离
	minDist, valid := findMinLIFORouteDistance(v.Lon, v.Lat, tasks, pois)
	if !valid {
		return 0, false // 没有任何合法的LIFO序列组合
	}

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
		}
	}

	// 把 LIFO 实际找出的最短作业总距离的影响加进去
	vCost += (minDist / 1000.0) * 10.0 // 将真实合并距离作为影响因子

	return vCost, true
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
