package services

import (
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
	Progress     *ProgressService
	Ebbinghaus   *EbbinghausService
	CheckIn      *CheckInService
	Interview    InterviewService
	GoJudge      *GoJudgeService
}

// NewServices 创建所有服务实例
func NewServices(repos *repositories.Repositories, db *gorm.DB) *Services {
	ebbinghausService := NewEbbinghausService(db)
	checkInService := NewCheckInService(repos.CheckIn, db)
	return &Services{
		User:         NewUserService(repos),
		QuestionBank: NewQuestionBankService(repos),
		Question:     NewQuestionService(repos),
		StudyPlan:    NewStudyPlanService(repos.StudyPlan, repos.UserQuestion, repos.Question),
		UserQuestion: NewUserQuestionService(repos.UserQuestion, repos.UserStats, checkInService),
		Share:        NewShareService(repos),
		Progress:     NewProgressService(repos, ebbinghausService, checkInService),
		Ebbinghaus:   ebbinghausService,
		CheckIn:      checkInService,
		Interview:    NewInterviewService(repos.Interview),
		GoJudge:      NewGoJudgeService(""),
	}
}
