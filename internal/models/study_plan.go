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

// StudyPlanProgress 学习计划进度结构体
type StudyPlanProgress struct {
	StudyPlanID    uint  `json:"study_plan_id"`
	TotalQuestions int64 `json:"total_questions"`
	StudiedCount   int64 `json:"studied_count"`
	CompletedCount int64 `json:"completed_count"`
	ReviewCount    int64 `json:"review_count"`
	ProgressRate   int   `json:"progress_rate"`
	MasteryRate    int   `json:"mastery_rate"`
}

// StudyStats 学习统计
type StudyStats struct {
	TotalStudied int64 `json:"total_studied"`
	Completed    int64 `json:"completed"`
	TodayReview  int64 `json:"today_review"`
}
