package repositories

import (
	"ggcode/internal/models"
	"time"

	"gorm.io/gorm"
)

// 用户-题目进度/统计相关
// 实现 struct 可为 userQuestionRepository

type UserQuestionRepository interface {
	GetUserQuestionProgress(userID, questionID uint, progress *models.UserQuestionProgress) error
	CreateUserQuestionProgress(progress *models.UserQuestionProgress) error
	UpdateUserQuestionProgress(progress *models.UserQuestionProgress) error
	GetReviewQuestionProgresses(userID, questionBankID uint, today time.Time) ([]models.UserQuestionProgress, error)
	GetNewQuestions(userID, questionBankID uint, count int, progresses *[]models.UserQuestionProgress) ([]models.Question, error)
	GetQuestionsByIDs(questionIDs []uint, questionBankID uint) ([]models.QuestionWithProgress, error)
	CalculateStartPosition(userID uint, questionIDs []uint, today time.Time) (int, error)
	GetStudiedQuestionCount(userID, questionBankID uint) (int64, error)
	GetCompletedQuestionCount(userID, questionBankID uint) (int64, error)
	GetReviewQuestionCount(userID, questionBankID uint) (int64, error)
}

type userQuestionRepository struct {
	db *gorm.DB
}

func NewUserQuestionRepository(db *gorm.DB) UserQuestionRepository {
	return &userQuestionRepository{db: db}
}

var _ UserQuestionRepository = &userQuestionRepository{nil}

func (r *userQuestionRepository) GetUserQuestionProgress(userID, questionID uint, progress *models.UserQuestionProgress) error {
	return r.db.Where("user_id = ? AND question_id = ?", userID, questionID).First(progress).Error
}

func (r *userQuestionRepository) CreateUserQuestionProgress(progress *models.UserQuestionProgress) error {
	return r.db.Create(progress).Error
}

func (r *userQuestionRepository) UpdateUserQuestionProgress(progress *models.UserQuestionProgress) error {
	return r.db.Save(progress).Error
}

func (r *userQuestionRepository) GetReviewQuestionProgresses(userID, questionBankID uint, today time.Time) ([]models.UserQuestionProgress, error) {
	var progresses []models.UserQuestionProgress
	err := r.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ? AND user_question_progresses.next_review_date <= ?",
			userID, questionBankID, false, today).
		Find(&progresses).Error
	return progresses, err
}

func (r *userQuestionRepository) GetNewQuestions(userID, questionBankID uint, count int, progresses *[]models.UserQuestionProgress) ([]models.Question, error) {
	var questions []models.Question
	if err := r.db.Table("questions").
		Where("question_bank_id = ?", questionBankID).
		Where("id NOT IN (SELECT question_id FROM user_question_progresses WHERE user_id = ?)", userID).
		Limit(count).Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

func (r *userQuestionRepository) GetQuestionsByIDs(questionIDs []uint, questionBankID uint) ([]models.QuestionWithProgress, error) {
	if len(questionIDs) == 0 {
		return []models.QuestionWithProgress{}, nil
	}
	var questions []models.Question
	if err := r.db.Where("id IN ? AND question_bank_id = ?", questionIDs, questionBankID).Find(&questions).Error; err != nil {
		return nil, err
	}
	questionMap := make(map[uint]models.Question)
	for _, q := range questions {
		questionMap[q.ID] = q
	}
	var result []models.QuestionWithProgress
	for _, questionID := range questionIDs {
		if question, exists := questionMap[questionID]; exists {
			var progress models.UserQuestionProgress
			r.db.Where("question_id = ?", questionID).First(&progress)
			result = append(result, models.QuestionWithProgress{
				Question: question,
				Progress: progress,
				IsReview: progress.ID != 0,
			})
		}
	}
	return result, nil
}

// CalculateStartPosition 计算开始位置
func (r *userQuestionRepository) CalculateStartPosition(userID uint, questionIDs []uint, today time.Time) (int, error) {
	var count int64
	err := r.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ? AND question_id IN ? AND DATE(last_review_date) = DATE(?)",
			userID, questionIDs, today).Count(&count).Error
	return int(count), err
}

// GetStudiedQuestionCount 获取已学习题目数
func (r *userQuestionRepository) GetStudiedQuestionCount(userID, questionBankID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ?", userID, questionBankID).
		Count(&count).Error
	return count, err
}

// GetCompletedQuestionCount 获取已完成题目数
func (r *userQuestionRepository) GetCompletedQuestionCount(userID, questionBankID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ?",
			userID, questionBankID, true).
		Count(&count).Error
	return count, err
}

// GetReviewQuestionCount 获取待复习题目数
func (r *userQuestionRepository) GetReviewQuestionCount(userID, questionBankID uint) (int64, error) {
	var count int64
	// 使用本地时区获取今天的开始时间
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	err := r.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ? AND user_question_progresses.next_review_date <= ?",
			userID, questionBankID, false, today).
		Count(&count).Error
	return count, err
}
