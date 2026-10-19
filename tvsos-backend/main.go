package main

import (
	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/router"
	"github.com/kiritosuki/mover/internal/task"
	_ "github.com/kiritosuki/mover/swagger-docs"
	"go.uber.org/zap"
)

// @title           Mover API
// @version         1.0
func main() {
	// 初始化全局日志对象 Logger
	Logger.InitLogger()
	Logger.Logger.Info("全局日志对象Logger加载成功")
	// 初始化全局 viper 对象并读取配置
	config.InitViper()
	// 数据库连接（含沙箱系统表与 experiment_id 维度列迁移）
	database.InitDB()
	// Redis 运行态连接（失败时自动降级为仅内存模式）
	database.InitRedis()
	// 沙箱系统：服务启动即中断残留实验（单实验串行，重启=abort，不做运行态恢复）
	task.AbortStaleExperiments()
	// 加载排放因子配置（绿色指标 G1/G2），无配置时使用默认估算因子
	task.LoadEmissionFactors()
	// 加载装卸时间系数配置（handling_config），无配置时保持默认 60 秒/吨
	task.LoadHandlingConfig()
	// 兼容迁移：为沙箱模板车辆(experiment_id=0)补绑司机
	task.EnsureDriversForVehicles()
	// 供应链链条货类幂等补种（吨位/方量趋势核算依赖）
	database.EnsureChainCargos()
	// 记录服务启动事件（expID=0 服务级；实验级 startup 由实验启动时记录）
	task.RecordSystemEvent(0, "startup", "info", "mover服务启动", "")

	// gin 框架 返回默认引擎
	r := gin.Default()

	// 注册路由
	router.SetupRouter(r)

	// 沙箱系统：仿真不再随进程常驻运行。
	// 派单/移动/事件/快照均由"实验运行时"(ExperimentRuntime)按虚拟时钟驱动，
	// 通过 POST /experiments/:id/start 启动、/stop 停止（见 experiment_runtime.go）。

	// 启动 mover 服务
	port := config.VP.GetString("server.port")
	err := r.Run("0.0.0.0:" + port)
	if err != nil {
		Logger.Logger.Error("服务启动失败！", zap.Error(err))
		panic(err)
	}
}
