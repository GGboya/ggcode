package services

import (
	"ggcode/internal/repositories"
)

type ProgressService struct {
	ebbinghausService *EbbinghausService
}

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

func NewProgressService(repos *repositories.Repositories, ebbinghausService *EbbinghausService) *ProgressService {
	return &ProgressService{
		ebbinghausService: ebbinghausService,
	}
}

// GetQuestionBankProgress 获取特定题库的学习进度
func (s *ProgressService) GetQuestionBankProgress(userID, bankID uint) (*QuestionBankProgress, error) {
	return s.ebbinghausService.GetQuestionBankProgress(userID, bankID)
}

// GetAllQuestionBanksProgress 获取所有题库的学习进度
func (s *ProgressService) GetAllQuestionBanksProgress(userID uint) ([]QuestionBankProgress, error) {
	return s.ebbinghausService.GetAllQuestionBanksProgress(userID)
}

// CheckInToday 今日打卡
func (s *ProgressService) CheckInToday(userID uint) error {
	return s.ebbinghausService.CheckInToday(userID)
}

// GetCheckInStats 获取打卡统计
func (s *ProgressService) GetCheckInStats(userID uint) (*CheckInStats, error) {
	return s.ebbinghausService.GetCheckInStats(userID)
}

// GetStudyHeatmap 获取学习活动热力图数据
func (s *ProgressService) GetStudyHeatmap(userID uint) (*HeatmapResponse, error) {
	return s.ebbinghausService.GetStudyHeatmap(userID)
}
