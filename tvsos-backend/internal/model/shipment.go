package model

import "time"

func (shipment Shipment) TableName() string {
	return "shipment"
}

// Shipment 订单
type Shipment struct {
	Id         uint      `gorm:"primaryKey" json:"id"`
	StartPoiId uint      `json:"startPoiId"`
	EndPoiId   uint      `json:"endPoiId"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
	Status     int       `json:"status"`
	CargoId    uint      `json:"cargoId"`
	Count      int       `json:"count"`
	ExperimentId uint       `json:"experimentId" gorm:"column:experiment_id;index"` // 所属实验id(0=沙箱模板/全局)
}
