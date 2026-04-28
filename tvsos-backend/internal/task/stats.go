package task

import (
	"math"
	"sort"
	"time"

	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
)

const (
	defaultVehicleSpeedMPS               = 10.0
	estimatedHandlingSecondsPerTon       = 60.0
	denseVehicleNearestNeighborThreshold = 1500.0
)

// Statistics 调度系统的核心内粗数据统计模块入口结构。
// 包含了所有当前大盘以及计算成本函数所需的中间态指标：
// 如利用率、运能分布、全局等待时间、空驶状况等。这是 SA 调度算法和报表展示的基石。
type Statistics struct {
	VehicleStats       VehicleOverview      // 车辆概况
	UtilizationStats   UtilizationMetrics   // 利用率指标
	VehicleWaitStats   TimeMetrics          // 车辆等待时间指标
	EmptyDistStats     DistanceMetrics      // 空驶/总里程指标
	EmptyTimeStats     TimeMetrics          // 空驶时间指标
	CargoStats         CargoOverviewMetrics // 货物概况
	CargoWaitStats     CargoWaitMetrics     // 货物等待时间指标
	ThroughputStats    ThroughputMetrics    // 吞吐量指标
	TransportWorkStats TransportWorkMetrics // 吨位 * 距离指标
	PendingLossStats   PendingLossMetrics   // 待处理损耗
	TransportTimeStats TimeMetrics          // 运输时间指标
	AdvancedStats      AdvancedMetrics      // 高阶比例
	LoadStats          LoadMetrics          // 装卸/载重指标
	GlobalCost         CostBreakdown        // 基于当前快照的全局 cost
	TotalCost          float64              // GlobalCost.F

	vehicleWaitByVehicle        map[uint]float64
	cargoWaitByShipment         map[uint]float64
	cargoWeightTonsByShipment   map[uint]float64
	vehicleCapacityGapByVehicle map[uint]float64
	vehicleDirLon               map[uint]float64
	vehicleDirLat               map[uint]float64
}

// VehicleOverview 车辆基础情况
type VehicleOverview struct {
	TotalCount           int                        // 车辆总数
	TypeCountMap         map[int]int                // 各类型车辆数量分布 (Tybe -> Count)
	CapacityStats        NumericMetrics             // 车辆运能分布
	CurrentLoadStats     NumericMetrics             // 当前货舱载货分布
	LocationDistribution LocationDistributionMetric // 车辆位置分布
}

// LocationDistributionMetric 车辆位置分布
type LocationDistributionMetric struct {
	CenterLon                  float64
	CenterLat                  float64
	AvgDistanceToCenter        float64
	MaxDistanceToCenter        float64
	AvgNearestNeighborDistance float64
	Pattern                    string // dense / dispersed / unknown
}

// UtilizationMetrics 利用率指标
type UtilizationMetrics struct {
	ByVehicleCount float64 // 按车辆数计算的利用率
	ByCapacity     float64 // 按载重/容量计算的利用率
	TotalCapacity  int     // 总运能
	UsedCapacity   int     // 实际已使用运能
}

// NumericMetrics 通用数值统计量
type NumericMetrics struct {
	Total  float64
	Max    float64
	Min    float64
	Avg    float64
	Median float64
	StdDev float64
}

// TimeMetrics 时间相关统计量
type TimeMetrics struct {
	Total  float64
	Max    float64
	Min    float64
	Avg    float64
	Median float64
	StdDev float64
}

// CargoOverviewMetrics 货物概况
type CargoOverviewMetrics struct {
	TotalCount         int
	AssignedCount      int
	PendingCount       int
	TotalDemandTons    float64
	AssignedDemandTons float64
	PendingDemandTons  float64
}

// CargoWaitMetrics 货物等待统计
type CargoWaitMetrics struct {
	TimeMetrics
	WeightedTotal float64 // 所有货物(吨 * 等待时间)
}

// ThroughputMetrics 吞吐量指标
type ThroughputMetrics struct {
	TotalTransportedTons    float64
	InTransitTons           float64
	CompletedTons           float64
	ReceiverEfficiency      NumericMetrics
	ReceiverEfficiencyByPoi map[uint]float64
}

