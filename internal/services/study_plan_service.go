package services

import (
	"encoding/json"
	"errors"
	"ggcode/internal/models"
	"ggcode/internal/pkg/logger"
	"ggcode/internal/repositories"
	"math"
	"sort"
	"time"

	"gorm.io/gorm"
)

type StudyPlanServiceInterface interface {
	CreateStudyPlan(userID, questionBankID uint, dailyCount int) (*models.UserStudyPlan, error)
	GetStudyPlan(planID, userID uint) (*models.UserStudyPlan, error)
	UpdateStudyPlan(planID, userID uint, dailyCount int) error
	DeleteStudyPlan(planID, userID uint) error
	GetAllStudyPlans(userID uint, page, limit int) ([]models.UserStudyPlan, int64, error)
	GetStudyPlanProgress(userID, planID uint) (*models.StudyPlanProgress, error)
	GetDailyQuestions(userID, planID uint) (*models.DailyQuestionsResponse, error)
}
type StudyPlanService struct {
	studyPlanRepo    repositories.StudyPlanRepository
	userQuestionRepo repositories.UserQuestionRepository
	questionRepo     repositories.QuestionRepository
}

var _ StudyPlanServiceInterface = &StudyPlanService{}

// 类型定义使用ebbinghaus服务中的定义

