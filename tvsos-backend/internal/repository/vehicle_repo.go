package repository

import (
	"fmt"
	"time"

	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
	"gorm.io/gorm"
)

// ListVehicles 筛选/获取车辆列表
func ListVehicles(status int) ([]*model.Vehicle, error) {
	// 存放返回结果
	var vehicles []*model.Vehicle
	// 获取数据库连接对象
	db := database.DB.Model(&model.Vehicle{})
	// 条件查询
	if status != 0 {
		db = db.Where("status = ?", status)
	}
	// 写入结果
	err := db.Find(&vehicles).Error
	return vehicles, err
}

// UpdateVehicleLocation 更新车辆在地图上的实时经纬度坐标。
// 同时重置车辆的 wait_time 为 0，并刷新 update_time。
func UpdateVehicleLocation(id int, lon float64, lat float64) error {
	// 获取数据库连接对象
	db := database.DB.Model(&model.Vehicle{})
	db = db.Where("id = ?", id)
	err := db.Updates(map[string]interface{}{
		"lon":         lon,
		"lat":         lat,
		"wait_time":   0,
		"update_time": time.Now(),
	}).Error
	return err
}

// AddVehicleTravelMetrics 累加车辆的总体运输指标（如空驶里程、运输总距离和总行驶时长）。
// 运用 gorm.Expr 表达式对数据库对应字段进行原子递增操作。
func AddVehicleTravelMetrics(id int, emptyMileageDelta float64, transportDistanceDelta float64, durationDeltaHour float64) error {
	updates := map[string]interface{}{
		"empty_mileage": gorm.Expr("empty_mileage + ?", emptyMileageDelta),
		"distance":      gorm.Expr("distance + ?", transportDistanceDelta),
		"duration":      gorm.Expr("duration + ?", durationDeltaHour),
		"update_time":   time.Now(),
	}

	db := database.DB.Model(&model.Vehicle{}).Where("id = ?", id)
	return db.Updates(updates).Error
}

// UpdateVehicleStatus 基于乐观锁（CAS - Compare And Swap）机制安全地更新车辆状态。
// 必须匹配预期的旧状态(`oldStatus`)，能有效防止并发调度中车辆状态被覆盖修改而导致逻辑错误。
// 若 RowsAffected == 0，即表示因状态变化导致的更新失败。
func UpdateVehicleStatus(id int, status int, oldStatus int) error {
	db := database.DB.Model(&model.Vehicle{})
	db = db.Where("id = ? and status = ?", id, oldStatus)
	res := db.Updates(map[string]interface{}{
		"status":      status,
		"update_time": time.Now(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("vehicle %d CAS update failed, status not equal %d", id, oldStatus)
	}
	return nil
}

// UpdateVehicleSize 更新车辆当前载重
func UpdateVehicleSize(id int, size int) error {
	db := database.DB.Model(&model.Vehicle{})
	db = db.Where("id = ?", id)
	err := db.Update("size", size).Error
	return err
}

// UpdateVehicleReservedSize 更新车辆被算法"预占"的未来预估载重，用于算法在决策排单并入排在队列尾部时作前置可用载重判定。
func UpdateVehicleReservedSize(id int, reservedSize int) error {
	db := database.DB.Model(&model.Vehicle{})
	db = db.Where("id = ?", id)
	err := db.Update("reserved_size", reservedSize).Error
	return err
}

// GetVehicle 根据 id 查询车辆
func GetVehicle(id int) (*model.Vehicle, error) {
	vehicle := model.Vehicle{}
	db := database.DB.Model(&model.Vehicle{})
	err := db.First(&vehicle, id).Error
	return &vehicle, err
}
