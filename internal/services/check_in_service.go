package services

import (
	"ggcode/internal/events"
	"ggcode/internal/models"
	"ggcode/internal/pkg/logger"
	"ggcode/internal/repositories"
	"time"
)

// HeatmapData 热力图数据结构
type HeatmapData struct {
	Date  string `json:"date"`  // YYYY-MM-DD 格式
	Count int    `json:"count"` // 当天学习题目数量
	Level int    `json:"level"` // 活跃度级别 0-4
}

// HeatmapResponse 热力图响应结构
type HeatmapResponse struct {
	Data          []HeatmapData `json:"data"`
	TotalCommits  int           `json:"total_commits"`  // 总学习天数
	CurrentStreak int           `json:"current_streak"` // 当前连续天数
	MaxStreak     int           `json:"max_streak"`     // 最长连续天数
	ThisYear      int           `json:"this_year"`      // 今年学习天数
}

type CheckInServiceInterface interface {
	CheckInToday(userID uint) error
	GetCheckInStats(userID uint) (*models.CheckInStat, error)
	GetStudyHeatmap(userID uint) (*HeatmapResponse, error)
}

type CheckInService struct {
	checkInRepo      repositories.CheckInRepository
	userQuestionRepo repositories.UserQuestionRepository
	bus              *events.EventBus
}

func NewCheckInService(checkInRepo repositories.CheckInRepository, userQuestionRepo repositories.UserQuestionRepository, bus *events.EventBus) *CheckInService {
	checkInService := &CheckInService{
		checkInRepo:      checkInRepo,
		userQuestionRepo: userQuestionRepo,
		bus:              bus,
	}
	checkInService.StartEventListener()
	return checkInService
}

// CheckInToday 用户今日打卡（支持连续天数和最大连续天数统计）
func (s *CheckInService) CheckInToday(userID uint) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 检查今日是否已打卡
	var existingCheckIn models.UserCheckIn
	err := s.checkInRepo.GetUserCheckInByDate(userID, today, &existingCheckIn)
	if err == nil {
		// 今日已打卡，更新学习数量
		studyCount, err := s.userQuestionRepo.GetUserDailyStudyCount(userID, today)
		if err != nil {
			studyCount = 0
		}
		existingCheckIn.StudyCount = int(studyCount)
		return s.checkInRepo.UpdateUserCheckIn(&existingCheckIn)
	}

	// 获取昨天的打卡记录来计算连续天数和最长连续天数
	yesterday := today.AddDate(0, 0, -1)
	var yesterdayCheckIn models.UserCheckIn
	consecutiveDays := 1 // 默认为1（今天是第一天）
	bestStreak := 1      // 默认为1

	err = s.checkInRepo.GetUserCheckInByDate(userID, yesterday, &yesterdayCheckIn)
	if err == nil {
		consecutiveDays = yesterdayCheckIn.ConsecutiveDays + 1
		bestStreak = max(consecutiveDays, yesterdayCheckIn.BestStreak)
	} else {
		var latestCheckIn models.UserCheckIn
		err = s.checkInRepo.GetLatestUserCheckIn(userID, &latestCheckIn)
		if err == nil {
			bestStreak = max(1, latestCheckIn.BestStreak)
		}
	}

	// 统计今天学习的题目数量
	studyCount, err := s.userQuestionRepo.GetUserDailyStudyCount(userID, today)
	if err != nil {
		studyCount = 0
	}

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
func (s *CheckInService) GetCheckInStats(userID uint) (*models.CheckInStat, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var stats models.CheckInStat

	var todayCheckIn models.UserCheckIn
	err := s.checkInRepo.GetUserCheckInByDate(userID, today, &todayCheckIn)
	if err == nil {
		stats.CheckedInToday = true
		stats.ConsecutiveDays = int64(todayCheckIn.ConsecutiveDays)
		stats.BestStreak = int64(todayCheckIn.BestStreak)
	} else {
		stats.CheckedInToday = false
		var latestCheckIn models.UserCheckIn
		err = s.checkInRepo.GetLatestUserCheckIn(userID, &latestCheckIn)
		if err == nil {
			stats.ConsecutiveDays = int64(latestCheckIn.ConsecutiveDays)
			stats.BestStreak = int64(latestCheckIn.BestStreak)
		}
	}

	totalCount, err := s.checkInRepo.GetTotalCheckInDays(userID)
	if err != nil {
		totalCount = 0
	}
	stats.TotalCheckInDays = totalCount

	return &stats, nil
}

// GetStudyHeatmap 获取学习活动热力图数据
func (s *CheckInService) GetStudyHeatmap(userID uint) (*HeatmapResponse, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(-1, 0, 0)

	checkInStats, err := s.checkInRepo.GetUserYearlyCheckInStats(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	dailyStats, err := s.userQuestionRepo.GetUserYearlyDailyStats(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	heatmapData := make([]HeatmapData, 0)
	currentDate := startDate

	checkInMap := make(map[string]int)
	for _, checkIn := range checkInStats {
		dateStr := checkIn.Date.Format("2006-01-02")
		checkInMap[dateStr] = int(checkIn.StudyCount)
	}

	statsMap := make(map[string]int64)
	for _, stat := range dailyStats {
		dateStr := stat.Date.Format("2006-01-02")
		statsMap[dateStr] = stat.StudyCount
	}

	now := time.Now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	for currentDate.Before(tomorrow) {
		dateStr := currentDate.Format("2006-01-02")
		count := checkInMap[dateStr]
		if count == 0 {
			count = int(statsMap[dateStr])
		}
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

	totalStudyDays := 0
	totalStudyCount := 0
	currentStreak := 0
	maxStreak := 0
	tempStreak := 0
	thisYearStudyCount := 0

	for _, data := range heatmapData {
		if data.Count > 0 {
			totalStudyDays++
			totalStudyCount += data.Count
		}
	}

	for i := len(heatmapData) - 1; i >= 0; i-- {
		if heatmapData[i].Count > 0 {
			currentStreak++
		} else {
			break
		}
	}

	for i := 0; i < len(heatmapData); i++ {
		if heatmapData[i].Count > 0 {
			tempStreak++
			if tempStreak > maxStreak {
				maxStreak = tempStreak
			}
		} else {
			tempStreak = 0
		}
	}

	currentYear := time.Now().Year()
	for _, data := range heatmapData {
		date, err := time.Parse("2006-01-02", data.Date)
		if err != nil {
			continue // 或者记录日志
		}
		if date.Year() == currentYear {
			thisYearStudyCount += data.Count
		}
	}

	return &HeatmapResponse{
		Data:          heatmapData,
		TotalCommits:  totalStudyCount,
		CurrentStreak: currentStreak,
		MaxStreak:     maxStreak,
		ThisYear:      thisYearStudyCount,
	}, nil
}

func (s *CheckInService) StartEventListener() {
	go func() {
		for event := range s.bus.UserCompletedQuestionChan {
			// 加 recover 防止 panic
			func() {
				err := s.CheckInToday(event.UserID)
				if err != nil {
					logger.Error("打卡失败", err)
				}
			}()
		}
	}()
}
