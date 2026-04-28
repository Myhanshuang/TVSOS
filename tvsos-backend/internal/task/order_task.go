package task

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/realtime"
	"github.com/kiritosuki/mover/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Point struct {
	Lon float64
	Lat float64
}

var (
	lastVehicleIdx = 0
	nfMu           sync.Mutex
)

const candidateVehicleLimit = 5

const (
	schedulerModeHybrid = "hybrid"
	schedulerModeNF     = "nf"

	schedulerFallbackNFFirst = "nf_first"
)

type VehicleCandidate struct {
	Vehicle *model.Vehicle
	Index   int
	NFRank  int
}

type SchedulerOptions struct {
	Mode         string
	CandidateK   int
	CostFallback string
}

// CreateOrderTask 后台守护常驻轮询任务：循环扫描调度休眠装运(Sleeping Shipment)集合。
// 定时去数据库把刚产生、还没指派车的运单拉出，交由模拟退火引擎（或备用的 Next Fit 引擎）批量分配并发车。
func CreateOrderTask() {
	for {
		shipments, err := repository.GetSleepingShipment()
		if err != nil {
			Logger.Logger.Error("查询待创建任务的订单时出错")
			time.Sleep(1 * time.Second)
			continue
		}

		if len(shipments) > 0 {
			doCreateOrderTaskBatch(shipments)
		}

		time.Sleep(constant.OrderTaskCreatingGap * time.Second)
	}
}

// doCreateOrderTaskBatch 全局最核心指派下发逻辑（单批次处理函数）。
// 使用互斥资源防止并发出错。在这里抓取当前全部车辆状态和当前的大盘评估快照 (statsSnapshot) ，
// 将该批次的待分配运单数组(shipments)输入 SA 求解器引擎进行全局代价最小的最优分配。
// 并循环地落实分配结果即创建 OrderTask。最后还会自动检查车斗满载/发车情况并驱动车子发车。
func doCreateOrderTaskBatch(shipments []*model.Shipment) {
	nfMu.Lock()
	defer nfMu.Unlock()

	vehicles, err := repository.ListVehicles(0) // 获取所有车辆以进行 NF
	if err != nil || len(vehicles) == 0 {
		Logger.Logger.Warn("没有可用车辆")
		return
	}

	statsSnapshot, err := CalculateGlobalStats()
	if err != nil {
		Logger.Logger.Warn("统计快照获取失败，将回退为NF首可行", zap.Error(err))
		statsSnapshot = nil
	}

	projectedLoads := make(map[uint]int, len(vehicles))
	for _, v := range vehicles {
		projectedLoads[v.Id] = v.Size + v.ReservedSize
	}

	cargoCache := make(map[uint]*model.Cargo)
	unassigned := make([]*model.Shipment, 0)
	assignedVehicles := make(map[uint]*model.Vehicle)

	// SA 求解订单分配
	pois, _ := repository.ListAllPois()
	poiMap := make(map[uint]*model.Poi)
	for _, p := range pois {
		poiMap[p.Id] = p
	}

	bestAssignment := assignBySimulatedAnnealing(shipments, vehicles, statsSnapshot, cargoCache, projectedLoads, poiMap)

	for _, shipment := range shipments {
		cargo := getCargoByCache(cargoCache, shipment.CargoId)
		if cargo == nil {
			Logger.Logger.Error("查询货物详情失败", zap.Uint("shipmentId", shipment.Id))
			unassigned = append(unassigned, shipment)
			continue
		}

		v := bestAssignment[shipment.Id]
		if v == nil {
			Logger.Logger.Warn("订单无法匹配到合适车辆 (SA)", zap.Uint("shipmentId", shipment.Id))
			unassigned = append(unassigned, shipment)
			continue
		}

		shipmentLoad := cargo.Weight * shipment.Count
		if !assignShipmentToVehicle(shipment, v, shipmentLoad) {
			Logger.Logger.Warn("订单分配失败", zap.Uint("shipmentId", shipment.Id), zap.Uint("vehicleId", v.Id))
			unassigned = append(unassigned, shipment)
			continue
		}

		projectedLoads[v.Id] += shipmentLoad
		assignedVehicles[v.Id] = v
	}

	for vehicleID, vehicle := range assignedVehicles {
		if !isVehicleClosedForBatch(vehicle, projectedLoads[vehicleID], unassigned, cargoCache) {
			continue
		}
		if !startNextPlannedTaskForVehicle(vehicleID) {
			Logger.Logger.Warn("车辆闭合后发车触发失败", zap.Uint("vehicleId", vehicleID))
		}
	}
}

