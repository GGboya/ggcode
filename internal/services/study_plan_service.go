package services

import (
	"encoding/json"
	"errors"
	"ggcode/internal/models"
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
}
type StudyPlanService struct {
	studyPlanRepo repositories.StudyPlanRepository
}

// 类型定义使用ebbinghaus服务中的定义

func NewStudyPlanService(repos *repositories.Repositories) *StudyPlanService {
	return &StudyPlanService{
		studyPlanRepo: repos.StudyPlan,
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
		localCount, err := s.studyPlanRepo.GetQuestionCount(questionBank.ID)
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
func (s *StudyPlanService) GetStudyPlanProgress(userID, planID uint) (*StudyPlanProgress, error) {
	// 获取学习计划
	studyPlan, err := s.studyPlanRepo.GetStudyPlan(planID, userID)
	if err != nil {
		return nil, errors.New("学习计划不存在")
	}

	var progress StudyPlanProgress
	progress.StudyPlanID = planID

	// 确定实际统计的题库ID
	actualQuestionBankID := studyPlan.QuestionBankID
	questionBank := studyPlan.QuestionBank
	if questionBank.ForkedFrom != nil {
		localCount, err := s.studyPlanRepo.GetQuestionCount(questionBank.ID)
		if err != nil {
			return nil, err
		}
		if localCount == 0 {
			// 还没有发生写时复制，使用原题库ID进行统计
			actualQuestionBankID = *questionBank.ForkedFrom
		}
	}

	// 获取题库总题目数
	totalQuestions, err := s.studyPlanRepo.GetQuestionCount(actualQuestionBankID)
	if err != nil {
		return nil, err
	}
	progress.TotalQuestions = totalQuestions

	// 获取已学习题目数（有学习记录的）
	studiedCount, err := s.studyPlanRepo.GetStudiedQuestionCount(userID, actualQuestionBankID)
	if err != nil {
		return nil, err
	}
	progress.StudiedCount = studiedCount

	// 获取已掌握题目数
	completedCount, err := s.studyPlanRepo.GetCompletedQuestionCount(userID, actualQuestionBankID)
	if err != nil {
		return nil, err
	}
	progress.CompletedCount = completedCount

	// 获取待复习题目数（未完成且到了复习时间）
	reviewCount, err := s.studyPlanRepo.GetReviewQuestionCount(userID, actualQuestionBankID)
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
	// 使用服务器本地时区（北京时间）
	today := time.Now().Truncate(24 * time.Hour)

	// 确定实际使用的题库ID
	actualQuestionBankID := questionBank.ID
	if questionBank.ForkedFrom != nil {
		localCount, err := s.studyPlanRepo.GetQuestionCount(questionBank.ID)
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
	questions, err := s.studyPlanRepo.GetQuestionsByIDs(questionIDs, actualQuestionBankID)
	if err != nil {
		return nil, err
	}

	// 计算 start 位置（今天已学习的题目数量）
	start, err := s.studyPlanRepo.CalculateStartPosition(userID, questionIDs, today)
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
	today := time.Now().Truncate(24 * time.Hour)

	// 1. 获取需要复习的题目
	reviewQuestions, err := s.getReviewQuestionProgresses(userID, questionBankID, today)
	if err != nil {
		return nil, err
	}

	// 2. 如果复习题目不足，添加新题目
	currentCount := len(reviewQuestions)
	if currentCount < dailyCount {
		newQuestions, err := s.getNewQuestions(userID, questionBankID, dailyCount-currentCount)
		if err != nil {
			return nil, err
		}
		reviewQuestions = append(reviewQuestions, newQuestions...)
	}

	// 3. 对所有题目进行打分
	s.scoreQuestions(reviewQuestions)

	// 4. 按照得分从高到低排序
	sort.Slice(reviewQuestions, func(i, j int) bool {
		return reviewQuestions[i].Score > reviewQuestions[j].Score
	})

	// 5. 确保不超过每日学习量
	if len(reviewQuestions) > dailyCount {
		reviewQuestions = reviewQuestions[:dailyCount]
	}

	return reviewQuestions, nil
}

// getReviewQuestionProgresses 获取今日需要复习的题目进度
func (s *StudyPlanService) getReviewQuestionProgresses(userID, questionBankID uint, today time.Time) ([]models.QuestionWithProgress, error) {
	progresses, err := s.studyPlanRepo.GetReviewQuestionProgresses(userID, questionBankID, today)
	if err != nil {
		return nil, err
	}
	var questionIDs []uint
	for _, p := range progresses {
		questionIDs = append(questionIDs, p.QuestionID)
	}
	return s.studyPlanRepo.GetQuestionsByIDs(questionIDs, questionBankID)
}

// getNewQuestions 获取未学过的新题目
func (s *StudyPlanService) getNewQuestions(userID, questionBankID uint, count int) ([]models.QuestionWithProgress, error) {
	questions, err := s.studyPlanRepo.GetNewQuestions(userID, questionBankID, count, nil)
	if err != nil {
		return nil, err
	}

	var result []models.QuestionWithProgress
	for _, question := range questions {
		result = append(result, models.QuestionWithProgress{
			Question: question,
			Progress: models.UserQuestionProgress{}, // 新题目没有进度
			IsReview: false,
		})
	}
	return result, nil
}

// scoreQuestions 对题目进行打分
func (s *StudyPlanService) scoreQuestions(questions []models.QuestionWithProgress) {
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

// CompleteQuestion 完成题目学习
// 对于fork题库，直接使用原题库的题目ID记录学习进度，避免不必要的数据复制
func (s *StudyPlanService) CompleteQuestion(userID, questionID uint, resultType string) error {
	// 查找或创建学习进度记录
	// 注意：这里直接使用传入的questionID，不需要进行写时复制
	var progress models.UserQuestionProgress
	err := s.studyPlanRepo.GetUserQuestionProgress(userID, questionID, &progress)

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		// 新题目，创建进度记录
		progress = models.UserQuestionProgress{
			UserID:         userID,
			QuestionID:     questionID,
			ReviewLevel:    0,
			LastReviewDate: now,
			IsCompleted:    false,
		}

		// 根据结果类型设置复习计划
		if resultType == "failed" {
			// 不会做：重新开始学习流程，使用完整的复习间隔
			progress.NextReviewDate = now.AddDate(0, 0, reviewIntervals[0]) // 1天后复习
		} else {
			// 独立AC：第一次就 AC，表示用户已经掌握了该题目，直接设置为已掌握
			progress.IsCompleted = true
		}

		if err := s.studyPlanRepo.CreateUserQuestionProgress(&progress); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// 更新现有进度
		progress.LastReviewDate = now

		if resultType == "failed" {
			// 不会做：缩短复习间隔
			progress.ReviewLevel--
			if progress.ReviewLevel < 0 {
				progress.ReviewLevel = 0
			}
			progress.NextReviewDate = now.AddDate(0, 0, reviewIntervals[progress.ReviewLevel])
			progress.IsCompleted = false
		} else {
			// 独立AC：正常推进复习
			progress.ReviewLevel++

			// 检查是否完成所有复习层级
			if progress.ReviewLevel >= len(reviewIntervals) {
				progress.IsCompleted = true
			} else {
				progress.NextReviewDate = now.AddDate(0, 0, reviewIntervals[progress.ReviewLevel])
			}
		}

		if err := s.studyPlanRepo.UpdateUserQuestionProgress(&progress); err != nil {
			return err
		}
	}

	// 完成学习后自动打卡
	if err := s.CheckInToday(userID); err != nil {
		// 打卡失败不影响学习记录，但记录日志
		// 常见情况：今日已打卡
		// 这里可以添加日志记录
		_ = err // 忽略打卡错误，不影响学习进度保存
	}

	return nil
}

// CheckInToday 用户今日打卡（支持连续天数和最大连续天数统计）
func (s *StudyPlanService) CheckInToday(userID uint) error {
	today := time.Now().Truncate(24 * time.Hour)

	// 检查今日是否已打卡
	var existingCheckIn models.UserCheckIn
	err := s.studyPlanRepo.GetUserCheckInByDate(userID, today, &existingCheckIn)
	if err == nil {
		return errors.New("今日已打卡")
	}

	// 获取昨天的打卡记录来计算连续天数和最长连续天数
	yesterday := today.AddDate(0, 0, -1)
	var yesterdayCheckIn models.UserCheckIn
	consecutiveDays := 1 // 默认为1（今天是第一天）
	bestStreak := 1      // 默认为1

	err = s.studyPlanRepo.GetUserCheckInByDate(userID, yesterday, &yesterdayCheckIn)
	if err == nil {
		// 昨天有打卡，连续天数 = 昨天的连续天数 + 1
		consecutiveDays = yesterdayCheckIn.ConsecutiveDays + 1
		// 最长连续天数 = max(当前连续天数, 昨天的最长连续天数)
		bestStreak = max(consecutiveDays, yesterdayCheckIn.BestStreak)
	} else {
		// 昨天没打卡，获取最近的一条记录来获取历史最长连续天数
		var latestCheckIn models.UserCheckIn
		err = s.studyPlanRepo.GetLatestUserCheckIn(userID, &latestCheckIn)
		if err == nil {
			// 有历史记录，使用历史最长连续天数
			bestStreak = max(1, latestCheckIn.BestStreak)
		}
		// 如果没有历史记录，bestStreak 保持默认值 1
	}

	// 创建打卡记录
	checkIn := models.UserCheckIn{
		UserID:          userID,
		CheckDate:       today,
		ConsecutiveDays: consecutiveDays,
		BestStreak:      bestStreak,
	}

	return s.studyPlanRepo.CreateUserCheckIn(&checkIn)
}

// GetStudyStats 获取学习统计信息
func (s *StudyPlanService) GetStudyStats(userID uint) (*StudyStats, error) {
	var stats StudyStats

	// 总学习题目数
	s.studyPlanRepo.CountUserStudiedQuestions(userID, &stats.TotalStudied)

	// 已完成题目数
	s.studyPlanRepo.CountUserCompletedQuestions(userID, &stats.Completed)

	// 今日需复习题目数
	today := time.Now().Truncate(24 * time.Hour)
	s.studyPlanRepo.CountUserTodayReviewQuestions(userID, today, &stats.TodayReview)

	return &stats, nil
}