// TransportWorkMetrics 运输能力体量/吨距离
type TransportWorkMetrics struct {
	TotalTonDistance    float64
	ShipmentTonDistance NumericMetrics
}

// PendingLossMetrics 待处理损耗
type PendingLossMetrics struct {
	TotalLoss        float64
	FactoryLoss      NumericMetrics
	FactoryLossByPoi map[uint]float64
}

// DistanceMetrics 距离/空驶指标
type DistanceMetrics struct {
	TotalEmpty     float64
	TotalLoaded    float64
	TotalDistance  float64
	MaxEmpty       float64
	MinEmpty       float64
	AvgEmpty       float64
	MedianEmpty    float64
	StdDevEmpty    float64
	EmptyFullRatio float64 // 空驶/重载
}

// AdvancedMetrics 高阶比例指标
type AdvancedMetrics struct {
	EmptyLoadedRatio   float64
	WaitTransportRatio float64
}

// LoadMetrics 装卸相关指标
type LoadMetrics struct {
	TimeMetrics
	Estimated bool // 当前无装卸事件日志，只能按重量比例估算
}

// CalculateGlobalStats 采集数据库并计算当前系统全局的所有统计指标。
// 包含了运能、在线率、总距离、等待延迟、运单分发比例等，
// 一次性构建完整的系统运行状况快照，是生成大屏看板数据和 Cost 动态算子的核心入口。
func CalculateGlobalStats() (*Statistics, error) {
	vehicles, err := repository.ListVehicles(0)
	if err != nil {
		return nil, err
	}
	tasks, err := repository.GetAllTasks()
	if err != nil {
		return nil, err
	}
	shipments, err := repository.ListShipments()
	if err != nil {
		return nil, err
	}
	cargos, err := repository.ListCargos()
	if err != nil {
		return nil, err
	}
	pois, err := repository.ListAllPois()
	if err != nil {
		return nil, err
	}

	cargoMap := make(map[uint]*model.Cargo, len(cargos))
	for _, cargo := range cargos {
		cargoMap[cargo.Id] = cargo
	}
	poiMap := make(map[uint]*model.Poi, len(pois))
	for _, poi := range pois {
		poiMap[poi.Id] = poi
	}

	return buildStatistics(vehicles, tasks, shipments, cargoMap, poiMap, time.Now()), nil
}

// buildStatistics 将入参中传递的内存字典组装映射为 `Statistics` 通用大结构。
// 分步骤执行所有的子维度指标汇总逻辑。
func buildStatistics(
	vehicles []*model.Vehicle,
	tasks []*model.OrderTask,
	shipments []*model.Shipment,
	cargos map[uint]*model.Cargo,
	pois map[uint]*model.Poi,
	now time.Time,
) *Statistics {
	stats := &Statistics{
		vehicleWaitByVehicle:        make(map[uint]float64),
		cargoWaitByShipment:         make(map[uint]float64),
		cargoWeightTonsByShipment:   make(map[uint]float64),
		vehicleCapacityGapByVehicle: make(map[uint]float64),
		vehicleDirLon:               make(map[uint]float64),
		vehicleDirLat:               make(map[uint]float64),
	}

	stats.calculateVehicleOverview(vehicles)
	stats.calculateUtilization(vehicles)
	stats.calculateTimeStats(vehicles, tasks, shipments, cargos, now)
	stats.calculateCargoOverview(shipments, tasks, cargos)
	stats.calculateDistanceStats(vehicles, tasks, shipments, pois)
	stats.calculateVehicleDirection(vehicles, tasks, shipments, pois)
	stats.calculateThroughputStats(vehicles, tasks, shipments, cargos, pois, now)
	stats.calculatePendingLossStats(shipments, cargos)
	stats.calculateLoadStats(shipments, cargos)
	stats.calculateAdvancedStats()
	stats.calculateSnapshotCost()
	return stats
}

