package model

// TableName 指明关联的数据库表 emission_factor
func (emissionFactor EmissionFactor) TableName() string {
	return "emission_factor"
}

// EmissionFactor 排放因子配置
// 按车型(vehicle.tybe)配置载重/空驶排放因子(kg CO2 / km)，用于绿色指标(G1/G2)的计算。
type EmissionFactor struct {
	Id          uint    `gorm:"primaryKey" json:"id"`
	VehicleType int     `json:"vehicleType" gorm:"column:vehicle_type;uniqueIndex"`
	FactorEmpty float64 `json:"factorEmpty" gorm:"column:factor_empty"`
	FactorLoad  float64 `json:"factorLoad" gorm:"column:factor_load"`
	Remark      string  `json:"remark" gorm:"column:remark"`
}
