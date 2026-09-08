package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/repository"
	"github.com/kiritosuki/mover/internal/result"
	"github.com/kiritosuki/mover/internal/task"
	"go.uber.org/zap"
)

// ListVehicles godoc
// @Summary 查询/筛选车辆列表
// @Tags Vehicle
// @Accept json
// @Produce json
// @Param status query int false "vehicle 状态"
// @Success 200 {object} result.Result{data=[]model.Vehicle}
// @Failure 400 {object} result.Result
// @Router /vehicles [get]
//
// ListVehicles 处理客户端获取车辆信息的请求。
// 支持按照 status 过滤查询数据库中各个状态的车辆详情。
// 该请求最终返回所有指定状态（或默认查状态0）的车队概览信息包装后的 VehicleView。
func ListVehicles(c *gin.Context) {
	Logger.Logger.Info("查询/筛选车辆列表...")
	statusStr := c.DefaultQuery("status", "0")
	status, err := strconv.Atoi(statusStr)
	if err != nil {
		result.Fail(c, "status需要为整数类型", err)
		return
	}
	Logger.Logger.Info("查询条件", zap.Any("status", status))
	// 沙箱系统：优先返回活跃实验的实例车辆；无运行中实验时回落沙箱模板车队(experiment_id=0)。
	// 也支持显式 experiment_id 查询历史实验车辆。
	expID := task.CurrentExperimentID()
	if expStr := c.Query("experiment_id"); expStr != "" {
		if v, parseErr := strconv.Atoi(expStr); parseErr == nil && v >= 0 {
			expID = uint(v)
		}
	}
	vehicles, err := repository.ListVehiclesByExp(expID, status)
	if err != nil {
		result.Fail(c, "数据库查询错误！", err)
		return
	}
	// 查询成功
	Logger.Logger.Info("查询/筛选车辆列表成功", zap.Any("vehicles", vehicles))
	result.Success(c, repository.BuildVehicleViews(vehicles))
}
