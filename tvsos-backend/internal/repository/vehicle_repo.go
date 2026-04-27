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

// UpdateVehicleLocation 更新车辆位置
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

// AddVehicleTravelMetrics 累加运输指标
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

// UpdateVehicleStatus 更新车辆状态，基于旧状态进行 CAS 更新
// 返回错误通常表示车辆不存在，或状态已变（RowsAffected == 0）
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

// UpdateVehicleReservedSize 更新车辆预占载重
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
