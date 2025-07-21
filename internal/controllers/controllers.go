package controllers

import "ggcode/internal/services"

// Controllers 包含所有控制器
type Controllers struct {
	User         *UserController
	QuestionBank *QuestionBankController
	Question     *QuestionController
	StudyPlan    *StudyPlanController
	Share        *ShareController
	Page         *PageController
	Interview    *InterviewController
	GoJudge      *GoJudgeController
	UserQuestion *UserQuestionController
	CheckIn      *CheckInController
}

// NewControllers 创建所有控制器实例
func NewControllers(services *services.Services) *Controllers {
	return &Controllers{
		User:         NewUserController(services.User),
		QuestionBank: NewQuestionBankController(services.QuestionBank),
		Question:     NewQuestionController(services.Question),
		StudyPlan:    NewStudyPlanController(services.StudyPlan),
		Share:        NewShareController(services.Share),
		Page:         NewPageController(),
		Interview:    NewInterviewController(services.Interview, services.User),
		GoJudge:      NewGoJudgeController(services.GoJudge, services.Interview),
		UserQuestion: NewUserQuestionController(services.UserQuestion),
		CheckIn:      NewCheckInController(services.CheckIn),
	}
}
