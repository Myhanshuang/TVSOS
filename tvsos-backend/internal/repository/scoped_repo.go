package repository

import (
	"time"

	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// ============================================================================
// 实验作用域（experiment_id）查询变体——沙箱系统多实验数据隔离的读取入口。
// experiment_id=0 表示沙箱模板/全局数据；>0 表示某次实验的实例数据。
// ============================================================================

// ListVehiclesByExp 按实验筛选车辆（status=0 表示全部状态）
func ListVehiclesByExp(experimentID uint, status int) ([]*model.Vehicle, error) {
	var vehicles []*model.Vehicle
	db := database.DB.Model(&model.Vehicle{}).Where("experiment_id = ?", experimentID)
	if status != 0 {
		db = db.Where("status = ?", status)
	}
	err := db.Find(&vehicles).Error
	return vehicles, err
}

// CreateVehicle 新增车辆（实验车队实例化用）
func CreateVehicle(vehicle *model.Vehicle) error {
	return database.DB.Model(&model.Vehicle{}).Create(vehicle).Error
}

// GetVehicleByLicense 按实验 + 车牌查询车辆（脚本事件目标定位用）
func GetVehicleByLicense(experimentID uint, license string) (*model.Vehicle, error) {
	vehicle := model.Vehicle{}
	err := database.DB.Model(&model.Vehicle{}).
		Where("experiment_id = ? AND license = ?", experimentID, license).
		First(&vehicle).Error
	return &vehicle, err
}

// UpdateVehicleLocationAt 以指定（虚拟）时刻更新车辆位置，并重置 wait_time。
func UpdateVehicleLocationAt(id int, lon float64, lat float64, now time.Time) error {
	return database.DB.Model(&model.Vehicle{}).Where("id = ?", id).Updates(map[string]interface{}{
		"lon":         lon,
		"lat":         lat,
		"wait_time":   0,
		"update_time": now,
	}).Error
}

// GetSleepingShipmentByExp 查询本实验待派单（Sleeping）运单
func GetSleepingShipmentByExp(experimentID uint) ([]*model.Shipment, error) {
	var shipments []*model.Shipment
	err := database.DB.Model(&model.Shipment{}).
		Where("experiment_id = ? AND status = 1", experimentID).
		Order("id asc").
		Find(&shipments).Error
	return shipments, err
}

// ListShipmentsByExp 查询本实验全部运单
func ListShipmentsByExp(experimentID uint) ([]*model.Shipment, error) {
	var shipments []*model.Shipment
	err := database.DB.Model(&model.Shipment{}).Where("experiment_id = ?", experimentID).Find(&shipments).Error
	return shipments, err
}

// GetRunningTasksOrderedByVehicleByExp 查询本实验进行中任务（按车辆与创建时间排序）
func GetRunningTasksOrderedByVehicleByExp(experimentID uint) ([]*model.OrderTask, error) {
	var tasks []*model.OrderTask
	err := database.DB.Model(&model.OrderTask{}).
		Where("experiment_id = ? AND sequential IN (2,3)", experimentID).
		Order("vehicle_id asc, create_time asc").
		Find(&tasks).Error
	return tasks, err
}

// GetAllTasksByExp 查询本实验全部任务
func GetAllTasksByExp(experimentID uint) ([]*model.OrderTask, error) {
	var tasks []*model.OrderTask
	err := database.DB.Model(&model.OrderTask{}).Where("experiment_id = ?", experimentID).Find(&tasks).Error
	return tasks, err
}

// ListDriversByExp 按实验筛选司机（status=0 表示全部）
func ListDriversByExp(experimentID uint, status int) ([]*model.Driver, error) {
	var drivers []*model.Driver
	db := database.DB.Model(&model.Driver{}).Where("experiment_id = ?", experimentID)
	if status != 0 {
		db = db.Where("status = ?", status)
	}
	err := db.Find(&drivers).Error
	return drivers, err
}

// ListRandomEventsByExp 查询本实验最近随机事件
func ListRandomEventsByExp(experimentID uint, limit int) ([]*model.RandomEvent, error) {
	var events []*model.RandomEvent
	db := database.DB.Model(&model.RandomEvent{}).Where("experiment_id = ?", experimentID).Order("event_time desc")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&events).Error
	return events, err
}

// ListActiveRandomEventsByExp 查询本实验活跃状态随机事件
func ListActiveRandomEventsByExp(experimentID uint) ([]*model.RandomEvent, error) {
	var events []*model.RandomEvent
	err := database.DB.Model(&model.RandomEvent{}).
		Where("experiment_id = ? AND status = 'active'", experimentID).
		Find(&events).Error
	return events, err
}

// ListRandomEventsByTypeByExp 按类型查询本实验随机事件
func ListRandomEventsByTypeByExp(experimentID uint, eventType string, limit int) ([]*model.RandomEvent, error) {
	var events []*model.RandomEvent
	db := database.DB.Model(&model.RandomEvent{}).
		Where("experiment_id = ? AND event_type = ?", experimentID, eventType).
		Order("event_time desc")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&events).Error
	return events, err
}

// CountRandomEventsByTypeByExp 统计本实验某类事件数量
func CountRandomEventsByTypeByExp(experimentID uint, eventType string) (int64, error) {
	var count int64
	err := database.DB.Model(&model.RandomEvent{}).
		Where("experiment_id = ? AND event_type = ?", experimentID, eventType).
		Count(&count).Error
	return count, err
}

// ListSystemEventsByExp 查询本实验系统事件（eventType 空=全部）
func ListSystemEventsByExp(experimentID uint, eventType string, limit int) ([]*model.SystemEvent, error) {
	var events []*model.SystemEvent
	db := database.DB.Model(&model.SystemEvent{}).Where("experiment_id = ?", experimentID)
	if eventType != "" {
		db = db.Where("event_type = ?", eventType)
	}
	db = db.Order("event_time desc")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&events).Error
	return events, err
}

// CountMatchingLogsSinceByExp 统计本实验自某（虚拟）时刻以来的匹配日志数
func CountMatchingLogsSinceByExp(experimentID uint, since time.Time) (int64, error) {
	var count int64
	err := database.DB.Model(&model.MatchingLog{}).
		Where("experiment_id = ? AND created_at >= ?", experimentID, since).
		Count(&count).Error
	return count, err
}

// ListMetricSnapshotsByExp 查询本实验指标快照（按时间升序，趋势图用）
func ListMetricSnapshotsByExp(experimentID uint, limit int) ([]*model.MetricSnapshot, error) {
	var snapshots []*model.MetricSnapshot
	db := database.DB.Model(&model.MetricSnapshot{}).
		Where("experiment_id = ?", experimentID).
		Order("snapshot_time asc")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&snapshots).Error
	return snapshots, err
}

// MarkStaleExperimentsAborted 服务启动时清理：残留 running→aborted，generating→failed。
func MarkStaleExperimentsAborted() error {
	if err := database.DB.Model(&model.Experiment{}).
		Where("status = ?", model.ExperimentStatusRunning).
		Updates(map[string]interface{}{"status": model.ExperimentStatusAborted, "finished_at": time.Now()}).Error; err != nil {
		return err
	}
	return database.DB.Model(&model.Experiment{}).
		Where("status = ?", model.ExperimentStatusGenerating).
		Updates(map[string]interface{}{"status": model.ExperimentStatusFailed, "finished_at": time.Now()}).Error
}
