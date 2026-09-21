package task

import (
	"sort"
	"sync"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
	"go.uber.org/zap"
)

type NameValue struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type NumericStatView struct {
	Max    float64 `json:"max"`
	Min    float64 `json:"min"`
	Avg    float64 `json:"avg"`
	Median float64 `json:"median"`
	P50    float64 `json:"p50"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
}

type RealtimeSummary struct {
	// ---- 沙箱系统：实验与虚拟时钟信息 ----
	ExperimentId   uint    `json:"experiment_id"`   // 当前实验 id
	ExperimentName string  `json:"experiment_name"` // 实验名
	VirtualTime    string  `json:"virtual_time"`    // 虚拟时钟当前时刻
	Progress       float64 `json:"progress"`        // 虚拟时钟进度 0~1
	Speed          float64 `json:"speed"`           // 当前时间加速比

	TotalThroughput     float64     `json:"total_throughput"`
	InTransitTons       float64     `json:"in_transit_tons"` // 在途货物量(吨)
	CompletedTons       float64     `json:"completed_tons"`   // 已完成货物量(吨)
	CostValue           float64     `json:"cost_value"`
	CargoPending        float64     `json:"cargo_pending"`
	TransportEfficiency float64     `json:"transport_efficiency"`
	TotalWaitingTime    float64     `json:"total_waiting_time"`
	TotalEmptyMileage   float64     `json:"total_empty_mileage"`
	TotalLossTh         float64     `json:"total_loss_th"`
	TotalCapacityTm     float64     `json:"total_capacity_tm"`
	VehicleTypes        []NameValue `json:"vehicle_types"`
	CargoDemand         []NameValue `json:"cargo_demand"`
	EmptyFullRatio      float64     `json:"empty_full_ratio"`
	WaitTransportRatio  float64     `json:"wait_transport_ratio"`
	WorkTime            WorkTime    `json:"work_time"`
	// ---- 科学指标体系扩展字段 ----
	UnitCost           float64 `json:"unit_cost"`           // C2 单位成本(成本/吨公里)
	EmptyRate          float64 `json:"empty_rate"`          // U4 空驶率
	LoadingRateAvg     float64 `json:"loading_rate_avg"`    // U2 装载率均值
	LoadingRateStdDev  float64 `json:"loading_rate_stddev"` // U2 装载率标准差
	OnlineRate         float64 `json:"online_rate"`         // U3 车辆在线率
	GiniCoefficient    float64 `json:"gini_coefficient"`    // F1 基尼系数
	MatchResponseP50   float64 `json:"match_response_p50"`  // T2 匹配响应 P50(秒)
	MatchResponseP95   float64 `json:"match_response_p95"`  // T2 匹配响应 P95(秒)
	MatchResponseP99   float64 `json:"match_response_p99"`  // T2 匹配响应 P99(秒)
	CarbonTotal        float64 `json:"carbon_total"`        // G1 总碳排放(kg)
	CarbonPerTonKm     float64 `json:"carbon_per_ton_km"`   // G2 单位吨公里碳排放
	OptimizationGap    float64 `json:"optimization_gap"`    // A3 优化 gap(最近批次，百分比)

	// ---- 可靠性/匹配吞吐/算法耗时扩展字段 ----
	MtbfHours           float64 `json:"mtbf_hours"`           // R1 平均无故障时间(小时, 自最近启动)
	MttrHours           float64 `json:"mttr_hours"`           // R2 平均修复时长(小时)
	ServiceAvailability float64 `json:"service_availability"` // R3 服务可用性(百分比)
	FaultCount          int64   `json:"fault_count"`          // 观测窗口内车祸故障次数
	MatchThroughput     float64 `json:"match_throughput"`     // T1 匹配吞吐量(单/小时, 自最近启动)
	MatchTotalCount     int64   `json:"match_total_count"`     // T1 观测窗口内匹配决策数
	SolveBatches        int     `json:"solve_batches"`        // A2 已计时的求解批次数
	SolveTimeAvgMs      float64 `json:"solve_time_avg_ms"`    // A2 单批求解平均耗时(毫秒)
	SolveTimeMaxMs      float64 `json:"solve_time_max_ms"`    // A2 单批求解峰值耗时(毫秒)

	// ---- 匹配精准度/基线对比(M1-M4/C1) ----
	MatchPrecision      float64 `json:"match_precision"`      // M1 精确率(%): 选中车=条件最优
	MatchRecall         float64 `json:"match_recall"`         // M2 召回率(%): 未派单计未命中
	MatchF1             float64 `json:"match_f1"`             // M3 匹配 F1
	MatchMrr            float64 `json:"match_mrr"`            // M4 平均倒数排名
	MatchEvaluated      int64   `json:"match_evaluated"`      // 已评估运单数
	BaselineImprovement float64 `json:"baseline_improvement"` // C1 SA相对贪心基线成本改善(%)

	// ---- 随机事件/司机扩展字段 ----
	MaintenanceVehicles int         `json:"maintenance_vehicles"` // 维修中车辆数
	RestingDrivers      int         `json:"resting_drivers"`      // 休息中司机数
	RefuseCount         int64       `json:"refuse_count"`         // 拒单事件计数
	AccidentCount       int64       `json:"accident_count"`       // 车祸事件计数
	RecentEvents        []EventView `json:"recent_events"`        // 最近随机事件列表

	// ---- 地图覆盖层面板扩展字段（指标展示建议：凹字形面板） ----
	VehicleStatus   []NameValue      `json:"vehicle_status"`    // 车辆状态分布(行驶中/空闲中/待发车/维修中)
	TopLoadVehicles []VehicleTopView `json:"top_load_vehicles"` // Top载货最重的车(Top3, 吨)
	TopTimeVehicles []VehicleTopView `json:"top_time_vehicles"` // Top载货时间最长的车(Top3, 小时)
	Radar           RadarMetrics     `json:"radar"`             // 调配效率雷达五维(0~100)
}

type WorkTime struct {
	TotalLoadingTime float64 `json:"total_loading_time"`
}

// EventView 随机事件的前端展示视图
// 类型徽标与描述供前端事件流面板直接渲染。
type EventView struct {
	Id          uint   `json:"id"`
	EventType   string `json:"event_type"`
	EventTime   string `json:"event_time"`
	VehicleId   uint   `json:"vehicle_id"`
	License     string `json:"license"`
	DriverId    uint   `json:"driver_id"`
	ShipmentId  uint   `json:"shipment_id"`
	Severity    int    `json:"severity"`
	DurationSec int    `json:"duration_seconds"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// VehicleTopView 车辆最值榜单条目（Top载货/Top载货时长）
type VehicleTopView struct {
	License string  `json:"license"`
	Value   float64 `json:"value"`
}

// RadarMetrics 调配效率雷达图五维指标（均为 0~100 的百分比，前端直接渲染）。
// 口径与文档一致：
// 里程利用率=载货里程/总里程；装载率=实际载重/最大载重；派单率=运输中订单/总订单；
// 货物流动率=在途货物量/总货物量；运力利用率=已使用运力吨公里/可用运力吨公里。
type RadarMetrics struct {
	MileageUtilization  float64 `json:"mileage_utilization"`
	LoadingRate         float64 `json:"loading_rate"`
	DispatchRate        float64 `json:"dispatch_rate"`
	CargoFlowRate       float64 `json:"cargo_flow_rate"`
	CapacityUtilization float64 `json:"capacity_utilization"`
}

type RealtimeIndividualStats struct {
	TruckCapacity NumericStatView `json:"truck_capacity"`
	LoadingTime   NumericStatView `json:"loading_time"`
}

type RealtimeDashboard struct {
	Summary         RealtimeSummary         `json:"summary"`
	IndividualStats RealtimeIndividualStats `json:"individual_stats"`
	DetailTrends    map[string][]float64    `json:"detail_trends"`
	TrendLabels     []string                `json:"trend_labels"` // 趋势点虚拟时刻标签(HH:MM:SS)
}

type trendStore struct {
	mu          sync.Mutex
	labels      []string
	throughput  []float64
	pending     []float64
	cost        []float64
	efficiency  []float64
	p95         []float64
	loadingRate []float64
	emptyRate   []float64
	onlineRate  []float64
	inTransit   []float64
	completed   []float64
}

// 趋势窗口容量：WS 推送周期 3s，240 点约覆盖最近 12 分钟虚拟运行过程。
const trendPointLimit = 240

var dashboardTrendStore = &trendStore{}

// BuildRealtimeDashboard 构建指定实验运行时的实时看板（experiment_id 作用域 + 虚拟时钟口径）。
// R1-R3/T1 观测窗口 = 本实验虚拟时钟起点（替代旧"最近一次 startup 事件"口径）。
func BuildRealtimeDashboard(rt *ExperimentRuntime) (*RealtimeDashboard, error) {
	stats, err := CalculateGlobalStats(rt.Exp.Id, rt.Clock.Now())
	if err != nil {
		return nil, err
	}

	// 单次拉取本实验车辆列表：维修计数 / 状态分布 / 车牌映射 / Top榜 共用
	allVehicles, _ := repository.ListVehiclesByExp(rt.Exp.Id, 0)
	maintenanceVehicleCount, restingDriverCount := 0, 0
	// 构建 车辆ID -> 车牌号 映射，供随机事件流展示真实车牌
	vehicleLicenseMap := make(map[uint]string)
	statusCountMap := map[int]int{
		constant.VehicleStatusRunning:     0,
		constant.VehicleStatusFree:        0,
		constant.VehicleStatusReady:       0,
		constant.VehicleStatusMaintenance: 0,
	}
	for _, v := range allVehicles {
		if v == nil {
			continue
		}
		license := v.License
		if license == "" {
			license = "车辆-" + uintToStringSafe(v.Id)
		}
		vehicleLicenseMap[v.Id] = license
		if _, ok := statusCountMap[v.Status]; ok {
			statusCountMap[v.Status]++
		}
		if v.Status == constant.VehicleStatusMaintenance {
			maintenanceVehicleCount++
		}
	}
	vehicleStatusDist := []NameValue{
		{Name: "行驶中", Value: float64(statusCountMap[constant.VehicleStatusRunning])},
		{Name: "空闲中", Value: float64(statusCountMap[constant.VehicleStatusFree])},
		{Name: "待发车", Value: float64(statusCountMap[constant.VehicleStatusReady])},
		{Name: "维修中", Value: float64(statusCountMap[constant.VehicleStatusMaintenance])},
	}
	if drivers, err := repository.ListDriversByExp(rt.Exp.Id, 2); err == nil {
		restingDriverCount = len(drivers)
	}
	refuseEventCount, _ := repository.CountRandomEventsByTypeByExp(rt.Exp.Id, "refuse")
	accidentEventCount, _ := repository.CountRandomEventsByTypeByExp(rt.Exp.Id, "accident")
	// R1-R3 可靠性 / T1 匹配吞吐量：观测窗口 = 本实验虚拟时钟起点至今（虚拟时间）
	now := rt.Clock.Now()
	windowStart := rt.Clock.Start()
	reliability := calculateReliabilityMetrics(rt.Exp.Id, len(vehicleLicenseMap), windowStart, now)
	matchTotalCount, _ := repository.CountMatchingLogsSinceByExp(rt.Exp.Id, windowStart)
	matchThroughput := 0.0
	if windowHours := now.Sub(windowStart).Hours(); windowHours > 0 && matchTotalCount > 0 {
		matchThroughput = float64(matchTotalCount) / windowHours
	}
	solveBatches, solveAvgMs, solveMaxMs := GetSolverTimingStats()
	matchP, matchR, matchF1, matchMrr, matchEval := GetMatchingQualityStats()

	var recentEvents []EventView
	if events, err := repository.ListRandomEventsByExp(rt.Exp.Id, 10); err == nil {
		recentEvents = make([]EventView, 0, len(events))
		for _, ev := range events {
			recentEvents = append(recentEvents, EventView{
				Id:          ev.Id,
				EventType:   ev.EventType,
				EventTime:   ev.EventTime.Format("2006-01-02 15:04:05"),
				VehicleId:   ev.VehicleId,
				License:     vehicleLicenseMap[ev.VehicleId],
				DriverId:    ev.DriverId,
				ShipmentId:  ev.ShipmentId,
				Severity:    ev.Severity,
				DurationSec: ev.DurationSeconds,
				Description: ev.Description,
				Status:      ev.Status,
			})
		}
	}

	cargoDemand, err := buildCargoDemandByType(rt.Exp.Id)
	if err != nil {
		return nil, err
	}

	vehicleTypes := make([]NameValue, 0, len(stats.VehicleStats.TypeCountMap))
	keys := make([]int, 0, len(stats.VehicleStats.TypeCountMap))
	for k := range stats.VehicleStats.TypeCountMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		vehicleTypes = append(vehicleTypes, NameValue{
			Name:  "类型" + intToString(k),
			Value: float64(stats.VehicleStats.TypeCountMap[k]),
		})
	}

	// ---- Top榜：载货最重 / 载货时间最长（各取Top3，完整列表由前端走 /vehicles） ----
	topLoadVehicles := make([]VehicleTopView, 0, 3)
	topTimeVehicles := make([]VehicleTopView, 0, 3)
	{
		byLoad := make([]*model.Vehicle, 0, len(allVehicles))
		byLoad = append(byLoad, allVehicles...)
		sort.Slice(byLoad, func(i, j int) bool { return byLoad[i].Size > byLoad[j].Size })
		for i := 0; i < len(byLoad) && i < 3; i++ {
			topLoadVehicles = append(topLoadVehicles, VehicleTopView{
				License: vehicleLicenseMap[byLoad[i].Id],
				Value:   float64(byLoad[i].Size) / 1000.0, // kg -> 吨
			})
		}
		byTime := make([]*model.Vehicle, 0, len(allVehicles))
		byTime = append(byTime, allVehicles...)
		sort.Slice(byTime, func(i, j int) bool { return byTime[i].LoadedDuration > byTime[j].LoadedDuration })
		for i := 0; i < len(byTime) && i < 3; i++ {
			topTimeVehicles = append(topTimeVehicles, VehicleTopView{
				License: vehicleLicenseMap[byTime[i].Id],
				Value:   byTime[i].LoadedDuration, // 小时
			})
		}
	}

	// ---- 调配效率雷达五维（0~100，口径见 RadarMetrics 注释） ----
	clamp100 := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 100 {
			return 100
		}
		return v
	}
	radar := RadarMetrics{
		LoadingRate: clamp100(stats.UtilizationStats.LoadingRateAvg * 100),
	}
	// 里程利用率 = 载货里程/总里程
	if stats.EmptyDistStats.TotalDistance > 0 {
		radar.MileageUtilization = clamp100(stats.EmptyDistStats.TotalLoaded / stats.EmptyDistStats.TotalDistance * 100)
	}
	// 派单率 = 实际运输中订单数/总订单数
	if shipments, sErr := repository.ListShipmentsByExp(rt.Exp.Id); sErr == nil && len(shipments) > 0 {
		workingCount := 0
		for _, s := range shipments {
			if s != nil && s.Status == constant.ShipmentStatusWorking {
				workingCount++
			}
		}
		radar.DispatchRate = clamp100(float64(workingCount) / float64(len(shipments)) * 100)
	}
	// 货物流动率 = 在途货物量/(在途+待处理+已完成)
	if totalCargoTons := stats.ThroughputStats.InTransitTons + stats.CargoStats.PendingDemandTons + stats.ThroughputStats.CompletedTons; totalCargoTons > 0 {
		radar.CargoFlowRate = clamp100(stats.ThroughputStats.InTransitTons / totalCargoTons * 100)
	}
	// 运力利用率 = 已使用运力吨公里/可用运力吨公里(Σ 车辆载重吨 × 已行驶里程km)
	availableTonKm := 0.0
	for _, v := range allVehicles {
		if v == nil || v.Distance <= 0 {
			continue
		}
		availableTonKm += (float64(v.Capacity) / 1000.0) * v.Distance
	}
	if availableTonKm > 0 {
		radar.CapacityUtilization = clamp100(stats.TransportWorkStats.TotalTonDistance / availableTonKm * 100)
	}

	dashboard := &RealtimeDashboard{
		Summary: RealtimeSummary{
			ExperimentId:        rt.Exp.Id,
			ExperimentName:      rt.Exp.Name,
			VirtualTime:         now.Format("2006-01-02 15:04:05"),
			Progress:            rt.Clock.Progress(),
			Speed:               rt.GetSpeed(),
			TotalThroughput:     stats.ThroughputStats.TotalTransportedTons,
			InTransitTons:       stats.ThroughputStats.InTransitTons,
			CompletedTons:       stats.ThroughputStats.CompletedTons,
			CostValue:           stats.TotalCost,
			CargoPending:        stats.CargoStats.PendingDemandTons,
			TransportEfficiency: stats.UtilizationStats.ByCapacity * 100,
			TotalWaitingTime:    stats.VehicleWaitStats.Total / 3600,
			TotalEmptyMileage:   stats.EmptyDistStats.TotalEmpty / 1000.0,
			TotalLossTh:         stats.PendingLossStats.TotalLoss,
			TotalCapacityTm:     stats.TransportWorkStats.TotalTonDistance,
			VehicleTypes:        vehicleTypes,
			CargoDemand:         cargoDemand,
			EmptyFullRatio:      stats.AdvancedStats.EmptyLoadedRatio,
			WaitTransportRatio:  stats.AdvancedStats.WaitTransportRatio,
			WorkTime: WorkTime{
				TotalLoadingTime: stats.LoadStats.Total / 60,
			},
			UnitCost:          stats.UnitCost,
			EmptyRate:         stats.EmptyDistStats.EmptyRate,
			LoadingRateAvg:    stats.UtilizationStats.LoadingRateAvg,
			LoadingRateStdDev: stats.UtilizationStats.LoadingRateStd,
			OnlineRate:        stats.UtilizationStats.OnlineRate,
			GiniCoefficient:   stats.UtilizationStats.GiniCoefficient,
			MatchResponseP50:  stats.MatchResponseStats.P50,
			MatchResponseP95:  stats.MatchResponseStats.P95,
			MatchResponseP99:  stats.MatchResponseStats.P99,
			CarbonTotal:       stats.CarbonStats.TotalCarbon,
			CarbonPerTonKm:    stats.CarbonStats.CarbonPerTonKm,
			OptimizationGap:   GetLastOptimizationGap(),
			MtbfHours:           reliability.MtbfHours,
			MttrHours:           reliability.MttrHours,
			ServiceAvailability: reliability.Availability,
			FaultCount:          reliability.FaultCount,
			MatchThroughput:     matchThroughput,
			MatchTotalCount:     matchTotalCount,
			SolveBatches:        solveBatches,
			SolveTimeAvgMs:      solveAvgMs,
			SolveTimeMaxMs:      solveMaxMs,
			MatchPrecision:      matchP,
			MatchRecall:         matchR,
			MatchF1:             matchF1,
			MatchMrr:            matchMrr,
			MatchEvaluated:      matchEval,
			BaselineImprovement: GetLastBaselineImprovement(),
			MaintenanceVehicles: maintenanceVehicleCount,
			RestingDrivers:      restingDriverCount,
			RefuseCount:         refuseEventCount,
			AccidentCount:       accidentEventCount,
			RecentEvents:        recentEvents,
			VehicleStatus:       vehicleStatusDist,
			TopLoadVehicles:     topLoadVehicles,
			TopTimeVehicles:     topTimeVehicles,
			Radar:               radar,
		},
		IndividualStats: RealtimeIndividualStats{
			TruckCapacity: NumericStatView{
				Max:    stats.VehicleStats.CapacityStats.Max,
				Min:    stats.VehicleStats.CapacityStats.Min,
				Avg:    stats.VehicleStats.CapacityStats.Avg,
				Median: stats.VehicleStats.CapacityStats.Median,
				P50:    stats.VehicleStats.CapacityStats.P50,
				P95:    stats.VehicleStats.CapacityStats.P95,
				P99:    stats.VehicleStats.CapacityStats.P99,
			},
			LoadingTime: NumericStatView{
				Max:    stats.LoadStats.Max,
				Min:    stats.LoadStats.Min,
				Avg:    stats.LoadStats.Avg,
				Median: stats.LoadStats.Median,
				P50:    stats.LoadStats.P50,
				P95:    stats.LoadStats.P95,
				P99:    stats.LoadStats.P99,
			},
		},
	}

	dashboard.DetailTrends, dashboard.TrendLabels = appendAndExportTrends(dashboard, now.Format("15:04:05"))
	return dashboard, nil
}