// calculateVehicleOverview 归纳车辆的基础全局大盘。
// 包括车辆总数、不同类型的映射数量以及全局的车舱空载/重载分布与位置散点等。
func (s *Statistics) calculateVehicleOverview(vehicles []*model.Vehicle) {
	s.VehicleStats.TotalCount = len(vehicles)
	s.VehicleStats.TypeCountMap = make(map[int]int)

	var capacities []float64
	var loads []float64
	for _, v := range vehicles {
		s.VehicleStats.TypeCountMap[v.Tybe]++
		capacities = append(capacities, float64(v.Capacity))
		loads = append(loads, float64(v.Size))
	}

	s.VehicleStats.CapacityStats = newNumericMetrics(capacities)
	s.VehicleStats.CurrentLoadStats = newNumericMetrics(loads)
	s.VehicleStats.LocationDistribution = computeLocationDistribution(vehicles)
}

// calculateUtilization 汇总整个车队系统维度的调度运力利用率表现。
// 计算在所有总运能当中的已占用部分，并得到总体利用率百分比。
func (s *Statistics) calculateUtilization(vehicles []*model.Vehicle) {
	if len(vehicles) == 0 {
		return
	}
	runningCount := 0
	totalCap := 0
	usedCap := 0
	for _, v := range vehicles {
		totalCap += v.Capacity
		usedCap += v.Size
		s.vehicleCapacityGapByVehicle[v.Id] = float64(v.Capacity - v.Size)
		if v.Status == constant.VehicleStatusRunning {
			runningCount++
		}
	}
	s.UtilizationStats.TotalCapacity = totalCap
	s.UtilizationStats.UsedCapacity = usedCap
	s.UtilizationStats.ByVehicleCount = float64(runningCount) / float64(len(vehicles))
	if totalCap > 0 {
		s.UtilizationStats.ByCapacity = float64(usedCap) / float64(totalCap)
	}
}

// calculateTimeStats 采集计算车辆与运单货物的各项时间延迟和等待开销。
// 分析所有处于空闲状态的车辆挂机时长及所有待处理/排队中运单货物的滞留时长，以此用作全局效率惩罚函数的参考。
func (s *Statistics) calculateTimeStats(
	vehicles []*model.Vehicle,
	tasks []*model.OrderTask,
	shipments []*model.Shipment,
	cargos map[uint]*model.Cargo,
	now time.Time,
) {
	var vehicleWaits []float64
	for _, v := range vehicles {
		if v.Status != constant.VehicleStatusFree {
			continue
		}
		wait := secondsSince(v.UpdateTime, now)
		s.vehicleWaitByVehicle[v.Id] = wait
		vehicleWaits = append(vehicleWaits, wait)
	}
	s.VehicleWaitStats = newTimeMetrics(vehicleWaits)

	var cargoWaits []float64
	var weightedTotal float64
	for _, shipment := range shipments {
		if shipment.Status != constant.ShipmentStatusSleeping &&
			shipment.Status != constant.ShipmentStatusQueued &&
			shipment.Status != constant.ShipmentStatusWaiting {
			continue
		}
		wait := secondsSince(shipment.CreateTime, now)
		s.cargoWaitByShipment[shipment.Id] = wait
		cargoWaits = append(cargoWaits, wait)

		weightTons := shipmentWeightTons(shipment, cargos[shipment.CargoId])
		s.cargoWeightTonsByShipment[shipment.Id] = weightTons
		weightedTotal += weightTons * wait
	}
	s.CargoWaitStats = CargoWaitMetrics{
		TimeMetrics:   newTimeMetrics(cargoWaits),
		WeightedTotal: weightedTotal,
	}

	var transportTimes []float64
	for _, task := range tasks {
		if task.Sequential != constant.OrderTaskSequentialTransporting {
			continue
		}
		transportTimes = append(transportTimes, secondsSince(task.UpdateTime, now))
	}
	s.TransportTimeStats = newTimeMetrics(transportTimes)
}

// calculateCargoOverview 计算货物需求与分配情况
func (s *Statistics) calculateCargoOverview(
	shipments []*model.Shipment,
	tasks []*model.OrderTask,
	cargos map[uint]*model.Cargo,
) {
	taskByShipment := make(map[uint]*model.OrderTask, len(tasks))
	for _, task := range tasks {
		taskByShipment[task.ShipmentId] = task
	}

	for _, shipment := range shipments {
		weightTons := shipmentWeightTons(shipment, cargos[shipment.CargoId])
		s.CargoStats.TotalCount++
		s.CargoStats.TotalDemandTons += weightTons

		if shipment.Status == constant.ShipmentStatusSleeping {
			s.CargoStats.PendingCount++
			s.CargoStats.PendingDemandTons += weightTons
		}

		if shipmentHasAssignedVehicle(shipment, taskByShipment[shipment.Id]) {
			s.CargoStats.AssignedCount++
			s.CargoStats.AssignedDemandTons += weightTons
		}
	}
}

