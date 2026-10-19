// Package gen 沙箱数据生成器：在仿真系统外部，按"沙箱世界规则 + 实验参数 + 种子"
// 确定性预生成本次实验的输入数据集（订单流 + 随机事件脚本）。
//
// 严格复现保证：生成器内部使用独立 rand.New(rand.NewSource(seed))，纯函数无 IO/时钟依赖；
// 同种子 + 同参数 + 同生成器版本 ⇒ 输出逐字节一致（有单测断言）。
package gen

import (
	"math"
	"math/rand"
	"time"

	"github.com/kiritosuki/mover/internal/model"
	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

// Input 数据生成器输入
type Input struct {
	ExperimentID  uint
	Seed          int64
	DurationHours float64
	StartTime     time.Time            // 虚拟时钟起点
	TickSeconds   int                  // 虚拟步长（事件按 tick 对齐）
	Orders        sandboxpkg.OrderWorldRules // 订单生成模型（已合并实验覆盖）
	Events        sandboxpkg.RandomEventRules // 随机事件概率模型（已合并实验覆盖）
	VehicleLicenses []string           // 本实验实例化车队车牌（事件目标）
	PoiIDs        []uint               // 可用 POI id 集合（白名单过滤后）
	CargoIDs      []uint               // 货类 id 集合
	Chains        []ChainInput         // 已解析的供应链需求链条（空=不生成链单）
}

// ChainInput 已解析的供应链（由调用方按世界规则解析：货类按名、POI 按类型+关键词）。
type ChainInput struct {
	Name  string          // 链条标识
	Share float64         // 根订单占日均单量比例(0-1)
	Steps []ChainStepInput // 有序运输段：步骤k = 节点k 的货物发往节点k+1
}

// ChainStepInput 一段链条运输：节点k 发货 → 节点k+1 收货并在其处加工（α/β/L）后发出下一段。
type ChainStepInput struct {
	CargoId     uint    // 本段货类
	CargoWeight int     // 本段货类单件重量(kg)，吨位→件数折算
	FromPoiIDs  []uint  // 发货点候选
	ToPoiIDs    []uint  // 收货点候选
	YieldWeight float64 // α：本段吨位 = 上段吨位 × α（首段=1）
	TriggerProb float64 // β：上段订单触发本段补货的概率（首段=1）
	DelaySec    int     // L：加工延迟(虚拟秒)，本段释放时刻 = 上段释放时刻 + L
	NodeLabel   string  // 发货节点标签（chain_name/node_label 统计口径）
}

// Output 数据生成器输出
type Output struct {
	Orders []*model.ExperimentOrder
	Events []*model.ExperimentEvent
	// DriverLevels 司机等级表（车牌→等级2-5）：数据生成期种子摇号产生，
	// 实例化建司机与车祸脚本概率修正共用同一份，保证全链路确定性一致。
	DriverLevels map[string]int
}

// Generate 确定性生成实验输入数据集。
func Generate(in Input) Output {
	rng := rand.New(rand.NewSource(in.Seed))
	out := Output{}
	if in.TickSeconds <= 0 {
		in.TickSeconds = 10
	}
	if in.DurationHours <= 0 {
		in.DurationHours = 24
	}
	// 摇号顺序固定（复现关键）：先司机等级，再订单流，再事件脚本
	out.DriverLevels = generateDriverLevels(rng, in.VehicleLicenses)
	out.Orders = generateOrders(rng, in)
	out.Events = generateEvents(rng, in, out.DriverLevels)
	return out
}

// generateDriverLevels 为车队逐车摇号司机等级（2~5），车牌顺序即 RNG 消费顺序。
func generateDriverLevels(rng *rand.Rand, licenses []string) map[string]int {
	levels := make(map[string]int, len(licenses))
	for _, license := range licenses {
		levels[license] = 2 + rng.Intn(4)
	}
	return levels
}

// generateOrders 生成订单流：基础随机单（泊松/均匀到达）+ 供应链链条单（根订单+级联）。
// RNG 消费序固定（复现关键）：先基础随机单循环（数量按 1-Σshare 缩放），
// 再按链条配置顺序逐链生成：根订单泊松到达（起终点/件数摇号）→ 逐级级联
// （β 摇号 → 释放时刻 = 上段时刻 + L → 件数 = 上段吨位 × α / 本段单件重量）。
func generateOrders(rng *rand.Rand, in Input) []*model.ExperimentOrder {
	if len(in.PoiIDs) < 2 || len(in.CargoIDs) == 0 {
		return nil
	}
	durationSec := in.DurationHours * 3600
	daily := in.Orders.DailyVolume
	if daily <= 0 {
		daily = 200
	}

	// 链条根订单份额合计：基础随机单等量缩减（总单量口径保持稳定）
	shareSum := 0.0
	for _, chain := range in.Chains {
		if chain.Share > 0 && len(chain.Steps) > 0 {
			shareSum += chain.Share
		}
	}
	if shareSum >= 1 {
		shareSum = 0.99
	}
	baseDaily := int(math.Round(float64(daily) * (1 - shareSum)))
	if baseDaily < 0 {
		baseDaily = 0
	}
	meanIntervalSec := 24 * 3600 / float64(maxInt(1, baseDaily)) // 泊松平均间隔(虚拟秒)

	orders := make([]*model.ExperimentOrder, 0, int(float64(daily)*in.DurationHours/24*1.4)+1)
	cursor := 0.0
	for {
		// 下一单到达间隔
		var gap float64
		if in.Orders.ArrivalModel == "uniform" {
			gap = meanIntervalSec
		} else { // 默认 poisson：指数分布间隔
			gap = -math.Log(1-rng.Float64()) * meanIntervalSec
		}
		cursor += gap
		if cursor >= durationSec {
			break
		}

		startIdx := rng.Intn(len(in.PoiIDs))
		endIdx := rng.Intn(len(in.PoiIDs) - 1)
		if endIdx >= startIdx {
			endIdx++
		}

		count := in.Orders.CountRange[0]
		if in.Orders.CountRange[1] > in.Orders.CountRange[0] {
			count += rng.Intn(in.Orders.CountRange[1] - in.Orders.CountRange[0] + 1)
		}

		orders = append(orders, &model.ExperimentOrder{
			ExperimentId: in.ExperimentID,
			VirtualTime:  in.StartTime.Add(time.Duration(cursor) * time.Second),
			StartPoiId:   in.PoiIDs[startIdx],
			EndPoiId:     in.PoiIDs[endIdx],
			CargoId:      in.CargoIDs[rng.Intn(len(in.CargoIDs))],
			Count:        count,
		})
	}

	// 供应链链条单：根订单 + 加工级联
	for _, chain := range in.Chains {
		generateChainOrders(rng, in, chain, durationSec, daily, &orders)
	}
	return orders
}

// chainSpawned 级联中的一个在途触发源（某段订单的释放时刻与吨位）。
type chainSpawned struct {
	stepIdx  int
	atSec    float64
	weightKg float64
}

// generateChainOrders 生成单条链条的根订单与逐级级联订单（确定性 RNG 消费序）。
func generateChainOrders(rng *rand.Rand, in Input, chain ChainInput, durationSec float64, daily int, orders *[]*model.ExperimentOrder) {
	if chain.Share <= 0 || len(chain.Steps) == 0 {
		return
	}
	rootCount := int(math.Round(float64(daily) * chain.Share * in.DurationHours / 24))
	if rootCount <= 0 {
		return
	}
	rootInterval := durationSec / float64(rootCount)

	// 根订单：泊松到达（与基础单同一间隔模型），起终点/件数在链内候选摇号
	cursor := 0.0
	frontier := make([]chainSpawned, 0, rootCount)
	for i := 0; i < rootCount; i++ {
		var gap float64
		if in.Orders.ArrivalModel == "uniform" {
			gap = rootInterval
		} else {
			gap = -math.Log(1-rng.Float64()) * rootInterval
		}
		cursor += gap
		if cursor >= durationSec {
			break
		}
		step := chain.Steps[0]
		count := pickCount(rng, in)
		weight := float64(count * step.CargoWeight)
		*orders = append(*orders, &model.ExperimentOrder{
			ExperimentId: in.ExperimentID,
			VirtualTime:  in.StartTime.Add(time.Duration(cursor) * time.Second),
			StartPoiId:   step.FromPoiIDs[rng.Intn(len(step.FromPoiIDs))],
			EndPoiId:     step.ToPoiIDs[rng.Intn(len(step.ToPoiIDs))],
			CargoId:      step.CargoId,
			Count:        count,
			ChainName:    chain.Name,
			NodeLabel:    step.NodeLabel,
		})
		frontier = append(frontier, chainSpawned{stepIdx: 0, atSec: cursor, weightKg: weight})
	}

	// 逐级级联：上段释放后经加工延迟 L，以概率 β 触发下段补货，吨位按 α 折算
	for stepIdx := 1; stepIdx < len(chain.Steps); stepIdx++ {
		step := chain.Steps[stepIdx]
		next := make([]chainSpawned, 0, len(frontier))
		for _, f := range frontier {
			if step.TriggerProb <= 0 || step.TriggerProb < 1 && rng.Float64() >= step.TriggerProb {
				continue
			}
			at := f.atSec + float64(step.DelaySec)
			if at >= durationSec {
				continue // 加工完成时仿真已结束
			}
			count := int(math.Round(f.weightKg * step.YieldWeight / float64(maxInt(1, step.CargoWeight))))
			if count < 1 {
				continue // 加工余量不足一单位，不触发补货运输
			}
			weight := float64(count * step.CargoWeight)
			*orders = append(*orders, &model.ExperimentOrder{
				ExperimentId: in.ExperimentID,
				VirtualTime:  in.StartTime.Add(time.Duration(at) * time.Second),
				StartPoiId:   step.FromPoiIDs[rng.Intn(len(step.FromPoiIDs))],
				EndPoiId:     step.ToPoiIDs[rng.Intn(len(step.ToPoiIDs))],
				CargoId:      step.CargoId,
				Count:        count,
				ChainName:    chain.Name,
				NodeLabel:    step.NodeLabel,
			})
			next = append(next, chainSpawned{stepIdx: stepIdx, atSec: at, weightKg: weight})
		}
		frontier = next
	}
}

// pickCount 按世界规则的件数区间摇号（与基础单同一口径）。
func pickCount(rng *rand.Rand, in Input) int {
	count := in.Orders.CountRange[0]
	if in.Orders.CountRange[1] > in.Orders.CountRange[0] {
		count += rng.Intn(in.Orders.CountRange[1] - in.Orders.CountRange[0] + 1)
	}
	return count
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// generateEvents 生成天气/车祸事件脚本：逐 tick 按概率摇号（预生成，运行期不再随机）。
// 车祸概率带司机等级修正（与运行期拒单判定同一套 level_factor 口径）；
// 天气为环境因素，不做等级修正。
func generateEvents(rng *rand.Rand, in Input, levels map[string]int) []*model.ExperimentEvent {
	if len(in.VehicleLicenses) == 0 {
		return nil
	}
	totalTicks := int64(in.DurationHours*3600) / int64(in.TickSeconds)
	events := make([]*model.ExperimentEvent, 0)
	for tick := int64(0); tick < totalTicks; tick++ {
		for _, license := range in.VehicleLicenses {
			virtualNow := in.StartTime.Add(time.Duration(tick*int64(in.TickSeconds)) * time.Second)
			if in.Events.WeatherProbPerTick > 0 && rng.Float64() < in.Events.WeatherProbPerTick {
				duration := rangeSeconds(rng, in.Events.WeatherDelaySeconds)
				events = append(events, &model.ExperimentEvent{
					ExperimentId:    in.ExperimentID,
					VirtualTime:     virtualNow,
					EventType:       "weather",
					VehicleLicense:  license,
					Severity:        2,
					DurationSeconds: duration,
					Detail:          "{}",
				})
			}
			// 车祸：司机等级修正（等级越高越稳，系数越小），与 random_event 的 levelFactor 口径一致
			lf := 1.0
			if level, ok := levels[license]; ok && level >= 1 && level <= 5 {
				lf = in.Events.LevelFactor[level-1]
			}
			if in.Events.AccidentProbPerTick > 0 && rng.Float64() < in.Events.AccidentProbPerTick*lf {
				duration := rangeSeconds(rng, in.Events.MaintenanceSeconds)
				events = append(events, &model.ExperimentEvent{
					ExperimentId:    in.ExperimentID,
					VirtualTime:     virtualNow,
					EventType:       "accident",
					VehicleLicense:  license,
					Severity:        4,
					DurationSeconds: duration,
					Detail:          "{}",
				})
			}
		}
	}
	return events
}

// rangeSeconds 区间内取整秒（确定性：使用生成器 RNG）。
func rangeSeconds(rng *rand.Rand, bounds [2]float64) int {
	if bounds[1] <= bounds[0] {
		return int(bounds[0])
	}
	return int(bounds[0] + rng.Float64()*(bounds[1]-bounds[0]))
}
