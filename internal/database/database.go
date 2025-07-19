package database

import (
	"fmt"
	"ggcode/internal/config"
	"ggcode/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Init 初始化数据库连接和表结构
func Init(cfg *config.Config) (*gorm.DB, error) {
	// 使用配置中的数据库连接信息
	dsn := cfg.Database.GetDSN()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %v", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 自动迁移表结构
	err = db.AutoMigrate(
		&models.User{},
		&models.QuestionBank{},
		&models.QuestionBankStar{},
		&models.Question{},
		&models.UserStudyPlan{},
		&models.UserQuestionProgress{},
		&models.UserCheckIn{},
		&models.InterviewIsland{},
		&models.InterviewLevel{},
		&models.UserLevelProgress{},
		&models.UserLevelSubmission{},
		&models.InterviewTestCase{},
		&models.AlgoTag{},
		&models.QuestionTag{},
		&models.UserUnlockedTag{},
		&models.DailyStudyPlanCache{},
		&models.ContestProblem{}, // 新增
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	// 初始化官方题库数据
	if err := initOfficialData(db); err != nil {
		return nil, fmt.Errorf("failed to initialize official data: %v", err)
	}

	return db, nil
}

// initOfficialData 初始化官方题库数据（LeetCode Hot 100）
func initOfficialData(db *gorm.DB) error {
	// 检查是否已经初始化过
	var count int64
	db.Model(&models.QuestionBank{}).Where("is_official = ?", true).Count(&count)
	if count > 0 {
		return nil // 已经初始化过了
	}

	// 创建官方题库
	officialBank := &models.QuestionBank{
		Name:        "LeetCode Hot 100",
		Description: "LeetCode 热门题目精选，包含最常考的算法题目",
		IsOfficial:  true,
		IsShared:    true,
		CreatedBy:   nil, // 系统创建，无创建者
	}

	if err := db.Create(officialBank).Error; err != nil {
		return fmt.Errorf("failed to create official question bank: %v", err)
	}

	// 这里可以添加具体的题目数据
	// 为了简化，这里只创建题库结构

	return nil
}
