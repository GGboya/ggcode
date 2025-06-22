package repositories

import (
	"ggcode/internal/database"

	"gorm.io/gorm"
)

type ShareRepository interface {
	CheckQuestionBankOwnership(bankID, userID uint) (bool, error)
	CheckQuestionBankShared(bankID uint) (bool, error)
	CheckQuestionBankStarred(bankID, userID uint) (bool, error)
	CheckQuestionBankForked(bankID, userID uint) (bool, error)
	ShareQuestionBank(bankID uint) error
	UnshareQuestionBank(bankID uint) error
	StarQuestionBank(bankID, userID uint) error
	UnstarQuestionBank(bankID, userID uint) error
	ForkQuestionBank(bankID, userID uint) (*database.QuestionBank, error)
	GetUserStarredBanks(userID uint, page, limit int) ([]database.QuestionBank, int64, error)
}

type shareRepository struct {
	db *gorm.DB
}

func NewShareRepository(db *gorm.DB) ShareRepository {
	return &shareRepository{db: db}
}

func (r *shareRepository) CheckQuestionBankOwnership(bankID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&database.QuestionBank{}).
		Where("id = ? AND created_by = ?", bankID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *shareRepository) CheckQuestionBankShared(bankID uint) (bool, error) {
	var count int64
	err := r.db.Model(&database.QuestionBank{}).
		Where("id = ? AND (is_official = ? OR is_shared = ?)", bankID, true, true).
		Count(&count).Error
	return count > 0, err
}

func (r *shareRepository) CheckQuestionBankStarred(bankID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&database.QuestionBankStar{}).
		Where("user_id = ? AND question_bank_id = ?", userID, bankID).
		Count(&count).Error
	return count > 0, err
}

func (r *shareRepository) CheckQuestionBankForked(bankID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&database.QuestionBank{}).
		Where("created_by = ? AND forked_from = ?", userID, bankID).
		Count(&count).Error
	return count > 0, err
}

func (r *shareRepository) ShareQuestionBank(bankID uint) error {
	return r.db.Model(&database.QuestionBank{}).
		Where("id = ?", bankID).
		Update("is_shared", true).Error
}

func (r *shareRepository) UnshareQuestionBank(bankID uint) error {
	return r.db.Model(&database.QuestionBank{}).
		Where("id = ?", bankID).
		Update("is_shared", false).Error
}

func (r *shareRepository) StarQuestionBank(bankID, userID uint) error {
	// 开始事务
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建Star记录
	star := database.QuestionBankStar{
		UserID:         userID,
		QuestionBankID: bankID,
	}
	if err := tx.Create(&star).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新题库的Star数量
	if err := tx.Model(&database.QuestionBank{}).
		Where("id = ?", bankID).
		Update("star_count", gorm.Expr("star_count + ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *shareRepository) UnstarQuestionBank(bankID, userID uint) error {
	// 开始事务
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除Star记录
	if err := tx.Where("user_id = ? AND question_bank_id = ?", userID, bankID).
		Delete(&database.QuestionBankStar{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新题库的Star数量
	if err := tx.Model(&database.QuestionBank{}).
		Where("id = ?", bankID).
		Update("star_count", gorm.Expr("star_count - ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *shareRepository) ForkQuestionBank(bankID, userID uint) (*database.QuestionBank, error) {
	// 开始事务
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取原题库
	var originalBank database.QuestionBank
	if err := tx.Where("id = ?", bankID).First(&originalBank).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 创建Fork的题库
	forkedBank := database.QuestionBank{
		Name:        originalBank.Name + " (Fork)",
		Description: originalBank.Description,
		CreatedBy:   &userID,
		ForkedFrom:  &originalBank.ID,
		IsOfficial:  false,
		IsShared:    false,
	}

	if err := tx.Create(&forkedBank).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 复制原题库的所有题目
	var originalQuestions []database.Question
	if err := tx.Where("question_bank_id = ?", bankID).Find(&originalQuestions).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 批量插入题目
	for _, question := range originalQuestions {
		newQuestion := database.Question{
			Title:          question.Title,
			LeetcodeURL:    question.LeetcodeURL,
			Difficulty:     question.Difficulty,
			QuestionBankID: forkedBank.ID,
		}
		if err := tx.Create(&newQuestion).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// 更新原题库的Fork数量
	if err := tx.Model(&originalBank).
		Update("fork_count", gorm.Expr("fork_count + ?", 1)).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &forkedBank, nil
}

func (r *shareRepository) GetUserStarredBanks(userID uint, page, limit int) ([]database.QuestionBank, int64, error) {
	var starredBanks []database.QuestionBank
	var total int64

	baseQuery := r.db.Table("question_banks").
		Joins("JOIN question_bank_stars ON question_banks.id = question_bank_stars.question_bank_id").
		Where("question_bank_stars.user_id = ?", userID)

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := baseQuery.Offset(offset).Limit(limit).
		Preload("Creator").
		Find(&starredBanks).Error; err != nil {
		return nil, 0, err
	}

	return starredBanks, total, nil
}
