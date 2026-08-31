package repository

import (
	"time"

	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// CreateMetricSnapshot 落库一条指标历史采样记录。
func CreateMetricSnapshot(snapshot *model.MetricSnapshot) error {
	if snapshot.SnapshotTime.IsZero() {
		snapshot.SnapshotTime = time.Now()
	}
	db := database.DB.Model(&model.MetricSnapshot{})
	return db.Create(snapshot).Error
}

// ListMetricSnapshots 按时间倒序分页拉取指标历史采样，用于时序曲线与成本效益对比(C1)。
func ListMetricSnapshots(limit int) ([]*model.MetricSnapshot, error) {
	if limit <= 0 {
		limit = 500
	}
	var snapshots []*model.MetricSnapshot
	db := database.DB.Model(&model.MetricSnapshot{})
	err := db.Order("snapshot_time DESC").Limit(limit).Find(&snapshots).Error
	return snapshots, err
}

// CreateSystemEvent 记录一条系统运行事件(启动/恢复/故障/修复)。
func CreateSystemEvent(event *model.SystemEvent) error {
	if event.EventTime.IsZero() {
		event.EventTime = time.Now()
	}
	db := database.DB.Model(&model.SystemEvent{})
	return db.Create(event).Error
}

// ListSystemEvents 按时间倒序拉取系统事件，用于 MTBF/MTTR/服务可用性(R1-R3)分析。
func ListSystemEvents(eventType string, limit int) ([]*model.SystemEvent, error) {
	if limit <= 0 {
		limit = 200
	}
	var events []*model.SystemEvent
	db := database.DB.Model(&model.SystemEvent{})
	if eventType != "" {
		db = db.Where("event_type = ?", eventType)
	}
	err := db.Order("event_time DESC").Limit(limit).Find(&events).Error
	return events, err
}
