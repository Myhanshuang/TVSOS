package task

import (
	"testing"
	"time"

	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
)

// TestEvaluateBatchOptimizationGap 验证 A3 优化 gap 评估：小规模批次 SA 代价应不小于精确最优。
func TestEvaluateBatchOptimizationGap(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)

	vehicles := []*model.Vehicle{
		{Id: 1, Lon: 0.00, Lat: 0, Speed: 10, Status: constant.VehicleStatusFree, UpdateTime: now.Add(-30 * time.Minute), Capacity: 1000, Size: 0, Tybe: 1},
		{Id: 2, Lon: 0.02, Lat: 0, Speed: 10, Status: constant.VehicleStatusFree, UpdateTime: now.Add(-30 * time.Minute), Capacity: 1000, Size: 0, Tybe: 1},
	}

	// 两个同向小单：车辆1 在起点附近更优
	shipments := []*model.Shipment{
		{Id: 1, Status: constant.ShipmentStatusSleeping, CreateTime: now.Add(-10 * time.Minute), CargoId: 1, Count: 1, StartPoiId: 1, EndPoiId: 3},
		{Id: 2, Status: constant.ShipmentStatusSleeping, CreateTime: now.Add(-10 * time.Minute), CargoId: 1, Count: 1, StartPoiId: 2, EndPoiId: 3},
	}

	cargos := map[uint]*model.Cargo{
		1: {Id: 1, Weight: 500},
	}

	pois := map[uint]*model.Poi{
		1: {Id: 1, Lon: 0.00, Lat: 0},
		2: {Id: 2, Lon: 0.02, Lat: 0},
		3: {Id: 3, Lon: 0.04, Lat: 0},
	}

	stats := buildStatistics(vehicles, nil, shipments, cargos, pois, now)
	cargoCache := map[uint]*model.Cargo{1: cargos[1]}
	projectedLoads := map[uint]int{1: 0, 2: 0}

	// SA 分配结果（桩运行时：固定种子，确定性）
	rt := newStubRuntime()
	sa := assignBySimulatedAnnealing(rt, shipments, vehicles, stats, cargoCache, projectedLoads, pois)
	if len(sa) != 2 {
		t.Fatalf("SA should assign all 2 shipments, got %d", len(sa))
	}

	saCost, bestCost, gap := evaluateBatchOptimizationGap(shipments, vehicles, stats, cargoCache, projectedLoads, pois, sa, now)
	if saCost <= 0 || bestCost <= 0 {
		t.Fatalf("expected positive costs, got saCost=%v bestCost=%v", saCost, bestCost)
	}
	if gap < 0 || gap > 100 {
		t.Fatalf("expected gap in [0,100], got %v", gap)
	}

	// 规模超限应跳过
	bigShipments := make([]*model.Shipment, maxExactShipments+1)
	for i := range bigShipments {
		bigShipments[i] = shipments[0]
	}
	if c, b, g := evaluateBatchOptimizationGap(bigShipments, vehicles, stats, cargoCache, projectedLoads, pois, sa, now); c != 0 || b != 0 || g != 0 {
		t.Fatalf("expected skip for oversized batch, got %v %v %v", c, b, g)
	}
}