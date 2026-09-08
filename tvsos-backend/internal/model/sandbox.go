package model

import "time"

// TableName 指明关联的数据库表 sandbox
func (sandbox Sandbox) TableName() string {
	return "sandbox"
}

// Sandbox 沙箱（世界设定：基础设施 + 世界规则，一次生命周期内不变）
// WorldRulesToml 按 geography/traffic/time/economy/world.orders/world.random_events 组织（对齐数据实验沙箱.pdf 四维环境）；
// FleetTemplate 为车队模板 JSON（车辆类型库/车队定义），实验启动时按实验参数实例化为本实验专属 vehicle/driver 行。
type Sandbox struct {
	Id             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `json:"name" gorm:"column:name"`
	Description    string    `json:"description" gorm:"column:description"`
	WorldRulesToml string    `json:"worldRulesToml" gorm:"column:world_rules_toml;type:mediumtext"`
	FleetTemplate  string    `json:"fleetTemplate" gorm:"column:fleet_template;type:mediumtext"`
	PoiWhitelist   string    `json:"poiWhitelist" gorm:"column:poi_whitelist;type:text"`
	AlgoVersion    string    `json:"algoVersion" gorm:"column:algo_version"`
	CreatedBy      uint      `json:"createdBy" gorm:"column:created_by"`
	CreatedAt      time.Time `json:"createdAt" gorm:"column:created_at"`
	Archived       bool      `json:"archived" gorm:"column:archived"`
}

// TableName 指明关联的数据库表 experiment_scheme
func (scheme ExperimentScheme) TableName() string {
	return "experiment_scheme"
}

// ExperimentScheme 实验方案（沙箱内命名的参数模板 TOML，可复用/编辑；"再次运行"= 用方案新建实验）
type ExperimentScheme struct {
	Id          uint      `gorm:"primaryKey" json:"id"`
	SandboxId   uint      `json:"sandboxId" gorm:"column:sandbox_id;index"`
	Name        string    `json:"name" gorm:"column:name"`
	Description string    `json:"description" gorm:"column:description"`
	ParamsToml  string    `json:"paramsToml" gorm:"column:params_toml;type:mediumtext"`
	CreatedBy   uint      `json:"createdBy" gorm:"column:created_by"`
	CreatedAt   time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdateTime  time.Time `json:"updateTime" gorm:"column:update_time"`
}

// 实验状态机
const (
	ExperimentStatusConfigured = "configured" // 已配置（未生成数据）
	ExperimentStatusGenerating = "generating" // 数据生成中
	ExperimentStatusReady      = "ready"      // 数据已生成，可启动
	ExperimentStatusRunning    = "running"    // 运行中
	ExperimentStatusFinished   = "finished"   // 正常完成（到达固定虚拟仿真时长）
	ExperimentStatusFailed     = "failed"     // 失败
	ExperimentStatusAborted    = "aborted"    // 人工中止/服务重启中断
)

// TableName 指明关联的数据库表 experiment
func (experiment Experiment) TableName() string {
	return "experiment"
}

// Experiment 实验（批次）：由某方案发起的一次运行，含参数快照 + 种子 + 状态 + 进度。
// 种子不同即不同实验；相同种子 + 相同参数 + 相同生成器版本 => 结果严格可复现（虚拟时钟）。
type Experiment struct {
	Id          uint       `gorm:"primaryKey" json:"id"`
	SandboxId   uint       `json:"sandboxId" gorm:"column:sandbox_id;index"`
	SchemeId    uint       `json:"schemeId" gorm:"column:scheme_id"`
	Name        string     `json:"name" gorm:"column:name"`
	ParamsToml  string     `json:"paramsToml" gorm:"column:params_toml;type:mediumtext"`
	Seed        int64      `json:"seed" gorm:"column:seed"`
	Status      string     `json:"status" gorm:"column:status"`
	DataVersion int        `json:"dataVersion" gorm:"column:data_version"`
	Progress    float64    `json:"progress" gorm:"column:progress"`
	CreatedBy   uint       `json:"createdBy" gorm:"column:created_by"`
	CreatedAt   time.Time  `json:"createdAt" gorm:"column:created_at"`
	StartedAt   *time.Time `json:"startedAt" gorm:"column:started_at"`
	FinishedAt  *time.Time `json:"finishedAt" gorm:"column:finished_at"`
}

// TableName 指明关联的数据库表 experiment_result
func (r ExperimentResult) TableName() string {
	return "experiment_result"
}

// ExperimentResult 实验终态结果（结果数据层，对齐 PDF p18：总成本/总里程/准时率/计算时间/利用率/约束违反）
// SummaryJson 为终态指标全量快照；PrimaryMetrics 冗余主目标便于对比查询；Convergence 为 SA 收敛曲线/算法性能。
type ExperimentResult struct {
	Id             uint      `gorm:"primaryKey" json:"id"`
	ExperimentId   uint      `json:"experimentId" gorm:"column:experiment_id;uniqueIndex:uk_experiment"`
	SummaryJson    string    `json:"summaryJson" gorm:"column:summary_json;type:mediumtext"`
	PrimaryMetrics string    `json:"primaryMetrics" gorm:"column:primary_metrics;type:text"`
	Convergence    string    `json:"convergence" gorm:"column:convergence;type:mediumtext"`
	CreatedAt      time.Time `json:"createdAt" gorm:"column:created_at"`
}
