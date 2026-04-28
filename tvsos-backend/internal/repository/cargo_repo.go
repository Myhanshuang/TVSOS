package repository

import (
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// GetCargo 根据 id 获取特定单件/种类货品的业务实体详情 (包含类型及所需运输容积等属性)。
func GetCargo(id int) (*model.Cargo, error) {
	var cargo model.Cargo
	db := database.DB.Model(&model.Cargo{})
	err := db.First(&cargo, id).Error
	return &cargo, err
}

// ListCargos 无条件拉取数据库中目前存在的所有种类货物映射配置信息。
func ListCargos() ([]*model.Cargo, error) {
	var cargos []*model.Cargo
	db := database.DB.Model(&model.Cargo{})
	err := db.Find(&cargos).Error
	return cargos, err
}