// calculateDistanceStats 基于任务快照估算空驶和重载里程/时间
func (s *Statistics) calculateDistanceStats(
	vehicles []*model.Vehicle,
	tasks []*model.OrderTask,
	shipments []*model.Shipment,
	pois map[uint]*model.Poi,
) {
	if len(pois) == 0 {
		return
	}

	vehicleMap := make(map[uint]*model.Vehicle, len(vehicles))
	for _, vehicle := range vehicles {
		vehicleMap[vehicle.Id] = vehicle
	}
	shipmentMap := make(map[uint]*model.Shipment, len(shipments))
	for _, shipment := range shipments {
		shipmentMap[shipment.Id] = shipment
	}

	var emptyDists []float64
	var emptyTimes []float64
	var totalEmpty, totalLoaded float64
	for _, task := range tasks {
		shipment := shipmentMap[task.ShipmentId]
		if shipment == nil {
			continue
		}

		switch task.Sequential {
		case constant.OrderTaskSequentialAccepting:
			vehicle := vehicleMap[task.VehicleId]
			startPoi := pois[shipment.StartPoiId]
			if vehicle == nil || startPoi == nil {
				continue
			}
			dist := haversine(vehicle.Lon, vehicle.Lat, startPoi.Lon, startPoi.Lat)
			emptyDists = append(emptyDists, dist)
			emptyTimes = append(emptyTimes, dist/effectiveVehicleSpeed(vehicle))
			totalEmpty += dist
		case constant.OrderTaskSequentialTransporting, constant.OrderTaskSequentialFinish:
			startPoi := pois[shipment.StartPoiId]
			endPoi := pois[shipment.EndPoiId]
			if startPoi == nil || endPoi == nil {
				continue
			}
			totalLoaded += haversine(startPoi.Lon, startPoi.Lat, endPoi.Lon, endPoi.Lat)
		}
	}

	emptyMetrics := computeBasicMetrics(emptyDists)
	s.EmptyDistStats = DistanceMetrics{
		TotalEmpty:    totalEmpty,
		TotalLoaded:   totalLoaded,
		TotalDistance: totalEmpty + totalLoaded,
		MaxEmpty:      emptyMetrics.Max,
		MinEmpty:      emptyMetrics.Min,
		AvgEmpty:      emptyMetrics.Avg,
		MedianEmpty:   emptyMetrics.Median,
		StdDevEmpty:   emptyMetrics.StdDev,
	}
	if totalLoaded > 0 {
		s.EmptyDistStats.EmptyFullRatio = totalEmpty / totalLoaded
	}
	s.EmptyTimeStats = newTimeMetrics(emptyTimes)
}

