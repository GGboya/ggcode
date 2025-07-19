package repositories

import (
	"encoding/json"
	"errors"
	"ggcode/internal/models"
	"time"

	"gorm.io/gorm"
)

// 学习计划相关
// 只保留学习计划本身的 CRUD、缓存等
// 实现 struct 可为 studyPlanRepository

type StudyPlanRepository interface {
	CheckStudyPlanExists(userID, questionBankID uint) (bool, error)
	CreateStudyPlan(studyPlan *models.UserStudyPlan) (*models.UserStudyPlan, error)
	GetStudyPlan(planID, userID uint) (*models.UserStudyPlan, error)
	UpdateStudyPlan(planID, userID uint, dailyCount int) error
	DeleteStudyPlan(planID, userID uint) error
	DeleteStudyPlanWithProgress(planID, userID, questionBankID uint) error
	GetAllStudyPlans(userID uint, page, limit int) ([]models.UserStudyPlan, int64, error)
	GetDailyCache(userID, planID uint, today time.Time) (*models.DailyStudyPlanCache, error)
	CacheDailyQuestions(userID, planID uint, questionIDs []uint, today time.Time) error
}

type studyPlanRepository struct {
	db *gorm.DB
}

var _ StudyPlanRepository = &studyPlanRepository{nil}

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

func (r *studyPlanRepository) DeleteStudyPlan(planID, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", planID, userID).
		Delete(&models.UserStudyPlan{}).Error
}

func (r *studyPlanRepository) DeleteStudyPlanWithProgress(planID, userID, questionBankID uint) error {
	// 开始事务
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除该用户在该题库中的所有学习进度记录
	if err := tx.Exec(`
		DELETE FROM user_question_progresses 
		WHERE user_id = ? AND question_id IN (
			SELECT id FROM questions WHERE question_bank_id = ?
		)
	`, userID, questionBankID).Error; err != nil {
		tx.Rollback()
		return errors.New("清空学习进度失败")
	}

	// 删除该学习计划的每日缓存记录
	if err := tx.Where("user_id = ? AND study_plan_id = ?", userID, planID).
		Delete(&models.DailyStudyPlanCache{}).Error; err != nil {
		tx.Rollback()
		return errors.New("清空学习缓存失败")
	}

	// 删除学习计划
	if err := tx.Where("id = ? AND user_id = ?", planID, userID).
		Delete(&models.UserStudyPlan{}).Error; err != nil {
		tx.Rollback()
		return errors.New("删除学习计划失败")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return errors.New("删除操作提交失败")
	}

	return nil
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

func (r *studyPlanRepository) GetDailyCache(userID, planID uint, today time.Time) (*models.DailyStudyPlanCache, error) {
	var cache models.DailyStudyPlanCache
	err := r.db.Where("user_id = ? AND study_plan_id = ? AND DATE(cache_date) = DATE(?)",
		userID, planID, today).First(&cache).Error
	return &cache, err
}

func (r *studyPlanRepository) CacheDailyQuestions(userID, planID uint, questionIDs []uint, today time.Time) error {
	// 序列化题目ID列表
	questionIDsJSON, err := json.Marshal(questionIDs)
	if err != nil {
		return err
	}

	// 创建缓存记录
	cache := models.DailyStudyPlanCache{
		UserID:      userID,
		StudyPlanID: planID,
		CacheDate:   today,
		QuestionIDs: string(questionIDsJSON),
	}

	return r.db.Create(&cache).Error
}
