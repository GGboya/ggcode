package repositories

import (
	"ggcode/internal/models"

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
	ForkQuestionBank(bankID, userID uint) (*models.QuestionBank, error)
	GetUserStarredBanks(userID uint, page, limit int) ([]models.QuestionBank, int64, error)
}

type shareRepository struct {
	db *gorm.DB
}

func NewShareRepository(db *gorm.DB) ShareRepository {
	return &shareRepository{db: db}
}

func (r *shareRepository) CheckQuestionBankOwnership(bankID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.QuestionBank{}).
		Where("id = ? AND created_by = ?", bankID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *shareRepository) CheckQuestionBankShared(bankID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.QuestionBank{}).
		Where("id = ? AND (is_official = ? OR is_shared = ?)", bankID, true, true).
		Count(&count).Error
	return count > 0, err
}

func (r *shareRepository) CheckQuestionBankStarred(bankID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.QuestionBankStar{}).
		Where("user_id = ? AND question_bank_id = ?", userID, bankID).
		Count(&count).Error
	return count > 0, err
}

func (r *shareRepository) CheckQuestionBankForked(bankID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.QuestionBank{}).
		Where("created_by = ? AND forked_from = ?", userID, bankID).
		Count(&count).Error
	return count > 0, err
}

func (r *shareRepository) ShareQuestionBank(bankID uint) error {
	return r.db.Model(&models.QuestionBank{}).
		Where("id = ?", bankID).
		Update("is_shared", true).Error
}

func (r *shareRepository) UnshareQuestionBank(bankID uint) error {
	// 开始事务
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 将题库设为非共享
	if err := tx.Model(&models.QuestionBank{}).
		Where("id = ?", bankID).
		Update("is_shared", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. 对所有 fork 自该题库且仍与原题库共享题目的题库执行写时复制
	var forkedBanks []models.QuestionBank
	if err := tx.Where("forked_from = ?", bankID).Find(&forkedBanks).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, fb := range forkedBanks {
		var localCount int64
		if err := tx.Model(&models.Question{}).Where("question_bank_id = ?", fb.ID).Count(&localCount).Error; err != nil {
			tx.Rollback()
			return err
		}

		// 仅当 fork 的题库尚未写时复制（本地题数为0）时，才进行复制
		if localCount == 0 {
			if err := duplicateQuestions(tx, bankID, fb.ID); err != nil {
				tx.Rollback()
				return err
			}

			// 与原题库解除关联，后续不再共享
			if err := tx.Model(&models.QuestionBank{}).Where("id = ?", fb.ID).Update("forked_from", nil).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

// duplicateQuestions 在写时复制触发时，将原题库的题目复制到目标题库
func duplicateQuestions(tx *gorm.DB, fromBankID, toBankID uint) error {
	var originalQuestions []models.Question
	if err := tx.Where("question_bank_id = ?", fromBankID).Find(&originalQuestions).Error; err != nil {
		return err
	}

	for _, q := range originalQuestions {
		newQ := models.Question{
			Title:          q.Title,
			LeetcodeURL:    q.LeetcodeURL,
			Difficulty:     q.Difficulty,
			Description:    q.Description,
			QuestionBankID: toBankID,
		}
		if err := tx.Create(&newQ).Error; err != nil {
			return err
		}
	}
	return nil
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
	star := models.QuestionBankStar{
		UserID:         userID,
		QuestionBankID: bankID,
	}
	if err := tx.Create(&star).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新题库的Star数量
	if err := tx.Model(&models.QuestionBank{}).
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
		Delete(&models.QuestionBankStar{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新题库的Star数量
	if err := tx.Model(&models.QuestionBank{}).
		Where("id = ?", bankID).
		Update("star_count", gorm.Expr("star_count - ?", 1)).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *shareRepository) ForkQuestionBank(bankID, userID uint) (*models.QuestionBank, error) {
	// 开始事务
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取原题库
	var originalBank models.QuestionBank
	if err := tx.Where("id = ?", bankID).First(&originalBank).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 创建 Fork 元题库记录（不复制题目）
	fName := originalBank.Name + " (Fork)"
	forkedBank := models.QuestionBank{
		Name:        fName,
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

	// 更新原题库的 Fork 数量
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

func (r *shareRepository) GetUserStarredBanks(userID uint, page, limit int) ([]models.QuestionBank, int64, error) {
	var starredBanks []models.QuestionBank
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
