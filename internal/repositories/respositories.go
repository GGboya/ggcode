package repositories

import "gorm.io/gorm"

// Repositories 包含所有仓库接口
type Repositories struct {
	User         UserRepository
	QuestionBank QuestionBankRepository
	Question     QuestionRepository
	StudyPlan    StudyPlanRepository
	Share        ShareRepository
	Interview    InterviewRepository
	// Progress     ProgressRepository
	// CheckIn      CheckInRepository
	// Star         StarRepository
}

// NewRepositories 创建所有仓库实例
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:         NewUserRepository(db),
		QuestionBank: NewQuestionBankRepository(db),
		Question:     NewQuestionRepository(db),
		StudyPlan:    NewStudyPlanRepository(db),
		Share:        NewShareRepository(db),
		Interview:    NewInterviewRepository(db),
		// Progress:     NewProgressRepository(db),
		// CheckIn:      NewCheckInRepository(db),
		// Star:         NewStarRepository(db),
	}
}
