package task

import (
	"sort"
	"sync"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/realtime"
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
	TotalThroughput     float64     `json:"total_throughput"`
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
	LoadingRateStdDev  float64 `json:"loading_rate_stddev"` // U2 装载率标准差
	GiniCoefficient    float64 `json:"gini_coefficient"`    // F1 基尼系数
	MatchResponseP50   float64 `json:"match_response_p50"`  // T2 匹配响应 P50(秒)
	MatchResponseP95   float64 `json:"match_response_p95"`  // T2 匹配响应 P95(秒)
	MatchResponseP99   float64 `json:"match_response_p99"`  // T2 匹配响应 P99(秒)
	CarbonTotal        float64 `json:"carbon_total"`        // G1 总碳排放(kg)
	CarbonPerTonKm     float64 `json:"carbon_per_ton_km"`   // G2 单位吨公里碳排放
	OptimizationGap    float64 `json:"optimization_gap"`    // A3 优化 gap(最近批次，百分比)
}

type WorkTime struct {
	TotalLoadingTime float64 `json:"total_loading_time"`
}

type RealtimeIndividualStats struct {
	TruckCapacity NumericStatView `json:"truck_capacity"`
	LoadingTime   NumericStatView `json:"loading_time"`
}

type RealtimeDashboard struct {
	Summary         RealtimeSummary         `json:"summary"`
	IndividualStats RealtimeIndividualStats `json:"individual_stats"`
	DetailTrends    map[string][]float64    `json:"detail_trends"`
}

type trendStore struct {
	mu         sync.Mutex
	throughput []float64
	pending    []float64
	cost       []float64
	efficiency []float64
}

var dashboardTrendStore = &trendStore{}

func BuildRealtimeDashboard() (*RealtimeDashboard, error) {
	stats, err := CalculateGlobalStats()
	if err != nil {
		return nil, err
	}

	cargoDemand, err := buildCargoDemandByType()
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

	dashboard := &RealtimeDashboard{
		Summary: RealtimeSummary{
			TotalThroughput:     stats.ThroughputStats.TotalTransportedTons,
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
			LoadingRateStdDev: stats.UtilizationStats.LoadingRateStd,
			GiniCoefficient:   stats.UtilizationStats.GiniCoefficient,
			MatchResponseP50:  stats.MatchResponseStats.P50,
			MatchResponseP95:  stats.MatchResponseStats.P95,
			MatchResponseP99:  stats.MatchResponseStats.P99,
			CarbonTotal:       stats.CarbonStats.TotalCarbon,
			CarbonPerTonKm:    stats.CarbonStats.CarbonPerTonKm,
			OptimizationGap:   GetLastOptimizationGap(),
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

	dashboard.DetailTrends = appendAndExportTrends(dashboard)
	return dashboard, nil
}

func BroadcastStatsLoop() {
	for {
		interval := getStatsPushInterval()
		if realtime.GlobalHub().StatsSubscriberCount() == 0 {
			time.Sleep(interval)
			continue
		}

		snapshot, err := BuildRealtimeDashboard()
		if err != nil {
			Logger.Logger.Warn("实时统计推送失败", zap.Error(err))
			time.Sleep(interval)
			continue
		}
		realtime.GlobalHub().BroadcastStats("snapshot", snapshot)
		time.Sleep(interval)
	}
}

func appendAndExportTrends(d *RealtimeDashboard) map[string][]float64 {
	dashboardTrendStore.mu.Lock()
	defer dashboardTrendStore.mu.Unlock()

	appendWithLimit := func(arr []float64, v float64) []float64 {
		arr = append(arr, v)
		if len(arr) > 24 {
			arr = arr[len(arr)-24:]
		}
		return arr
	}

	dashboardTrendStore.throughput = appendWithLimit(dashboardTrendStore.throughput, d.Summary.TotalThroughput)
	dashboardTrendStore.pending = appendWithLimit(dashboardTrendStore.pending, d.Summary.CargoPending)
	dashboardTrendStore.cost = appendWithLimit(dashboardTrendStore.cost, d.Summary.CostValue)
	dashboardTrendStore.efficiency = appendWithLimit(dashboardTrendStore.efficiency, d.Summary.TransportEfficiency)

	trends := map[string][]float64{
		"throughput": copySlice(dashboardTrendStore.throughput),
		"pending":    copySlice(dashboardTrendStore.pending),
		"cost":       copySlice(dashboardTrendStore.cost),
		"efficiency": copySlice(dashboardTrendStore.efficiency),
	}
	return trends
}

func buildCargoDemandByType() ([]NameValue, error) {
	shipments, err := repository.ListShipments()
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
// 事件落库失败仅记录日志，不影响主流程。
func RecordSystemEvent(eventType string, level string, message string, detail string) {
	err := repository.CreateSystemEvent(&model.SystemEvent{
		EventTime: time.Now(),
		EventType: eventType,
		Level:     level,
		Message:   message,
		Detail:    detail,
	})
	if err != nil {
		Logger.Logger.Warn("记录系统事件失败", zap.String("eventType", eventType), zap.Error(err))
	}
}

// MetricSnapshotLoop 周期采样协程：将全局核心指标落库到 metric_snapshot 表。
// 支撑历史趋势、成本效益对比(C1)与可用性分析(R3)等时序类指标；同时刷新排放因子缓存。
func MetricSnapshotLoop() {
	for {
		snapshot, err := BuildRealtimeDashboard()
		if err != nil {
			Logger.Logger.Warn("指标采样失败", zap.Error(err))
			time.Sleep(getStatsPushInterval())
			continue
		}

		record := &model.MetricSnapshot{
			SnapshotTime:        time.Now(),
			CostValue:           snapshot.Summary.CostValue,
			TotalThroughput:     snapshot.Summary.TotalThroughput,
			CargoPending:        snapshot.Summary.CargoPending,
			TransportEfficiency: snapshot.Summary.TransportEfficiency,
			TotalWaitingTime:    snapshot.Summary.TotalWaitingTime,
			TotalEmptyMileage:   snapshot.Summary.TotalEmptyMileage,
			UnitCost:            snapshot.Summary.UnitCost,
			EmptyRate:           snapshot.Summary.EmptyRate,
			LoadingRateStdDev:   snapshot.Summary.LoadingRateStdDev,
			MatchResponseP50:    snapshot.Summary.MatchResponseP50,
			MatchResponseP95:    snapshot.Summary.MatchResponseP95,
			MatchResponseP99:    snapshot.Summary.MatchResponseP99,
			GiniCoefficient:     snapshot.Summary.GiniCoefficient,
			CarbonTotal:         snapshot.Summary.CarbonTotal,
			CarbonPerTonKm:      snapshot.Summary.CarbonPerTonKm,
		}
		if err := repository.CreateMetricSnapshot(record); err != nil {
			Logger.Logger.Warn("指标采样落库失败", zap.Error(err))
		}

		time.Sleep(getStatsPushInterval())
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


