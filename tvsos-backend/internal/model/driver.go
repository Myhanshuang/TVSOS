package model

import "time"

// TableName 指明关联的数据库表 driver
func (driver Driver) TableName() string {
	return "driver"
}

// Driver 司机对象（一车一司机）
// 支撑司机拒单、疲劳休息、车祸概率等随机事件建模，现实依据：
// 货运行业司机连续驾驶 4 小时须休息不少于 20 分钟（《道路交通安全法实施条例》第 62 条）；
// 资深司机对不划算订单拒绝率更低、事故率更低。
type Driver struct {
	Id               uint       `gorm:"primaryKey" json:"id"`
	Name             string     `json:"name" gorm:"column:name"`            // 姓名
	Phone            string     `json:"phone" gorm:"column:phone"`          // 电话
	LicenseNo        string     `json:"licenseNo" gorm:"column:license_no"` // 驾驶证号
	Level            int        `json:"level" gorm:"column:level"`          // 驾驶等级 1-5（1 学徒 / 5 资深）
	Fatigue          float64    `json:"fatigue" gorm:"column:fatigue"`      // 疲劳度 0~1
	ContinuousHours  float64    `json:"continuousHours" gorm:"column:continuous_hours"` // 连续驾驶小时数
	RestUntil        *time.Time `json:"restUntil" gorm:"column:rest_until"` // 休息截止时刻(休息中暂停移动)
	Status           int        `json:"status" gorm:"column:status"`        // 0 未知 / 1 可用 / 2 休息中
	RefuseCount      int        `json:"refuseCount" gorm:"column:refuse_count"`   // 累计拒单次数
	VehicleId        uint       `json:"vehicleId" gorm:"column:vehicle_id"` // 绑定车辆 id
	UpdateTime       time.Time  `json:"updateTime" gorm:"column:update_time"`
	ExperimentId uint       `json:"experimentId" gorm:"column:experiment_id;index"` // 所属实验id(0=沙箱模板/全局)
}