package services

import (
	"ggcode/internal/repositories"
)

// Services 包含所有业务服务
type Services struct {
	User         *UserService
	QuestionBank *QuestionBankService
	Question     *QuestionService
	StudyPlan    *StudyPlanService
	Ebbinghaus   *EbbinghausService
	Progress     *ProgressService
	CheckIn      *CheckInService
}

// NewServices 创建所有服务实例
func NewServices(repos *repositories.Repositories) *Services {
	return &Services{
		User:         NewUserService(repos),
		QuestionBank: NewQuestionBankService(repos),
		Question:     NewQuestionService(repos),
		StudyPlan:    NewStudyPlanService(repos),
		Ebbinghaus:   NewEbbinghausService(repos),
		Progress:     NewProgressService(repos),
		CheckIn:      NewCheckInService(repos),
	}
}
