package models

import (
	"time"
)

// User 用户模型
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Email     string    `json:"email" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	IsAdmin   bool      `json:"is_admin" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserCheckIn 用户打卡记录
type UserCheckIn struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"user_id" gorm:"uniqueIndex:idx_user_date"` // 联合唯一索引的一部分
	User            User      `json:"user" gorm:"foreignKey:UserID"`
	CheckDate       time.Time `json:"check_date" gorm:"uniqueIndex:idx_user_date"` // 联合唯一索引的一部分
	ConsecutiveDays int       `json:"consecutive_days" gorm:"default:0"`           // 到当前日期的连续打卡天数
	BestStreak      int       `json:"best_streak" gorm:"default:0"`                // 到当前日期为止的最长连续天数
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
