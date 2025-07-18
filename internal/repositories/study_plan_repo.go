package repositories

import (
	"encoding/json"
	"errors"
	"ggcode/internal/models"
	"time"

	"gorm.io/gorm"
)

type StudyPlanRepository interface {
	CheckStudyPlanExists(userID, questionBankID uint) (bool, error)
	CreateStudyPlan(studyPlan *models.UserStudyPlan) (*models.UserStudyPlan, error)
	GetStudyPlan(planID, userID uint) (*models.UserStudyPlan, error)
	UpdateStudyPlan(planID, userID uint, dailyCount int) error
	DeleteStudyPlan(planID, userID uint) error
	DeleteStudyPlanWithProgress(planID, userID, questionBankID uint) error
	GetAllStudyPlans(userID uint, page, limit int) ([]models.UserStudyPlan, int64, error)
	GetRandomMasteredQuestions(userID, questionBankID uint, count int) ([]models.UserQuestionProgress, error)
	GetQuestionCount(questionBankID uint) (int64, error)
	GetStudiedQuestionCount(userID, questionBankID uint) (int64, error)
	GetCompletedQuestionCount(userID, questionBankID uint) (int64, error)
	GetReviewQuestionCount(userID, questionBankID uint) (int64, error)
	GetDailyCache(userID, planID uint, today time.Time) (*models.DailyStudyPlanCache, error)
	GenerateDailyQuestions(userID, questionBankID uint, dailyCount int) ([]models.Question, error)
	CacheDailyQuestions(userID, planID uint, questionIDs []uint, today time.Time) error
	GetQuestionsByIDs(questionIDs []uint, questionBankID uint) ([]models.QuestionWithProgress, error)
	CalculateStartPosition(userID uint, questionIDs []uint, today time.Time) (int, error)
	GetReviewQuestionProgresses(userID, questionBankID uint, today time.Time) ([]models.UserQuestionProgress, error)
	GetNewQuestions(userID, questionBankID uint, count int, progresses *[]models.UserQuestionProgress) ([]models.Question, error)
	GetUserQuestionProgress(userID, questionID uint, progress *models.UserQuestionProgress) error
	CreateUserQuestionProgress(progress *models.UserQuestionProgress) error
	UpdateUserQuestionProgress(progress *models.UserQuestionProgress) error
	CountCheckInToday(userID uint, today time.Time, count *int64) error
	CreateCheckInRecord(userID uint, today time.Time) error
	GetUserCheckInByDate(userID uint, date time.Time, checkIn *models.UserCheckIn) error
	GetLatestUserCheckIn(userID uint, checkIn *models.UserCheckIn) error
	CreateUserCheckIn(checkIn *models.UserCheckIn) error
	CountUserStudiedQuestions(userID uint, count *int64) error
	CountUserCompletedQuestions(userID uint, count *int64) error
	CountUserTodayReviewQuestions(userID uint, today time.Time, count *int64) error
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

func (r *studyPlanRepository) GetQuestionCount(questionBankID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Question{}).Where("question_bank_id = ?", questionBankID).Count(&count).Error
	return count, err
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

func (r *studyPlanRepository) GetRandomMasteredQuestions(userID, questionBankID uint, count int) ([]models.UserQuestionProgress, error) {
	var progresses []models.UserQuestionProgress
	err := r.db.Where("user_id = ? AND is_completed = ?", userID, true).
		Preload("Question", "question_bank_id = ?", questionBankID).
		Order("RANDOM()").
		Limit(count).
		Find(&progresses).Error

	return progresses, err
}

func (r *studyPlanRepository) GetStudiedQuestionCount(userID, questionBankID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ?", userID, questionBankID).
		Count(&count).Error
	return count, err
}

func (r *studyPlanRepository) GetCompletedQuestionCount(userID, questionBankID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ?",
			userID, questionBankID, true).
		Count(&count).Error
	return count, err
}

func (r *studyPlanRepository) GetReviewQuestionCount(userID, questionBankID uint) (int64, error) {
	var count int64
	today := time.Now().Truncate(24 * time.Hour)
	err := r.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ? AND user_question_progresses.next_review_date <= ?",
			userID, questionBankID, false, today).
		Count(&count).Error
	return count, err
}

func (r *studyPlanRepository) GetDailyCache(userID, planID uint, today time.Time) (*models.DailyStudyPlanCache, error) {
	var cache models.DailyStudyPlanCache
	err := r.db.Where("user_id = ? AND study_plan_id = ? AND DATE(cache_date) = DATE(?)",
		userID, planID, today).First(&cache).Error
	return &cache, err
}

