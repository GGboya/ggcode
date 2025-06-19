package database

import (
	"time"
)

// User 用户模型
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Email     string    `json:"email" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// QuestionBank 题库模型
type QuestionBank struct {
	ID           uint          `json:"id" gorm:"primaryKey"`
	Name         string        `json:"name" gorm:"not null"`
	Description  string        `json:"description"`
	IsOfficial   bool          `json:"is_official" gorm:"default:false"`
	IsShared     bool          `json:"is_shared" gorm:"default:false"` // 是否为共享题库
	CreatedBy    *uint         `json:"created_by"`                     // 使用指针类型，允许为空
	Creator      User          `json:"creator" gorm:"foreignKey:CreatedBy"`
	ForkedFrom   *uint         `json:"forked_from"`                                          // Fork来源题库ID
	OriginalBank *QuestionBank `json:"original_bank,omitempty" gorm:"foreignKey:ForkedFrom"` // 原始题库
	StarCount    int           `json:"star_count" gorm:"default:0"`                          // Star数量
	ForkCount    int           `json:"fork_count" gorm:"default:0"`                          // Fork数量
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// QuestionBankStar 题库Star关系模型
type QuestionBankStar struct {
	ID             uint         `json:"id" gorm:"primaryKey"`
	UserID         uint         `json:"user_id" gorm:"uniqueIndex:idx_user_questionbank_star"`
	User           User         `json:"user" gorm:"foreignKey:UserID"`
	QuestionBankID uint         `json:"question_bank_id" gorm:"uniqueIndex:idx_user_questionbank_star"`
	QuestionBank   QuestionBank `json:"question_bank" gorm:"foreignKey:QuestionBankID"`
	CreatedAt      time.Time    `json:"created_at"`
}

// Question 题目模型
type Question struct {
	ID             uint         `json:"id" gorm:"primaryKey"`
	Title          string       `json:"title" gorm:"not null"`
	LeetcodeURL    string       `json:"leetcode_url" gorm:"not null"`
	Difficulty     string       `json:"difficulty" gorm:"not null"` // Easy, Medium, Hard
	QuestionBankID uint         `json:"question_bank_id"`
	QuestionBank   QuestionBank `json:"question_bank" gorm:"foreignKey:QuestionBankID"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

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

// UserQuestionProgress 用户题目学习进度（艾宾浩斯遗忘曲线）
type UserQuestionProgress struct {
	ID         uint     `json:"id" gorm:"primaryKey"`
	UserID     uint     `json:"user_id"`
	User       User     `json:"user" gorm:"foreignKey:UserID"`
	QuestionID uint     `json:"question_id"`
	Question   Question `json:"question" gorm:"foreignKey:QuestionID"`

	// 艾宾浩斯相关字段
	ReviewLevel    int       `json:"review_level" gorm:"default:0"`     // 复习层级 (0-6)
	LastReviewDate time.Time `json:"last_review_date"`                  // 上次复习时间
	NextReviewDate time.Time `json:"next_review_date"`                  // 下次复习时间
	IsCompleted    bool      `json:"is_completed" gorm:"default:false"` // 是否完成学习

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserCheckIn 用户打卡记录
type UserCheckIn struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex:idx_user_date"` // 联合唯一索引的一部分
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	CheckDate time.Time `json:"check_date" gorm:"uniqueIndex:idx_user_date"` // 联合唯一索引的一部分
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
