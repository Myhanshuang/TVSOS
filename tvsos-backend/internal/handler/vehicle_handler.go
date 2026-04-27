package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/repository"
	"github.com/kiritosuki/mover/internal/result"
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
// 目前查询条件有 status
// TODO 后续可能有更新的查询条件
func ListVehicles(c *gin.Context) {
	Logger.Logger.Info("查询/筛选车辆列表...")
	statusStr := c.DefaultQuery("status", "0")
	status, err := strconv.Atoi(statusStr)
	if err != nil {
		result.Fail(c, "status需要为整数类型", err)
		return
	}
	Logger.Logger.Info("查询条件", zap.Any("status", status))
	// 查询数据库
	vehicles, err := repository.ListVehicles(status)
	if err != nil {
		result.Fail(c, "数据库查询错误！", err)
		return
	}
	// 查询成功
	Logger.Logger.Info("查询/筛选车辆列表成功", zap.Any("vehicles", vehicles))
	result.Success(c, repository.BuildVehicleViews(vehicles))
}
