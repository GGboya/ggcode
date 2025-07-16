package models

import "time"

// Question 题目模型
type Question struct {
	ID             uint         `json:"id" gorm:"primaryKey"`
	Title          string       `json:"title" gorm:"not null"`
	URL            string       `json:"url" gorm:"not null"`
	Difficulty     string       `json:"difficulty" gorm:"not null"` // Easy, Medium, Hard
	QuestionBankID uint         `json:"question_bank_id"`
	QuestionBank   QuestionBank `json:"question_bank" gorm:"foreignKey:QuestionBankID"`
	TestCases      []TestCase   `json:"test_cases" gorm:"foreignKey:QuestionID"` // 关联的测试用例
	// 与算法知识点的多对多关系
	Tags      []AlgoTag `json:"tags" gorm:"many2many:question_tags;"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TestCase 测试用例模型
type TestCase struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	QuestionID     uint      `json:"question_id" gorm:"not null"`
	Question       Question  `json:"question" gorm:"foreignKey:QuestionID"`
	Input          string    `json:"input" gorm:"type:text"`           // 测试输入
	ExpectedOutput string    `json:"expected_output" gorm:"type:text"` // 期望输出
	Description    string    `json:"description"`                      // 测试用例描述
	IsHidden       bool      `json:"is_hidden" gorm:"default:false"`   // 是否为隐藏测试用例
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserQuestionProgress 用户题目学习进度（艾宾浩斯遗忘曲线）
type UserQuestionProgress struct {
	ID         uint     `json:"id" gorm:"primaryKey"`
	UserID     uint     `json:"user_id"`
	User       User     `json:"user" gorm:"foreignKey:UserID"`
	QuestionID uint     `json:"question_id"`
	Question   Question `json:"question" gorm:"foreignKey:QuestionID"`

	// 艾宾浩斯相关字段
	ReviewLevel    int       `json:"review_level" gorm:"default:0"`        // 复习层级 (0-6)
	LastReviewDate time.Time `json:"last_review_date"`                     // 上次复习时间
	NextReviewDate time.Time `json:"next_review_date" gorm:"default:NULL"` // 下次复习时间
	IsCompleted    bool      `json:"is_completed" gorm:"default:false"`    // 是否完成学习

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DailyStudyPlanCache 每日学习计划缓存
type DailyStudyPlanCache struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"not null"`
	StudyPlanID uint      `json:"study_plan_id" gorm:"not null"`
	CacheDate   time.Time `json:"cache_date" gorm:"not null"`    // 缓存日期（按天）
	QuestionIDs string    `json:"question_ids" gorm:"type:text"` // JSON格式存储题目ID列表
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联关系
	User      User          `json:"user" gorm:"foreignKey:UserID"`
	StudyPlan UserStudyPlan `gorm:"foreignKey:StudyPlanID"`
}

// ContestProblem 比赛题目模型
// 用于从外部平台导入题库
// Source: leetcode, atcoder, codeforces, nowcoder
// Score: 题目分数（如100, 200等）
type ContestProblem struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Source    string    `json:"source" gorm:"not null"` // leetcode, atcoder, codeforces, nowcoder
	Title     string    `json:"title" gorm:"not null"`
	Score     *float64  `json:"score" gorm:"not null"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
