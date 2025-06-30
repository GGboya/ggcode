package repositories

import (
	"errors"
	"ggcode/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InterviewRepository 面试岛仓库接口
type InterviewRepository interface {
	// 岛屿管理
	GetAllIslands() ([]models.InterviewIsland, error)
	GetIslandByID(id uint) (*models.InterviewIsland, error)
	CreateIsland(island *models.InterviewIsland) error
	UpdateIsland(island *models.InterviewIsland) error
	DeleteIsland(id uint) error

	// 关卡管理
	GetLevelsByIslandID(islandID uint) ([]models.InterviewLevel, error)
	GetLevelByID(id uint) (*models.InterviewLevel, error)
	CreateLevel(level *models.InterviewLevel) error
	UpdateLevel(level *models.InterviewLevel) error
	DeleteLevel(id uint) error
	GetMaxLevelNum(islandID uint) (int, error)

	// 用户进度管理
	GetUserIslandProgress(userID uint) ([]IslandProgressInfo, error)
	GetUserLevelProgress(userID, levelID uint) (*models.UserLevelProgress, error)
	CreateOrUpdateLevelProgress(progress *models.UserLevelProgress) error
	UnlockNextLevel(userID, currentLevelID uint) error

	// 提交管理
	CreateSubmission(submission *models.UserLevelSubmission) error
	GetUserSubmissions(userID, levelID uint, limit int) ([]models.UserLevelSubmission, error)
	GetBestSubmission(userID, levelID uint) (*models.UserLevelSubmission, error)

	// 测试用例管理
	GetTestCasesByLevelID(levelID uint) ([]models.InterviewTestCase, error)
	GetSampleTestCases(levelID uint) ([]models.InterviewTestCase, error)
	CreateTestCase(testCase *models.InterviewTestCase) error
	GetTestCase(id uint) (*models.InterviewTestCase, error)
	UpdateTestCase(testCase *models.InterviewTestCase) error
	DeleteTestCase(id uint) error
	AddUserUnlockedTag(userID, tagID uint) error
}

// IslandProgressInfo 岛屿进度信息
type IslandProgressInfo struct {
	Island         models.InterviewIsland     `json:"island"`
	Levels         []models.InterviewLevel    `json:"levels"`
	UserProgress   []models.UserLevelProgress `json:"user_progress"`
	CompletedCount int                        `json:"completed_count"`
	TotalCount     int                        `json:"total_count"`
	TotalStars     int                        `json:"total_stars"`
	MaxStars       int                        `json:"max_stars"`
}

type interviewRepository struct {
	db *gorm.DB
}

func NewInterviewRepository(db *gorm.DB) InterviewRepository {
	return &interviewRepository{db: db}
}

// GetAllIslands 获取所有激活的面试岛
func (r *interviewRepository) GetAllIslands() ([]models.InterviewIsland, error) {
	var islands []models.InterviewIsland
	err := r.db.Where("is_active = ?", true).Order("`order` ASC, created_at ASC").Find(&islands).Error
	return islands, err
}

// GetIslandByID 根据ID获取面试岛
func (r *interviewRepository) GetIslandByID(id uint) (*models.InterviewIsland, error) {
	var island models.InterviewIsland
	err := r.db.Where("id = ? AND is_active = ?", id, true).First(&island).Error
	if err != nil {
		return nil, err
	}
	return &island, nil
}

// CreateIsland 创建面试岛
func (r *interviewRepository) CreateIsland(island *models.InterviewIsland) error {
	return r.db.Create(island).Error
}

// UpdateIsland 更新面试岛信息
func (r *interviewRepository) UpdateIsland(island *models.InterviewIsland) error {
	return r.db.Save(island).Error
}

// DeleteIsland 删除面试岛
func (r *interviewRepository) DeleteIsland(id uint) error {
	return r.db.Delete(&models.InterviewIsland{}, id).Error
}

// GetLevelsByIslandID 获取指定岛屿的所有关卡
func (r *interviewRepository) GetLevelsByIslandID(islandID uint) ([]models.InterviewLevel, error) {
	var levels []models.InterviewLevel
	err := r.db.Where("island_id = ?", islandID).
		Preload("Island").
		Order("level_num ASC").
		Find(&levels).Error
	return levels, err
}

// GetLevelByID 根据ID获取关卡
func (r *interviewRepository) GetLevelByID(id uint) (*models.InterviewLevel, error) {
	var level models.InterviewLevel
	err := r.db.Where("id = ?", id).
		Preload("Island").
		First(&level).Error
	if err != nil {
		return nil, err
	}
	return &level, nil
}

// CreateLevel 创建关卡
func (r *interviewRepository) CreateLevel(level *models.InterviewLevel) error {
	return r.db.Create(level).Error
}

// UpdateLevel 更新关卡
func (r *interviewRepository) UpdateLevel(level *models.InterviewLevel) error {
	return r.db.Save(level).Error
}

// DeleteLevel 删除关卡
func (r *interviewRepository) DeleteLevel(id uint) error {
	return r.db.Delete(&models.InterviewLevel{}, id).Error
}

// GetMaxLevelNum 获取岛屿的最大关卡数
func (r *interviewRepository) GetMaxLevelNum(islandID uint) (int, error) {
	var maxLevelNum int
	err := r.db.Model(&models.InterviewLevel{}).Where("island_id = ?", islandID).Select("COALESCE(MAX(level_num), 0)").Row().Scan(&maxLevelNum)
	if err != nil {
		return 0, err
	}
	return maxLevelNum, nil
}

// GetUserIslandProgress 获取用户所有岛屿的进度
func (r *interviewRepository) GetUserIslandProgress(userID uint) ([]IslandProgressInfo, error) {
	islands, err := r.GetAllIslands()
	if err != nil {
		return nil, err
	}

	var progressInfos []IslandProgressInfo

	for _, island := range islands {
		// 获取岛屿的所有关卡
		levels, err := r.GetLevelsByIslandID(island.ID)
		if err != nil {
			return nil, err
		}

		// 获取用户在这些关卡的进度
		var userProgress []models.UserLevelProgress
		if len(levels) > 0 {
			levelIDs := make([]uint, len(levels))
			for i, level := range levels {
				levelIDs[i] = level.ID
			}

			r.db.Where("user_id = ? AND level_id IN ?", userID, levelIDs).
				Find(&userProgress)
		}

		// 初始化第一关卡为解锁状态
		if len(levels) > 0 && len(userProgress) == 0 {
			firstProgress := models.UserLevelProgress{
				UserID:  userID,
				LevelID: levels[0].ID,
				Status:  "unlocked",
			}
			r.db.Create(&firstProgress)
			userProgress = append(userProgress, firstProgress)
		}

		// 计算统计信息
		completedCount := 0
		totalStars := 0
		for _, progress := range userProgress {
			if progress.Status == "completed" {
				completedCount++
				totalStars += progress.Stars
			}
		}

		progressInfo := IslandProgressInfo{
			Island:         island,
			Levels:         levels,
			UserProgress:   userProgress,
			CompletedCount: completedCount,
			TotalCount:     len(levels),
			TotalStars:     totalStars,
			MaxStars:       len(levels) * 3, // 每关最多3星
		}

		progressInfos = append(progressInfos, progressInfo)
	}

	return progressInfos, nil
}

// GetUserLevelProgress 获取用户关卡进度
func (r *interviewRepository) GetUserLevelProgress(userID, levelID uint) (*models.UserLevelProgress, error) {
	var progress models.UserLevelProgress
	err := r.db.Where("user_id = ? AND level_id = ?", userID, levelID).
		Preload("Level").
		First(&progress).Error

	if err == gorm.ErrRecordNotFound {
		// 如果没有记录，检查是否应该创建
		level, err := r.GetLevelByID(levelID)
		if err != nil {
			return nil, err
		}

		// 检查前置关卡是否已完成
		if level.LevelNum == 1 {
			// 第一关自动解锁
			progress = models.UserLevelProgress{
				UserID:  userID,
				LevelID: levelID,
				Status:  "unlocked",
			}
			r.db.Create(&progress)
		} else {
			// 其他关卡默认锁定
			progress = models.UserLevelProgress{
				UserID:  userID,
				LevelID: levelID,
				Status:  "locked",
			}
		}
		return &progress, nil
	}

	return &progress, err
}

// CreateOrUpdateLevelProgress 创建或更新关卡进度
func (r *interviewRepository) CreateOrUpdateLevelProgress(progress *models.UserLevelProgress) error {
	var existing models.UserLevelProgress
	err := r.db.Where("user_id = ? AND level_id = ?", progress.UserID, progress.LevelID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return r.db.Create(progress).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录
	existing.Status = progress.Status
	if progress.Stars > existing.Stars {
		existing.Stars = progress.Stars
	}
	if progress.BestTime == 0 || (existing.BestTime > 0 && progress.BestTime < existing.BestTime) {
		existing.BestTime = progress.BestTime
	}
	existing.AttemptCount++
	if progress.Status == "completed" && existing.CompletedAt == nil {
		now := time.Now()
		existing.CompletedAt = &now
	}

	return r.db.Save(&existing).Error
}

// UnlockNextLevel 解锁下一关卡
func (r *interviewRepository) UnlockNextLevel(userID, currentLevelID uint) error {
	// 获取当前关卡信息
	currentLevel, err := r.GetLevelByID(currentLevelID)
	if err != nil {
		return err
	}

	// 查找下一关卡
	var nextLevel models.InterviewLevel
	err = r.db.Where("island_id = ? AND level_num = ?",
		currentLevel.IslandID, currentLevel.LevelNum+1).First(&nextLevel).Error

	if err == gorm.ErrRecordNotFound {
		// 没有下一关卡了
		return nil
	} else if err != nil {
		return err
	}

	// 检查下一关卡的进度
	var nextProgress models.UserLevelProgress
	err = r.db.Where("user_id = ? AND level_id = ?", userID, nextLevel.ID).First(&nextProgress).Error

	if err == gorm.ErrRecordNotFound {
		// 创建下一关卡的进度记录
		nextProgress = models.UserLevelProgress{
			UserID:  userID,
			LevelID: nextLevel.ID,
			Status:  "unlocked",
		}
		return r.db.Create(&nextProgress).Error
	} else if err != nil {
		return err
	}

	// 如果已存在但是锁定状态，解锁它
	if nextProgress.Status == "locked" {
		nextProgress.Status = "unlocked"
		return r.db.Save(&nextProgress).Error
	}

	return nil
}

// CreateSubmission 创建提交记录
func (r *interviewRepository) CreateSubmission(submission *models.UserLevelSubmission) error {
	return r.db.Create(submission).Error
}

// GetUserSubmissions 获取用户关卡提交记录
func (r *interviewRepository) GetUserSubmissions(userID, levelID uint, limit int) ([]models.UserLevelSubmission, error) {
	var submissions []models.UserLevelSubmission
	query := r.db.Where("user_id = ? AND level_id = ?", userID, levelID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&submissions).Error
	return submissions, err
}

// GetBestSubmission 获取用户关卡最佳提交
func (r *interviewRepository) GetBestSubmission(userID, levelID uint) (*models.UserLevelSubmission, error) {
	var submission models.UserLevelSubmission
	err := r.db.Where("user_id = ? AND level_id = ? AND status = ?", userID, levelID, "AC").
		Order("stars DESC, submit_time ASC").
		First(&submission).Error

	if err != nil {
		return nil, err
	}
	return &submission, nil
}

// GetTestCasesByLevelID 获取关卡的所有测试用例
func (r *interviewRepository) GetTestCasesByLevelID(levelID uint) ([]models.InterviewTestCase, error) {
	var testCases []models.InterviewTestCase
	err := r.db.Where("level_id = ?", levelID).Order("`order` ASC").Find(&testCases).Error
	return testCases, err
}

// GetSampleTestCases 获取关卡的样例测试用例
func (r *interviewRepository) GetSampleTestCases(levelID uint) ([]models.InterviewTestCase, error) {
	var testCases []models.InterviewTestCase
	err := r.db.Where("level_id = ? AND is_sample = ?", levelID, true).Order("`order` ASC").Find(&testCases).Error
	return testCases, err
}

// CreateTestCase 创建测试用例
func (r *interviewRepository) CreateTestCase(testCase *models.InterviewTestCase) error {
	return r.db.Create(testCase).Error
}

// GetTestCase 获取单个测试用例
func (r *interviewRepository) GetTestCase(id uint) (*models.InterviewTestCase, error) {
	var tc models.InterviewTestCase
	if err := r.db.First(&tc, id).Error; err != nil {
		return nil, err
	}
	return &tc, nil
}

// UpdateTestCase 更新测试用例
func (r *interviewRepository) UpdateTestCase(tc *models.InterviewTestCase) error {
	return r.db.Save(tc).Error
}

// DeleteTestCase 删除测试用例
func (r *interviewRepository) DeleteTestCase(id uint) error {
	result := r.db.Delete(&models.InterviewTestCase{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("测试用例不存在或已被删除")
	}
	return nil
}

// AddUserUnlockedTag 写入用户已解锁知识点 (忽略已存在)
func (r *interviewRepository) AddUserUnlockedTag(userID, tagID uint) error {
	record := models.UserUnlockedTag{
		UserID:    userID,
		AlgoTagID: tagID,
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error
}
