package model

// TableName 指明关联的数据库表 handling_config
func (handlingConfig HandlingConfig) TableName() string {
	return "handling_config"
}

// HandlingConfig 装卸/调度参数配置
// 将原先写死的经验系数(如装卸 60 秒/吨)配置化，支持不同货种/包装/场站的差异化标定。
type HandlingConfig struct {
	Id          uint    `gorm:"primaryKey" json:"id"`
	ConfigKey   string  `json:"configKey" gorm:"column:config_key;uniqueIndex"`
	ConfigValue float64 `json:"configValue" gorm:"column:config_value"`
	Remark      string  `json:"remark" gorm:"column:remark"`
}
