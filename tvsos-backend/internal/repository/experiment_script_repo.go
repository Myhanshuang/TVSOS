package repository

import (
	"time"

	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// ---------- ExperimentOrder（预生成订单脚本） ----------

// CreateExperimentOrders 批量写入预生成订单
func CreateExperimentOrders(orders []*model.ExperimentOrder) error {
	if len(orders) == 0 {
		return nil
	}
	return database.DB.Model(&model.ExperimentOrder{}).Create(&orders).Error
}

// ListDueExperimentOrders 查询某实验到点（virtual_time <= now）且未释放的订单
func ListDueExperimentOrders(experimentID uint, now time.Time) ([]*model.ExperimentOrder, error) {
	var orders []*model.ExperimentOrder
	err := database.DB.Model(&model.ExperimentOrder{}).
		Where("experiment_id = ? AND released = 0 AND virtual_time <= ?", experimentID, now).
		Order("virtual_time asc, id asc").
		Find(&orders).Error
	return orders, err
}

// MarkExperimentOrdersReleased 批量标记订单已释放
func MarkExperimentOrdersReleased(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return database.DB.Model(&model.ExperimentOrder{}).Where("id IN ?", ids).Update("released", true).Error
}

// CountExperimentOrders 统计实验预生成订单总数（数据生成结果展示用）
func CountExperimentOrders(experimentID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&model.ExperimentOrder{}).Where("experiment_id = ?", experimentID).Count(&count).Error
	return count, err
}

// DeleteExperimentOrdersByExperiment 删除实验订单脚本（重新生成前清理）
func DeleteExperimentOrdersByExperiment(experimentID uint) error {
	return database.DB.Where("experiment_id = ?", experimentID).Delete(&model.ExperimentOrder{}).Error
}

// ---------- ExperimentEvent（预生成事件脚本） ----------

// CreateExperimentEvents 批量写入预生成事件
func CreateExperimentEvents(events []*model.ExperimentEvent) error {
	if len(events) == 0 {
		return nil
	}
	return database.DB.Model(&model.ExperimentEvent{}).Create(&events).Error
}

// ListDueExperimentEvents 查询某实验到点且未消费的事件
func ListDueExperimentEvents(experimentID uint, now time.Time) ([]*model.ExperimentEvent, error) {
	var events []*model.ExperimentEvent
	err := database.DB.Model(&model.ExperimentEvent{}).
		Where("experiment_id = ? AND consumed = 0 AND virtual_time <= ?", experimentID, now).
		Order("virtual_time asc, id asc").
		Find(&events).Error
	return events, err
}

// MarkExperimentEventsConsumed 批量标记事件已消费
func MarkExperimentEventsConsumed(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return database.DB.Model(&model.ExperimentEvent{}).Where("id IN ?", ids).Update("consumed", true).Error
}

// CountExperimentEvents 统计实验预生成事件总数
func CountExperimentEvents(experimentID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&model.ExperimentEvent{}).Where("experiment_id = ?", experimentID).Count(&count).Error
	return count, err
}

// DeleteExperimentEventsByExperiment 删除实验事件脚本（重新生成前清理）
func DeleteExperimentEventsByExperiment(experimentID uint) error {
	return database.DB.Where("experiment_id = ?", experimentID).Delete(&model.ExperimentEvent{}).Error
}
