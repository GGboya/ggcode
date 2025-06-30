package repositories

import (
	"errors"
	"ggcode/internal/models"

	"gorm.io/gorm"
)

// QuestionListResult 题目列表结果
type QuestionListResult struct {
	Data       []models.Question
	Total      int64
	TotalPages int
}

type QuestionRepository interface {
	GetQuestions(bankID uint, page, limit int) (*QuestionListResult, error)
	GetAllQuestions() ([]models.Question, error)
	CreateQuestion(userID, bankID uint, title, leetcodeURL, difficulty string) (*models.Question, error)
	GetQuestion(questionID uint) (*models.Question, error)
	UpdateQuestion(userID, questionID uint, title, leetcodeURL, difficulty string) (*models.Question, error)
	UpdateQuestionWithDescription(userID, questionID uint, title, leetcodeURL, difficulty, description string) (*models.Question, error)
	DeleteQuestion(userID, questionID uint) error
}

type questionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) QuestionRepository {
	return &questionRepository{db: db}
}

// GetQuestions 获取题库下的题目列表
func (r *questionRepository) GetQuestions(bankID uint, page, limit int) (*QuestionListResult, error) {
	var questions []models.Question
	var total int64

	query := r.db.Model(&models.Question{}).Where("question_bank_id = ?", bankID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页和排序
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at ASC").Find(&questions).Error; err != nil {
		return nil, err
	}

	// 计算分页信息
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &QuestionListResult{
		Data:       questions,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// GetAllQuestions gets all questions from the database.
func (r *questionRepository) GetAllQuestions() ([]models.Question, error) {
	var questions []models.Question
	if err := r.db.Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

// CreateQuestion 在题库中创建题目
func (r *questionRepository) CreateQuestion(userID, bankID uint, title, leetcodeURL, difficulty string) (*models.Question, error) {
	// 检查题库是否存在且属于当前用户
	var questionBank models.QuestionBank
	if err := r.db.Where("id = ? AND created_by = ?", bankID, userID).First(&questionBank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("题库不存在或无权限添加题目")
		}
		return nil, errors.New("查询题库失败")
	}

	// 创建题目
	question := models.Question{
		Title:          title,
		LeetcodeURL:    leetcodeURL,
		Difficulty:     difficulty,
		QuestionBankID: bankID,
	}

	if err := r.db.Create(&question).Error; err != nil {
		return nil, errors.New("创建题目失败")
	}

	return &question, nil
}

// GetQuestion 获取单个题目
func (r *questionRepository) GetQuestion(questionID uint) (*models.Question, error) {
	var question models.Question
	if err := r.db.Where("id = ?", questionID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("题目不存在")
		}
		return nil, errors.New("查询题目失败")
	}

	return &question, nil
}

// UpdateQuestion 更新题目信息
func (r *questionRepository) UpdateQuestion(userID, questionID uint, title, leetcodeURL, difficulty string) (*models.Question, error) {
	var question models.Question
	if err := r.db.Where("id = ?", questionID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("题目不存在")
		}
		return nil, errors.New("查询题目失败")
	}

	// 更新题目字段
	question.Title = title
	question.LeetcodeURL = leetcodeURL
	question.Difficulty = difficulty

	if err := r.db.Save(&question).Error; err != nil {
		return nil, errors.New("更新题目失败")
	}

	return &question, nil
}

// UpdateQuestionWithDescription 更新题目信息（包含描述）
func (r *questionRepository) UpdateQuestionWithDescription(userID, questionID uint, title, leetcodeURL, difficulty, description string) (*models.Question, error) {
	var question models.Question
	if err := r.db.Where("id = ?", questionID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("题目不存在")
		}
		return nil, errors.New("查询题目失败")
	}

	// 更新题目字段
	question.Title = title
	question.LeetcodeURL = leetcodeURL
	question.Difficulty = difficulty
	question.Description = description

	if err := r.db.Save(&question).Error; err != nil {
		return nil, errors.New("更新题目失败")
	}

	return &question, nil
}

// DeleteQuestion 删除题目
func (r *questionRepository) DeleteQuestion(userID, questionID uint) error {
	// 检查题目是否存在且属于用户创建的题库
	var question models.Question
	if err := r.db.Preload("QuestionBank").Where("id = ?", questionID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("题目不存在")
		}
		return errors.New("查询题目失败")
	}

	// 检查权限：只能删除自己创建的题库中的题目
	if question.QuestionBank.CreatedBy == nil || *question.QuestionBank.CreatedBy != userID {
		return errors.New("无权限删除此题目")
	}

	// 开始事务删除题目及相关学习进度
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 定义一个通用的学习进度结构体
	type UserQuestionProgress struct {
		ID         uint `gorm:"primaryKey"`
		UserID     uint
		QuestionID uint
	}

	// 删除该题目的所有学习进度记录
	if err := tx.Where("question_id = ?", questionID).Delete(&UserQuestionProgress{}).Error; err != nil {
		tx.Rollback()
		return errors.New("删除学习进度失败")
	}

	// 删除题目
	if err := tx.Delete(&question).Error; err != nil {
		tx.Rollback()
		return errors.New("删除题目失败")
	}

	return tx.Commit().Error
}