// calculateThroughputStats 计算吞吐量与吨距离体量
func (s *Statistics) calculateThroughputStats(
	vehicles []*model.Vehicle,
	tasks []*model.OrderTask,
	shipments []*model.Shipment,
	cargos map[uint]*model.Cargo,
	pois map[uint]*model.Poi,
	now time.Time,
) {
	vehicleMap := make(map[uint]*model.Vehicle, len(vehicles))
	for _, vehicle := range vehicles {
		vehicleMap[vehicle.Id] = vehicle
	}
	taskByShipment := make(map[uint]*model.OrderTask, len(tasks))
	for _, task := range tasks {
		taskByShipment[task.ShipmentId] = task
	}

	receiverTons := make(map[uint]float64)
	receiverSeconds := make(map[uint]float64)
	var receiverEfficiencyValues []float64
	var shipmentTonDistances []float64

	for _, shipment := range shipments {
		if shipment.Status != constant.ShipmentStatusWorking && shipment.Status != constant.ShipmentStatusFinish {
			continue
		}

		weightTons := shipmentWeightTons(shipment, cargos[shipment.CargoId])
		s.ThroughputStats.TotalTransportedTons += weightTons
		if shipment.Status == constant.ShipmentStatusWorking {
			s.ThroughputStats.InTransitTons += weightTons
		}
		if shipment.Status == constant.ShipmentStatusFinish {
			s.ThroughputStats.CompletedTons += weightTons
		}

		startPoi := pois[shipment.StartPoiId]
		endPoi := pois[shipment.EndPoiId]
		if startPoi != nil && endPoi != nil {
			tonDistance := weightTons * haversine(startPoi.Lon, startPoi.Lat, endPoi.Lon, endPoi.Lat)
			s.TransportWorkStats.TotalTonDistance += tonDistance
			shipmentTonDistances = append(shipmentTonDistances, tonDistance)
		}

		duration := estimateShipmentTransportDuration(
			shipment,
			taskByShipment[shipment.Id],
			vehicleMap[taskVehicleID(taskByShipment[shipment.Id])],
			pois,
			now,
		)
		if duration > 0 {
			receiverTons[shipment.EndPoiId] += weightTons
			receiverSeconds[shipment.EndPoiId] += duration
		}
	}

	s.TransportWorkStats.ShipmentTonDistance = newNumericMetrics(shipmentTonDistances)
	s.ThroughputStats.ReceiverEfficiencyByPoi = make(map[uint]float64, len(receiverTons))
	for poiID, tons := range receiverTons {
		seconds := receiverSeconds[poiID]
		if seconds <= 0 {
			continue
		}
		efficiency := tons / seconds
		s.ThroughputStats.ReceiverEfficiencyByPoi[poiID] = efficiency
		receiverEfficiencyValues = append(receiverEfficiencyValues, efficiency)
	}
	s.ThroughputStats.ReceiverEfficiency = newNumericMetrics(receiverEfficiencyValues)
}

// calculatePendingLossStats 计算待处理损耗
func (s *Statistics) calculatePendingLossStats(
	shipments []*model.Shipment,
	cargos map[uint]*model.Cargo,
) {
	s.PendingLossStats.FactoryLossByPoi = make(map[uint]float64)

	var losses []float64
	for _, shipment := range shipments {
		if shipment.Status != constant.ShipmentStatusSleeping &&
			shipment.Status != constant.ShipmentStatusQueued &&
			shipment.Status != constant.ShipmentStatusWaiting {
			continue
		}
		loss := shipmentWeightTons(shipment, cargos[shipment.CargoId]) * s.cargoWaitByShipment[shipment.Id]
		s.PendingLossStats.TotalLoss += loss
		s.PendingLossStats.FactoryLossByPoi[shipment.StartPoiId] += loss
	}
	for _, loss := range s.PendingLossStats.FactoryLossByPoi {
		losses = append(losses, loss)
	}
	s.PendingLossStats.FactoryLoss = newNumericMetrics(losses)
}

// calculateLoadStats 计算装卸耗时估算
func (s *Statistics) calculateLoadStats(
	shipments []*model.Shipment,
	cargos map[uint]*model.Cargo,
) {
	var handlingTimes []float64
	for _, shipment := range shipments {
		weightTons := shipmentWeightTons(shipment, cargos[shipment.CargoId])
		if weightTons <= 0 {
			continue
		}

		operations := 0.0
		switch shipment.Status {
		case constant.ShipmentStatusWorking:
			operations = 1 // 已完成装货
		case constant.ShipmentStatusFinish:
			operations = 2 // 已完成装货 + 卸货
		default:
			continue
		}

		handlingTimes = append(handlingTimes, weightTons*estimatedHandlingSecondsPerTon*operations)
	}
	s.LoadStats = LoadMetrics{
		TimeMetrics: newTimeMetrics(handlingTimes),
		Estimated:   true,
	}
}

// calculateAdvancedStats 计算高阶比例
func (s *Statistics) calculateAdvancedStats() {
	s.AdvancedStats.EmptyLoadedRatio = s.EmptyDistStats.EmptyFullRatio
	if s.TransportTimeStats.Total > 0 {
		s.AdvancedStats.WaitTransportRatio = (s.VehicleWaitStats.Total + s.CargoWaitStats.Total) / s.TransportTimeStats.Total
	}
}

