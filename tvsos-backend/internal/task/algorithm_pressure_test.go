package task

import (
	"fmt"
	"testing"
	"time"

	"github.com/kiritosuki/mover/internal/model"
)

// BenchmarkSchedulerVehicleScale 用固定的 5 笔运单测量车辆规模扩大时的算法计算开销。
// 使用 -benchtime=1x 是因为一次 SA 已包含约 9000 次状态评估。
func BenchmarkSchedulerVehicleScale(b *testing.B) {
	for _, vehicleCount := range []int{20, 50, 100, 200, 500} {
		vehicles, shipments, cargos, loads, pois := schedulerBenchmarkFixture(vehicleCount)

		b.Run(fmt.Sprintf("NF/vehicles_%d", vehicleCount), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				projected := cloneProjectedLoads(loads)
				for i, shipment := range shipments {
					cargo := cargos[shipment.CargoId]
					candidates := collectNFCandidates(vehicles, projected, shipment, cargo, i%len(vehicles), candidateVehicleLimit)
					if len(candidates) > 0 {
						projected[candidates[0].Vehicle.Id] += cargo.Weight * shipment.Count
					}
				}
			}
		})

		b.Run(fmt.Sprintf("SA/vehicles_%d", vehicleCount), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_ = assignBySimulatedAnnealing(shipments, vehicles, nil, cargos, cloneProjectedLoads(loads), pois)
			}
		})
	}
}

func schedulerBenchmarkFixture(vehicleCount int) (
	[]*model.Vehicle,
	[]*model.Shipment,
	map[uint]*model.Cargo,
	map[uint]int,
	map[uint]*model.Poi,
) {
	vehicles := make([]*model.Vehicle, 0, vehicleCount)
	loads := make(map[uint]int, vehicleCount)
	for i := 1; i <= vehicleCount; i++ {
		id := uint(i)
		vehicles = append(vehicles, &model.Vehicle{
			Id: id, Tybe: 1, Capacity: 10000, Speed: 50,
			Lon:        104.00 + float64(i%10)*0.005,
			Lat:        30.50 + float64(i%7)*0.005,
			UpdateTime: time.Unix(1_700_000_000, 0),
		})
		loads[id] = 0
	}

	pois := make(map[uint]*model.Poi, 10)
	for i := 1; i <= 10; i++ {
		id := uint(i)
		pois[id] = &model.Poi{Id: id, Lon: 104.00 + float64(i)*0.01, Lat: 30.50 + float64(i%4)*0.01}
	}

	cargos := map[uint]*model.Cargo{
		1: {Id: 1, Tybe: 1, Weight: 100},
	}
	shipments := make([]*model.Shipment, 0, 5)
	for i := 1; i <= 5; i++ {
		shipments = append(shipments, &model.Shipment{
			Id: uint(i), StartPoiId: uint(i), EndPoiId: uint(i + 5),
			CargoId: 1, Count: i, CreateTime: time.Unix(1_700_000_000, 0),
		})
	}
	return vehicles, shipments, cargos, loads, pois
}

func cloneProjectedLoads(src map[uint]int) map[uint]int {
	dst := make(map[uint]int, len(src))
	for id, load := range src {
		dst[id] = load
	}
	return dst
}
