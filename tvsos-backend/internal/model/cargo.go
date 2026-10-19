package model

func (cargo Cargo) TableName() string {
	return "cargo"
}

type Cargo struct {
	Id     uint    `gorm:"primaryKey" json:"id"`
	Name   string  `json:"name"`
	Tybe   int     `json:"tybe"`
	Pack   int     `json:"pack"`
	Weight int     `json:"weight"`  // 单件重量(kg)
	Volume float64 `json:"volume" gorm:"column:volume"` // 单件方量(m³)，供应链吨位/方量趋势核算用
}
