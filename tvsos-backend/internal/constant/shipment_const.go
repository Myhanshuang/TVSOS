package constant

const (
	ShipmentGetCount = 5 // 获取5条数据

	// 订单状态
	ShipmentStatusSleeping = 1 // 待创建任务
	ShipmentStatusQueued   = 2 // 已分配待发车
	ShipmentStatusWaiting  = 3 // 待取货
	ShipmentStatusWorking  = 4 // 运输中
	ShipmentStatusFinish   = 5 // 已完成
)
