package services

import (
	"ggcode/internal/config"
	"ggcode/internal/events"
	"ggcode/internal/repositories"

	"gorm.io/gorm"
)

// Services 包含所有业务服务
type Services struct {
	User         *UserService
	QuestionBank *QuestionBankService
	Question     *QuestionService
	StudyPlan    *StudyPlanService
	UserQuestion *UserQuestionService
	Share        *ShareService
	Ebbinghaus   *EbbinghausService
	CheckIn      *CheckInService
	Interview    InterviewService
	GoJudge      *GoJudgeService
}

// NewServices 创建所有服务实例
func NewServices(repos *repositories.Repositories, db *gorm.DB, cfg *config.Config) *Services {
	ebbinghausService := NewEbbinghausService(db)
	bus := events.NewEventBus()
	return &Services{
		User:         NewUserService(repos, cfg),
		QuestionBank: NewQuestionBankService(repos),
		Question:     NewQuestionService(repos),
		StudyPlan:    NewStudyPlanService(repos.StudyPlan, repos.UserQuestion, repos.Question),
		UserQuestion: NewUserQuestionService(repos.UserQuestion, repos.UserStats, bus),
		Share:        NewShareService(repos),
		Ebbinghaus:   ebbinghausService,
		CheckIn:      NewCheckInService(repos.CheckIn, repos.UserQuestion, bus),
		Interview:    NewInterviewService(repos.Interview),
		GoJudge:      NewGoJudgeService(""),
	}
}
