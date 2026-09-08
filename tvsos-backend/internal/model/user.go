package model

import "time"

// TableName 指明关联的数据库表 user
func (user User) TableName() string {
	return "user"
}

// User 研究人员/实验者（沙箱系统登录主体，简单演示级鉴权）
type User struct {
	Id           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `json:"username" gorm:"column:username;uniqueIndex:uk_username"`
	PasswordHash string    `json:"-" gorm:"column:password_hash"`
	DisplayName  string    `json:"displayName" gorm:"column:display_name"`
	CreatedAt    time.Time `json:"createdAt" gorm:"column:created_at"`
}
