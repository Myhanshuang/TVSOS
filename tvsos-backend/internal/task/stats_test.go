package task

import (
	"testing"
	"time"

	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
)

func TestBuildStatisticsComputesFullSnapshotMetrics(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)

	vehicles := []*model.Vehicle{
		{Id: 1, Lon: 0.03, Lat: 0, Speed: 10, Status: constant.VehicleStatusFree, UpdateTime: now.Add(-30 * time.Minute), Capacity: 1000, Size: 0, Tybe: 1},
		{Id: 2, Lon: 0.005, Lat: 0, Speed: 10, Status: constant.VehicleStatusFree, UpdateTime: now.Add(-10 * time.Minute), Capacity: 800, Size: 0, Tybe: 1},
		{Id: 3, Lon: 0.01, Lat: 0, Speed: 10, Status: constant.VehicleStatusRunning, UpdateTime: now.Add(-5 * time.Minute), Capacity: 1200, Size: 400, Tybe: 2},
	}

	tasks := []*model.OrderTask{
		{Id: 1, ShipmentId: 11, VehicleId: 2, Sequential: constant.OrderTaskSequentialAccepting, UpdateTime: now.Add(-5 * time.Minute)},
		{Id: 2, ShipmentId: 12, VehicleId: 3, Sequential: constant.OrderTaskSequentialTransporting, UpdateTime: now.Add(-20 * time.Minute)},
		{Id: 3, ShipmentId: 13, VehicleId: 1, Sequential: constant.OrderTaskSequentialFinish, UpdateTime: now.Add(-2 * time.Minute)},
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

	assertClose(t, stats.VehicleWaitStats.Total, 2400)
	assertClose(t, stats.VehicleWaitStats.Max, 1800)
	assertClose(t, stats.VehicleWaitStats.Min, 600)
	assertClose(t, stats.VehicleWaitStats.Median, 1200)
	assertClose(t, stats.VehicleWaitStats.StdDev, 600)

	assertClose(t, stats.CargoWaitStats.Total, 3300)
	assertClose(t, stats.CargoWaitStats.Max, 2400)
	assertClose(t, stats.CargoWaitStats.Min, 900)
	assertClose(t, stats.CargoWaitStats.Median, 1650)
	assertClose(t, stats.CargoWaitStats.StdDev, 750)
	assertClose(t, stats.CargoWaitStats.WeightedTotal, 3750)

	assertClose(t, stats.TransportTimeStats.Total, 1200)
	assertClose(t, stats.TransportTimeStats.Max, 1200)

	if stats.VehicleStats.TotalCount != 3 {
		t.Fatalf("got vehicle count %d, want 3", stats.VehicleStats.TotalCount)
	}
	if stats.VehicleStats.TypeCountMap[1] != 2 || stats.VehicleStats.TypeCountMap[2] != 1 {
		t.Fatalf("unexpected vehicle type distribution: %+v", stats.VehicleStats.TypeCountMap)
	}
	assertClose(t, stats.VehicleStats.CapacityStats.Total, 3000)
	assertClose(t, stats.VehicleStats.CapacityStats.Max, 1200)
	assertClose(t, stats.VehicleStats.CapacityStats.Min, 800)
	assertClose(t, stats.VehicleStats.CurrentLoadStats.Total, 400)

	centerDistV1 := haversine(0.03, 0, 0.015, 0)
	centerDistV2 := haversine(0.005, 0, 0.015, 0)
	centerDistV3 := haversine(0.01, 0, 0.015, 0)
	assertClose(t, stats.VehicleStats.LocationDistribution.CenterLon, 0.015)
	assertClose(t, stats.VehicleStats.LocationDistribution.AvgDistanceToCenter, (centerDistV1+centerDistV2+centerDistV3)/3)
	assertClose(t, stats.VehicleStats.LocationDistribution.MaxDistanceToCenter, centerDistV1)
	assertClose(t, stats.VehicleStats.LocationDistribution.AvgNearestNeighborDistance, (haversine(0.03, 0, 0.01, 0)+haversine(0.005, 0, 0.01, 0)+haversine(0.01, 0, 0.005, 0))/3)
	if stats.VehicleStats.LocationDistribution.Pattern != "dense" {
		t.Fatalf("got vehicle location pattern %q, want dense", stats.VehicleStats.LocationDistribution.Pattern)
	}

	emptyDistance := haversine(0.005, 0, 0.01, 0)
	loadedDistance := haversine(0, 0, 0.02, 0) + haversine(0.02, 0, 0, 0)
	assertClose(t, stats.EmptyDistStats.TotalEmpty, emptyDistance)
	assertClose(t, stats.EmptyDistStats.TotalLoaded, loadedDistance)
	assertClose(t, stats.EmptyTimeStats.Total, emptyDistance/10)
	assertClose(t, stats.AdvancedStats.EmptyLoadedRatio, emptyDistance/loadedDistance)
	assertClose(t, stats.AdvancedStats.WaitTransportRatio, (2400+3300)/1200.0)

	if stats.CargoStats.TotalCount != 4 || stats.CargoStats.AssignedCount != 3 || stats.CargoStats.PendingCount != 1 {
		t.Fatalf("unexpected cargo counts: %+v", stats.CargoStats)
	}
	assertClose(t, stats.CargoStats.TotalDemandTons, 4.5)
	assertClose(t, stats.CargoStats.AssignedDemandTons, 3.5)
	assertClose(t, stats.CargoStats.PendingDemandTons, 1.0)

	assertClose(t, stats.ThroughputStats.TotalTransportedTons, 2.0)
	assertClose(t, stats.ThroughputStats.InTransitTons, 0.5)
	assertClose(t, stats.ThroughputStats.CompletedTons, 1.5)
	assertClose(t, stats.ThroughputStats.ReceiverEfficiencyByPoi[3], 0.5/1200.0)
	assertClose(t, stats.ThroughputStats.ReceiverEfficiencyByPoi[1], 1.5/(haversine(0.02, 0, 0, 0)/10))

	tonDistance1 := 0.5 * haversine(0, 0, 0.02, 0)
	tonDistance2 := 1.5 * haversine(0.02, 0, 0, 0)
	assertClose(t, stats.TransportWorkStats.TotalTonDistance, tonDistance1+tonDistance2)
	assertClose(t, stats.TransportWorkStats.ShipmentTonDistance.Max, tonDistance2)
	assertClose(t, stats.TransportWorkStats.ShipmentTonDistance.Min, tonDistance1)
	assertClose(t, stats.TransportWorkStats.ShipmentTonDistance.Median, (tonDistance1+tonDistance2)/2)

	assertClose(t, stats.PendingLossStats.TotalLoss, 3750)
	assertClose(t, stats.PendingLossStats.FactoryLossByPoi[1], 2400)
	assertClose(t, stats.PendingLossStats.FactoryLossByPoi[2], 1350)
	assertClose(t, stats.PendingLossStats.FactoryLoss.Max, 2400)
	assertClose(t, stats.PendingLossStats.FactoryLoss.Min, 1350)
	assertClose(t, stats.PendingLossStats.FactoryLoss.Median, 1875)

	if !stats.LoadStats.Estimated {
		t.Fatalf("expected load stats to be estimated")
	}
	assertClose(t, stats.LoadStats.Total, 210)
	assertClose(t, stats.LoadStats.Max, 180)
	assertClose(t, stats.LoadStats.Min, 30)
	assertClose(t, stats.LoadStats.Median, 105)

	assertClose(t, stats.GlobalCost.A, 0.5*2400+0.5*emptyDistance)
	assertClose(t, stats.GlobalCost.B, 0.6*3750+0.4*2400)
	assertClose(t, stats.TotalCost, stats.GlobalCost.F)
}
