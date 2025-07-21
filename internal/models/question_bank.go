package models

import "time"

// QuestionBank 题库模型
type QuestionBank struct {
	ID           uint          `json:"id" gorm:"primaryKey"`
	Name         string        `json:"name" gorm:"not null"`
	Description  string        `json:"description"`
	IsOfficial   bool          `json:"is_official" gorm:"default:false"`
	IsShared     bool          `json:"is_shared" gorm:"default:false"`     // 是否为共享题库
	IsWrongBook  bool          `json:"is_wrong_book" gorm:"default:false"` // 是否为错题本
	CreatedBy    *uint         `json:"created_by"`                         // 使用指针类型，允许为空
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

type QuestionBankProgress struct {
	QuestionBankID uint  `json:"question_bank_id"`
	TotalQuestions int64 `json:"total_questions"` // 题库总题目数
	StudiedCount   int64 `json:"studied_count"`   // 已学习题目数
	CompletedCount int64 `json:"completed_count"` // 已掌握题目数
	ReviewCount    int64 `json:"review_count"`    // 待复习题目数
	ProgressRate   int   `json:"progress_rate"`   // 学习进度百分比
	MasteryRate    int   `json:"mastery_rate"`    // 掌握率百分比
}
