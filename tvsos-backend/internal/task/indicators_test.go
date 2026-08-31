package task

import (
	"testing"
	"time"

	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
)

// TestComputePercentile 验证 P50/P95/P99 分位计算(报告 T2/A2 尾部特征)。
func TestComputePercentile(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 5}
	assertClose(t, computePercentile(sorted, 50), 3)
	assertClose(t, computePercentile(sorted, 95), 4.8)
	assertClose(t, computePercentile(sorted, 99), 4.96)

	single := []float64{7}
	assertClose(t, computePercentile(single, 95), 7)

	if got := computePercentile(nil, 95); got != 0 {
		t.Fatalf("expected 0 for empty input, got %v", got)
	}
}

// TestGiniCoefficient 验证基尼系数(F1)：完全均衡=0，极端不均趋近于 1。
func TestGiniCoefficient(t *testing.T) {
	if got := giniCoefficient([]float64{100, 100, 100}); got != 0 {
		t.Fatalf("expected 0 for equal distribution, got %v", got)
	}
	if got := giniCoefficient([]float64{0, 0, 0}); got != 0 {
		t.Fatalf("expected 0 for zero values, got %v", got)
	}
	if got := giniCoefficient([]float64{1, 1, 1, 9}); got <= 0 || got >= 1 {
		t.Fatalf("expected gini in (0,1), got %v", got)
	}
	// 单元素恒为 0
	if got := giniCoefficient([]float64{42}); got != 0 {
		t.Fatalf("expected 0 for single element, got %v", got)
	}
}

// TestNewMetricsPercentiles 验证 newTimeMetrics / newNumericMetrics 会带出分位字段。
func TestNewMetricsPercentiles(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5}
	nm := newNumericMetrics(vals)
	assertClose(t, nm.P50, 3)
	assertClose(t, nm.P95, 4.8)
	assertClose(t, nm.P99, 4.96)

	tm := newTimeMetrics(vals)
	assertClose(t, tm.P50, 3)
	assertClose(t, tm.P95, 4.8)
}

// TestAdvancedIndicatorsInSnapshot 验证快照中的 U2/U4/C2/F1 与 T2 匹配响应分布。
func TestAdvancedIndicatorsInSnapshot(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)

	vehicles := []*model.Vehicle{
		{Id: 1, Lon: 0.03, Lat: 0, Speed: 10, Status: constant.VehicleStatusFree, UpdateTime: now.Add(-30 * time.Minute), Capacity: 1000, Size: 0, Tybe: 1, Distance: 100, EmptyMileage: 50},
		{Id: 2, Lon: 0.005, Lat: 0, Speed: 10, Status: constant.VehicleStatusFree, UpdateTime: now.Add(-10 * time.Minute), Capacity: 800, Size: 0, Tybe: 1},
		{Id: 3, Lon: 0.01, Lat: 0, Speed: 10, Status: constant.VehicleStatusRunning, UpdateTime: now.Add(-5 * time.Minute), Capacity: 1200, Size: 400, Tybe: 2, Distance: 200, EmptyMileage: 80},
	}

	tasks := []*model.OrderTask{
		{Id: 1, ShipmentId: 11, VehicleId: 2, Sequential: constant.OrderTaskSequentialAccepting, CreateTime: now.Add(-10 * time.Minute), UpdateTime: now.Add(-5 * time.Minute)},
		{Id: 2, ShipmentId: 12, VehicleId: 3, Sequential: constant.OrderTaskSequentialTransporting, CreateTime: now.Add(-50 * time.Minute), UpdateTime: now.Add(-20 * time.Minute)},
		{Id: 3, ShipmentId: 13, VehicleId: 1, Sequential: constant.OrderTaskSequentialFinish, CreateTime: now.Add(-110 * time.Minute), UpdateTime: now.Add(-2 * time.Minute)},
	}

	shipments := []*model.Shipment{
		{Id: 10, Status: constant.ShipmentStatusSleeping, CreateTime: now.Add(-40 * time.Minute), CargoId: 1, Count: 2, StartPoiId: 1, EndPoiId: 2},
		{Id: 11, Status: constant.ShipmentStatusWaiting, CreateTime: now.Add(-15 * time.Minute), CargoId: 2, Count: 1, StartPoiId: 2, EndPoiId: 3},
		{Id: 12, Status: constant.ShipmentStatusWorking, CreateTime: now.Add(-60 * time.Minute), CargoId: 1, Count: 1, StartPoiId: 1, EndPoiId: 3},
		{Id: 13, Status: constant.ShipmentStatusFinish, CreateTime: now.Add(-120 * time.Minute), CargoId: 1, Count: 3, StartPoiId: 3, EndPoiId: 1},
	}

	cargos := map[uint]*model.Cargo{
		1: {Id: 1, Weight: 500},
		2: {Id: 2, Weight: 1500},
	}

	pois := map[uint]*model.Poi{
		1: {Id: 1, Lon: 0, Lat: 0},
		2: {Id: 2, Lon: 0.01, Lat: 0},
		3: {Id: 3, Lon: 0.02, Lat: 0},
	}

	stats := buildStatistics(vehicles, tasks, shipments, cargos, pois, now)

	// U2 装载率标准差：装载率为 [0,0,400/1200≈0.3333]
	assertClose(t, stats.UtilizationStats.LoadingRateAvg, (0+0+0.3333333)/3)
	assertClose(t, stats.UtilizationStats.LoadingRateStd, 0.15713484)

	// F1 基尼系数：基于装载率 [0,0,0.3333] 应处于 (0,1)
	if stats.UtilizationStats.GiniCoefficient <= 0 || stats.UtilizationStats.GiniCoefficient >= 1 {
		t.Fatalf("expected gini in (0,1), got %v", stats.UtilizationStats.GiniCoefficient)
	}

	// U4 空驶率 = 空驶/总里程
	emptyDist := haversine(0.005, 0, 0.01, 0)
	loadedDist := haversine(0, 0, 0.02, 0) + haversine(0.02, 0, 0, 0)
	assertClose(t, stats.EmptyDistStats.EmptyRate, emptyDist/(emptyDist+loadedDist))

	// C2 单位成本 = TotalCost / 总吨公里
	tonDistance := 0.5*haversine(0, 0, 0.02, 0) + 1.5*haversine(0.02, 0, 0, 0)
	assertClose(t, stats.UnitCost, stats.TotalCost/tonDistance)

	// T2 匹配响应时间：shipment 11/12/13 分别有任务
	// 11: now-10min - (now-15min) = 300s；12: now-50min - (now-60min)=600s；13: now-110min-(now-120min)=600s
	assertClose(t, stats.MatchResponseStats.Total, 1500)
	assertClose(t, stats.MatchResponseStats.Min, 300)
	assertClose(t, stats.MatchResponseStats.Max, 600)

	// G1/G2 碳排放：基于车辆累计里程与默认因子
	// v1: 50*0.6+100*1.1=140；v3: 80*0.8+200*1.5=364 => total 504
	assertClose(t, stats.CarbonStats.TotalCarbon, 504)
	assertClose(t, stats.CarbonStats.CarbonPerTonKm, 504/tonDistance)
	if !stats.CarbonStats.Estimated {
		t.Fatalf("expected carbon estimated flag to be true without configured factors")
	}
}
