package models

import "time"

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
