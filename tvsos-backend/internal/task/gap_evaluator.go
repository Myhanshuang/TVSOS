package task

import (
	"math"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/model"
	"go.uber.org/zap"
)

// ---- 报告 A3：优化质量 Gap 评估（小规模精确枚举）----
// 当批次规模足够小时，通过穷举所有"货-车"指派方案，得到该问题在
// 当前成本口径(evaluateStateCost)下的全局最优代价，再与调度算法(SA)实际输出的
// 代价对比，得到优化 gap = (SA代价 - 最优代价) / 最优代价。
// 该指标是算法无关的（黑盒/结果层），SA 切换为全局最优算法后依然有效（报告 §3.7）。

const (
	// maxExactShipments / maxExactVehicles 限制精确枚举规模，防止组合爆炸
	maxExactShipments = 4
	maxExactVehicles  = 6
)

// lastOptimizationGap 最近一次小规模批次评估得到的优化 gap(百分比)。
// 通过 GetLastOptimizationGap 暴露给实时看板(A3)。
var lastOptimizationGap = 0.0

// GetLastOptimizationGap 返回最近一次批次级优化 gap 评估结果(百分比，0 表示无数据)。
func GetLastOptimizationGap() float64 {
	return lastOptimizationGap
}

// evaluateBatchOptimizationGap 对小规模批次执行精确枚举评估，返回 (SA代价, 最优代价, gap百分比)。
// 只有当批次规模与车队规模都小于阈值时才执行；否则返回 (0,0,0) 表示跳过。
func evaluateBatchOptimizationGap(
	shipments []*model.Shipment,
	vehicles []*model.Vehicle,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	projectedLoads map[uint]int,
	pois map[uint]*model.Poi,
	saAssignment map[uint]*model.Vehicle,
) (float64, float64, float64) {
	if len(shipments) == 0 || len(shipments) > maxExactShipments || len(vehicles) == 0 || len(vehicles) > maxExactVehicles {
		return 0, 0, 0
	}
	if saAssignment == nil || stats == nil {
		return 0, 0, 0
	}

	// 将 SA 结果映射回索引数组并评估其代价
	saIdx := make([]int, len(shipments))
	for i, ship := range shipments {
		v := saAssignment[ship.Id]
		if v == nil {
			return 0, 0, 0 // 存在未分配运单，无法公平对比
		}
		saIdx[i] = vehicleIndexByID(vehicles, v.Id)
		if saIdx[i] < 0 {
			return 0, 0, 0
		}
	}
	saCost := evaluateStateCost(saIdx, shipments, vehicles, stats, cargoCache, projectedLoads, pois)

	// 穷举所有指派组合求全局最优(深度受限的迭代枚举)
	bestCost := math.MaxFloat64
	found := false
	cur := make([]int, len(shipments))
	var dfs func(idx int)
	dfs = func(idx int) {
		if idx == len(shipments) {
			cost := evaluateStateCost(cur, shipments, vehicles, stats, cargoCache, projectedLoads, pois)
			if cost < bestCost {
				bestCost = cost
				found = true
			}
			return
		}
		for vIdx := 0; vIdx < len(vehicles); vIdx++ {
			cur[idx] = vIdx
			dfs(idx + 1)
		}
	}
	dfs(0)

	if !found || bestCost <= 0 || math.IsInf(bestCost, 0) || math.IsInf(saCost, 0) {
		return saCost, bestCost, 0
	}
	gap := (saCost - bestCost) / bestCost * 100.0
	if gap < 0 {
		gap = 0
	}
	return saCost, bestCost, gap
}

func vehicleIndexByID(vehicles []*model.Vehicle, id uint) int {
	for i, v := range vehicles {
		if v != nil && v.Id == id {
			return i
		}
	}
	return -1
}

// evaluateAndLogOptimizationGap 评估并记录批次优化 gap，供 A3 指标与日志使用。
func evaluateAndLogOptimizationGap(
	shipments []*model.Shipment,
	vehicles []*model.Vehicle,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	projectedLoads map[uint]int,
	pois map[uint]*model.Poi,
	saAssignment map[uint]*model.Vehicle,
) {
	if len(shipments) == 0 || len(shipments) > maxExactShipments || len(vehicles) > maxExactVehicles {
		return
	}
	saCost, bestCost, gap := evaluateBatchOptimizationGap(shipments, vehicles, stats, cargoCache, projectedLoads, pois, saAssignment)
	if gap > 0 || saCost > 0 {
		lastOptimizationGap = gap
		Logger.Logger.Info("A3 优化质量评估", zap.Int("shipments", len(shipments)), zap.Int("vehicles", len(vehicles)),
			zap.Float64("saCost", saCost), zap.Float64("optimalCost", bestCost), zap.Float64("gapPct", gap))
	}
}