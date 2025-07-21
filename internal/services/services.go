package services

import (
	"ggcode/internal/config"
	"ggcode/internal/events"
	"ggcode/internal/repositories"

	"gorm.io/gorm"
)

// Services 包含所有业务服务
type Services struct {
	User         UserServiceInterface
	QuestionBank QuestionBankServiceInterface
	Question     QuestionServiceInterface
	StudyPlan    StudyPlanServiceInterface
	UserQuestion UserQuestionServiceInterface
	Share        ShareServiceInterface
	CheckIn      CheckInServiceInterface
	Interview    InterviewService
	GoJudge      *GoJudgeService
}

// NewServices 创建所有服务实例
func NewServices(repos *repositories.Repositories, db *gorm.DB, cfg *config.Config) *Services {
	bus := events.NewEventBus()
	return &Services{
		User:         NewUserService(repos.User, cfg),
		QuestionBank: NewQuestionBankService(repos.QuestionBank, repos.ContestProblem, repos.Question),
		Question:     NewQuestionService(repos.Question),
		StudyPlan:    NewStudyPlanService(repos.StudyPlan, repos.UserQuestion, repos.Question),
		UserQuestion: NewUserQuestionService(repos.UserQuestion, repos.UserStats, bus),
		Share:        NewShareService(repos.Share),
		CheckIn:      NewCheckInService(repos.CheckIn, repos.UserQuestion, bus),
		Interview:    NewInterviewService(repos.Interview),
		GoJudge:      NewGoJudgeService(""),
	}
}
