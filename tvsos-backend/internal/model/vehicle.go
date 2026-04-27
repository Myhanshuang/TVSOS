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
	Status        int       `json:"status"`
	Tybe          int       `json:"tybe"`
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
	Angle         float64   `json:"angle"`
}