// calculateSnapshotCost 计算当前全局统计口径下的总 cost
func (s *Statistics) calculateSnapshotCost() {
	weights := DefaultWeights

	costA := weights.K1A*s.VehicleWaitStats.Total + weights.K2A*s.EmptyDistStats.TotalEmpty
	costB := weights.K1B*s.CargoWaitStats.WeightedTotal + weights.K2B*s.CargoWaitStats.Max

	costC := 0.0
	if s.EmptyDistStats.TotalDistance > 0 {
		costC += weights.K1C * (s.EmptyDistStats.TotalEmpty / s.EmptyDistStats.TotalDistance)
	}
	costC += weights.K2C*s.AdvancedStats.WaitTransportRatio + weights.K3C*s.VehicleWaitStats.Max

	minGap := minValueFromMap(s.vehicleCapacityGapByVehicle)
	costD := weights.K1D*float64(s.UtilizationStats.TotalCapacity-s.UtilizationStats.UsedCapacity) + weights.K2D*minGap

	baseCost := (costA + costB + costC + costD) / 4
	costE := baseCost +
		weights.R1*s.VehicleWaitStats.StdDev +
		weights.R2*s.CargoWaitStats.StdDev +
		weights.R3*s.TransportTimeStats.StdDev

	costF := weights.Lambda1*costA +
		weights.Lambda2*costB +
		weights.Lambda3*costC +
		weights.Lambda4*costD +
		weights.Lambda5*costE

	s.GlobalCost = CostBreakdown{
		A: costA,
		B: costB,
		C: costC,
		D: costD,
		E: costE,
		F: costF,
	}
	s.TotalCost = costF
}

func computeLocationDistribution(vehicles []*model.Vehicle) LocationDistributionMetric {
	if len(vehicles) == 0 {
		return LocationDistributionMetric{Pattern: "unknown"}
	}

	var centerLon, centerLat float64
	for _, vehicle := range vehicles {
		centerLon += vehicle.Lon
		centerLat += vehicle.Lat
	}
	centerLon /= float64(len(vehicles))
	centerLat /= float64(len(vehicles))

	var distanceToCenter []float64
	var nearestNeighbor []float64
	for i, vehicle := range vehicles {
		dist := haversine(vehicle.Lon, vehicle.Lat, centerLon, centerLat)
		distanceToCenter = append(distanceToCenter, dist)

		minNeighbor := math.MaxFloat64
		for j, other := range vehicles {
			if i == j {
				continue
			}
			neighborDist := haversine(vehicle.Lon, vehicle.Lat, other.Lon, other.Lat)
			if neighborDist < minNeighbor {
				minNeighbor = neighborDist
			}
		}
		if minNeighbor < math.MaxFloat64 {
			nearestNeighbor = append(nearestNeighbor, minNeighbor)
		}
	}

	distanceMetrics := newNumericMetrics(distanceToCenter)
	neighborMetrics := newNumericMetrics(nearestNeighbor)
	pattern := "unknown"
	if len(nearestNeighbor) > 0 {
		pattern = "dispersed"
		if neighborMetrics.Avg <= denseVehicleNearestNeighborThreshold {
			pattern = "dense"
		}
	}

	return LocationDistributionMetric{
		CenterLon:                  centerLon,
		CenterLat:                  centerLat,
		AvgDistanceToCenter:        distanceMetrics.Avg,
		MaxDistanceToCenter:        distanceMetrics.Max,
		AvgNearestNeighborDistance: neighborMetrics.Avg,
		Pattern:                    pattern,
	}
}

func shipmentHasAssignedVehicle(shipment *model.Shipment, task *model.OrderTask) bool {
	if shipment == nil {
		return false
	}
	if task != nil {
		return true
	}
	return shipment.Status != constant.ShipmentStatusSleeping
}

func effectiveVehicleSpeed(vehicle *model.Vehicle) float64 {
	if vehicle != nil && vehicle.Speed > 0 {
		return vehicle.Speed
	}
	return defaultVehicleSpeedMPS
}