// resetDashboardTrends 重置趋势累计（实验启动时调用，新实验新窗口）
func resetDashboardTrends() {
	dashboardTrendStore.mu.Lock()
	defer dashboardTrendStore.mu.Unlock()
	dashboardTrendStore.labels = nil
	dashboardTrendStore.throughput = nil
	dashboardTrendStore.pending = nil
	dashboardTrendStore.cost = nil
	dashboardTrendStore.efficiency = nil
	dashboardTrendStore.p95 = nil
	dashboardTrendStore.loadingRate = nil
	dashboardTrendStore.emptyRate = nil
	dashboardTrendStore.onlineRate = nil
	dashboardTrendStore.inTransit = nil
	dashboardTrendStore.completed = nil
}

// appendAndExportTrends 追加当前快照到趋势窗口并导出（labels 与所有序列等长对齐）。
// label 为虚拟时刻(HH:MM:SS)，供覆盖层健康图/运力效率图的X轴与事件标记定位。
func appendAndExportTrends(d *RealtimeDashboard, label string) (map[string][]float64, []string) {
	dashboardTrendStore.mu.Lock()
	defer dashboardTrendStore.mu.Unlock()

	appendWithLimit := func(arr []float64, v float64) []float64 {
		arr = append(arr, v)
		if len(arr) > trendPointLimit {
			arr = arr[len(arr)-trendPointLimit:]
		}
		return arr
	}

	dashboardTrendStore.labels = append(dashboardTrendStore.labels, label)
	if len(dashboardTrendStore.labels) > trendPointLimit {
		dashboardTrendStore.labels = dashboardTrendStore.labels[len(dashboardTrendStore.labels)-trendPointLimit:]
	}
	dashboardTrendStore.throughput = appendWithLimit(dashboardTrendStore.throughput, d.Summary.TotalThroughput)
	dashboardTrendStore.pending = appendWithLimit(dashboardTrendStore.pending, d.Summary.CargoPending)
	dashboardTrendStore.cost = appendWithLimit(dashboardTrendStore.cost, d.Summary.CostValue)
	dashboardTrendStore.efficiency = appendWithLimit(dashboardTrendStore.efficiency, d.Summary.TransportEfficiency)
	dashboardTrendStore.p95 = appendWithLimit(dashboardTrendStore.p95, d.Summary.MatchResponseP95)
	dashboardTrendStore.loadingRate = appendWithLimit(dashboardTrendStore.loadingRate, d.Summary.LoadingRateAvg*100)
	dashboardTrendStore.emptyRate = appendWithLimit(dashboardTrendStore.emptyRate, d.Summary.EmptyRate*100)
	dashboardTrendStore.onlineRate = appendWithLimit(dashboardTrendStore.onlineRate, d.Summary.OnlineRate*100)
	dashboardTrendStore.inTransit = appendWithLimit(dashboardTrendStore.inTransit, d.Summary.InTransitTons)
	dashboardTrendStore.completed = appendWithLimit(dashboardTrendStore.completed, d.Summary.CompletedTons)

	trends := map[string][]float64{
		"throughput":   copySlice(dashboardTrendStore.throughput),
		"pending":      copySlice(dashboardTrendStore.pending),
		"cost":         copySlice(dashboardTrendStore.cost),
		"efficiency":   copySlice(dashboardTrendStore.efficiency),
		"p95":          copySlice(dashboardTrendStore.p95),
		"loading_rate": copySlice(dashboardTrendStore.loadingRate),
		"empty_rate":   copySlice(dashboardTrendStore.emptyRate),
		"online_rate":  copySlice(dashboardTrendStore.onlineRate),
		"in_transit":   copySlice(dashboardTrendStore.inTransit),
		"completed":    copySlice(dashboardTrendStore.completed),
	}
	labels := make([]string, len(dashboardTrendStore.labels))
	copy(labels, dashboardTrendStore.labels)
	return trends, labels
}

