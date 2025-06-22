package controllers

import "ggcode/internal/services"

// Controllers 包含所有控制器
type Controllers struct {
	User         *UserController
	QuestionBank *QuestionBankController
	Question     *QuestionController
	// StudyPlan    *StudyPlanController
	// Study        *StudyController
	// Progress     *ProgressController
	// CheckIn      *CheckInController
}

// NewControllers 创建所有控制器实例
func NewControllers(services *services.Services) *Controllers {
	return &Controllers{
		User:         NewUserController(services),
		QuestionBank: NewQuestionBankController(services),
		Question:     NewQuestionController(services),
		// StudyPlan:    NewStudyPlanController(services),
		// Study:        NewStudyController(services),
		// Progress:     NewProgressController(services),
		// CheckIn:      NewCheckInController(services),
	}
}
