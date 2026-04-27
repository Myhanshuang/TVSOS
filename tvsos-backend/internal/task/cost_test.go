package task

import (
	"math"
	"testing"
	"time"

	"github.com/kiritosuki/mover/internal/model"
)

func TestCalculateCostBreakdownUsesProjectedWhiteboardMetrics(t *testing.T) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)

	v := &model.Vehicle{
		Id:       1,
		Lon:      116.397428,
		Lat:      39.90923,
		Speed:    10.0,
		Capacity: 2000,
		Size:     400,
	}

	shipment := &model.Shipment{
		Id:         10,
		CreateTime: now.Add(-5 * time.Minute),
		Count:      2,
	}

	stats := &Statistics{
		VehicleWaitStats: TimeMetrics{
			Total:  200,
			Max:    120,
			StdDev: 15,
		},
		CargoWaitStats: CargoWaitMetrics{
			TimeMetrics: TimeMetrics{
				Total:  400,
				Max:    300,
				StdDev: 25,
			},
			WeightedTotal: 600,
		},
		TransportTimeStats: TimeMetrics{
			Total:  500,
			StdDev: 35,
		},
		EmptyDistStats: DistanceMetrics{
			TotalEmpty:    1000,
			TotalLoaded:   4000,
			TotalDistance: 5000,
		},
		UtilizationStats: UtilizationMetrics{
			TotalCapacity: 3000,
			UsedCapacity:  900,
		},
		vehicleWaitByVehicle: map[uint]float64{
			1: 120,
			2: 80,
		},
		cargoWaitByShipment: map[uint]float64{
			10: 300,
			11: 100,
		},
		cargoWeightTonsByShipment: map[uint]float64{
			10: 2,
		},
		vehicleCapacityGapByVehicle: map[uint]float64{
			1: 1600,
			2: 500,
		},
	}

	cargo := &model.Cargo{
		Weight: 1000,
	}

	startPoi := &model.Poi{
		Lon: 116.407428,
		Lat: 39.91923,
	}
	endPoi := &model.Poi{
		Lon: 116.427428,
		Lat: 39.92923,
	}

	breakdown := calculateCostBreakdown(v, shipment, stats, cargo, startPoi, endPoi, now)

	distToStart := haversine(v.Lon, v.Lat, startPoi.Lon, startPoi.Lat)
	tripDistance := haversine(startPoi.Lon, startPoi.Lat, endPoi.Lon, endPoi.Lat)
	timeToStart := distToStart / v.Speed
	tripTime := tripDistance / v.Speed
	shipmentLoadKg := float64(cargo.Weight * shipment.Count)

	expectedA := 0.5*(stats.VehicleWaitStats.Total-120) + 0.5*(stats.EmptyDistStats.TotalEmpty+distToStart)
	expectedB := 0.6*(stats.CargoWaitStats.WeightedTotal+2*timeToStart) + 0.4*math.Max(stats.CargoWaitStats.Max, 300+timeToStart)
	expectedC := 0.4*((stats.EmptyDistStats.TotalEmpty+distToStart)/(stats.EmptyDistStats.TotalDistance+distToStart+tripDistance)) +
		0.3*((80+stats.CargoWaitStats.Total+timeToStart)/(stats.TransportTimeStats.Total+tripTime)) +
		0.3*80
	expectedD := 0.5 * ((3000 - (900 + shipmentLoadKg)) + math.Min(1600-shipmentLoadKg, 500))
	expectedBase := (expectedA + expectedB + expectedC + expectedD) / 4
	expectedE := expectedBase + 0.1*stats.VehicleWaitStats.StdDev + 0.1*stats.CargoWaitStats.StdDev + 0.1*stats.TransportTimeStats.StdDev
	expectedF := 0.2*expectedA + 0.2*expectedB + 0.2*expectedC + 0.2*expectedD + 0.2*expectedE

	assertClose(t, breakdown.A, expectedA)
	assertClose(t, breakdown.B, expectedB)
	assertClose(t, breakdown.C, expectedC)
	assertClose(t, breakdown.D, expectedD)
	assertClose(t, breakdown.E, expectedE)
	assertClose(t, breakdown.F, expectedF)
}

func assertClose(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("got %.10f, want %.10f", got, want)
	}
}