func buildCargoDemandByType(experimentID uint) ([]NameValue, error) {
	shipments, err := repository.ListShipmentsByExp(experimentID)
	if err != nil {
		return nil, err
	}
	cargos, err := repository.ListCargos()
	if err != nil {
		return nil, err
	}
	cargoMap := make(map[uint]int, len(cargos))
	for _, cargo := range cargos {
		cargoMap[cargo.Id] = cargo.Tybe
	}

	demandByType := make(map[int]float64)
	for _, shipment := range shipments {
		if shipment.Status != constant.ShipmentStatusSleeping &&
			shipment.Status != constant.ShipmentStatusQueued &&
			shipment.Status != constant.ShipmentStatusWaiting {
			continue
		}
		cargo, err := repository.GetCargo(int(shipment.CargoId))
		if err != nil || cargo == nil {
			continue
		}
		cargoType := cargoMap[shipment.CargoId]
		demandByType[cargoType] += float64(cargo.Weight*shipment.Count) / 1000
	}

	keys := make([]int, 0, len(demandByType))
	for k := range demandByType {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	result := make([]NameValue, 0, len(keys))
	for _, k := range keys {
		result = append(result, NameValue{
			Name:  "货类" + intToString(k),
			Value: demandByType[k],
		})
	}
	if len(result) == 0 {
		result = append(result, NameValue{Name: "暂无", Value: 0})
	}
	return result, nil
}

func copySlice(src []float64) []float64 {
	dst := make([]float64, len(src))
	copy(dst, src)
	return dst
}

func intToString(v int) string {
	if v == 0 {
		return "0"
	}
	negative := false
	if v < 0 {
		negative = true
		v = -v
	}
	buf := make([]byte, 0, 12)
	for v > 0 {
		digit := byte(v%10) + '0'
		buf = append([]byte{digit}, buf...)
		v /= 10
	}
	if negative {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func getStatsPushInterval() time.Duration {
	if config.VP == nil {
		return 3 * time.Second
	}
	sec := config.VP.GetInt("realtime.stats_push_interval_sec")
	if sec <= 0 {
		sec = 3
	}
	return time.Duration(sec) * time.Second
}

// RecordSystemEvent 记录一条系统运行事件(启动/恢复/故障/修复)，供 R1-R3 可靠性指标计算。
// experimentID=0 表示服务级事件；>0 表示某次实验的事件（观测窗口按实验生命周期）。
// 事件落库失败仅记录日志，不影响主流程。
func RecordSystemEvent(experimentID uint, eventType string, level string, message string, detail string) {
	err := repository.CreateSystemEvent(&model.SystemEvent{
		EventTime:    time.Now(),
		EventType:    eventType,
		Level:        level,
		Message:      message,
		Detail:       detail,
		ExperimentId: experimentID,
	})
	if err != nil {
		Logger.Logger.Warn("记录系统事件失败", zap.String("eventType", eventType), zap.Error(err))
	}
}

// LoadEmissionFactors 从 emission_factor 配置表加载排放因子并注入统计模块。
// 无配置时使用内置默认估算因子。
func LoadEmissionFactors() {
	factors, err := repository.ListEmissionFactors()
	if err != nil || len(factors) == 0 {
		Logger.Logger.Info("未加载到排放因子配置，使用默认估算因子")
		return
	}
	factorMap := make(map[int]*model.EmissionFactor, len(factors))
	for _, f := range factors {
		factorMap[f.VehicleType] = f
	}
	SetEmissionFactorProvider(func(vehicleType int) (float64, float64) {
		if f, ok := factorMap[vehicleType]; ok {
			return f.FactorEmpty, f.FactorLoad
		}
		return 0, 0
	})
	Logger.Logger.Info("排放因子配置已加载", zap.Int("count", len(factors)))
}

// LoadHandlingConfig 从 handling_config 配置表加载调度经验系数并注入统计模块。
// 目前支持 handling_seconds_per_ton（装卸秒/吨），无配置时保持默认 60 秒/吨。
func LoadHandlingConfig() {
	configEntry, err := repository.GetHandlingConfig("handling_seconds_per_ton")
	if err == nil && configEntry != nil && configEntry.ConfigValue > 0 {
		estimatedHandlingSecondsPerTon = configEntry.ConfigValue
		Logger.Logger.Info("装卸时间系数已配置化", zap.Float64("handling_seconds_per_ton", estimatedHandlingSecondsPerTon))
		return
	}
	Logger.Logger.Info("未配置 handling_seconds_per_ton，使用默认装卸系数 60 秒/吨")
}

// uintToStringSafe 将 uint 转为十进制字符串，供车牌兜底等场景使用。
func uintToStringSafe(v uint) string {
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 12)
	for v > 0 {
		b = append([]byte{byte(v%10) + '0'}, b...)
		v /= 10
	}
	return string(b)
}



