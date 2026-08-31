package repository

import (
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// ListEmissionFactors 拉取全部车型排放因子配置。
func ListEmissionFactors() ([]*model.EmissionFactor, error) {
	var factors []*model.EmissionFactor
	db := database.DB.Model(&model.EmissionFactor{})
	err := db.Find(&factors).Error
	return factors, err
}

// GetEmissionFactorByType 按车型获取排放因子配置(未配置时返回 nil)。
func GetEmissionFactorByType(vehicleType int) (*model.EmissionFactor, error) {
	var factor model.EmissionFactor
	db := database.DB.Model(&model.EmissionFactor{})
	err := db.Where("vehicle_type = ?", vehicleType).First(&factor).Error
	if err != nil {
		return nil, err
	}
	return &factor, nil
}

// ListHandlingConfigs 拉取全部装卸/调度参数配置。
func ListHandlingConfigs() ([]*model.HandlingConfig, error) {
	var configs []*model.HandlingConfig
	db := database.DB.Model(&model.HandlingConfig{})
	err := db.Find(&configs).Error
	return configs, err
}

// GetHandlingConfig 按配置键读取参数(未配置时返回零值)。
func GetHandlingConfig(configKey string) (*model.HandlingConfig, error) {
	var config model.HandlingConfig
	db := database.DB.Model(&model.HandlingConfig{})
	err := db.Where("config_key = ?", configKey).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// CreateMatchingLog 落库一条匹配决策日志。
func CreateMatchingLog(log *model.MatchingLog) error {
	db := database.DB.Model(&model.MatchingLog{})
	return db.Create(log).Error
}

// ListMatchingLogs 分页拉取匹配决策日志，用于匹配精准度(M1-M4)评估。
func ListMatchingLogs(limit int) ([]*model.MatchingLog, error) {
	if limit <= 0 {
		limit = 500
	}
	var logs []*model.MatchingLog
	db := database.DB.Model(&model.MatchingLog{})
	err := db.Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}