func estimateShipmentTransportDuration(
	shipment *model.Shipment,
	task *model.OrderTask,
	vehicle *model.Vehicle,
	pois map[uint]*model.Poi,
	now time.Time,
) float64 {
	if shipment == nil {
		return 0
	}

	if task != nil && task.Sequential == constant.OrderTaskSequentialTransporting {
		return secondsSince(task.UpdateTime, now)
	}

	startPoi := pois[shipment.StartPoiId]
	endPoi := pois[shipment.EndPoiId]
	if startPoi == nil || endPoi == nil {
		return 0
	}

	dist := haversine(startPoi.Lon, startPoi.Lat, endPoi.Lon, endPoi.Lat)
	return dist / effectiveVehicleSpeed(vehicle)
}

func taskVehicleID(task *model.OrderTask) uint {
	if task == nil {
		return 0
	}
	return task.VehicleId
}

func shipmentWeightTons(shipment *model.Shipment, cargo *model.Cargo) float64 {
	if shipment == nil || cargo == nil {
		return 0
	}
	return float64(cargo.Weight*shipment.Count) / 1000.0
}

func secondsSince(start time.Time, now time.Time) float64 {
	if start.IsZero() {
		return 0
	}
	seconds := now.Sub(start).Seconds()
	if seconds < 0 {
		return 0
	}
	return seconds
}

func newTimeMetrics(data []float64) TimeMetrics {
	numeric := newNumericMetrics(data)
	return TimeMetrics(numeric)
}

func newNumericMetrics(data []float64) NumericMetrics {
	m := computeBasicMetrics(data)
	total := 0.0
	for _, value := range data {
		total += value
	}
	return NumericMetrics{
		Total:  total,
		Max:    m.Max,
		Min:    m.Min,
		Avg:    m.Avg,
		Median: m.Median,
		StdDev: m.StdDev,
	}
}

func minValueFromMap(values map[uint]float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minValue := math.MaxFloat64
	for _, value := range values {
		if value < minValue {
			minValue = value
		}
	}
	if minValue == math.MaxFloat64 {
		return 0
	}
	return minValue
}

// --- 通用数学辅助函数 ---

type basicMetrics struct {
	Max    float64
	Min    float64
	Avg    float64
	Median float64
	StdDev float64
}

func computeBasicMetrics(data []float64) basicMetrics {
	if len(data) == 0 {
		return basicMetrics{}
	}

	values := append([]float64(nil), data...)
	sort.Float64s(values)

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	avg := sum / float64(len(values))

	var sqSum float64
	for _, v := range values {
		sqSum += math.Pow(v-avg, 2)
	}
	stdDev := math.Sqrt(sqSum / float64(len(values)))

	median := values[len(values)/2]
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + values[len(values)/2]) / 2
	}

	return basicMetrics{
		Max:    values[len(values)-1],
		Min:    values[0],
		Avg:    avg,
		Median: median,
		StdDev: stdDev,
	}
}

// calculateVehicleDirection 计算车辆当前规划的整体行进方向矢量
func (s *Statistics) calculateVehicleDirection(
	vehicles []*model.Vehicle,
	tasks []*model.OrderTask,
	shipments []*model.Shipment,
	pois map[uint]*model.Poi,
) {
	if len(pois) == 0 {
		return
	}

	lastEndPoi := make(map[uint]*model.Poi)
	shipmentMap := make(map[uint]*model.Shipment, len(shipments))
	for _, shipment := range shipments {
		shipmentMap[shipment.Id] = shipment
	}

	for _, task := range tasks {
		if task.Sequential >= constant.OrderTaskSequentialFinish {
			continue
		}

		shipment := shipmentMap[task.ShipmentId]
		if shipment == nil {
			continue
		}
		endPoi := pois[shipment.EndPoiId]
		if endPoi == nil {
			continue
		}
		lastEndPoi[task.VehicleId] = endPoi
	}

	for _, vehicle := range vehicles {
		endTarget := lastEndPoi[vehicle.Id]
		if endTarget == nil {
			continue
		}
		dLon := endTarget.Lon - vehicle.Lon
		dLat := endTarget.Lat - vehicle.Lat

		if dLon*dLon+dLat*dLat > 1e-8 {
			s.vehicleDirLon[vehicle.Id] = dLon
			s.vehicleDirLat[vehicle.Id] = dLat
		}
	}
}
