// Package sandbox 提供沙箱系统的领域对象：世界规则/实验参数的 TOML 模式定义、
// 默认值、解析与校验。沙箱级参数（世界设定）与实验级参数（可调旋钮）在此分层定义，
// 对齐《数据实验沙箱.pdf》的地理/交通/时间/经济四维环境与算法变量记录要求。
package sandbox

import (
	"github.com/pelletier/go-toml/v2"
)

// DataGeneratorVersion 数据生成器版本号：生成逻辑发生不兼容变化时 +1。
// 种子导入时校验该版本，不同版本生成的数据不保证可复现。
const DataGeneratorVersion = 2

// ---------------- 沙箱级：世界规则（World Rules） ----------------

// WorldRules 沙箱"世界设定"：基础设施 + 世界规则（沙箱生命周期内不变）。
type WorldRules struct {
	Meta         MetaRules         `toml:"meta"`
	Geography    GeographyRules    `toml:"geography"`
	Traffic      TrafficRules      `toml:"traffic"`
	Time         TimeRules         `toml:"time"`
	Economy      EconomyRules      `toml:"economy"`
	Orders       OrderWorldRules   `toml:"orders"`
	RandomEvents RandomEventRules  `toml:"random_events"`
}

type MetaRules struct {
	Name        string `toml:"name"`
	AlgoVersion string `toml:"algo_version"` // 绑定调度算法版本
}

// GeographyRules 地理环境：POI 白名单与道路基准速度（PDF p6/p10）。
type GeographyRules struct {
	PoiWhitelist []uint  `toml:"poi_whitelist"` // 空=全部 POI
	AvgSpeedKmh  float64 `toml:"avg_speed_kmh"` // 道路基准速度（原硬编码 35）
}

// TrafficRules 交通环境：确定性分时段速度系数（PDF p7/p10，初版简化拥堵模型）。
type TrafficRules struct {
	RushHours        []int   `toml:"rush_hours"`         // 高峰小时(虚拟时间,0-23)
	RushSpeedFactor  float64 `toml:"rush_speed_factor"`  // 高峰速度系数
}

// TimeRules 时间环境：虚拟时钟起点与步长（PDF p8/p10）。加速比在实验级 speed 中。
type TimeRules struct {
	StartTime   string `toml:"start_time"`   // 虚拟时钟起点 "2006-01-02 15:04:05"
	TickSeconds int    `toml:"tick_seconds"` // 虚拟时间步长(秒/tick)
}

// EconomyRules 经济环境：成本参数（PDF p9/p10），供 Cost A-F 计算读取。
type EconomyRules struct {
	FuelPricePerLiter  float64 `toml:"fuel_price_per_liter"`
	TollPerKm          float64 `toml:"toll_per_km"`
	DriverWagePerHour  float64 `toml:"driver_wage_per_hour"`
	HandlingFeePerTon  float64 `toml:"handling_fee_per_ton"`
}

// OrderWorldRules 订单生成模型（世界规则，数据生成器输入）。
type OrderWorldRules struct {
	ArrivalModel  string     `toml:"arrival_model"` // poisson / uniform
	DailyVolume   int        `toml:"daily_volume"`   // 日均单量基准
	WeightRangeKg [2]int     `toml:"weight_range_kg"` // 单件货物重量区间
	CountRange    [2]int     `toml:"count_range"`     // 每单件数区间
}

// RandomEventRules 随机事件概率模型（世界规则；实验级 [random_events] 可覆盖）。
type RandomEventRules struct {
	DriverRefuseProb     float64    `toml:"driver_refuse_prob"`
	RefuseMaxPerShipment int        `toml:"refuse_max_per_shipment"`
	RestAfterHours       float64    `toml:"rest_after_hours"`
	RestMinutes          float64    `toml:"rest_minutes"`
	WeatherProbPerTick   float64    `toml:"weather_prob_per_tick"`
	WeatherDelaySeconds  [2]float64 `toml:"weather_delay_seconds"`
	AccidentProbPerTick  float64    `toml:"accident_prob_per_tick"`
	MaintenanceSeconds   [2]float64 `toml:"maintenance_seconds"`
	LevelFactor          [5]float64 `toml:"level_factor"`
}

// ---------------- 实验级：实验参数（每次实验可调的"旋钮"） ----------------

// ExperimentParams 实验级参数（对齐需求：算法/车辆/货物/随机事件几率/优化目标）。
// 留空(零值)的覆盖项回落到沙箱世界规则。
type ExperimentParams struct {
	Seed         int64                 `toml:"seed"`
	Simulation   SimulationParams      `toml:"simulation"`
	Fleet        FleetParams           `toml:"fleet"`
	Orders       OrderParams           `toml:"orders"`
	Algorithm    AlgorithmParams       `toml:"algorithm"`
	Objectives   ObjectiveParams       `toml:"objectives"`
	RandomEvents *RandomEventRules     `toml:"random_events,omitempty"` // 可空=沿用世界规则
	Constraints  ConstraintParams      `toml:"constraints"`
}

type SimulationParams struct {
	DurationHours float64 `toml:"duration_hours"` // 固定虚拟仿真时长（实验结束条件）
	Speed         float64 `toml:"speed"`          // 初始时间加速比
}

// FleetParams 运力规模：实例化车队时按类型配比取模板车辆。
type FleetParams struct {
	VehicleCount int            `toml:"vehicle_count"` // 0=使用模板全量
	TypeMix      map[string]int `toml:"type_mix,omitempty"`
}

// OrderParams 订单负载（覆盖世界规则基准）。
type OrderParams struct {
	DailyVolume int `toml:"daily_volume"` // 0=沿用世界规则
}

