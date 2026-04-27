package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/handler"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter(r *gin.Engine) {
	// r.Group() 返回 Group 对象 设置请求统一前缀
	// group.GET("", func)
	// 第一个参数拼接剩余请求路径 第二个参数是传递给哪个函数处理请求
	// 下面 {} 只是为了好看和规范

	// poi接口
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

	// vehicle接口
	vehicleGroup := r.Group("/vehicles")
	{
		vehicleGroup.GET("", handler.ListVehicles)
	}

	// shipment接口
	shipmentGroup := r.Group("/shipments")
	{
		shipmentGroup.GET("", handler.ListShipments)
		shipmentGroup.POST("/mock/:count", handler.MockShipments)
	}

	// simulation接口
	simulationGroup := r.Group("/simulation")
	{
		simulationGroup.GET("/speed", handler.GetSimulationSpeed)
		simulationGroup.POST("/speed", handler.SetSimulationSpeed)
	}

	// report接口
	reportGroup := r.Group("/report")
	{
		reportGroup.GET("/realtime", handler.GetRealtimeReport)
	}

	// websocket接口
	wsGroup := r.Group("/ws")
	{
		wsGroup.GET("/vehicles", handler.WsVehicles)
		wsGroup.GET("/paths/:vehicleId", handler.WsVehiclePath)
		wsGroup.GET("/stats", handler.WsStats)
	}

	// swagger接口
	swaggerGroup := r.Group("/swagger")
	{
		swaggerGroup.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
}
