package repository

import (
	"time"

	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// ---------- Sandbox ----------

// ListSandboxes 沙箱列表（includeArchived 控制是否含归档）
func ListSandboxes(includeArchived bool) ([]*model.Sandbox, error) {
	var sandboxes []*model.Sandbox
	db := database.DB.Model(&model.Sandbox{})
	if !includeArchived {
		db = db.Where("archived = 0")
	}
	err := db.Order("id desc").Find(&sandboxes).Error
	return sandboxes, err
}

// GetSandbox 根据 id 查询沙箱
func GetSandbox(id uint) (*model.Sandbox, error) {
	sandbox := model.Sandbox{}
	err := database.DB.Model(&model.Sandbox{}).First(&sandbox, id).Error
	return &sandbox, err
}

// CreateSandbox 新增沙箱
func CreateSandbox(sandbox *model.Sandbox) error {
	return database.DB.Model(&model.Sandbox{}).Create(sandbox).Error
}

// UpdateSandboxFields 按字段名与值映射更新沙箱
func UpdateSandboxFields(id uint, fields map[string]interface{}) error {
	return database.DB.Model(&model.Sandbox{}).Where("id = ?", id).Updates(fields).Error
}

// CountSandboxes 沙箱总数（默认沙箱迁移判定用）
func CountSandboxes() (int64, error) {
	var count int64
	err := database.DB.Model(&model.Sandbox{}).Count(&count).Error
	return count, err
}

// ---------- ExperimentScheme ----------

// ListSchemesBySandbox 查询沙箱下的实验方案列表
func ListSchemesBySandbox(sandboxID uint) ([]*model.ExperimentScheme, error) {
	var schemes []*model.ExperimentScheme
	err := database.DB.Model(&model.ExperimentScheme{}).Where("sandbox_id = ?", sandboxID).Order("id desc").Find(&schemes).Error
	return schemes, err
}

// GetScheme 根据 id 查询实验方案
func GetScheme(id uint) (*model.ExperimentScheme, error) {
	scheme := model.ExperimentScheme{}
	err := database.DB.Model(&model.ExperimentScheme{}).First(&scheme, id).Error
	return &scheme, err
}

// CreateScheme 新增实验方案
func CreateScheme(scheme *model.ExperimentScheme) error {
	scheme.CreatedAt = time.Now()
	scheme.UpdateTime = scheme.CreatedAt
	return database.DB.Model(&model.ExperimentScheme{}).Create(scheme).Error
}

// UpdateSchemeFields 按字段名与值映射更新实验方案
func UpdateSchemeFields(id uint, fields map[string]interface{}) error {
	fields["update_time"] = time.Now()
	return database.DB.Model(&model.ExperimentScheme{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteScheme 删除实验方案
func DeleteScheme(id uint) error {
	return database.DB.Where("id = ?", id).Delete(&model.ExperimentScheme{}).Error
}

// ---------- Experiment ----------

// ListExperimentsBySandbox 查询沙箱下的实验列表
func ListExperimentsBySandbox(sandboxID uint) ([]*model.Experiment, error) {
	var experiments []*model.Experiment
	err := database.DB.Model(&model.Experiment{}).Where("sandbox_id = ?", sandboxID).Order("id desc").Find(&experiments).Error
	return experiments, err
}

// ListExperimentsByScheme 查询某方案下的实验（批次）列表
func ListExperimentsByScheme(schemeID uint) ([]*model.Experiment, error) {
	var experiments []*model.Experiment
	err := database.DB.Model(&model.Experiment{}).Where("scheme_id = ?", schemeID).Order("id desc").Find(&experiments).Error
	return experiments, err
}

// GetExperiment 根据 id 查询实验
func GetExperiment(id uint) (*model.Experiment, error) {
	experiment := model.Experiment{}
	err := database.DB.Model(&model.Experiment{}).First(&experiment, id).Error
	return &experiment, err
}

// CreateExperiment 新增实验
func CreateExperiment(experiment *model.Experiment) error {
	return database.DB.Model(&model.Experiment{}).Create(experiment).Error
}

// UpdateExperimentFields 按字段名与值映射更新实验
func UpdateExperimentFields(id uint, fields map[string]interface{}) error {
	return database.DB.Model(&model.Experiment{}).Where("id = ?", id).Updates(fields).Error
}

// GetRunningExperiment 查询当前运行中的实验（单实验串行模型下最多一个）
func GetRunningExperiment() (*model.Experiment, error) {
	experiment := model.Experiment{}
	err := database.DB.Model(&model.Experiment{}).Where("status = ?", model.ExperimentStatusRunning).First(&experiment).Error
	return &experiment, err
}

// ---------- ExperimentResult ----------

// GetResultByExperiment 查询实验终态结果
func GetResultByExperiment(experimentID uint) (*model.ExperimentResult, error) {
	result := model.ExperimentResult{}
	err := database.DB.Model(&model.ExperimentResult{}).Where("experiment_id = ?", experimentID).First(&result).Error
	return &result, err
}

// UpsertResult 写入/覆盖实验终态结果
func UpsertResult(result *model.ExperimentResult) error {
	var existing model.ExperimentResult
	err := database.DB.Model(&model.ExperimentResult{}).Where("experiment_id = ?", result.ExperimentId).First(&existing).Error
	if err == nil {
		result.Id = existing.Id
		return database.DB.Model(&model.ExperimentResult{}).Where("id = ?", existing.Id).Updates(map[string]interface{}{
			"summary_json":    result.SummaryJson,
			"primary_metrics": result.PrimaryMetrics,
			"convergence":     result.Convergence,
			"created_at":      time.Now(),
		}).Error
	}
	return database.DB.Model(&model.ExperimentResult{}).Create(result).Error
}

// ListResultsByExperimentIDs 批量查询实验结果（对比视图用）
func ListResultsByExperimentIDs(experimentIDs []uint) ([]*model.ExperimentResult, error) {
	var results []*model.ExperimentResult
	err := database.DB.Model(&model.ExperimentResult{}).Where("experiment_id IN ?", experimentIDs).Find(&results).Error
	return results, err
}
