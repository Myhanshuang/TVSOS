package repository

import (
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// CreateRandomEvent 新增一条随机事件记录
func CreateRandomEvent(ev *model.RandomEvent) error {
	return database.DB.Model(&model.RandomEvent{}).Create(ev).Error
}

// ListRandomEvents 查询最近的事件列表（limit<=0 返回全部，通常按时间倒序）
func ListRandomEvents(limit int) ([]*model.RandomEvent, error) {
	var events []*model.RandomEvent
	db := database.DB.Model(&model.RandomEvent{}).Order("event_time DESC")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&events).Error
	return events, err
}

// ListActiveRandomEvents 查询尚未解除的事件（如进行中的天气/维修/休息）
func ListActiveRandomEvents() ([]*model.RandomEvent, error) {
	var events []*model.RandomEvent
	err := database.DB.Model(&model.RandomEvent{}).Where("status = ?", "active").Order("event_time DESC").Find(&events).Error
	return events, err
}

// ResolveRandomEvent 将事件标记为已解决（用于恢复后回写状态）
func ResolveRandomEvent(id uint) error {
	return database.DB.Model(&model.RandomEvent{}).Where("id = ?", id).Update("status", "resolved").Error
}

// CountRefuseByShipment 统计某运单被拒单的次数（防死循环上限判定）
func CountRefuseByShipment(shipmentID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&model.RandomEvent{}).
		Where("event_type = ? AND shipment_id = ?", "refuse", shipmentID).
		Count(&count).Error
	return count, err
}

// CountRandomEventsByType 统计指定类型的事件数量（今日或全部），供 R 系指标与实时面板使用
func CountRandomEventsByType(eventType string, since string) (int64, error) {
	var count int64
	db := database.DB.Model(&model.RandomEvent{})
	if eventType != "" {
		db = db.Where("event_type = ?", eventType)
	}
	if since != "" {
		db = db.Where("event_time >= ?", since)
	}
	err := db.Count(&count).Error
	return count, err
}