package services

import (
	"ggcode/internal/models"
	"ggcode/internal/repositories"
	"time"

	"gorm.io/gorm"
)

type UserQuestionServiceInterface interface {
	CompleteQuestion(userID, questionID uint, resultType string) error
	GetStudyStats(userID uint) (*StudyStats, error)
}

type UserQuestionService struct {
	userQuestionRepo repositories.UserQuestionRepository
	userStatsRepo    repositories.UserStatsRepository
	checkInService   *CheckInService // 新增
}

func NewUserQuestionService(userQuestionRepo repositories.UserQuestionRepository, userStatsRepo repositories.UserStatsRepository, checkInService *CheckInService) *UserQuestionService {
	return &UserQuestionService{
		userQuestionRepo: userQuestionRepo,
		userStatsRepo:    userStatsRepo,
		checkInService:   checkInService,
	}
}

// CompleteQuestion 完成题目学习
func (s *UserQuestionService) CompleteQuestion(userID, questionID uint, resultType string) error {
	var progress models.UserQuestionProgress
	err := s.userQuestionRepo.GetUserQuestionProgress(userID, questionID, &progress)

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		progress = models.UserQuestionProgress{
			UserID:         userID,
			QuestionID:     questionID,
			ReviewLevel:    0,
			LastReviewDate: now,
			IsCompleted:    false,
		}

		if resultType == "failed" {
			progress.NextReviewDate = now.AddDate(0, 0, reviewIntervals[0])
		} else {
			progress.IsCompleted = true
		}

		if err := s.userQuestionRepo.CreateUserQuestionProgress(&progress); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		progress.LastReviewDate = now

		if resultType == "failed" {
			progress.ReviewLevel--
			if progress.ReviewLevel < 0 {
				progress.ReviewLevel = 0
			}
			progress.NextReviewDate = now.AddDate(0, 0, reviewIntervals[progress.ReviewLevel])
			progress.IsCompleted = false
		} else {
			progress.ReviewLevel++
			if progress.ReviewLevel >= len(reviewIntervals) {
				progress.IsCompleted = true
			} else {
				progress.NextReviewDate = now.AddDate(0, 0, reviewIntervals[progress.ReviewLevel])
			}
		}

		if err := s.userQuestionRepo.UpdateUserQuestionProgress(&progress); err != nil {
			return err
		}
	}

	// 自动打卡
	if s.checkInService != nil {
		_ = s.checkInService.CheckInToday(userID)
	}

	return nil
}

// GetStudyStats 获取学习统计信息
func (s *UserQuestionService) GetStudyStats(userID uint) (*StudyStats, error) {
	var stats StudyStats
	s.userStatsRepo.CountUserStudiedQuestions(userID, &stats.TotalStudied)
	s.userStatsRepo.CountUserCompletedQuestions(userID, &stats.Completed)
	// 使用本地时区获取今天的开始时间
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	s.userStatsRepo.CountUserTodayReviewQuestions(userID, today, &stats.TodayReview)
	return &stats, nil
}
