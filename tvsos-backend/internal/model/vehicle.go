package model

import "time"

// TableName 指明关联的数据库
// grom 中默认会关联复数表名 否则需要手动指明
func (vehicle Vehicle) TableName() string {
	return "vehicle"
}

// Vehicle 车辆
type Vehicle struct {
	Id            uint      `gorm:"primaryKey" json:"id"`
	License       string    `json:"license"`
	Lon           float64   `json:"lon"`
	Lat           float64   `json:"lat"`
	Speed         float64   `json:"speed"`
	UpdateTime    time.Time `json:"updateTime"`
	Status        int        `json:"status"`
	DriverId      uint       `json:"driverId" gorm:"column:driver_id"`      // 绑定司机
	PauseUntil    *time.Time `json:"pauseUntil" gorm:"column:pause_until"` // 暂停截止(休息/天气/维修统一暂停)
	Tybe          int        `json:"tybe"`
	Size          int       `json:"size"`
	ReservedSize  int       `json:"reservedSize" gorm:"column:reserved_size"`
	Capacity      int       `json:"capacity"`
	Length        float64   `json:"length"`
	Width         float64   `json:"width"`
	Height        float64   `json:"height"`
	WaitTime      float64   `json:"waitTime" gorm:"column:wait_time"`
	TotalWaitTime float64   `json:"totalWaitTime" gorm:"column:total_wait_time"`
	EmptyMileage  float64   `json:"emptyMileage" gorm:"column:empty_mileage"`
	Distance      float64   `json:"distance"`
	Duration      float64   `json:"duration"`
	LoadedDuration float64  `json:"loadedDuration" gorm:"column:loaded_duration"` // 累计载货时长(小时,仅运输段)
	Angle         float64   `json:"angle"`
	ExperimentId uint      `json:"experimentId" gorm:"column:experiment_id;index"` // 所属实验id(0=沙箱模板/全局)
}
