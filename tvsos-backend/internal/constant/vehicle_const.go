package constant

const (
	// 模拟车辆移动更新间隔时间
	VehicleSimulateMovingGap = 10 // 10s执行一次

	// vehicle 的类型 (6种)
	VehicleTybeVan          = 1 // 普通厢式货车
	VehicleTybeRefrigerated = 2 // 冷藏车
	VehicleTybeFlatbed      = 3 // 平板车
	VehicleTybeTanker       = 4 // 危化品罐车
	VehicleTybeHighSided    = 5 // 高栏车
	VehicleTybeMinivan      = 6 // 微型面包车

	// vehicle 的状态
	VehicleStatusRunning = 1 // 行驶中
	VehicleStatusFree    = 2 // 空闲中
	VehicleStatusReady   = 3 // 已分配待发车
)
