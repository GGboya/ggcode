package repositories

import (
	"errors"
	"ggcode/internal/models"

	"gorm.io/gorm"
)

// QuestionBankQueryOptions 题库查询选项
type QuestionBankQueryOptions struct {
	UserID   uint
	BankType string // "official", "shared", "personal"
	SortBy   string // "star_count", "fork_count", "created_at"
	Page     int
	Limit    int
}

// QuestionBankListResult 题库列表结果
type QuestionBankListResult struct {
	Data       []models.QuestionBank
	Total      int64
	TotalPages int
}

type QuestionBankUpdateData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type QuestionBankRepository interface {
	GetQuestionBanks(options QuestionBankQueryOptions) (*QuestionBankListResult, error)
	GetStarredBankIDs(userID uint, bankIDs []uint) ([]uint, error)
	CreateQuestionBank(name, description string, userID uint) (*models.QuestionBank, error)
	UpdateQuestionBank(bankID, userID uint, updateData QuestionBankUpdateData) error
	DeleteQuestionBank(bankID, userID uint) error
	GetOrCreateWrongQuestionBook(userID uint) (*models.QuestionBank, error)
	AddQuestionToWrongBook(userID, questionID uint) error
}

type questionBankRepository struct {
	db *gorm.DB
}

func NewQuestionBankRepository(db *gorm.DB) QuestionBankRepository {
	return &questionBankRepository{db: db}
}

func (r *questionBankRepository) GetQuestionBanks(options QuestionBankQueryOptions) (*QuestionBankListResult, error) {
	var questionBanks []models.QuestionBank
	var total int64

	query := r.db.Model(&models.QuestionBank{})

	// 根据题库类型过滤
	switch options.BankType {
	case "official":
		query = query.Where("is_official = ?", true)
	case "shared":
		query = query.Where("is_official = ? AND is_shared = ?", false, true)
	case "personal":
		query = query.Where("created_by = ?", options.UserID)
	default:
		query = query.Where("is_official = ?", true)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序
	switch options.SortBy {
	case "star_count":
		query = query.Order("star_count DESC")
	case "fork_count":
		query = query.Order("fork_count DESC")
	default:
		query = query.Order("created_at DESC")
	}

	// 分页
	offset := (options.Page - 1) * options.Limit
	if err := query.Offset(offset).Limit(options.Limit).
		Preload("Creator").
		Preload("OriginalBank").
		Find(&questionBanks).Error; err != nil {
		return nil, err
	}

	// 计算分页信息
	totalPages := int((total + int64(options.Limit) - 1) / int64(options.Limit))

	return &QuestionBankListResult{
		Data:       questionBanks,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (r *questionBankRepository) GetStarredBankIDs(userID uint, bankIDs []uint) ([]uint, error) {
	var starredBankIDs []uint
	if len(bankIDs) == 0 {
		return starredBankIDs, nil
	}

	// user_id 和 question_bank_id 联合唯一索引，找到用户star的题库id
	err := r.db.Model(&models.QuestionBankStar{}).
		Where("user_id = ? AND question_bank_id IN ?", userID, bankIDs).
		Pluck("question_bank_id", &starredBankIDs).Error

	return starredBankIDs, err
}

func (r *questionBankRepository) CreateQuestionBank(name, description string, userID uint) (*models.QuestionBank, error) {
	questionBank := &models.QuestionBank{
		Name:        name,
		Description: description,
		CreatedBy:   &userID,
		IsOfficial:  false,
	}

	err := r.db.Create(questionBank).Error
	if err != nil {
		return nil, err
	}

	return questionBank, nil
}

func (r *questionBankRepository) UpdateQuestionBank(bankID, userID uint, updateData QuestionBankUpdateData) error {
	return r.db.Model(&models.QuestionBank{}).
		Where("id = ? AND created_by = ?", bankID, userID).
		Updates(updateData).Error
}

func (r *questionBankRepository) DeleteQuestionBank(bankID, userID uint) error {
	// 检查题库是否存在且属于当前用户
	var questionBank models.QuestionBank
	if err := r.db.Where("id = ? AND created_by = ?", bankID, userID).First(&questionBank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("题库不存在或无权限删除")
		}
		return errors.New("查询题库失败")
	}

	// 检查是否为错题本，错题本不允许删除
	if questionBank.IsWrongBook {
		return errors.New("错题本不允许删除")
	}

	// 定义一个通用的学习计划结构体（根据database模型修改）
	type UserStudyPlan struct {
		ID             uint `gorm:"primaryKey"`
		UserID         uint
		QuestionBankID uint
	}

	// 检查是否有用户正在使用此题库的学习计划
	var studyPlanCount int64
	r.db.Model(&UserStudyPlan{}).Where("question_bank_id = ?", bankID).Count(&studyPlanCount)
	if studyPlanCount > 0 {
		return errors.New("该题库正在被学习计划使用，无法删除")
	}

	// 开始事务删除题库及其题目
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 定义一个通用的题目结构体
	type Question struct {
		ID             uint `gorm:"primaryKey"`
		QuestionBankID uint
	}

	// 删除题库中的所有题目
	if err := tx.Where("question_bank_id = ?", bankID).Delete(&Question{}).Error; err != nil {
		tx.Rollback()
		return errors.New("删除题目失败")
	}

	// 删除题库
	if err := tx.Delete(&questionBank).Error; err != nil {
		tx.Rollback()
		return errors.New("删除题库失败")
	}

	return tx.Commit().Error
}

// GetOrCreateWrongQuestionBook 获取或创建用户的错题本
func (r *questionBankRepository) GetOrCreateWrongQuestionBook(userID uint) (*models.QuestionBank, error) {
	var wrongBook models.QuestionBank

	// 先尝试查找现有的错题本
	err := r.db.Where("created_by = ? AND is_wrong_book = ?", userID, true).First(&wrongBook).Error
	if err == nil {
		return &wrongBook, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 如果不存在，创建新的错题本
	wrongBook = models.QuestionBank{
		Name:        "我的错题本",
		Description: "系统自动创建的错题本，收录学习中未掌握的题目",
		CreatedBy:   &userID,
		IsOfficial:  false,
		IsShared:    false,
		IsWrongBook: true,
	}

	err = r.db.Create(&wrongBook).Error
	if err != nil {
		return nil, err
	}

	return &wrongBook, nil
}

// AddQuestionToWrongBook 添加题目到错题本
func (r *questionBankRepository) AddQuestionToWrongBook(userID, questionID uint) error {
	// 获取或创建错题本
	wrongBook, err := r.GetOrCreateWrongQuestionBook(userID)
	if err != nil {
		return err
	}

	// 获取原题目信息
	var originalQuestion models.Question
	err = r.db.First(&originalQuestion, questionID).Error
	if err != nil {
		return err
	}

	// 检查题目是否已经在错题本中
	var existingQuestion models.Question
	err = r.db.Where("question_bank_id = ? AND title = ? AND leetcode_url = ?",
		wrongBook.ID, originalQuestion.Title, originalQuestion.LeetcodeURL).First(&existingQuestion).Error

	if err == nil {
		// 题目已存在，不重复添加
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return err
	}

	// 创建新的错题记录
	wrongQuestion := models.Question{
		Title:          originalQuestion.Title,
		LeetcodeURL:    originalQuestion.LeetcodeURL,
		Difficulty:     originalQuestion.Difficulty,
		QuestionBankID: wrongBook.ID,
	}

	return r.db.Create(&wrongQuestion).Error
}
