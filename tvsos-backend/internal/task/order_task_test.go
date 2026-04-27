package task

import (
	"testing"

	"github.com/kiritosuki/mover/internal/model"
)

func TestCollectNFCandidatesRespectsRingOrderAndLimit(t *testing.T) {
	vehicles := []*model.Vehicle{
		{Id: 1, Tybe: 1, Capacity: 100, Size: 0},
		{Id: 2, Tybe: 1, Capacity: 100, Size: 0},
		{Id: 3, Tybe: 1, Capacity: 100, Size: 0},
		{Id: 4, Tybe: 1, Capacity: 100, Size: 0},
	}
	projectedLoads := map[uint]int{1: 0, 2: 0, 3: 0, 4: 0}
	shipment := &model.Shipment{Count: 1}
	cargo := &model.Cargo{Tybe: 1, Weight: 10}

	candidates := collectNFCandidates(vehicles, projectedLoads, shipment, cargo, 2, 2)
	if len(candidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(candidates))
	}
	if candidates[0].Vehicle.Id != 3 || candidates[1].Vehicle.Id != 4 {
		t.Fatalf("unexpected ring order: got %d,%d want 3,4", candidates[0].Vehicle.Id, candidates[1].Vehicle.Id)
	}
}

func TestSelectBestCandidateModesAndFallback(t *testing.T) {
	candidates := []VehicleCandidate{
		{Vehicle: &model.Vehicle{Id: 1}, Index: 0},
		{Vehicle: &model.Vehicle{Id: 2}, Index: 1},
	}
	shipment := &model.Shipment{}
	cargo := &model.Cargo{}
	start := &model.Poi{}
	end := &model.Poi{}

	nf := selectBestCandidate(SchedulerOptions{Mode: schedulerModeNF}, candidates, shipment, cargo, start, end, nil)
	if nf.Vehicle.Id != 1 {
		t.Fatalf("nf mode picked vehicle %d, want 1", nf.Vehicle.Id)
	}

	hybridFallback := selectBestCandidate(SchedulerOptions{Mode: schedulerModeHybrid, CostFallback: schedulerFallbackNFFirst}, candidates, shipment, cargo, start, end, nil)
	if hybridFallback.Vehicle.Id != 1 {
		t.Fatalf("hybrid fallback picked vehicle %d, want 1", hybridFallback.Vehicle.Id)
	}
}

func TestIsVehicleFeasibleChecksTypeAndCapacity(t *testing.T) {
	v := &model.Vehicle{Id: 1, Tybe: 1, Capacity: 100, Size: 0}
	shipment := &model.Shipment{Count: 5}
	cargo := &model.Cargo{Tybe: 1, Weight: 10}

	if !isVehicleFeasible(v, shipment, cargo, 40) {
		t.Fatal("expected vehicle feasible with enough capacity")
	}
	if isVehicleFeasible(v, shipment, &model.Cargo{Tybe: 2, Weight: 10}, 0) {
		t.Fatal("expected vehicle infeasible due to type mismatch")
	}
	if isVehicleFeasible(v, shipment, cargo, 60) {
		t.Fatal("expected vehicle infeasible due to capacity")
	}
}
