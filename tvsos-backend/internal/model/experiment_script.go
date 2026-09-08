package model

import "time"

// TableName 指明关联的数据库表 experiment_order
func (o ExperimentOrder) TableName() string {
	return "experiment_order"
}

// ExperimentOrder 实验预生成订单（数据生成器输出，仿真运行的输入脚本）。
// 运行时按虚拟时钟到点释放为真正的 shipment 行（experiment_id 相同），本表全程不可变，保证可复现。
type ExperimentOrder struct {
	Id           uint      `gorm:"primaryKey" json:"id"`
	ExperimentId uint      `json:"experimentId" gorm:"column:experiment_id;index:idx_exp_time,priority:1"`
	VirtualTime  time.Time `json:"virtualTime" gorm:"column:virtual_time;index:idx_exp_time,priority:2"` // 虚拟释放时刻
	StartPoiId   uint      `json:"startPoiId" gorm:"column:start_poi_id"`
	EndPoiId     uint      `json:"endPoiId" gorm:"column:end_poi_id"`
	CargoId      uint      `json:"cargoId" gorm:"column:cargo_id"`
	Count        int       `json:"count" gorm:"column:count"`
	Released     bool      `json:"released" gorm:"column:released"` // 是否已释放为 shipment
}

// TableName 指明关联的数据库表 experiment_event
func (e ExperimentEvent) TableName() string {
	return "experiment_event"
}

// ExperimentEvent 实验预生成随机事件脚本（天气/车祸等到点必发事件）。
// 运行时按虚拟时钟到点消费；VehicleLicense 关联本实验实例化车辆（车牌在沙箱模板内唯一）。
type ExperimentEvent struct {
	Id             uint      `gorm:"primaryKey" json:"id"`
	ExperimentId   uint      `json:"experimentId" gorm:"column:experiment_id;index:idx_exp_time,priority:1"`
	VirtualTime    time.Time `json:"virtualTime" gorm:"column:virtual_time;index:idx_exp_time,priority:2"` // 虚拟触发时刻
	EventType      string    `json:"eventType" gorm:"column:event_type"`                                   // weather/accident
	VehicleLicense string    `json:"vehicleLicense" gorm:"column:vehicle_license"`                          // 目标车辆车牌(空=随机/全体)
	Severity       int       `json:"severity" gorm:"column:severity"`
	DurationSeconds int      `json:"durationSeconds" gorm:"column:duration_seconds"` // 影响时长(虚拟秒)
	Detail         string    `json:"detail" gorm:"column:detail;type:text"`
	Consumed       bool      `json:"consumed" gorm:"column:consumed"` // 是否已消费
}
