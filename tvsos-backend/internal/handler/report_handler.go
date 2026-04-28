package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/result"
	"github.com/kiritosuki/mover/internal/task"
)

// GetRealtimeReport 收集并返回实时的全局调度运行状态和总计指标快照报告。
// 该数据服务于前端的数据大屏组件 (如统计图表与饼图等)，由运行时统计算法层组装并返回。
func GetRealtimeReport(c *gin.Context) {
	snapshot, err := task.BuildRealtimeDashboard()
	if err != nil {
		result.Fail(c, "统计快照生成失败", err)
		return
	}
	result.Success(c, snapshot)
}
