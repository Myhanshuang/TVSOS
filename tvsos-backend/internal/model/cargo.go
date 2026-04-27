package model

func (cargo Cargo) TableName() string {
	return "cargo"
}

type Cargo struct {
	Id     uint `gorm:"primaryKey" json:"id"`
	Name   uint `json:"name"`
	Tybe   int  `json:"tybe"`
	Pack   int  `json:"pack"`
	Weight int  `json:"weight"`
}
