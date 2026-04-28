package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/handler"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter 注册和初始化所有 HTTP 和 WebSocket 路由规则。
// 包含了 POI、车辆、运单、模拟调度控制、实时报表以及 Swagger 接口挂载。
func SetupRouter(r *gin.Engine) {
	// r.Group() 返回 Group 对象 设置请求统一前缀
	// group.GET("", func)
	// 第一个参数拼接剩余请求路径 第二个参数是传递给哪个函数处理请求
	// 下面 {} 只是为了好看和规范

	// poi接口 - 兴趣点(如发货/收货仓库等)数据接口
	poiGroup := r.Group("/pois")
	{
		poiGroup.GET("", handler.ListPois)
		poiGroup.GET("/:id", handler.GetPoi)
	}
	// 兼容旧前端路径
	legacyPoiGroup := r.Group("/poi")
	{
		legacyPoiGroup.GET("", handler.ListPois)
		legacyPoiGroup.GET("/:id", handler.GetPoi)
	}

	// vehicle接口 - 返回当前所有车辆列表等基础信息
	vehicleGroup := r.Group("/vehicles")
	{
		vehicleGroup.GET("", handler.ListVehicles)
	}

	// shipment接口 - 管理当前系统运单(如列表和模拟生成发单)
	shipmentGroup := r.Group("/shipments")
	{
		shipmentGroup.GET("", handler.ListShipments)
		shipmentGroup.POST("/mock/:count", handler.MockShipments)
	}

	// simulation接口 - 调度可视化演示模拟调速控制接口
	simulationGroup := r.Group("/simulation")
	{
		simulationGroup.GET("/speed", handler.GetSimulationSpeed)
		simulationGroup.POST("/speed", handler.SetSimulationSpeed)
	}

	// report接口 - 获取调度大屏所需要的核心实时统计报表
	reportGroup := r.Group("/report")
	{
		reportGroup.GET("/realtime", handler.GetRealtimeReport)
	}

	// websocket接口 - 服务于前端看板的实时推流(车辆坐标/局部路径/统计数据)
	wsGroup := r.Group("/ws")
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
