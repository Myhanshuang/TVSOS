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
}
