package services

import (
	"ggcode/internal/repositories"
)

type ProgressService struct {
	ebbinghausService *EbbinghausService
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