// assignShipmentToVehicle 原子核心落地操作：当订单分配在算法层面敲定车牌后执行的事务处理。
// 新增 OrderTask，同时增加该车的预占预估载重、并原子更新所属 Shipment 状态(为排队待拉走)。
func assignShipmentToVehicle(shipment *model.Shipment, v *model.Vehicle, shipmentLoad int) bool {
	// 创建任务
	newTask := &model.OrderTask{
		ShipmentId: shipment.Id,
		VehicleId:  v.Id,
		Sequential: constant.OrderTaskSequentialPlanned,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	err := repository.CreateOrderTask(newTask)
	if err != nil {
		Logger.Logger.Error("创建任务失败")
		return false
	}

	// 更新订单状态：待创建 -> 已分配待发车
	err = repository.UpdateShipmentStatus(int(shipment.Id), constant.ShipmentStatusQueued, shipment.Status)
	if err != nil {
		Logger.Logger.Error("更新订单状态失败")
	}

	// 标记车辆为可发车 (如果原来是空闲)
	if v.Status == constant.VehicleStatusFree {
		_ = repository.UpdateVehicleStatus(int(v.Id), constant.VehicleStatusReady, constant.VehicleStatusFree)
		v.Status = constant.VehicleStatusReady
	}

	v.ReservedSize += shipmentLoad
	if err = repository.UpdateVehicleReservedSize(int(v.Id), v.ReservedSize); err != nil {
		Logger.Logger.Error("更新车辆预占载重失败", zap.Uint("vehicle", v.Id), zap.Error(err))
		return false
	}

	Logger.Logger.Info("任务分配成功 (NF+Cost)", zap.Uint("shipment", shipment.Id), zap.Uint("vehicle", v.Id))
	return true
}

func collectNFCandidates(
	vehicles []*model.Vehicle,
	projectedLoads map[uint]int,
	shipment *model.Shipment,
	cargo *model.Cargo,
	startIdx int,
	limit int,
) []VehicleCandidate {
	candidates := make([]VehicleCandidate, 0, limit)
	for i := 0; i < len(vehicles) && len(candidates) < limit; i++ {
		currIdx := (startIdx + i) % len(vehicles)
		v := vehicles[currIdx]
		if !isVehicleFeasible(v, shipment, cargo, projectedLoads[v.Id]) {
			continue
		}
		candidates = append(candidates, VehicleCandidate{
			Vehicle: v,
			Index:   currIdx,
			NFRank:  len(candidates),
		})
	}
	return candidates
}

func selectBestCandidateByCost(
	candidates []VehicleCandidate,
	shipment *model.Shipment,
	cargo *model.Cargo,
	startPoi *model.Poi,
	endPoi *model.Poi,
	stats *Statistics,
) VehicleCandidate {
	if len(candidates) == 0 {
		return VehicleCandidate{}
	}
	if stats == nil {
		return candidates[0]
	}

	best := candidates[0]
	bestCost := math.MaxFloat64
	found := false

	for _, c := range candidates {
		cost := CalculateCost(c.Vehicle, shipment, stats, cargo, startPoi, endPoi)
		if math.IsNaN(cost) || math.IsInf(cost, 0) {
			continue
		}
		if !found || cost < bestCost {
			best = c
			bestCost = cost
			found = true
		}
	}

	if !found {
		return candidates[0]
	}
	return best
}

func selectBestCandidate(
	options SchedulerOptions,
	candidates []VehicleCandidate,
	shipment *model.Shipment,
	cargo *model.Cargo,
	startPoi *model.Poi,
	endPoi *model.Poi,
	stats *Statistics,
) VehicleCandidate {
	if len(candidates) == 0 {
		return VehicleCandidate{}
	}

	if options.Mode == schedulerModeNF {
		return candidates[0]
	}

	if stats == nil && options.CostFallback == schedulerFallbackNFFirst {
		return candidates[0]
	}

	return selectBestCandidateByCost(candidates, shipment, cargo, startPoi, endPoi, stats)
}

func isVehicleFeasible(v *model.Vehicle, shipment *model.Shipment, cargo *model.Cargo, projectedSize int) bool {
	if v == nil || shipment == nil || cargo == nil {
		return false
	}

	if !isVehicleTypeCompatible(v.Tybe, cargo.Tybe) {
		return false
	}

	load := cargo.Weight * shipment.Count
	return v.Capacity-projectedSize >= load
}

func isVehicleTypeCompatible(vehicleType int, cargoType int) bool {
	if vehicleType == cargoType {
		return true
	}
	return vehicleType == 2 && cargoType == 1
}

func getCargoByCache(cache map[uint]*model.Cargo, cargoID uint) *model.Cargo {
	if cargo, ok := cache[cargoID]; ok {
		return cargo
	}
	cargo, err := repository.GetCargo(int(cargoID))
	if err != nil {
		return nil
	}
	cache[cargoID] = cargo
	return cargo
}

func isVehicleClosedForBatch(vehicle *model.Vehicle, projectedSize int, remaining []*model.Shipment, cargoCache map[uint]*model.Cargo) bool {
	if vehicle == nil {
		return false
	}
	for _, shipment := range remaining {
		cargo := getCargoByCache(cargoCache, shipment.CargoId)
		if cargo == nil {
			continue
		}
		if isVehicleFeasible(vehicle, shipment, cargo, projectedSize) {
			return false
		}
	}
	return true
}

// startNextPlannedTaskForVehicle 将某辆车最早的Planned任务切到Accepting并下发首段路径。
func startNextPlannedTaskForVehicle(vehicleID uint) bool {
	task, err := repository.GetFirstTaskByVehicleAndSequential(vehicleID, constant.OrderTaskSequentialPlanned)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			Logger.Logger.Error("查询车辆Planned任务失败", zap.Uint("vehicleId", vehicleID), zap.Error(err))
		}
		return false
	}

	vehicle, err := repository.GetVehicle(int(vehicleID))
	if err != nil {
		Logger.Logger.Error("查询车辆失败", zap.Uint("vehicleId", vehicleID), zap.Error(err))
		return false
	}

	shipment, err := repository.GetShipment(int(task.ShipmentId))
	if err != nil {
		Logger.Logger.Error("查询订单失败", zap.Uint("shipmentId", task.ShipmentId), zap.Error(err))
		return false
	}

	startPoi, err := repository.GetPoi(int(shipment.StartPoiId))
	if err != nil {
		Logger.Logger.Error("查询起点POI失败", zap.Uint("shipmentId", task.ShipmentId), zap.Error(err))
		return false
	}

	points := planRouteWithFallback(vehicle.Lon, vehicle.Lat, startPoi.Lon, startPoi.Lat)
	if len(points) == 0 {
		Logger.Logger.Warn("发车路径为空且兜底失败", zap.Uint("shipmentId", task.ShipmentId))
		return false
	}

	err = repository.UpdateTaskSequential(int(task.Id), constant.OrderTaskSequentialAccepting, constant.OrderTaskSequentialPlanned)
	if err != nil {
		Logger.Logger.Error("更新任务阶段为Accepting失败", zap.Uint("taskId", task.Id), zap.Error(err))
		return false
	}

	err = repository.UpdateShipmentStatus(int(task.ShipmentId), constant.ShipmentStatusWaiting, constant.ShipmentStatusQueued)
	if err != nil {
		Logger.Logger.Warn("更新订单状态为Waiting失败", zap.Uint("shipmentId", task.ShipmentId), zap.Error(err))
	}

	if vehicle.Status != constant.VehicleStatusRunning {
		_ = repository.UpdateVehicleStatus(int(vehicleID), constant.VehicleStatusRunning, constant.VehicleStatusReady)
		_ = repository.UpdateVehicleStatus(int(vehicleID), constant.VehicleStatusRunning, constant.VehicleStatusFree)
	}

	meta, saveErr := persistRuntimeRoute(vehicleID, task.ShipmentId, constant.OrderTaskSequentialAccepting, points, 0, vehicle.Lon, vehicle.Lat)
	if saveErr != nil {
		Logger.Logger.Error("写入接单阶段运行态路径失败", zap.Uint("vehicleId", vehicleID), zap.Error(saveErr))
		return false
	}
	realtime.GlobalHub().BroadcastPath(vehicleID, "full_path", buildFullPathPayload(meta, points))

	broadcastVehicleEvent("vehicle_update", vehicleID)

	Logger.Logger.Info("车辆发车成功", zap.Uint("vehicle", vehicleID), zap.Uint("shipment", task.ShipmentId), zap.Uint("task", task.Id))
	return true
}

