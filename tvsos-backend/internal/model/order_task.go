package model

import "time"

func (orderTask OrderTask) TableName() string {
	return "order_task"
}

// OrderTask 任务
type OrderTask struct {
	Id         uint      `gorm:"primaryKey" json:"id"`
	ShipmentId uint      `json:"shipmentId"`
	VehicleId  uint      `json:"vehicleId"`
	Sequential int       `json:"sequential"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`

	// ---- 任务阶段时间戳（用于还原调度时序，支撑 T2 响应时间分位/瓶颈分析/装卸时间标定）----
	// 使用 *time.Time 以便未发生时写入 NULL，避免 MySQL 严格模式拒绝零值 '0000-00-00'。
	AssignTime         *time.Time `json:"assignTime" gorm:"column:assign_time"`          // 订单被分配车辆时间
	ReachPickupTime    *time.Time `json:"reachPickupTime" gorm:"column:reach_pickup_time"`  // 车辆到达提货点时间
	LoadStartTime      *time.Time `json:"loadStartTime" gorm:"column:load_start_time"`    // 开始装货时间
	LoadEndTime        *time.Time `json:"loadEndTime" gorm:"column:load_end_time"`      // 完成装货时间
	TransportStartTime *time.Time `json:"transportStartTime" gorm:"column:transport_start_time"` // 开始运输时间
	ReachDeliveryTime  *time.Time `json:"reachDeliveryTime" gorm:"column:reach_delivery_time"`  // 到达收货点时间
	UnloadStartTime    *time.Time `json:"unloadStartTime" gorm:"column:unload_start_time"`    // 开始卸货时间
	UnloadEndTime      *time.Time `json:"unloadEndTime" gorm:"column:unload_end_time"`      // 完成卸货时间
	SignoffTime        *time.Time `json:"signoffTime" gorm:"column:signoff_time"`        // 签收完成时间
	ExperimentId uint       `json:"experimentId" gorm:"column:experiment_id;index"` // 所属实验id(0=沙箱模板/全局)
}
