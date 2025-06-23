package services

import (
	"ggcode/internal/repositories"

	"gorm.io/gorm"
)

// Services 包含所有业务服务
type Services struct {
	User          *UserService
	QuestionBank  *QuestionBankService
	Question      *QuestionService
	StudyPlan     *StudyPlanService
	Share         *ShareService
	Progress      *ProgressService
	Ebbinghaus    *EbbinghausService
	Interview     InterviewService
	DockerJudge   *DockerJudgeService
	ContainerPool *SimpleContainerPool
	// CheckIn      *CheckInService
}

// NewServices 创建所有服务实例
func NewServices(repos *repositories.Repositories, db *gorm.DB) *Services {
	// 首先创建EbbinghausService
	ebbinghausService := NewEbbinghausService(db)

	// 创建Hydro评测服务

	// 创建容器池（这里会自动启动！）
	containerPool := NewSimpleContainerPool()

	// 创建Docker评测服务，并传入容器池
	dockerJudgeService := NewDockerJudgeServiceWithPool(containerPool)

	return &Services{
		User:          NewUserService(repos),
		QuestionBank:  NewQuestionBankService(repos),
		Question:      NewQuestionService(repos),
		StudyPlan:     NewStudyPlanService(repos, ebbinghausService),
		Share:         NewShareService(repos),
		Progress:      NewProgressService(repos, ebbinghausService),
		Ebbinghaus:    ebbinghausService,
		Interview:     NewInterviewService(repos.Interview),
		DockerJudge:   dockerJudgeService,
		ContainerPool: containerPool,
	}
}
