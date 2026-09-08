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
}

// Output 数据生成器输出
type Output struct {
	Orders []*model.ExperimentOrder
	Events []*model.ExperimentEvent
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
	out.Orders = generateOrders(rng, in)
	out.Events = generateEvents(rng, in)
	return out
}

// generateOrders 生成订单流：泊松(指数间隔)或均匀分布到达，起终点/货类/件数随机。
func generateOrders(rng *rand.Rand, in Input) []*model.ExperimentOrder {
	if len(in.PoiIDs) < 2 || len(in.CargoIDs) == 0 {
		return nil
	}
	durationSec := in.DurationHours * 3600
	daily := in.Orders.DailyVolume
	if daily <= 0 {
		daily = 200
	}
	meanIntervalSec := 24 * 3600 / float64(daily) // 泊松平均间隔(虚拟秒)

	orders := make([]*model.ExperimentOrder, 0, int(float64(daily)*in.DurationHours/24*1.2)+1)
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
	return orders
}

// generateEvents 生成天气/车祸事件脚本：逐 tick 按概率摇号（预生成，运行期不再随机）。
// 注：脚本事件使用基准概率（不含司机等级修正），等级修正仅作用于运行期拒单判定。
func generateEvents(rng *rand.Rand, in Input) []*model.ExperimentEvent {
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
			if in.Events.AccidentProbPerTick > 0 && rng.Float64() < in.Events.AccidentProbPerTick {
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