func NewStudyPlanService(studyPlanRepo repositories.StudyPlanRepository, userQuestionRepo repositories.UserQuestionRepository, questionRepo repositories.QuestionRepository) *StudyPlanService {
	return &StudyPlanService{
		studyPlanRepo:    studyPlanRepo,
		userQuestionRepo: userQuestionRepo,
		questionRepo:     questionRepo,
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
	// 先拿到学习计划
	studyPlan, err := s.studyPlanRepo.GetStudyPlan(planID, userID)
	if err != nil {
		return err
	}

	// 确定实际的题库ID
	actualQuestionBankID := studyPlan.QuestionBankID
	questionBank := studyPlan.QuestionBank
	if questionBank.ForkedFrom != nil {
		// 使用新的 GetQuestionCount 函数
		localCount, err := s.questionRepo.GetQuestionCount(questionBank.ID)
		if err != nil {
			return err
		}
		if localCount == 0 {
			// 还没有发生写时复制，使用原题库ID
			actualQuestionBankID = *questionBank.ForkedFrom
		}
	}

	// 删除学习计划及其相关数据
	return s.studyPlanRepo.DeleteStudyPlanWithProgress(planID, userID, actualQuestionBankID)
}

// GetAllStudyPlans 获取所有学习计划
func (s *StudyPlanService) GetAllStudyPlans(userID uint, page, limit int) ([]models.UserStudyPlan, int64, error) {
	return s.studyPlanRepo.GetAllStudyPlans(userID, page, limit)
}

// GetStudyPlanProgress 获取学习计划进度
func (s *StudyPlanService) GetStudyPlanProgress(userID, planID uint) (*models.StudyPlanProgress, error) {
	// 获取学习计划
	studyPlan, err := s.studyPlanRepo.GetStudyPlan(planID, userID)
	if err != nil {
		return nil, errors.New("学习计划不存在")
	}

	var progress models.StudyPlanProgress
	progress.StudyPlanID = planID

	// 确定实际统计的题库ID
	actualQuestionBankID := studyPlan.QuestionBankID
	questionBank := studyPlan.QuestionBank
	if questionBank.ForkedFrom != nil {
		localCount, err := s.questionRepo.GetQuestionCount(questionBank.ID)
		if err != nil {
			return nil, err
		}
		if localCount == 0 {
			// 还没有发生写时复制，使用原题库ID进行统计
			actualQuestionBankID = *questionBank.ForkedFrom
		}
	}

	// 获取题库总题目数
	totalQuestions, err := s.questionRepo.GetQuestionCount(actualQuestionBankID)
	if err != nil {
		return nil, err
	}
	progress.TotalQuestions = totalQuestions

	// 获取已学习题目数（有学习记录的）
	studiedCount, err := s.userQuestionRepo.GetStudiedQuestionCount(userID, actualQuestionBankID)
	if err != nil {
		return nil, err
	}
	progress.StudiedCount = studiedCount

	// 获取已掌握题目数
	completedCount, err := s.userQuestionRepo.GetCompletedQuestionCount(userID, actualQuestionBankID)
	if err != nil {
		return nil, err
	}
	progress.CompletedCount = completedCount

	// 获取待复习题目数（未完成且到了复习时间）
	reviewCount, err := s.userQuestionRepo.GetReviewQuestionCount(userID, actualQuestionBankID)
	if err != nil {
		return nil, err
	}
	progress.ReviewCount = reviewCount

	// 计算进度百分比
	if progress.TotalQuestions > 0 {
		progress.ProgressRate = int((progress.StudiedCount * 100) / progress.TotalQuestions)
		progress.MasteryRate = int((progress.CompletedCount * 100) / progress.TotalQuestions)
	}

	return &progress, nil
}

// GetDailyQuestions 获取每日学习题目
func (s *StudyPlanService) GetDailyQuestions(userID, planID uint) (*models.DailyQuestionsResponse, error) {
	// 获取指定的学习计划
	studyPlan, err := s.studyPlanRepo.GetStudyPlan(planID, userID)
	if err != nil {
		return nil, err
	}

	questionBank := studyPlan.QuestionBank
	// 使用本地时区获取今天的开始时间
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 确定实际使用的题库ID
	actualQuestionBankID := questionBank.ID
	if questionBank.ForkedFrom != nil {
		localCount, err := s.questionRepo.GetQuestionCount(questionBank.ID)
		if err != nil {
			return nil, err
		}
		if localCount == 0 {
			actualQuestionBankID = *questionBank.ForkedFrom
		}
	}

	cache, err := s.studyPlanRepo.GetDailyCache(userID, planID, today)

	var questionIDs []uint

	if err == gorm.ErrRecordNotFound {
		// 今天第一次学习，生成新的学习计划并缓存
		questions, err := s.GenerateDailyQuestions(userID, actualQuestionBankID, studyPlan.DailyCount)
		if err != nil {
			return nil, err
		}

		// 提取题目ID
		for _, q := range questions {
			questionIDs = append(questionIDs, q.Question.ID)
		}

		// 缓存题目ID列表
		if err := s.studyPlanRepo.CacheDailyQuestions(userID, planID, questionIDs, today); err != nil {
			// 缓存失败不影响学习，记录日志即可
			// 可以在这里添加日志记录
			logger.Error("缓存每日学习题目失败", err)
		}
	} else if err != nil {
		return nil, err
	} else {
		// 从缓存中获取题目ID列表
		if err := json.Unmarshal([]byte(cache.QuestionIDs), &questionIDs); err != nil {
			return nil, errors.New("解析缓存数据失败")
		}
	}

	// 根据题目ID获取完整的题目信息
	questions, err := s.userQuestionRepo.GetQuestionsByIDs(questionIDs, actualQuestionBankID)
	if err != nil {
		return nil, err
	}

	// 计算 start 位置（今天已学习的题目数量）
	start, err := s.userQuestionRepo.CalculateStartPosition(userID, questionIDs, today)
	if err != nil {
		return nil, err
	}

	return &models.DailyQuestionsResponse{
		Questions: questions,
		Start:     start,
		Total:     len(questions),
	}, nil
}

// GenerateDailyQuestions 生成每日学习题目（艾宾浩斯算法）
func (s *StudyPlanService) GenerateDailyQuestions(userID, questionBankID uint, dailyCount int) ([]models.QuestionWithProgress, error) {
	// 使用本地时区获取今天的开始时间
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 1. 获取需要复习的题目进度
	reviewProgresses, err := s.userQuestionRepo.GetReviewQuestionProgresses(userID, questionBankID, today)
	if err != nil {
		return nil, err
	}

	// 2. 将复习进度转换为 QuestionWithProgress
	var reviewQuestions []models.QuestionWithProgress
	for _, progress := range reviewProgresses {
		// 获取题目信息
		question, err := s.questionRepo.GetQuestion(progress.QuestionID)
		if err != nil {
			continue // 跳过找不到的题目
		}
		reviewQuestions = append(reviewQuestions, models.QuestionWithProgress{
			Question: *question,
			Progress: progress,
			IsReview: true,
		})
	}

	// 3. 如果复习题目不足，添加新题目
	currentCount := len(reviewQuestions)
	if currentCount < dailyCount {
		newQuestions, err := s.userQuestionRepo.GetNewQuestions(userID, questionBankID, dailyCount-currentCount, nil)
		if err != nil {
			return nil, err
		}
		for _, question := range newQuestions {
			reviewQuestions = append(reviewQuestions, models.QuestionWithProgress{
				Question: question,
				Progress: models.UserQuestionProgress{}, // 新题目没有进度
				IsReview: false,
			})
		}
	}

	// 4. 对所有题目进行打分
	scoreQuestions(reviewQuestions)

	// 5. 按照得分从高到低排序
	sort.Slice(reviewQuestions, func(i, j int) bool {
		return reviewQuestions[i].Score > reviewQuestions[j].Score
	})

	// 6. 确保不超过每日学习量
	if len(reviewQuestions) > dailyCount {
		reviewQuestions = reviewQuestions[:dailyCount]
	}

	return reviewQuestions, nil
}

// scoreQuestions 对题目进行打分
func scoreQuestions(questions []models.QuestionWithProgress) {
	now := time.Now()

	// 1. 计算最大分数（只考虑没有 Difficulty 的题目）
	maxScore := 0.0
	for _, q := range questions {
		if q.Question.Difficulty == "" && q.Question.Score > maxScore {
			maxScore = q.Question.Score
		}
	}
	if maxScore == 0 {
		maxScore = 100 // 防止除零，默认100
	}

	for i := range questions {
		question := &questions[i]
		timeScore := calculateTimeScore(question.Progress, now)
		difficultyScore := calculateDifficultyScore(question.Question.Difficulty, question.Question.Score, maxScore)
		finalScore := 0.5*timeScore + 0.5*difficultyScore
		question.Score = int(finalScore * 100)
	}
}

// calculateTimeScore 计算时间维度得分
func calculateTimeScore(progress models.UserQuestionProgress, now time.Time) float64 {
	if progress.LastReviewDate.IsZero() {
		return 0.8 // 新题目给较高基础分
	}
	daysSinceLastReview := int(now.Sub(progress.LastReviewDate).Hours() / 24)
	if daysSinceLastReview <= 0 {
		return 0.1
	}
	if daysSinceLastReview >= 14 {
		return 1.0
	}
	normalized := math.Log(1+float64(daysSinceLastReview)) / math.Log(1+14)
	return 0.2 + 0.8*normalized
}

// calculateDifficultyScore 支持分数归一化
func calculateDifficultyScore(difficulty string, score, maxScore float64) float64 {
	switch difficulty {
	case "Easy":
		return 0.3
	case "Medium":
		return 0.6
	case "Hard":
		return 1.0
	}
	// 没有难度，用分数归一化
	if score > 0 && maxScore > 0 {
		normalized := score / maxScore
		if normalized > 1.0 {
			normalized = 1.0
		}
		return normalized
	}
	return 0.5
}
