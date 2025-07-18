package services

import (
	"errors"
	"ggcode/internal/models"
	"ggcode/internal/repositories"
	"time"

	"gorm.io/gorm"
)

type CheckInServiceInterface interface {
	CheckInToday(userID uint) error
	GetCheckInStats(userID uint) (*CheckInStats, error)
	GetStudyHeatmap(userID uint) (*HeatmapResponse, error)
}

type CheckInService struct {
	checkInRepo repositories.CheckInRepository
	db          *gorm.DB
}

func NewCheckInService(checkInRepo repositories.CheckInRepository, db *gorm.DB) *CheckInService {
	return &CheckInService{
		checkInRepo: checkInRepo,
		db:          db,
	}
}

// CheckInToday 用户今日打卡（支持连续天数和最大连续天数统计）
func (s *CheckInService) CheckInToday(userID uint) error {
	// 使用本地时区获取今天的开始时间
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 检查今日是否已打卡
	var existingCheckIn models.UserCheckIn
	err := s.checkInRepo.GetUserCheckInByDate(userID, today, &existingCheckIn)
	if err == nil {
		return errors.New("今日已打卡")
	}

	// 获取昨天的打卡记录来计算连续天数和最长连续天数
	yesterday := today.AddDate(0, 0, -1)
	var yesterdayCheckIn models.UserCheckIn
	consecutiveDays := 1 // 默认为1（今天是第一天）
	bestStreak := 1      // 默认为1

	err = s.checkInRepo.GetUserCheckInByDate(userID, yesterday, &yesterdayCheckIn)
	if err == nil {
		// 昨天有打卡，连续天数 = 昨天的连续天数 + 1
		consecutiveDays = yesterdayCheckIn.ConsecutiveDays + 1
		// 最长连续天数 = max(当前连续天数, 昨天的最长连续天数)
		bestStreak = int(max(int64(consecutiveDays), int64(yesterdayCheckIn.BestStreak)))
	} else {
		// 昨天没打卡，获取最近的一条记录来获取历史最长连续天数
		var latestCheckIn models.UserCheckIn
		err = s.checkInRepo.GetLatestUserCheckIn(userID, &latestCheckIn)
		if err == nil {
			// 有历史记录，使用历史最长连续天数
			bestStreak = int(max(1, int64(latestCheckIn.BestStreak)))
		}
		// 如果没有历史记录，bestStreak 保持默认值 1
	}

	// 创建打卡记录
	checkIn := models.UserCheckIn{
		UserID:          userID,
		CheckDate:       today,
		ConsecutiveDays: consecutiveDays,
		BestStreak:      bestStreak,
	}

	return s.checkInRepo.CreateUserCheckIn(&checkIn)
}

// GetCheckInStats 获取打卡统计信息
func (s *CheckInService) GetCheckInStats(userID uint) (*CheckInStats, error) {
	// 使用本地时区获取今天的开始时间
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var stats CheckInStats

	// 检查今日是否已打卡
	var todayCheckIn models.UserCheckIn
	err := s.checkInRepo.GetUserCheckInByDate(userID, today, &todayCheckIn)
	if err == nil {
		stats.CheckedInToday = true
		stats.ConsecutiveDays = int64(todayCheckIn.ConsecutiveDays)
		stats.BestStreak = int64(todayCheckIn.BestStreak)
	} else {
		stats.CheckedInToday = false
		// 获取最近的一条记录来获取连续天数和最长连续天数
		var latestCheckIn models.UserCheckIn
		err = s.checkInRepo.GetLatestUserCheckIn(userID, &latestCheckIn)
		if err == nil {
			stats.ConsecutiveDays = int64(latestCheckIn.ConsecutiveDays)
			stats.BestStreak = int64(latestCheckIn.BestStreak)
		}
	}

	// 统计总打卡天数
	var totalCount int64
	s.db.Model(&models.UserCheckIn{}).Where("user_id = ?", userID).Count(&totalCount)
	stats.TotalCheckInDays = totalCount

	return &stats, nil
}

// GetStudyHeatmap 获取学习活动热力图数据
func (s *CheckInService) GetStudyHeatmap(userID uint) (*HeatmapResponse, error) {
	// 获取过去一年的学习活动数据
	oneYearAgo := time.Now().AddDate(-1, 0, 0)

	var heatmapData []HeatmapData
	var totalCommits int
	var currentStreak int
	var maxStreak int
	var thisYear int

	// 查询过去一年的学习活动
	err := s.db.Table("user_question_progresses").
		Select("DATE(last_review_date) as date, COUNT(*) as count").
		Where("user_id = ? AND last_review_date >= ?", userID, oneYearAgo).
		Group("DATE(last_review_date)").
		Order("date ASC").
		Scan(&heatmapData).Error

	if err != nil {
		return nil, err
	}

	// 计算活跃度级别
	for i := range heatmapData {
		count := heatmapData[i].Count
		switch {
		case count == 0:
			heatmapData[i].Level = 0
		case count <= 3:
			heatmapData[i].Level = 1
		case count <= 6:
			heatmapData[i].Level = 2
		case count <= 10:
			heatmapData[i].Level = 3
		default:
			heatmapData[i].Level = 4
		}
	}

	// 计算总学习天数
	totalCommits = len(heatmapData)

	// 计算当前连续天数和最长连续天数
	currentStreak = 0
	maxStreak = 0
	thisYear = 0
	currentDate := time.Now()
	yearStart := time.Date(currentDate.Year(), 1, 1, 0, 0, 0, 0, currentDate.Location())

	// 创建日期映射，用于快速查找
	dateMap := make(map[string]bool)
	for _, data := range heatmapData {
		dateMap[data.Date] = true
	}

	// 计算连续天数
	streak := 0
	for i := 0; i < 365; i++ {
		checkDate := currentDate.AddDate(0, 0, -i)
		dateStr := checkDate.Format("2006-01-02")

		if dateMap[dateStr] {
			streak++
			if checkDate.After(yearStart) {
				thisYear++
			}
		} else {
			if streak > maxStreak {
				maxStreak = streak
			}
			if currentStreak == 0 && streak > 0 {
				currentStreak = streak
			}
			streak = 0
		}
	}

	// 处理边界情况
	if streak > maxStreak {
		maxStreak = streak
	}
	if currentStreak == 0 && streak > 0 {
		currentStreak = streak
	}

	return &HeatmapResponse{
		Data:          heatmapData,
		TotalCommits:  totalCommits,
		CurrentStreak: currentStreak,
		MaxStreak:     maxStreak,
		ThisYear:      thisYear,
	}, nil
}

// max 返回两个整数中的较大值
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
