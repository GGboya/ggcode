package database

import (
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Init 初始化数据库连接和表结构
func Init() (*gorm.DB, error) {
	// 从环境变量获取数据库配置，如果没有则使用默认值
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "ggcode")

	// 构建MySQL DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %v", err)
	}

	// 自动迁移表结构
	err = db.AutoMigrate(
		&User{},
		&QuestionBank{},
		&Question{},
		&UserStudyPlan{},
		&UserQuestionProgress{},
		&UserCheckIn{},
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

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// initOfficialData 初始化官方题库数据（LeetCode Hot 100）
func initOfficialData(db *gorm.DB) error {
	// 检查是否已存在官方题库
	var count int64
	db.Model(&QuestionBank{}).Where("is_official = ?", true).Count(&count)
	if count > 0 {
		return nil // 已存在，不重复初始化
	}

	// 创建官方题库
	officialBank := QuestionBank{
		Name:        "LeetCode Hot 100",
		Description: "LeetCode 热题 HOT 100",
		IsOfficial:  true,
		CreatedBy:   nil, // 系统创建，无创建者
	}

	if err := db.Create(&officialBank).Error; err != nil {
		return err
	}

	// 添加部分 Hot 100 题目（示例）
	questions := []Question{
		{Title: "两数之和", LeetcodeURL: "https://leetcode.cn/problems/two-sum/", Difficulty: "Easy", QuestionBankID: officialBank.ID},
		{Title: "两数相加", LeetcodeURL: "https://leetcode.cn/problems/add-two-numbers/", Difficulty: "Medium", QuestionBankID: officialBank.ID},
		{Title: "无重复字符的最长子串", LeetcodeURL: "https://leetcode.cn/problems/longest-substring-without-repeating-characters/", Difficulty: "Medium", QuestionBankID: officialBank.ID},
		{Title: "寻找两个正序数组的中位数", LeetcodeURL: "https://leetcode.cn/problems/median-of-two-sorted-arrays/", Difficulty: "Hard", QuestionBankID: officialBank.ID},
		{Title: "最长回文子串", LeetcodeURL: "https://leetcode.cn/problems/longest-palindromic-substring/", Difficulty: "Medium", QuestionBankID: officialBank.ID},
		{Title: "盛最多水的容器", LeetcodeURL: "https://leetcode.cn/problems/container-with-most-water/", Difficulty: "Medium", QuestionBankID: officialBank.ID},
		{Title: "三数之和", LeetcodeURL: "https://leetcode.cn/problems/3sum/", Difficulty: "Medium", QuestionBankID: officialBank.ID},
		{Title: "电话号码的字母组合", LeetcodeURL: "https://leetcode.cn/problems/letter-combinations-of-a-phone-number/", Difficulty: "Medium", QuestionBankID: officialBank.ID},
		{Title: "四数之和", LeetcodeURL: "https://leetcode.cn/problems/4sum/", Difficulty: "Medium", QuestionBankID: officialBank.ID},
		{Title: "删除链表的倒数第N个结点", LeetcodeURL: "https://leetcode.cn/problems/remove-nth-node-from-end-of-list/", Difficulty: "Medium", QuestionBankID: officialBank.ID},
	}

	for _, question := range questions {
		if err := db.Create(&question).Error; err != nil {
			return err
		}
	}

	return nil
}
