package repository

import (
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// GetSleepingShipment 获取处于休眠(Sleeping)状态、等待生成任务的运单集合。
// 该函数按照运单的创建时间(create_time)升序排序，并限制最大拉取数量。
// 主要被调度引擎或轮询任务循环调用，用以从数据库中分批捞取待处理订单。
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

// UpdateShipmentStatus 基于乐观锁（CAS）更新运单状态。
// 更新时需匹配当前记录中的旧状态 (oldStatus)，避免高并发下的状态相互覆盖和错误转移。
func UpdateShipmentStatus(id int, status int, oldStatus int) error {
	// 获取数据库连接对象
	db := database.DB.Model(&model.Shipment{})
	// CAS锁
	db = db.Where("id = ? and status = ?", id, oldStatus)
	db = db.Update("status", status)
	return db.Error
}

// GetShipment 根据给定的运单ID，从数据库单查对应的一笔运单(Shipment)详情。
func GetShipment(id int) (*model.Shipment, error) {
	// 存放返回结果
	shipment := model.Shipment{}
	// 获取数据库连接对象
	db := database.DB.Model(&model.Shipment{})
	// 根据主键查询
	err := db.First(&shipment, id).Error
	return &shipment, err
}

// ListShipments 获取数据库中全量的运单记录及对应属性，不做状态和分页限制。
// 大体量场景下此接口开销较大，主要用于调试及系统完全初始化。
func ListShipments() ([]*model.Shipment, error) {
	var shipments []*model.Shipment
	db := database.DB.Model(&model.Shipment{})
	err := db.Find(&shipments).Error
	return shipments, err
}

// CreateShipment 插入一个新运单 (Shipment)
// 包含起终点信息、货物挂载信息以及状态等，实现持久化入库以便算法线程消费。
func CreateShipment(shipment *model.Shipment) error {
	db := database.DB.Model(&model.Shipment{})
	err := db.Create(shipment).Error
	return err
}