// AlgorithmParams 调度算法超参。
type AlgorithmParams struct {
	Mode            string  `toml:"mode"` // hybrid / nf
	SAInitialTemp   float64 `toml:"sa_initial_temp"`
	SACoolingRate   float64 `toml:"sa_cooling_rate"`
	SAMinTemp       float64 `toml:"sa_min_temp"`
	SAItersPerTemp  int     `toml:"sa_iters_per_temp"`
	CandidateK      int     `toml:"candidate_k"`
}

// ObjectiveParams 全局优化目标指标权重（需求1：可改变全局优化目标）。
type ObjectiveParams struct {
	Weights         map[string]float64 `toml:"weights"`          // 如 C2/U2/U4/T2
	SoftConstraints map[string]float64 `toml:"soft_constraints"` // 如 F1_gini_max/G2_carbon_max
}

// ConstraintParams 约束条件。
type ConstraintParams struct {
	DriverMaxHours       float64 `toml:"driver_max_hours"`       // 0=不限制
	VehicleMaxMileageKm  float64 `toml:"vehicle_max_mileage_km"` // 0=不限制
}

// ---------------- 默认值 ----------------

// DefaultWorldRules 默认世界规则（对齐当前 config-dev.yml 的随机事件口径与硬编码常数）。
func DefaultWorldRules() WorldRules {
	return WorldRules{
		Meta:      MetaRules{Name: "新沙箱", AlgoVersion: "sa-v2.1"},
		Geography: GeographyRules{PoiWhitelist: []uint{}, AvgSpeedKmh: 35},
		Traffic:   TrafficRules{RushHours: []int{8, 9, 17, 18}, RushSpeedFactor: 0.6},
		Time:      TimeRules{StartTime: "2026-01-01 08:00:00", TickSeconds: 10},
		Economy:   EconomyRules{FuelPricePerLiter: 7.5, TollPerKm: 0.5, DriverWagePerHour: 25, HandlingFeePerTon: 15},
		Orders:    OrderWorldRules{ArrivalModel: "poisson", DailyVolume: 200, WeightRangeKg: [2]int{500, 12000}, CountRange: [2]int{1, 5}},
		RandomEvents: RandomEventRules{
			DriverRefuseProb:     0.06,
			RefuseMaxPerShipment: 2,
			RestAfterHours:       4,
			RestMinutes:          20,
			WeatherProbPerTick:   0.0005,
			WeatherDelaySeconds:  [2]float64{1800, 7200},
			AccidentProbPerTick:  0.0002,
			MaintenanceSeconds:   [2]float64{3600, 21600},
			LevelFactor:          [5]float64{1.4, 1.2, 1.0, 0.85, 0.7},
		},
	}
}

// DefaultExperimentParams 默认实验参数（新建方案/实验的表单初值）。
func DefaultExperimentParams() ExperimentParams {
	return ExperimentParams{
		Seed:       12345,
		Simulation: SimulationParams{DurationHours: 24, Speed: 1},
		Fleet:      FleetParams{VehicleCount: 0},
		Orders:     OrderParams{DailyVolume: 0},
		Algorithm: AlgorithmParams{
			Mode:           "hybrid",
			SAInitialTemp:  1000,
			SACoolingRate:  0.95,
			SAMinTemp:      0.1,
			SAItersPerTemp: 50,
			CandidateK:     5,
		},
		Objectives: ObjectiveParams{
			Weights:         map[string]float64{"C2": 1.0, "U2": 0.5, "U4": 0.3, "T2": 0.3},
			SoftConstraints: map[string]float64{"F1_gini_max": 0.4, "G2_carbon_max": 99999},
		},
		RandomEvents: nil, // 默认沿用世界规则
		Constraints:  ConstraintParams{DriverMaxHours: 0, VehicleMaxMileageKm: 0},
	}
}

// ---------------- 解析 / 序列化 ----------------

// ParseWorldRules 解析沙箱世界规则 TOML，缺省字段回落默认值。
func ParseWorldRules(text string) (WorldRules, error) {
	rules := DefaultWorldRules()
	if text == "" {
		return rules, nil
	}
	err := toml.Unmarshal([]byte(text), &rules)
	return rules, err
}

// MarshalWorldRules 序列化世界规则为 TOML 文本。
func MarshalWorldRules(rules WorldRules) (string, error) {
	out, err := toml.Marshal(rules)
	return string(out), err
}

// ParseExperimentParams 解析实验参数 TOML，缺省字段回落默认值。
func ParseExperimentParams(text string) (ExperimentParams, error) {
	params := DefaultExperimentParams()
	if text == "" {
		return params, nil
	}
	err := toml.Unmarshal([]byte(text), &params)
	return params, err
}

// MarshalExperimentParams 序列化实验参数为 TOML 文本。
func MarshalExperimentParams(params ExperimentParams) (string, error) {
	out, err := toml.Marshal(params)
	return string(out), err
}

// EffectiveRandomEvents 实验有效随机事件规则：实验级覆盖优先，否则回落世界规则。
func EffectiveRandomEvents(params ExperimentParams, rules WorldRules) RandomEventRules {
	if params.RandomEvents != nil {
		return *params.RandomEvents
	}
	return rules.RandomEvents
}

// EffectiveDailyVolume 实验有效日均订单量。
func EffectiveDailyVolume(params ExperimentParams, rules WorldRules) int {
	if params.Orders.DailyVolume > 0 {
		return params.Orders.DailyVolume
	}
	return rules.Orders.DailyVolume
}
