package models

import "time"

// AlgoTag 知识点标签，如 "动态规划"、"拓扑排序" 等
// 题目与标签是多对多关系，通过 question_tags 绑定
// 用户解锁某题后，会向 user_unlocked_tags 写入已解锁标签

type AlgoTag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"unique;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// 反向关联
	Questions []Question `json:"questions" gorm:"many2many:question_tags;"`
}

// QuestionTag 题目-标签多对多关系表 (自定义中间表以便将来扩展字段)
// gorm 会自动识别 "question_id", "algo_tag_id" 作为外键

type QuestionTag struct {
	QuestionID uint `gorm:"primaryKey"`
	AlgoTagID  uint `gorm:"primaryKey"`
}

// UserUnlockedTag 用户已解锁的标签
// 当用户AC含某标签的题目后写入，用于基于知识点的题目解锁/推荐

type UserUnlockedTag struct {
	UserID    uint `gorm:"primaryKey"`
	AlgoTagID uint `gorm:"primaryKey"`
	CreatedAt time.Time
}
