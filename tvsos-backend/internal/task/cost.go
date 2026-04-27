package task

import (
	"math"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/model"
	"go.uber.org/zap"
)

// CostWeights 成本计算中的各类权重
type CostWeights struct {
	// A: 直接成本权重
	K1A float64
	K2A float64

	// B: 运送快权重
	K1B float64
	K2B float64

	// C: 效率+公平权重
	K1C float64
	K2C float64
	K3C float64

	// D: 平台权重
	K1D float64
	K2D float64

	// E: 风险权重
	R1 float64
	R2 float64
	R3 float64

	// F: 大综合权重
	Lambda1 float64
	Lambda2 float64
	Lambda3 float64
	Lambda4 float64
	Lambda5 float64
}

type CostBreakdown struct {
	A float64
	B float64
	C float64
	D float64
	E float64
	F float64
}

var DefaultWeights = CostWeights{
	K1A: 0.5, K2A: 0.5,
	K1B: 0.6, K2B: 0.4,
	K1C: 0.4, K2C: 0.3, K3C: 0.3,
	K1D: 0.5, K2D: 0.5,
	R1: 0.1, R2: 0.1, R3: 0.1,
	Lambda1: 0.2, Lambda2: 0.2, Lambda3: 0.2, Lambda4: 0.2, Lambda5: 0.2,
}

// CalculateCost 根据白板公式计算综合成本
func CalculateCost(
	v *model.Vehicle,
	shipment *model.Shipment,
	stats *Statistics,
	cargo *model.Cargo,
	startPoi *model.Poi,
	endPoi *model.Poi,
) float64 {
	return calculateCostBreakdown(v, shipment, stats, cargo, startPoi, endPoi, time.Now()).F
}

func calculateCostBreakdown(
	v *model.Vehicle,
	shipment *model.Shipment,
	stats *Statistics,
	cargo *model.Cargo,
	startPoi *model.Poi,
	endPoi *model.Poi,
	now time.Time,
) CostBreakdown {
	weights := DefaultWeights
	stats = normalizeStats(stats)

	distToStart := 0.0
	if v != nil && startPoi != nil {
		distToStart = haversine(v.Lon, v.Lat, startPoi.Lon, startPoi.Lat)
	}

	speed := 10.0
	if v != nil && v.Speed > 0 {
		speed = v.Speed
	}
	timeToStart := distToStart / speed

	tripDistance := 0.0
	if startPoi != nil && endPoi != nil {
		tripDistance = haversine(startPoi.Lon, startPoi.Lat, endPoi.Lon, endPoi.Lat)
	}
	tripTime := tripDistance / speed

	selectedVehicleWait := stats.vehicleWaitByVehicle[v.Id]
	if selectedVehicleWait == 0 && v != nil {
		selectedVehicleWait = secondsSince(v.UpdateTime, now)
	}
	projectedVehicleWaitTotal := math.Max(0, stats.VehicleWaitStats.Total-selectedVehicleWait)
	projectedLongestVehicleWait := projectedVehicleWaitTotal
	if longest := maxValueExcluding(stats.vehicleWaitByVehicle, v.Id); longest > 0 {
		projectedLongestVehicleWait = longest
	}

	shipmentWait := stats.cargoWaitByShipment[shipment.Id]
	if shipmentWait == 0 && shipment != nil {
		shipmentWait = secondsSince(shipment.CreateTime, now)
	}
	cargoWeightTons := stats.cargoWeightTonsByShipment[shipment.Id]
	if cargoWeightTons == 0 {
		cargoWeightTons = shipmentWeightTons(shipment, cargo)
	}
	projectedCargoWaitTotal := stats.CargoWaitStats.Total + timeToStart
	projectedWeightedCargoWait := stats.CargoWaitStats.WeightedTotal + cargoWeightTons*timeToStart
	projectedSlowestCargoWait := math.Max(stats.CargoWaitStats.Max, shipmentWait+timeToStart)

	projectedTotalEmpty := stats.EmptyDistStats.TotalEmpty + distToStart
	projectedTotalDistance := stats.EmptyDistStats.TotalDistance + distToStart + tripDistance
	if projectedTotalDistance <= 0 {
		projectedTotalDistance = projectedTotalEmpty + tripDistance
	}

	projectedTransportTotal := stats.TransportTimeStats.Total + tripTime
	totalWaitForEfficiency := projectedVehicleWaitTotal + projectedCargoWaitTotal
	waitTransportRatio := 0.0
	if projectedTransportTotal > 0 {
		waitTransportRatio = totalWaitForEfficiency / projectedTransportTotal
	}

	shipmentLoadKg := 0.0
	if shipment != nil && cargo != nil {
		shipmentLoadKg = float64(cargo.Weight * shipment.Count)
	}
	projectedUsedCapacity := float64(stats.UtilizationStats.UsedCapacity) + shipmentLoadKg
	projectedRemainingCapacity := float64(stats.UtilizationStats.TotalCapacity) - projectedUsedCapacity
	projectedMinGap := minCapacityGapAfterAssign(stats.vehicleCapacityGapByVehicle, v, shipmentLoadKg)

	costA := weights.K1A*projectedVehicleWaitTotal + weights.K2A*projectedTotalEmpty
	costB := weights.K1B*projectedWeightedCargoWait + weights.K2B*projectedSlowestCargoWait
	costC := 0.0
	if projectedTotalDistance > 0 {
		costC += weights.K1C * (projectedTotalEmpty / projectedTotalDistance)
	}
	costC += weights.K2C*waitTransportRatio + weights.K3C*projectedLongestVehicleWait
	costD := weights.K1D*projectedRemainingCapacity + weights.K2D*projectedMinGap

	// 白板里 E 的“基础成本(A/B/C/D)”没有指定唯一底座，这里取 A-D 的均值作为风险底座。
	baseCost := (costA + costB + costC + costD) / 4
	costE := baseCost +
		weights.R1*stats.VehicleWaitStats.StdDev +
		weights.R2*stats.CargoWaitStats.StdDev +
		weights.R3*stats.TransportTimeStats.StdDev

	costF := weights.Lambda1*costA +
		weights.Lambda2*costB +
		weights.Lambda3*costC +
		weights.Lambda4*costD +
		weights.Lambda5*costE

	if Logger.Logger != nil {
		Logger.Logger.Debug("Cost Calculation Breakdown",
			zap.Uint("VehicleID", v.Id),
			zap.Float64("CostA", costA),
			zap.Float64("CostB", costB),
			zap.Float64("CostC", costC),
			zap.Float64("CostD", costD),
			zap.Float64("CostE", costE),
			zap.Float64("TotalCostF", costF),
			zap.Float64("ProjectedVehicleWaitTotal", projectedVehicleWaitTotal),
			zap.Float64("ProjectedWeightedCargoWait", projectedWeightedCargoWait),
			zap.Float64("ProjectedTotalEmpty", projectedTotalEmpty),
			zap.Float64("ProjectedRemainingCapacity", projectedRemainingCapacity),
		)
	}

	return CostBreakdown{
		A: costA,
		B: costB,
		C: costC,
		D: costD,
		E: costE,
		F: costF,
	}
}

