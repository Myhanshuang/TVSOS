package model

// TableName 指明关联的数据库
// grom 中默认会关联复数表名 (pois) 否则需要手动指明
func (poi Poi) TableName() string {
	return "poi"
}

// Poi poi 点
type Poi struct {
	Id     uint    `gorm:"primaryKey" json:"id"`
	Name   string  `json:"name"`
	Tybe   int     `json:"tybe"`
	Lon    float64 `json:"lon"`
	Lat    float64 `json:"lat"`
	Status int     `json:"status"`
}
