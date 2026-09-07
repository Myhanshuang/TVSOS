package task

import "sync"

// ---- 报告 A2：单批求解耗时 ----
// 在派单批处理中对 SA 求解器计时（毫秒），内存累计均值/峰值，
// 通过 GetSolverTimingStats 暴露给实时看板。进程重启后重新累计。

var (
	solverTimingMu sync.Mutex
	solverBatches  int
	solverTotalMs  float64
	solverMaxMs    float64
)

// RecordSolverTiming 记录一次批次求解耗时（elapsedMs 为毫秒，负值按 0 处理）。
func RecordSolverTiming(elapsedMs float64) {
	if elapsedMs < 0 {
		elapsedMs = 0
	}
	solverTimingMu.Lock()
	solverBatches++
	solverTotalMs += elapsedMs
	if elapsedMs > solverMaxMs {
		solverMaxMs = elapsedMs
	}
	solverTimingMu.Unlock()
}

// GetSolverTimingStats 返回 (已计时批次数, 平均耗时ms, 峰值耗时ms)。
func GetSolverTimingStats() (int, float64, float64) {
	solverTimingMu.Lock()
	defer solverTimingMu.Unlock()
	if solverBatches == 0 {
		return 0, 0, 0
	}
	return solverBatches, solverTotalMs / float64(solverBatches), solverMaxMs
}