func interpolatePoints(points []Point, maxDist float64) []Point {
	if len(points) < 2 {
		return points
	}
	var res []Point
	res = append(res, points[0])
	for i := 1; i < len(points); i++ {
		prev := res[len(res)-1]
		curr := points[i]
		dx := curr.Lon - prev.Lon
		dy := curr.Lat - prev.Lat
		distKM := haversineKM(prev.Lon, prev.Lat, curr.Lon, curr.Lat)
		if distKM > maxDist {
			numSegments := int(math.Ceil(distKM / maxDist))
			stepX := dx / float64(numSegments)
			stepY := dy / float64(numSegments)
			for j := 1; j < numSegments; j++ {
				res = append(res, Point{
					Lon: prev.Lon + float64(j)*stepX,
					Lat: prev.Lat + float64(j)*stepY,
				})
			}
		}
		res = append(res, curr)
	}
	return res
}

func planRouteWithFallback(startLon float64, startLat float64, endLon float64, endLat float64) []Point {
	var points []Point
	routeResp, err := PlanRoute(startLon, startLat, endLon, endLat)
	if err == nil {
		points = ExtractPoints(routeResp)
		if len(points) == 0 {
			Logger.Logger.Warn("路径规划返回空路径，使用直连兜底")
		}
	} else {
		Logger.Logger.Warn("路径规划失败，使用直连兜底", zap.Error(err))
	}

	if len(points) == 0 {
		points = make([]Point, 0, 2)
		points = append(points, Point{Lon: startLon, Lat: startLat})
		if startLon != endLon || startLat != endLat {
			points = append(points, Point{Lon: endLon, Lat: endLat})
		}
	}
	// 50m max dist interpolation => 0.05 km
	return interpolatePoints(points, 0.05)
}

