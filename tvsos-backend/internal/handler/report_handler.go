package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/result"
	"github.com/kiritosuki/mover/internal/task"
)

func GetRealtimeReport(c *gin.Context) {
	snapshot, err := task.BuildRealtimeDashboard()
	if err != nil {
		result.Fail(c, "统计快照生成失败", err)
		return
	}
	result.Success(c, snapshot)
}
