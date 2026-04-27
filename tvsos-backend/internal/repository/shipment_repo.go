package repository

import (
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// GetSleepingShipment 根据更新时间 获取前ShipmentGetCount条待创建任务的订单
func GetSleepingShipment() ([]*model.Shipment, error) {
	// 存放返回结果
	var shipments []*model.Shipment
	// 获取数据库连接对象
	db := database.DB.Model(&model.Shipment{})
	// 条件查询
	db = db.Where("status = ?", constant.ShipmentStatusSleeping)
	db = db.Order("create_time ASC") // 早的在前
	db = db.Limit(constant.ShipmentGetCount)
	// 写入结果
	err := db.Find(&shipments).Error
	return shipments, err
}

// UpdateShipmentStatus 更新订单状态
func UpdateShipmentStatus(id int, status int, oldStatus int) error {
	// 获取数据库连接对象
	db := database.DB.Model(&model.Shipment{})
	// CAS锁
	db = db.Where("id = ? and status = ?", id, oldStatus)
	db = db.Update("status", status)
	return db.Error
}

// GetShipment 根据id查询shipment
func GetShipment(id int) (*model.Shipment, error) {
	// 存放返回结果
	shipment := model.Shipment{}
	// 获取数据库连接对象
	db := database.DB.Model(&model.Shipment{})
	// 根据主键查询
	err := db.First(&shipment, id).Error
	return &shipment, err
}

// ListShipments 查询全部订单
func ListShipments() ([]*model.Shipment, error) {
	var shipments []*model.Shipment
	db := database.DB.Model(&model.Shipment{})
	err := db.Find(&shipments).Error
	return shipments, err
}

// CreateShipment 创建订单
func CreateShipment(shipment *model.Shipment) error {
	db := database.DB.Model(&model.Shipment{})
	err := db.Create(shipment).Error
	return err
}
