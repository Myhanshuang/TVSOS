package task

import (
	"sort"
	"sync"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/constant"
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
			TotalEmptyMileage:   stats.EmptyDistStats.TotalEmpty,
			TotalLossTh:         stats.PendingLossStats.TotalLoss,
			TotalCapacityTm:     stats.TransportWorkStats.TotalTonDistance,
			VehicleTypes:        vehicleTypes,
			CargoDemand:         cargoDemand,
			EmptyFullRatio:      stats.AdvancedStats.EmptyLoadedRatio,
			WaitTransportRatio:  stats.AdvancedStats.WaitTransportRatio,
			WorkTime: WorkTime{
				TotalLoadingTime: stats.LoadStats.Total / 60,
			},
		},
		IndividualStats: RealtimeIndividualStats{
			TruckCapacity: NumericStatView{
				Max:    stats.VehicleStats.CapacityStats.Max,
				Min:    stats.VehicleStats.CapacityStats.Min,
				Avg:    stats.VehicleStats.CapacityStats.Avg,
				Median: stats.VehicleStats.CapacityStats.Median,
			},
			LoadingTime: NumericStatView{
				Max:    stats.LoadStats.Max,
				Min:    stats.LoadStats.Min,
				Avg:    stats.LoadStats.Avg,
				Median: stats.LoadStats.Median,
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
