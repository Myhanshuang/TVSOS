package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/handler"
	"github.com/kiritosuki/mover/internal/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter 注册和初始化所有 HTTP 和 WebSocket 路由规则。
// 包含了 POI、车辆、运单、模拟调度控制、实时报表以及 Swagger 接口挂载。
func SetupRouter(r *gin.Engine) {
	// 认证接口（公开）：登录/注册；其余全部路由要求登录 token
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", handler.Login)
		authGroup.POST("/register", handler.Register)
	}

	// 以下路由均挂载鉴权中间件
	authed := r.Group("")
	authed.Use(middleware.AuthRequired())
	{
		authed.POST("/auth/logout", handler.Logout)
		authed.GET("/auth/profile", handler.Profile)
	}

	// r.Group() 返回 Group 对象 设置请求统一前缀
	// group.GET("", func)
	// 第一个参数拼接剩余请求路径 第二个参数是传递给哪个函数处理请求
	// 下面 {} 只是为了好看和规范

	// poi接口 - 兴趣点(如发货/收货仓库等)数据接口
	poiGroup := authed.Group("/pois")
	{
		poiGroup.GET("", handler.ListPois)
		poiGroup.GET("/:id", handler.GetPoi)
	}
	// 兼容旧前端路径
	legacyPoiGroup := authed.Group("/poi")
	{
		legacyPoiGroup.GET("", handler.ListPois)
		legacyPoiGroup.GET("/:id", handler.GetPoi)
	}

	// vehicle接口 - 返回当前所有车辆列表等基础信息
	vehicleGroup := authed.Group("/vehicles")
	{
		vehicleGroup.GET("", handler.ListVehicles)
	}

	// shipment接口 - 管理当前系统运单(列表；mock生成接口已被沙箱数据生成器取代并移除)
	shipmentGroup := authed.Group("/shipments")
	{
		shipmentGroup.GET("", handler.ListShipments)
	}

	// simulation接口 - 调度可视化演示模拟调速控制接口
	simulationGroup := authed.Group("/simulation")
	{
		simulationGroup.GET("/speed", handler.GetSimulationSpeed)
		simulationGroup.POST("/speed", handler.SetSimulationSpeed)
	}

	// report接口 - 获取调度大屏所需要的核心实时统计报表
	reportGroup := authed.Group("/report")
	{
		reportGroup.GET("/realtime", handler.GetRealtimeReport)
	}

	// 沙箱系统接口 - 沙箱/实验方案/实验(批次)/数据生成/生命周期/种子导入导出/对比
	sandboxGroup := authed.Group("/sandboxes")
	{
		sandboxGroup.GET("", handler.ListSandboxes)
		sandboxGroup.POST("", handler.CreateSandbox)
		sandboxGroup.GET("/:id", handler.GetSandbox)
		sandboxGroup.PUT("/:id", handler.UpdateSandbox)
		sandboxGroup.DELETE("/:id", handler.ArchiveSandbox)
		sandboxGroup.GET("/:id/schemes", handler.ListSchemes)
		sandboxGroup.POST("/:id/schemes", handler.CreateScheme)
		sandboxGroup.GET("/:id/experiments", handler.ListExperiments)
		sandboxGroup.POST("/:id/experiments", handler.CreateExperiment)
		sandboxGroup.POST("/:id/import", handler.ImportExperiment)
	}
	schemeGroup := authed.Group("/schemes")
	{
		schemeGroup.PUT("/:schemeId", handler.UpdateScheme)
		schemeGroup.DELETE("/:schemeId", handler.DeleteScheme)
	}
	experimentGroup := authed.Group("/experiments")
	{
		experimentGroup.GET("/:id", handler.GetExperiment)
		experimentGroup.PUT("/:id/params", handler.UpdateExperimentParams)
		experimentGroup.POST("/:id/generate", handler.GenerateExperimentData)
		experimentGroup.POST("/:id/start", handler.StartExperiment)
		experimentGroup.POST("/:id/stop", handler.StopExperiment)
		experimentGroup.GET("/:id/status", handler.GetExperimentStatus)
		experimentGroup.GET("/:id/export", handler.ExportExperiment)
		experimentGroup.GET("/:id/result", handler.GetExperimentResult)
	}
	authed.GET("/compare", handler.CompareExperiments)

	// websocket接口 - 服务于前端看板的实时推流(车辆坐标/局部路径/统计数据)
	// WS 握手不支持自定义头，token 经 ?token= 传递（AuthRequired 已兼容）
	wsGroup := authed.Group("/ws")
	{
		wsGroup.GET("/vehicles", handler.WsVehicles)
		wsGroup.GET("/paths/:vehicleId", handler.WsVehiclePath)
		wsGroup.GET("/stats", handler.WsStats)
	}

	// swagger接口 - API 接口自动生成文档 (通过 /swagger/index.html 访问)
	swaggerGroup := r.Group("/swagger")
	{
		swaggerGroup.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
}
