package repository

import (
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// ListDrivers 获取司机列表（status=0 表示全部）
func ListDrivers(status int) ([]*model.Driver, error) {
	var drivers []*model.Driver
	db := database.DB.Model(&model.Driver{})
	if status != 0 {
		db = db.Where("status = ?", status)
	}
	err := db.Find(&drivers).Error
	return drivers, err
}

// GetDriver 根据 id 查询司机
func GetDriver(id int) (*model.Driver, error) {
	driver := model.Driver{}
	err := database.DB.Model(&model.Driver{}).First(&driver, id).Error
	return &driver, err
}

// GetDriverByVehicle 根据绑定的车辆 id 查询司机
func GetDriverByVehicle(vehicleID uint) (*model.Driver, error) {
	driver := model.Driver{}
	err := database.DB.Model(&model.Driver{}).Where("vehicle_id = ?", vehicleID).First(&driver).Error
	return &driver, err
}

// CreateDriver 新增司机记录
func CreateDriver(driver *model.Driver) error {
	return database.DB.Model(&model.Driver{}).Create(driver).Error
}

// UpdateDriverFields 按字段名与值映射批量更新司机记录
func UpdateDriverFields(id uint, fields map[string]interface{}) error {
	return database.DB.Model(&model.Driver{}).Where("id = ?", id).Updates(fields).Error
}

// ListDriversWithoutVehicle 查询尚未绑定车辆的司机(兼容迁移用)
func ListDriversWithoutVehicle() ([]*model.Driver, error) {
	var drivers []*model.Driver
	err := database.DB.Model(&model.Driver{}).Where("vehicle_id = 0").Find(&drivers).Error
	return drivers, err
}