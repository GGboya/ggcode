package repositories

import (
	"ggcode/internal/models"

	"gorm.io/gorm"
)

type StudyPlanRepository interface {
	CheckStudyPlanExists(userID, questionBankID uint) (bool, error)
	CreateStudyPlan(studyPlan *models.UserStudyPlan) (*models.UserStudyPlan, error)
	GetStudyPlan(planID, userID uint) (*models.UserStudyPlan, error)
	UpdateStudyPlan(planID, userID uint, dailyCount int) error
	GetAllStudyPlans(userID uint, page, limit int) ([]models.UserStudyPlan, int64, error)
	GetRandomMasteredQuestions(userID, questionBankID uint, count int) ([]models.UserQuestionProgress, error)
}

type studyPlanRepository struct {
	db *gorm.DB
}

func NewStudyPlanRepository(db *gorm.DB) StudyPlanRepository {
	return &studyPlanRepository{db: db}
}

func (r *studyPlanRepository) CheckStudyPlanExists(userID, questionBankID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.UserStudyPlan{}).
		Where("user_id = ? AND question_bank_id = ?", userID, questionBankID).
		Count(&count).Error
	return count > 0, err
}

func (r *studyPlanRepository) CreateStudyPlan(studyPlan *models.UserStudyPlan) (*models.UserStudyPlan, error) {
	if err := r.db.Create(studyPlan).Error; err != nil {
		return nil, err
	}

	// 预加载题库信息
	r.db.Preload("QuestionBank").First(studyPlan, studyPlan.ID)

	return studyPlan, nil
}

func (r *studyPlanRepository) GetStudyPlan(planID, userID uint) (*models.UserStudyPlan, error) {
	var studyPlan models.UserStudyPlan
	err := r.db.Where("id = ? AND user_id = ?", planID, userID).
		Preload("QuestionBank").First(&studyPlan).Error
	if err != nil {
		return nil, err
	}
	return &studyPlan, nil
}

func (r *studyPlanRepository) UpdateStudyPlan(planID, userID uint, dailyCount int) error {
	return r.db.Model(&models.UserStudyPlan{}).
		Where("id = ? AND user_id = ?", planID, userID).
		Update("daily_count", dailyCount).Error
}

func (r *studyPlanRepository) GetAllStudyPlans(userID uint, page, limit int) ([]models.UserStudyPlan, int64, error) {
	var studyPlans []models.UserStudyPlan
	var total int64

	query := r.db.Model(&models.UserStudyPlan{}).Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).
		Preload("QuestionBank").
		Order("created_at DESC").
		Find(&studyPlans).Error; err != nil {
		return nil, 0, err
	}

	return studyPlans, total, nil
}

func (r *studyPlanRepository) GetRandomMasteredQuestions(userID, questionBankID uint, count int) ([]models.UserQuestionProgress, error) {
	var progresses []models.UserQuestionProgress
	err := r.db.Where("user_id = ? AND is_completed = ?", userID, true).
		Preload("Question", "question_bank_id = ?", questionBankID).
		Order("RANDOM()").
		Limit(count).
		Find(&progresses).Error

	return progresses, err
}
