package repositories

import (
	"ggcode/internal/models"
	"time"

	"gorm.io/gorm"
)

// 用户统计相关
// 实现 struct 可为 userStatsRepository

type UserStatsRepository interface {
	CountUserStudiedQuestions(userID uint, count *int64) error
	CountUserCompletedQuestions(userID uint, count *int64) error
	CountUserTodayReviewQuestions(userID uint, today time.Time, count *int64) error
}

type userStatsRepository struct {
	db *gorm.DB
}

func NewUserStatsRepository(db *gorm.DB) UserStatsRepository {
	return &userStatsRepository{db: db}
}

var _ UserStatsRepository = &userStatsRepository{nil}

// CountUserStudiedQuestions 统计用户总学习题目数
func (r *userStatsRepository) CountUserStudiedQuestions(userID uint, count *int64) error {
	return r.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ?", userID).Count(count).Error
}

// CountUserCompletedQuestions 统计用户已完成题目数
func (r *userStatsRepository) CountUserCompletedQuestions(userID uint, count *int64) error {
	return r.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ? AND is_completed = ?", userID, true).Count(count).Error
}

// CountUserTodayReviewQuestions 统计用户今日需复习题目数
func (r *userStatsRepository) CountUserTodayReviewQuestions(userID uint, today time.Time, count *int64) error {
	return r.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ? AND next_review_date <= ? AND is_completed = ?", userID, today, false).Count(count).Error
}
