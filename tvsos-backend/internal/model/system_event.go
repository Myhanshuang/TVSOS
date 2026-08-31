package model

import "time"

// TableName 指明关联的数据库表 system_event
func (systemEvent SystemEvent) TableName() string {
	return "system_event"
}

// SystemEvent 系统运行事件
// 记录服务启动、运行时恢复、故障与修复等事件，用于可靠性指标(MTBF/MTTR/服务可用性 R1-R3)的计算。
type SystemEvent struct {
	Id        uint      `gorm:"primaryKey" json:"id"`
	EventTime time.Time `json:"eventTime" gorm:"column:event_time;index"`
	EventType string    `json:"eventType" gorm:"column:event_type"`
	Level     string    `json:"level" gorm:"column:level"`
	Message   string    `json:"message" gorm:"column:message"`
	Detail    string    `json:"detail" gorm:"column:detail;type:text"`
}
