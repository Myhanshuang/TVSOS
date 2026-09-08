package model

import "time"

// TableName 指明关联的数据库表 random_event
func (randomEvent RandomEvent) TableName() string {
	return "random_event"
}

// RandomEvent 随机事件记录（司机拒单/休息、极端天气、车祸受损等）
// 供前端事件流展示，并作为可靠性指标(R1-R3：事故数/维修时长/准时送达)的数据源。
type RandomEvent struct {
	Id              uint      `gorm:"primaryKey" json:"id"`
	EventTime       time.Time `json:"eventTime" gorm:"column:event_time;index"` // 事件发生时间
	EventType       string    `json:"eventType" gorm:"column:event_type"`       // refuse/rest/weather/accident
	VehicleId       uint      `json:"vehicleId" gorm:"column:vehicle_id"`
	DriverId        uint      `json:"driverId" gorm:"column:driver_id"`
	ShipmentId      uint      `json:"shipmentId" gorm:"column:shipment_id"`
	TaskId          uint      `json:"taskId" gorm:"column:task_id"`
	Severity        int       `json:"severity" gorm:"column:severity"`             // 严重程度 1-5
	DurationSeconds int       `json:"durationSeconds" gorm:"column:duration_seconds"` // 影响时长(秒)
	Description     string    `json:"description" gorm:"column:description"`       // 描述
	Detail          string    `json:"detail" gorm:"column:detail;type:text"`       // 详情(JSON)
	Status          string    `json:"status" gorm:"column:status"`                 // active/pending/resolved
	ExperimentId uint       `json:"experimentId" gorm:"column:experiment_id;index"` // 所属实验id(0=沙箱模板/全局)
}