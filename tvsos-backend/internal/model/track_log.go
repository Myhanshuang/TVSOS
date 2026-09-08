package model

import "time"

// TableName 指明关联的数据库表 track_log
func (trackLog TrackLog) TableName() string {
	return "track_log"
}

// TrackLog 车辆轨迹/里程日志
// 逐段记录车辆的轨迹点(经纬度)及空驶/载重标记，用于里程利用率(U1)与碳排放(G1)的真实化计算。
type TrackLog struct {
	Id         uint      `gorm:"primaryKey" json:"id"`
	VehicleId  uint      `json:"vehicleId" gorm:"column:vehicle_id;index:idx_vehicle_time"`
	Lon        float64   `json:"lon" gorm:"column:lon"`
	Lat        float64   `json:"lat" gorm:"column:lat"`
	IsEmpty    bool      `json:"isEmpty" gorm:"column:is_empty"`
	RecordTime time.Time `json:"recordTime" gorm:"column:record_time;index:idx_vehicle_time"`
	ExperimentId uint       `json:"experimentId" gorm:"column:experiment_id;index"` // 所属实验id(0=沙箱模板/全局)
}
