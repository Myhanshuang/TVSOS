package task

import (
	"math"
	"sync/atomic"
)

// 使用原子变量保存仿真倍速，避免多个协程并发读写竞争。
var simulationSpeedBits atomic.Uint64

func init() {
	simulationSpeedBits.Store(math.Float64bits(1.0))
}

func GetSimulationSpeed() float64 {
	s := math.Float64frombits(simulationSpeedBits.Load())
	if s <= 0 {
		return 1.0
	}
	return s
}

func SetSimulationSpeed(multiplier float64) {
	if multiplier <= 0 {
		return
	}
	simulationSpeedBits.Store(math.Float64bits(multiplier))
}
