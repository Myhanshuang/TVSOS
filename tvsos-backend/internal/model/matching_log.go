package model

import "time"

// TableName 指明关联的数据库表 matching_log
func (matchingLog MatchingLog) TableName() string {
	return "matching_log"
}

// MatchingLog 匹配决策日志
// 记录每次"货找车"分配的目标运单、选中车辆与候选排名(JSON)，是匹配精准度指标(M1-M4, MRR)的评估数据源。
type MatchingLog struct {
	Id                 uint      `gorm:"primaryKey" json:"id"`
	ShipmentId         uint      `json:"shipmentId" gorm:"column:shipment_id;index"`
	ChosenVehicleId    uint      `json:"chosenVehicleId" gorm:"column:chosen_vehicle_id"`
	CandidateRanking   string    `json:"candidateRanking" gorm:"column:candidate_ranking;type:text"`
	GroundTruthVehicle uint      `json:"groundTruthVehicle" gorm:"column:ground_truth_vehicle"`
	CreatedAt          time.Time `json:"createdAt" gorm:"column:created_at"`
	ExperimentId uint       `json:"experimentId" gorm:"column:experiment_id;index"` // 所属实验id(0=沙箱模板/全局)
}