func getSchedulerOptions() SchedulerOptions {
	options := SchedulerOptions{
		Mode:         schedulerModeHybrid,
		CandidateK:   candidateVehicleLimit,
		CostFallback: schedulerFallbackNFFirst,
	}

	if config.VP == nil {
		return options
	}

	mode := config.VP.GetString("scheduler.mode")
	if mode == schedulerModeNF || mode == schedulerModeHybrid {
		options.Mode = mode
	}

	k := config.VP.GetInt("scheduler.candidate_k")
	if k > 0 {
		options.CandidateK = k
	}

	fallback := config.VP.GetString("scheduler.cost_fallback")
	if fallback == schedulerFallbackNFFirst {
		options.CostFallback = fallback
	}

	return options
}

// TODO 这部分是AI写的 提取路径规划返回值的路径点 暂时不用动
func ExtractPoints(resp *RouteResponse) []Point {
	var points []Point
	if resp == nil || len(resp.Route.Paths) == 0 {
		return points
	}
	path := resp.Route.Paths[0]
	var last Point
	first := true
	for _, step := range path.Steps {
		pairs := strings.Split(step.Polyline, ";")

		for _, p := range pairs {
			xy := strings.Split(p, ",")
			if len(xy) != 2 {
				continue
			}

			lon, err1 := strconv.ParseFloat(xy[0], 64)
			lat, err2 := strconv.ParseFloat(xy[1], 64)

			if err1 != nil || err2 != nil {
				continue
			}

			curr := Point{Lon: lon, Lat: lat}

			// 去重：和上一个点一样就跳过
			if !first && curr == last {
				continue
			}

			points = append(points, curr)
			last = curr
			first = false
		}
	}

	return points
}
