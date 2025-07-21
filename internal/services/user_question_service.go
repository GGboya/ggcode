package services

import (
	"ggcode/internal/events"
	"ggcode/internal/models"
	"ggcode/internal/repositories"
	"time"

	"gorm.io/gorm"
)

var reviewIntervals = []int{1, 2, 4, 7, 15, 30, 60}

type UserQuestionServiceInterface interface {
	CompleteQuestion(userID, questionID uint, resultType string) error
	GetStudyStats(userID uint) (*models.StudyStats, error)
}

type UserQuestionService struct {
	userQuestionRepo repositories.UserQuestionRepository
	userStatsRepo    repositories.UserStatsRepository
	bus              *events.EventBus
}

func NewUserQuestionService(userQuestionRepo repositories.UserQuestionRepository, userStatsRepo repositories.UserStatsRepository, bus *events.EventBus) *UserQuestionService {
	return &UserQuestionService{
		userQuestionRepo: userQuestionRepo,
		userStatsRepo:    userStatsRepo,
		bus:              bus,
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

	// 通过 events 包的 channel 发布事件
	s.bus.UserCompletedQuestionChan <- events.UserCompletedQuestionEvent{
		UserID:     userID,
		QuestionID: questionID,
	}

	return nil
}

// GetStudyStats 获取学习统计信息
func (s *UserQuestionService) GetStudyStats(userID uint) (*models.StudyStats, error) {
	var stats models.StudyStats
	if err := s.userStatsRepo.CountUserStudiedQuestions(userID, &stats.TotalStudied); err != nil {
		return nil, err
	}
	if err := s.userStatsRepo.CountUserCompletedQuestions(userID, &stats.Completed); err != nil {
		return nil, err
	}
	// 使用本地时区获取今天的开始时间
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := s.userStatsRepo.CountUserTodayReviewQuestions(userID, today, &stats.TodayReview); err != nil {
		return nil, err
	}
	return &stats, nil
}