func normalizeStats(stats *Statistics) *Statistics {
	if stats != nil {
		if stats.vehicleWaitByVehicle == nil {
			stats.vehicleWaitByVehicle = make(map[uint]float64)
		}
		if stats.cargoWaitByShipment == nil {
			stats.cargoWaitByShipment = make(map[uint]float64)
		}
		if stats.cargoWeightTonsByShipment == nil {
			stats.cargoWeightTonsByShipment = make(map[uint]float64)
		}
		if stats.vehicleCapacityGapByVehicle == nil {
			stats.vehicleCapacityGapByVehicle = make(map[uint]float64)
		}
		return stats
	}
	return &Statistics{
		vehicleWaitByVehicle:        make(map[uint]float64),
		cargoWaitByShipment:         make(map[uint]float64),
		cargoWeightTonsByShipment:   make(map[uint]float64),
		vehicleCapacityGapByVehicle: make(map[uint]float64),
	}
}

func maxValueExcluding(values map[uint]float64, exclude uint) float64 {
	maxValue := 0.0
	for id, value := range values {
		if id == exclude {
			continue
		}
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func minCapacityGapAfterAssign(values map[uint]float64, v *model.Vehicle, shipmentLoadKg float64) float64 {
	if len(values) == 0 {
		if v == nil {
			return 0
		}
		return float64(v.Capacity-v.Size) - shipmentLoadKg
	}

	minGap := math.MaxFloat64
	for id, gap := range values {
		if v != nil && id == v.Id {
			gap -= shipmentLoadKg
		}
		if gap < minGap {
			minGap = gap
		}
	}
	if minGap == math.MaxFloat64 {
		return 0
	}
	return minGap
}

// haversine 计算两点间的球面距离 (单位: 米)
func haversine(lon1, lat1, lon2, lat2 float64) float64 {
	const R = 6371000 // 地球半径
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	deltaPhi := (lat2 - lat1) * math.Pi / 180
	deltaLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
