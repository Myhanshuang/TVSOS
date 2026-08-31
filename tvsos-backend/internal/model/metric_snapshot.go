package model

import "time"

// TableName 指明关联的数据库表 metric_snapshot
func (metricSnapshot MetricSnapshot) TableName() string {
	return "metric_snapshot"
}

// MetricSnapshot 指标历史采样快照
// 周期性地将全局核心指标落库，用于生成历史趋势、成本效益对比(C1)与可用性分析(R3)等时序类指标。
type MetricSnapshot struct {
	Id                  uint      `gorm:"primaryKey" json:"id"`
	SnapshotTime        time.Time `json:"snapshotTime" gorm:"column:snapshot_time;index"`
	CostValue           float64   `json:"costValue" gorm:"column:cost_value"`
	TotalThroughput     float64   `json:"totalThroughput" gorm:"column:total_throughput"`
	CargoPending        float64   `json:"cargoPending" gorm:"column:cargo_pending"`
	TransportEfficiency float64   `json:"transportEfficiency" gorm:"column:transport_efficiency"`
	TotalWaitingTime    float64   `json:"totalWaitingTime" gorm:"column:total_waiting_time"`
	TotalEmptyMileage   float64   `json:"totalEmptyMileage" gorm:"column:total_empty_mileage"`
	UnitCost            float64   `json:"unitCost" gorm:"column:unit_cost"`
	EmptyRate           float64   `json:"emptyRate" gorm:"column:empty_rate"`
	LoadingRateStdDev   float64   `json:"loadingRateStdDev" gorm:"column:loading_rate_stddev"`
	MatchResponseP50    float64   `json:"matchResponseP50" gorm:"column:match_response_p50"`
	MatchResponseP95    float64   `json:"matchResponseP95" gorm:"column:match_response_p95"`
	MatchResponseP99    float64   `json:"matchResponseP99" gorm:"column:match_response_p99"`
	GiniCoefficient     float64   `json:"giniCoefficient" gorm:"column:gini_coefficient"`
	CarbonTotal         float64   `json:"carbonTotal" gorm:"column:carbon_total"`
	CarbonPerTonKm      float64   `json:"carbonPerTonKm" gorm:"column:carbon_per_ton_km"`
}
