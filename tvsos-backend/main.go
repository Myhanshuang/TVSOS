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
	// 数据库连接
	database.InitDB()
	// Redis 运行态连接（失败时自动降级为仅内存模式）
	database.InitRedis()
	// 恢复运行时任务与车辆状态，避免数据库状态与内存路线脱节
	task.RecoverRuntimeState()

	// gin 框架 返回默认引擎
	// 默认引擎注册了默认的 Logger() 和 Recovery
	// Logger(): 自动日志打印
	// Recovery(): 自处理 panic
	// 它的主要作用是注册路由
	r := gin.Default()

	// 注册路由
	router.SetupRouter(r)

	Logger.Logger.Info("mover服务启动成功")
	Logger.Logger.Info("启动任务创建go程...")
	go task.CreateOrderTask()
	Logger.Logger.Info("任务启动 done")
	Logger.Logger.Info("启动模拟车辆移动go程...")
	go task.SimulateMoving()
	Logger.Logger.Info("车辆移动 done")
	Logger.Logger.Info("启动实时统计推送go程...")
	go task.BroadcastStatsLoop()
	Logger.Logger.Info("实时统计推送 done")

	// 启动 mover 服务
	port := config.VP.GetString("server.port")
	// Run 的参数表示为 ip:port
	// ip: 服务运行的ip地址 准确说是监听设置哪些网卡可以收到请求 可以通过它限制访问范围
	// 如果写成 192.168.64.1 那么很显然只有局域网内的设备才能访问到这个ip 借此限制其他人访问
	// 如果写成 127.0.0.1 那么压缩成只有自己能访问
	// 0.0.0.0 表示允许别人通过任何形式的目标 ip 访问
	err := r.Run("0.0.0.0:" + port)
	if err != nil {
		Logger.Logger.Error("服务启动失败！", zap.Error(err))
		panic(err)
	}
}
