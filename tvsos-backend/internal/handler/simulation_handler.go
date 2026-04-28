package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/result"
	"github.com/kiritosuki/mover/internal/task"
)

// GetSimulationSpeed 获取仿真倍速
// 处理客户端获取当前调度演练的时间倍频（如：x2, x5）的读取请求，
// 便于前端展示或同步状态。
func GetSimulationSpeed(c *gin.Context) {
	result.Success(c, task.GetSimulationSpeed())
}

// SetSimulationSpeed 设置仿真倍速
// 解析 URL 查询参数中的 multiplier（倍频数）并调用核心调度模块的加速方法，
// 主要用于让前端控制地图上车辆推演或发单速度。
func SetSimulationSpeed(c *gin.Context) {
	multiplierStr := c.Query("multiplier")
	if multiplierStr == "" {
		result.Fail(c, "参数multiplier不能为空", errors.New("multiplier is empty"))
		return
	}
	multiplier, err := strconv.ParseFloat(multiplierStr, 64)
	if err != nil || multiplier <= 0 {
		result.Fail(c, "参数multiplier必须为正数", errors.New("invalid multiplier"))
		return
	}
	task.SetSimulationSpeed(multiplier)
	result.Success(c, multiplier)
}
