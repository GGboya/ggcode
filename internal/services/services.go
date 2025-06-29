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
	Share        *ShareService
	Progress     *ProgressService
	Ebbinghaus   *EbbinghausService
	Interview    InterviewService
	GoJudge      *GoJudgeService
	// CheckIn      *CheckInService
}

// NewServices 创建所有服务实例
func NewServices(repos *repositories.Repositories, db *gorm.DB) *Services {
	// 首先创建EbbinghausService
	ebbinghausService := NewEbbinghausService(db)

	// 创建 go-judge 服务
	goJudgeService := NewGoJudgeService("")

	return &Services{
		User:         NewUserService(repos),
		QuestionBank: NewQuestionBankService(repos),
		Question:     NewQuestionService(repos),
		StudyPlan:    NewStudyPlanService(repos, ebbinghausService),
		Share:        NewShareService(repos),
		Progress:     NewProgressService(repos, ebbinghausService),
		Ebbinghaus:   ebbinghausService,
		Interview:    NewInterviewService(repos.Interview),
		GoJudge:      goJudgeService,
	}
}
