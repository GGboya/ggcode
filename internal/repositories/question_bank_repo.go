package repositories

import (
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

type QuestionBankRepository interface {
	GetQuestionBanks(options QuestionBankQueryOptions) (*QuestionBankListResult, error)
	GetStarredBankIDs(userID uint, bankIDs []uint) ([]uint, error)
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
