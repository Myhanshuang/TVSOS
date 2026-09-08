package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/result"
	"github.com/kiritosuki/mover/internal/task"
)

// GetRealtimeReport 收集并返回当前活跃实验的实时调度运行状态和总计指标快照报告。
// 该数据服务于前端的数据大屏组件 (如统计图表与饼图等)，由运行时统计算法层组装并返回。
// 沙箱系统：无运行中实验时返回失败提示（统计口径均为实验作用域 + 虚拟时钟）。
func GetRealtimeReport(c *gin.Context) {
	rt := task.CurrentRuntime()
	if rt == nil {
		result.Fail(c, "当前没有运行中的实验", errors.New("no active experiment"))
		return
	}
	snapshot, err := task.BuildRealtimeDashboard(rt)
	if err != nil {
		result.Fail(c, "统计快照生成失败", err)
		return
	}
	result.Success(c, snapshot)
}
