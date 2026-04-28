package repository

import (
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/dto"
	"github.com/kiritosuki/mover/internal/model"
)

// ListPois 筛选/获取poi列表
// 接收 ListPoisReq 结构体参数，对数据库的 Poi 数据进行条件查询。
// 对于 Name 字段执行模糊匹配 LIKE 查询，对类型和状态进行精确查询，
// 当参数为零值时自动忽略该条件的过滤。
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

// GetPoi 根据id获取poi (Interest Point)
// 查询数据库，按主键获取特定的单个兴趣点数据(如车场、收发货仓等)。
// 返回指定的 Poi 模型实例或错误信息。
func GetPoi(id int) (*model.Poi, error) {
	// 存放返回结果
	poi := model.Poi{}
	// 获取数据库连接对象
	db := database.DB.Model(&model.Poi{})
	// 根据主键查询
	err := db.First(&poi, id).Error
	return &poi, err
}

// ListAllPois 查询全部 POI 列表集
// 不包含过滤条件，查询并返回全量 Poi 指标对象切片。
// 通常在预热加载、随机生成测试数据时调用。
func ListAllPois() ([]*model.Poi, error) {
	var pois []*model.Poi
	db := database.DB.Model(&model.Poi{})
	err := db.Find(&pois).Error
	return pois, err
}
