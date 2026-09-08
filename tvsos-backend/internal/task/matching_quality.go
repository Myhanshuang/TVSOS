package task

import (
	"encoding/json"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/kiritosuki/mover/internal/model"
)

// ---- 报告 M1-M4 / C1：匹配精准度与基线对比 ----
// 参考答案(ground truth)口径：**条件最优** —— 固定 SA 其余指派不变，将该运单
// 改派给任一车辆后的整批成本最低者，即该运单在当前批次语境下的单条件最优车。
// 全局精确枚举在 20 辆车队下组合爆炸(见 gap_evaluator 的规模限制)，
// 而条件枚举为 O(单数×车数) 次成本评估，任意批次规模均可实时计算。
// 基线(C1)：贪心首可行策略(按车辆列表顺序取第一辆可行车，不可行强指派首车由成本惩罚体现)。

const (
	// maxQualityShipments / maxQualityVehicles 匹配质量评估的规模上限，防止评估拖慢派单
	maxQualityShipments = 30
	maxQualityVehicles  = 30
	// qualityRankingLimit 候选排名 JSON 记录的候选数上限
	qualityRankingLimit = 8
)

// shipmentQuality 单运单的匹配质量评估结果(供 matching_log 回填)
type shipmentQuality struct {
	RankingJSON   string // 成本升序候选排名 JSON(vehicle_id/cost)
	GroundTruthID uint   // 条件最优车辆 ID(排名首位)
	ChosenRank    int    // SA 实际选中车辆在排名中的位次(1 起)
}

// matchingQualityResult 一批运单的质量评估结果集合
type matchingQualityResult struct {
	ByShipment map[uint]*shipmentQuality
}

// 匹配质量滚动累计：自进程启动(与 A2/R1-R3 同观测窗口，重启后重新累计)
var (
	qualityMu           sync.Mutex
	qualityEvaluated    int64   // 已评估的已派单运单数
	qualityRank1        int64   // 选中车恰为条件最优车的次数
	qualityRecallTotal  int64   // 召回分母：批次运单总数(含未派单)
	qualityRecallHit    int64   // 召回分子：派单且选中条件最优车
	qualityRecipSum     float64 // MRR 倒数排名累计
	lastBaselineImprove float64 // C1 最近完整批次的成本改善百分比
)

// ResetMatchingQuality 清空匹配质量滚动累计（实验启动时调用，保证 M1-M4/C1 按实验窗口统计）。
func ResetMatchingQuality() {
	qualityMu.Lock()
	qualityEvaluated, qualityRank1, qualityRecallTotal, qualityRecallHit = 0, 0, 0, 0
	qualityRecipSum, lastBaselineImprove = 0, 0
	qualityMu.Unlock()
}

// GetMatchingQualityStats 返回 M1-M4 滚动指标(P/R/F1 为百分比)与已评估运单数。
func GetMatchingQualityStats() (precision, recall, f1, mrr float64, evaluated int64) {
	qualityMu.Lock()
	defer qualityMu.Unlock()
	evaluated = qualityEvaluated
	if evaluated > 0 {
		precision = float64(qualityRank1) / float64(evaluated) * 100
		mrr = qualityRecipSum / float64(evaluated)
	}
	if qualityRecallTotal > 0 {
		recall = float64(qualityRecallHit) / float64(qualityRecallTotal) * 100
	}
	if precision > 0 || recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	return
}

// GetLastBaselineImprovement 返回 C1：最近一次完整批次 SA 相对贪心基线的成本改善(百分比)。
func GetLastBaselineImprovement() float64 {
	qualityMu.Lock()
	defer qualityMu.Unlock()
	return lastBaselineImprove
}

// qualityRankEntry 候选排名 JSON 条目(与 matching_log.candidate_ranking 对应)
type qualityRankEntry struct {
	VehicleID uint    `json:"vehicle_id"`
	Cost      float64 `json:"cost"`
}

type qualityCand struct {
	idx  int
	cost float64
}

