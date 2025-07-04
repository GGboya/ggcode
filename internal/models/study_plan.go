package models

import "time"

// UserStudyPlan 用户学习计划
type UserStudyPlan struct {
	ID             uint         `json:"id" gorm:"primaryKey"`
	UserID         uint         `json:"user_id" gorm:"uniqueIndex:idx_user_questionbank"`
	User           User         `json:"user" gorm:"foreignKey:UserID"`
	QuestionBankID uint         `json:"question_bank_id" gorm:"uniqueIndex:idx_user_questionbank"`
	QuestionBank   QuestionBank `json:"question_bank" gorm:"foreignKey:QuestionBankID"`
	DailyCount     int          `json:"daily_count" gorm:"default:5"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// 添加复合唯一索引，确保一个用户只能对一个题库创建一个学习计划
func (UserStudyPlan) TableName() string {
	return "user_study_plans"
}
