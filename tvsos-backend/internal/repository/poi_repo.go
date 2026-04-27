package repository

import (
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/dto"
	"github.com/kiritosuki/mover/internal/model"
)

// ListPois 筛选/获取poi列表
func ListPois(listPoisReq *dto.ListPoisReq) ([]*model.Poi, error) {
	// 存放返回结果
	var pois []*model.Poi
	// 获取数据库连接对象
	db := database.DB.Model(&model.Poi{})
	// 名字单独做模糊查询
	if listPoisReq.Name != "" {
		db = db.Where("name like ?", "%"+listPoisReq.Name+"%")
	}
	filter := &dto.ListPoisReq{
		Name:   "",
		Tybe:   listPoisReq.Tybe,
		Status: listPoisReq.Status,
	}
	// gorm 当查询参数是结构体时
	// 默认会忽略结构体中的零值 并精确查询
	db = db.Where(filter)
	// 写入结果
	err := db.Find(&pois).Error
	return pois, err
}

// GetPoi 根据id获取poi
func GetPoi(id int) (*model.Poi, error) {
	// 存放返回结果
	poi := model.Poi{}
	// 获取数据库连接对象
	db := database.DB.Model(&model.Poi{})
	// 根据主键查询
	err := db.First(&poi, id).Error
	return &poi, err
}

// ListAllPois 查询全部 POI
func ListAllPois() ([]*model.Poi, error) {
	var pois []*model.Poi
	db := database.DB.Model(&model.Poi{})
	err := db.Find(&pois).Error
	return pois, err
}
