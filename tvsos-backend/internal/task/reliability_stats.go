package task

import (
	"time"

	"github.com/kiritosuki/mover/internal/repository"
)

// ---- 报告 R1-R3：可靠性指标（MTBF / MTTR / 服务可用性）----
// 观测窗口：自最近一次服务启动（system_event.startup）至今；
// 故障口径：车祸（accident）导致车辆进入维修，其发生时刻与维修时长
// 直接取自 random_event 表的 event_time / duration_seconds。

type reliabilityMetrics struct {
	ObservationHours float64 // 观测窗口时长（小时）
	FaultCount       int64   // 窗口内故障（车祸）次数
	MtbfHours        float64 // R1 平均无故障时间（小时）
	MttrHours        float64 // R2 平均修复时长（小时）
	Availability     float64 // R3 服务可用性（百分比 0-100）
}

// observationWindowStart 返回可靠性/吞吐类指标的观测起点：最近一次 startup 事件时刻。
// 无 startup 记录时退化为当前时刻（窗口为零，指标无数据）。
func observationWindowStart(now time.Time) time.Time {
	if evs, err := repository.ListSystemEvents("startup", 1); err == nil && len(evs) > 0 && evs[0] != nil {
		return evs[0].EventTime
	}
	return now
}

// calculateReliabilityMetrics 计算车队级可靠性指标。
// fleetSize 为车辆总数（可用运力池的分母）；windowStart 为观测起点。
func calculateReliabilityMetrics(fleetSize int, windowStart time.Time, now time.Time) reliabilityMetrics {
	m := reliabilityMetrics{}
	windowSec := now.Sub(windowStart).Seconds()
	if windowSec <= 0 {
		return m
	}
	m.ObservationHours = windowSec / 3600.0

	accidents, err := repository.ListRandomEventsByType("accident", 1000)
	if err != nil {
		return m
	}

	downtimeSec := 0.0
	for _, ev := range accidents {
		if ev == nil || ev.EventTime.Before(windowStart) {
			continue // 只统计本次运行观测窗口内的故障
		}
		m.FaultCount++
		downtimeSec += float64(ev.DurationSeconds)
	}

	if m.FaultCount > 0 {
		m.MtbfHours = m.ObservationHours / float64(m.FaultCount)
		m.MttrHours = downtimeSec / float64(m.FaultCount) / 3600.0
	}

	m.Availability = 100.0
	if fleetSize > 0 && downtimeSec > 0 {
		m.Availability = 100.0 * (1.0 - downtimeSec/(float64(fleetSize)*windowSec))
		if m.Availability < 0 {
			m.Availability = 0
		}
	}
	return m
}
