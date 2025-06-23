package services

import (
	"errors"
	"ggcode/internal/models"
	"ggcode/internal/repositories"
)

type StudyPlanService struct {
	studyPlanRepo     repositories.StudyPlanRepository
	ebbinghausService *EbbinghausService
}

// 类型定义使用ebbinghaus服务中的定义

func NewStudyPlanService(repos *repositories.Repositories, ebbinghausService *EbbinghausService) *StudyPlanService {
	return &StudyPlanService{
		studyPlanRepo:     repos.StudyPlan,
		ebbinghausService: ebbinghausService,
	}
}

// CreateStudyPlan 创建学习计划
func (s *StudyPlanService) CreateStudyPlan(userID, questionBankID uint, dailyCount int) (*models.UserStudyPlan, error) {
	// 检查用户是否已经为该题库创建了学习计划
	exists, err := s.studyPlanRepo.CheckStudyPlanExists(userID, questionBankID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("您已经为该题库创建了学习计划，一个题库只能创建一个学习计划")
	}

	// 创建学习计划
	studyPlan := &models.UserStudyPlan{
		UserID:         userID,
		QuestionBankID: questionBankID,
		DailyCount:     dailyCount,
	}

	createdPlan, err := s.studyPlanRepo.CreateStudyPlan(studyPlan)
	if err != nil {
		return nil, err
	}

	return createdPlan, nil
}

// GetStudyPlan 获取学习计划
func (s *StudyPlanService) GetStudyPlan(planID, userID uint) (*models.UserStudyPlan, error) {
	return s.studyPlanRepo.GetStudyPlan(planID, userID)
}

// UpdateStudyPlan 更新学习计划
func (s *StudyPlanService) UpdateStudyPlan(planID, userID uint, dailyCount int) error {
	return s.studyPlanRepo.UpdateStudyPlan(planID, userID, dailyCount)
}

// DeleteStudyPlan 删除学习计划
func (s *StudyPlanService) DeleteStudyPlan(planID, userID uint) error {
	return s.ebbinghausService.DeleteStudyPlanWithProgress(userID, planID)
}

// GetAllStudyPlans 获取所有学习计划
func (s *StudyPlanService) GetAllStudyPlans(userID uint, page, limit int) ([]models.UserStudyPlan, int64, error) {
	return s.studyPlanRepo.GetAllStudyPlans(userID, page, limit)
}

// GetStudyPlanProgress 获取学习计划进度
func (s *StudyPlanService) GetStudyPlanProgress(userID, planID uint) (*StudyPlanProgress, error) {
	return s.ebbinghausService.GetStudyPlanProgress(userID, planID)
}

// GetDailyQuestions 获取每日学习题目
func (s *StudyPlanService) GetDailyQuestions(userID, planID uint) ([]QuestionWithProgress, error) {
	return s.ebbinghausService.GetDailyQuestions(userID, planID)
}

// GetRandomMasteredQuestions 获取随机掌握题目
func (s *StudyPlanService) GetRandomMasteredQuestions(userID, planID uint, count int) ([]QuestionWithProgress, error) {
	// 获取指定的学习计划
	studyPlan, err := s.studyPlanRepo.GetStudyPlan(planID, userID)
	if err != nil {
		return nil, errors.New("学习计划不存在")
	}

	// 获取已掌握的题目，随机排序
	masteredQuestions, err := s.studyPlanRepo.GetRandomMasteredQuestions(userID, studyPlan.QuestionBankID, count)
	if err != nil {
		return nil, err
	}

	// 转换为QuestionWithProgress格式
	var questions []QuestionWithProgress
	for _, progress := range masteredQuestions {
		if progress.Question.QuestionBankID == studyPlan.QuestionBankID {
			questions = append(questions, QuestionWithProgress{
				Question: progress.Question,
				Progress: progress,
				IsReview: true, // 标记为复习题目
			})
		}
	}

	return questions, nil
}

// CompleteQuestion 完成题目
func (s *StudyPlanService) CompleteQuestion(userID, questionID uint, resultType string) error {
	// 验证结果类型
	if resultType != "ac" && resultType != "failed" {
		return errors.New("无效的结果类型")
	}

	return s.ebbinghausService.CompleteQuestion(userID, questionID, resultType)
}

// GetStudyStats 获取学习统计
func (s *StudyPlanService) GetStudyStats(userID uint) (*StudyStats, error) {
	return s.ebbinghausService.GetStudyStats(userID)
}