func (r *studyPlanRepository) GenerateDailyQuestions(userID, questionBankID uint, dailyCount int) ([]models.Question, error) {
	return nil, errors.New("not implemented: use service layer GenerateDailyQuestions instead")
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

func (r *studyPlanRepository) GetQuestionsByIDs(questionIDs []uint, questionBankID uint) ([]models.QuestionWithProgress, error) {
	if len(questionIDs) == 0 {
		return []models.QuestionWithProgress{}, nil
	}

	var questions []models.Question
	if err := r.db.Where("id IN ? AND question_bank_id = ?", questionIDs, questionBankID).Find(&questions).Error; err != nil {
		return nil, err
	}

	// 创建ID到题目的映射，保持原有顺序
	questionMap := make(map[uint]models.Question)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	var result []models.QuestionWithProgress
	for _, questionID := range questionIDs {
		if question, exists := questionMap[questionID]; exists {
			// 获取学习进度（如果有的话）
			var progress models.UserQuestionProgress
			r.db.Where("question_id = ?", questionID).First(&progress)

			result = append(result, models.QuestionWithProgress{
				Question: question,
				Progress: progress,
				IsReview: progress.ID != 0, // 如果有进度记录，说明是复习题目
			})
		}
	}
	return result, nil
}

func (r *studyPlanRepository) CalculateStartPosition(userID uint, questionIDs []uint, today time.Time) (int, error) {
	var count int64
	err := r.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ? AND question_id IN ? AND DATE(last_review_date) = DATE(?)",
			userID, questionIDs, today).Count(&count).Error
	return int(count), err
}

// GetReviewQuestionProgresses 查询今日需要复习的题目进度
func (r *studyPlanRepository) GetReviewQuestionProgresses(userID, questionBankID uint, today time.Time) ([]models.UserQuestionProgress, error) {
	var progresses []models.UserQuestionProgress
	err := r.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ? AND user_question_progresses.next_review_date <= ?",
			userID, questionBankID, false, today).
		Find(&progresses).Error
	return progresses, err
}

// GetNewQuestions 查询未学过的新题目进度（即没有进度记录的题目）
func (r *studyPlanRepository) GetNewQuestions(userID, questionBankID uint, count int, progresses *[]models.UserQuestionProgress) ([]models.Question, error) {
	var questions []models.Question

	// 获取用户没有学习过的题目
	if err := r.db.Table("questions").
		Where("question_bank_id = ?", questionBankID).
		Where("id NOT IN (SELECT question_id FROM user_question_progresses WHERE user_id = ?)", userID).
		Limit(count).Find(&questions).Error; err != nil {
		return nil, err
	}

	return questions, nil
}

// GetUserQuestionProgress 查询用户某题目的进度
func (r *studyPlanRepository) GetUserQuestionProgress(userID, questionID uint, progress *models.UserQuestionProgress) error {
	return r.db.Where("user_id = ? AND question_id = ?", userID, questionID).First(progress).Error
}

// CreateUserQuestionProgress 创建进度
func (r *studyPlanRepository) CreateUserQuestionProgress(progress *models.UserQuestionProgress) error {
	return r.db.Create(progress).Error
}

// UpdateUserQuestionProgress 更新进度
func (r *studyPlanRepository) UpdateUserQuestionProgress(progress *models.UserQuestionProgress) error {
	return r.db.Save(progress).Error
}

// CountCheckInToday 查询用户今日是否已打卡
func (r *studyPlanRepository) CountCheckInToday(userID uint, today time.Time, count *int64) error {
	return r.db.Model(&models.UserCheckIn{}).
		Where("user_id = ? AND DATE(check_date) = DATE(?)", userID, today).
		Count(count).Error
}

// CreateCheckInRecord 创建打卡记录
func (r *studyPlanRepository) CreateCheckInRecord(userID uint, today time.Time) error {
	record := models.UserCheckIn{
		UserID:    userID,
		CheckDate: today,
	}
	return r.db.Create(&record).Error
}

// GetUserCheckInByDate 查询用户某天的打卡记录
func (r *studyPlanRepository) GetUserCheckInByDate(userID uint, date time.Time, checkIn *models.UserCheckIn) error {
	return r.db.Where("user_id = ? AND check_date = ?", userID, date).First(checkIn).Error
}

// GetLatestUserCheckIn 查询用户最近的一条打卡记录
func (r *studyPlanRepository) GetLatestUserCheckIn(userID uint, checkIn *models.UserCheckIn) error {
	return r.db.Where("user_id = ?", userID).Order("check_date DESC").First(checkIn).Error
}

// CreateUserCheckIn 创建打卡记录
func (r *studyPlanRepository) CreateUserCheckIn(checkIn *models.UserCheckIn) error {
	return r.db.Create(checkIn).Error
}

// CountUserStudiedQuestions 统计用户总学习题目数
func (r *studyPlanRepository) CountUserStudiedQuestions(userID uint, count *int64) error {
	return r.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ?", userID).Count(count).Error
}

// CountUserCompletedQuestions 统计用户已完成题目数
func (r *studyPlanRepository) CountUserCompletedQuestions(userID uint, count *int64) error {
	return r.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ? AND is_completed = ?", userID, true).Count(count).Error
}

// CountUserTodayReviewQuestions 统计用户今日需复习题目数
func (r *studyPlanRepository) CountUserTodayReviewQuestions(userID uint, today time.Time, count *int64) error {
	return r.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ? AND next_review_date <= ? AND is_completed = ?", userID, today, false).Count(count).Error
}