// evaluateMatchingQuality 对一个 SA 批次做后验匹配质量评估：
// 1) 逐运单条件枚举所有车辆的整批成本，得到成本升序候选排名与条件最优车(ground truth)；
// 2) 累计 M1 精确率(选中=条件最优)/M2 召回率(未派单计未命中)/M4 MRR(选中车排名倒数)；
// 3) 完整指派的批次额外与贪心首可行基线对比，更新 C1 改善百分比。
// 返回逐运单结果供 matching_log 回填；规模超限或统计快照缺失时返回 nil(跳过评估)。
func evaluateMatchingQuality(
	shipments []*model.Shipment,
	vehicles []*model.Vehicle,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	projectedLoads map[uint]int,
	pois map[uint]*model.Poi,
	saAssignment map[uint]*model.Vehicle,
	now time.Time,
) *matchingQualityResult {
	if stats == nil || len(shipments) == 0 || len(shipments) > maxQualityShipments ||
		len(vehicles) == 0 || len(vehicles) > maxQualityVehicles || saAssignment == nil {
		return nil
	}

	saIdx := make([]int, len(shipments))
	complete := true
	for i, ship := range shipments {
		v := saAssignment[ship.Id]
		if v == nil {
			complete = false // 存在未派单运单，召回分母包含它
			saIdx[i] = -1
			continue
		}
		saIdx[i] = vehicleIndexByID(vehicles, v.Id)
		if saIdx[i] < 0 {
			return nil
		}
	}

	byShipment := make(map[uint]*shipmentQuality, len(shipments))
	var evaluated, rank1 int64
	recipSum := 0.0
	variant := make([]int, len(shipments))

	for i, ship := range shipments {
		if saIdx[i] < 0 {
			continue // 未派单运单无选中排名，仅计入召回分母
		}
		copy(variant, saIdx)
		cands := make([]qualityCand, 0, len(vehicles))
		for vi := range vehicles {
			variant[i] = vi
			cands = append(cands, qualityCand{vi, evaluateStateCost(variant, shipments, vehicles, stats, cargoCache, projectedLoads, pois, now)})
		}
		sort.Slice(cands, func(a, b int) bool { return cands[a].cost < cands[b].cost })

		chosenRank := 0
		for ri, c := range cands {
			if c.idx == saIdx[i] {
				chosenRank = ri + 1
				break
			}
		}
		if chosenRank == 0 {
			continue
		}

		entries := make([]qualityRankEntry, 0, qualityRankingLimit)
		for _, c := range cands {
			if len(entries) >= qualityRankingLimit {
				break
			}
			entries = append(entries, qualityRankEntry{vehicles[c.idx].Id, math.Round(c.cost*100) / 100})
		}
		js, err := json.Marshal(entries)
		if err != nil {
			continue
		}
		byShipment[ship.Id] = &shipmentQuality{
			RankingJSON:   string(js),
			GroundTruthID: vehicles[cands[0].idx].Id,
			ChosenRank:    chosenRank,
		}
		evaluated++
		if chosenRank == 1 {
			rank1++
		}
		recipSum += 1.0 / float64(chosenRank)
	}

	// C1：完整指派批次与贪心首可行基线对比
	blImprove, blOK := 0.0, false
	if complete {
		blImprove, blOK = evaluateBaselineImprovement(saIdx, shipments, vehicles, stats, cargoCache, projectedLoads, pois, now)
	}

	qualityMu.Lock()
	qualityEvaluated += evaluated
	qualityRank1 += rank1
	qualityRecallTotal += int64(len(shipments))
	qualityRecallHit += rank1
	qualityRecipSum += recipSum
	if blOK {
		lastBaselineImprove = blImprove
	}
	qualityMu.Unlock()

	return &matchingQualityResult{ByShipment: byShipment}
}

// evaluateBaselineImprovement 计算贪心首可行基线相对 SA 的成本改善百分比。
// 基线：按运单顺序取车辆列表中第一辆可行车(不可行强指派首车，由 SAPenaltyInf 体现)；
// 任一方成本含不可行惩罚或非正时跳过(返回 ok=false)。
func evaluateBaselineImprovement(
	saIdx []int,
	shipments []*model.Shipment,
	vehicles []*model.Vehicle,
	stats *Statistics,
	cargoCache map[uint]*model.Cargo,
	projectedLoads map[uint]int,
	pois map[uint]*model.Poi,
	now time.Time,
) (float64, bool) {
	blIdx := make([]int, len(shipments))
	blLoads := make(map[uint]int, len(vehicles))
	for _, v := range vehicles {
		blLoads[v.Id] = projectedLoads[v.Id]
	}
	for i, ship := range shipments {
		cargo := getCargoByCache(cargoCache, ship.CargoId)
		chosen := 0
		for vi, v := range vehicles {
			if isVehicleFeasible(v, ship, cargo, blLoads[v.Id]) {
				chosen = vi
				break
			}
		}
		blIdx[i] = chosen
		if cargo != nil {
			blLoads[vehicles[chosen].Id] += cargo.Weight * ship.Count
		}
	}

	saCost := evaluateStateCost(saIdx, shipments, vehicles, stats, cargoCache, projectedLoads, pois, now)
	blCost := evaluateStateCost(blIdx, shipments, vehicles, stats, cargoCache, projectedLoads, pois, now)
	if saCost <= 0 || blCost <= 0 || saCost >= SAPenaltyInf || blCost >= SAPenaltyInf {
		return 0, false
	}
	return (blCost - saCost) / blCost * 100.0, true
}
