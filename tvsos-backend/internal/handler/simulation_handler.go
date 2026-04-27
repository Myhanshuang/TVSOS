package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/result"
	"github.com/kiritosuki/mover/internal/task"
)

// GetSimulationSpeed 获取仿真倍速
func GetSimulationSpeed(c *gin.Context) {
	result.Success(c, task.GetSimulationSpeed())
}

// SetSimulationSpeed 设置仿真倍速
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
