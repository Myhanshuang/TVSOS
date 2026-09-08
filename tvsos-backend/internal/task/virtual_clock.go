package task

import (
	"sync/atomic"
	"time"
)

// VirtualClock 虚拟仿真时钟（严格复现的根基）：
// 虚拟时间 = 起点 + tick数 × 步长，只随主循环 tick 推进，与真实时钟解耦。
// 时间加速比（speed）只改变真实世界中 tick 的触发频率，不改变虚拟时间序列本身。
type VirtualClock struct {
	start       time.Time
	tickSeconds int64
	tick        atomic.Int64
	totalTicks  int64 // 0=不限
}

// NewVirtualClock 创建虚拟时钟。startText 形如 "2006-01-02 15:04:05"。
func NewVirtualClock(startText string, tickSeconds int, durationHours float64) *VirtualClock {
	start, err := time.ParseInLocation("2006-01-02 15:04:05", startText, time.Local)
	if err != nil {
		start = time.Now()
	}
	if tickSeconds <= 0 {
		tickSeconds = 10
	}
	c := &VirtualClock{start: start, tickSeconds: int64(tickSeconds)}
	if durationHours > 0 {
		c.totalTicks = int64(durationHours*3600) / int64(tickSeconds)
	}
	return c
}

// Now 当前虚拟时间
func (c *VirtualClock) Now() time.Time {
	return c.start.Add(time.Duration(c.tick.Load()*c.tickSeconds) * time.Second)
}

// Advance 推进一个虚拟 tick，返回推进后的 tick 数
func (c *VirtualClock) Advance() int64 {
	return c.tick.Add(1)
}

// Tick 当前 tick 数
func (c *VirtualClock) Tick() int64 {
	return c.tick.Load()
}

// TickSeconds 虚拟步长（秒）
func (c *VirtualClock) TickSeconds() int64 {
	return c.tickSeconds
}

// Start 虚拟时钟起点时刻（可靠性/吞吐指标的观测窗口起点）
func (c *VirtualClock) Start() time.Time {
	return c.start
}

// Expired 是否已到达固定虚拟仿真时长（实验结束条件）
func (c *VirtualClock) Expired() bool {
	return c.totalTicks > 0 && c.tick.Load() >= c.totalTicks
}

// Progress 虚拟时钟进度 0~1（totalTicks=0 时恒为 0）
func (c *VirtualClock) Progress() float64 {
	if c.totalTicks <= 0 {
		return 0
	}
	p := float64(c.tick.Load()) / float64(c.totalTicks)
	if p > 1 {
		return 1
	}
	return p
}

// AfterVirtual 从当前虚拟时间起 afterSec 秒后的虚拟时刻（用于 pause_until 等暂停截止）
func (c *VirtualClock) AfterVirtual(afterSec float64) time.Time {
	return c.Now().Add(time.Duration(afterSec) * time.Second)
}
