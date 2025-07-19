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

	// 统计今天学习的题目数量
	var studyCount int64
	err = s.db.Table("user_question_progresses").
		Where("user_id = ? AND DATE(last_review_date) = DATE(?)", userID, today).
		Count(&studyCount).Error
	if err != nil {
		studyCount = 0 // 如果查询失败，默认为0
	}

	// 创建打卡记录
	checkIn := models.UserCheckIn{
		UserID:          userID,
		CheckDate:       today,
		ConsecutiveDays: consecutiveDays,
		BestStreak:      bestStreak,
		StudyCount:      int(studyCount),
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
	// 获取过去一年的日期范围
	endDate := time.Now()
	startDate := endDate.AddDate(-1, 0, 0) // 一年前

	// 查询用户在过去一年的打卡记录（包含学习数量）
	var checkInStats []struct {
		Date       time.Time `json:"date"`
		StudyCount int       `json:"study_count"`
	}

	err := s.db.Table("user_check_ins").
		Select("DATE(check_date) as date, study_count").
		Where("user_id = ? AND DATE(check_date) >= DATE(?) AND DATE(check_date) < DATE(?)", userID, startDate.Format("2006-01-02"), endDate.AddDate(0, 0, 1).Format("2006-01-02")).
		Order("date").
		Scan(&checkInStats).Error

	if err != nil {
		return nil, err
	}

	// 查询用户在过去一年的学习记录（作为补充数据）
	var dailyStats []struct {
		Date  time.Time `json:"date"`
		Count int64     `json:"count"`
	}

	err = s.db.Table("user_question_progresses").
		Select("DATE(last_review_date) as date, COUNT(DISTINCT question_id) as count").
		Where("user_id = ? AND DATE(last_review_date) >= DATE(?) AND DATE(last_review_date) < DATE(?) AND last_review_date IS NOT NULL", userID, startDate.Format("2006-01-02"), endDate.AddDate(0, 0, 1).Format("2006-01-02")).
		Group("DATE(last_review_date)").
		Order("date").
		Scan(&dailyStats).Error

	if err != nil {
		return nil, err
	}

	// 构建热力图数据
	heatmapData := make([]HeatmapData, 0)
	currentDate := startDate

	// 创建日期到打卡记录学习次数的映射
	checkInMap := make(map[string]int)
	for _, checkIn := range checkInStats {
		dateStr := checkIn.Date.Format("2006-01-02")
		checkInMap[dateStr] = checkIn.StudyCount
	}

	// 创建日期到学习记录次数的映射
	statsMap := make(map[string]int64)
	for _, stat := range dailyStats {
		dateStr := stat.Date.Format("2006-01-02")
		statsMap[dateStr] = stat.Count
	}

	// 填充过去一年的每一天，包括今天
	now := time.Now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	for currentDate.Before(tomorrow) {
		dateStr := currentDate.Format("2006-01-02")

		// 优先使用打卡记录中的学习数量，如果没有则使用学习记录表的数据
		count := checkInMap[dateStr]
		if count == 0 {
			// 如果打卡记录中没有学习数量，尝试从学习记录表获取
			count = int(statsMap[dateStr])
		}

		// 根据学习次数确定活跃度级别 (0-4)
		level := 0
		if count > 0 {
			if count >= 10 {
				level = 4
			} else if count >= 7 {
				level = 3
			} else if count >= 4 {
				level = 2
			} else {
				level = 1
			}
		}

		heatmapData = append(heatmapData, HeatmapData{
			Date:  dateStr,
			Count: count,
			Level: level,
		})

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// 计算统计信息
	totalStudyDays := 0     // 总学习天数
	totalStudyCount := 0    // 总学习次数（题目数）
	currentStreak := 0      // 当前连续学习天数
	maxStreak := 0          // 最长连续学习天数
	tempStreak := 0         // 临时连续天数
	thisYearStudyCount := 0 // 今年学习次数

	// 计算总学习天数和总学习次数
	for _, data := range heatmapData {
		if data.Count > 0 {
			totalStudyDays++
			totalStudyCount += data.Count
		}
	}

	// 计算当前连续学习天数（从今天往前推）
	for i := len(heatmapData) - 1; i >= 0; i-- {
		if heatmapData[i].Count > 0 {
			currentStreak++
		} else {
			break // 遇到没有学习的天数就停止
		}
	}

	// 计算最长连续学习天数
	for i := 0; i < len(heatmapData); i++ {
		if heatmapData[i].Count > 0 {
			tempStreak++
			if tempStreak > maxStreak {
				maxStreak = tempStreak
			}
		} else {
			tempStreak = 0 // 重置临时连续天数
		}
	}

	// 计算今年的学习次数
	currentYear := time.Now().Year()
	for _, data := range heatmapData {
		date, _ := time.Parse("2006-01-02", data.Date)
		if date.Year() == currentYear {
			thisYearStudyCount += data.Count
		}
	}

	return &HeatmapResponse{
		Data:          heatmapData,
		TotalCommits:  totalStudyCount,    // 总学习次数（题目数）
		CurrentStreak: currentStreak,      // 当前连续学习天数
		MaxStreak:     maxStreak,          // 最长连续学习天数
		ThisYear:      thisYearStudyCount, // 今年学习次数
	}, nil
}

// max 返回两个整数中的较大值
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
