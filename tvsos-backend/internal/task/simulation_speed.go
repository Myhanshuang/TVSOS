package task

import (
	"math"
	"sync/atomic"
)

// 仿真倍速：有活跃实验运行时时委托给运行时（每实验独立），否则用进程级兜底值。
var simulationSpeedBits atomic.Uint64

func init() {
	simulationSpeedBits.Store(math.Float64bits(1.0))
}

// GetSimulationSpeed 当前时间加速比（优先取活跃实验运行时）
func GetSimulationSpeed() float64 {
	if rt := CurrentRuntime(); rt != nil {
		return rt.GetSpeed()
	}
	s := math.Float64frombits(simulationSpeedBits.Load())
	if s <= 0 {
		return 1.0
	}
	return s
}

// SetSimulationSpeed 设置时间加速比（作用于活跃实验；无实验时写兜底值）
func SetSimulationSpeed(multiplier float64) {
	if multiplier <= 0 {
		return
	}
	if rt := CurrentRuntime(); rt != nil {
		rt.SetSpeed(multiplier)
		return
	}
	simulationSpeedBits.Store(math.Float64bits(multiplier))
}
