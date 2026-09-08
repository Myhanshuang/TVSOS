package repository

import (
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// GetUserByUsername 按登录名查询用户（登录鉴权用）
func GetUserByUsername(username string) (*model.User, error) {
	user := model.User{}
	err := database.DB.Model(&model.User{}).Where("username = ?", username).First(&user).Error
	return &user, err
}

// GetUser 根据 id 查询用户
func GetUser(id uint) (*model.User, error) {
	user := model.User{}
	err := database.DB.Model(&model.User{}).First(&user, id).Error
	return &user, err
}

// CreateUser 新增用户
func CreateUser(user *model.User) error {
	return database.DB.Model(&model.User{}).Create(user).Error
}

// CountUsers 用户总数（种子账号判定用）
func CountUsers() (int64, error) {
	var count int64
	err := database.DB.Model(&model.User{}).Count(&count).Error
	return count, err
}
