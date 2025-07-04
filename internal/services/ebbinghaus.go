package services

import (
	"encoding/json"
	"errors"
	"ggcode/internal/models"
	"time"

	"gorm.io/gorm"
)

/*
Fork 题库的优化设计：

1. 懒加载策略：
   - Fork 题库时不立即复制题目，只创建题库关联关系
   - 直接复用原题库的题目进行学习，避免不必要的数据复制

2. 学习进度记录：
   - 用户学习时，直接使用原题库的题目ID记录学习进度
   - 由于学习进度表有 UserID 字段，不同用户的进度天然分离

3. 写时复制触发：
   - 只有在用户真正需要修改题库时（添加、删除、编辑题目）才进行写时复制
   - 写时复制时会同步更新所有相关的学习进度记录，确保数据一致性

4. 统计兼容性：
   - 所有统计函数都能智能识别 fork 题库的状态
   - 根据是否发生写时复制来选择正确的题库ID进行统计

这种设计既避免了数据冗余，又保证了功能的完整性和数据的一致性。
*/

type EbbinghausService struct {
	db *gorm.DB
}

func NewEbbinghausService(db *gorm.DB) *EbbinghausService {
	return &EbbinghausService{db: db}
}

// 艾宾浩斯遗忘曲线复习间隔（天数）
var reviewIntervals = []int{1, 2, 4, 7, 15, 30, 60}

// DailyQuestionsResponse 每日学习题目响应
type DailyQuestionsResponse struct {
	Questions []QuestionWithProgress `json:"questions"` // 今天的学习题目列表
	Start     int                    `json:"start"`     // 从第几道题开始展示（0-based）
	Total     int                    `json:"total"`     // 总题目数量
}

// GetDailyQuestions 获取指定学习计划的当天需要学习的题目（支持断点续学）
func (s *EbbinghausService) GetDailyQuestions(userID, studyPlanID uint) (*DailyQuestionsResponse, error) {
	// 获取指定的学习计划
	var studyPlan models.UserStudyPlan
	if err := s.db.Where("id = ? AND user_id = ?", studyPlanID, userID).
		Preload("QuestionBank").First(&studyPlan).Error; err != nil {
		return nil, errors.New("学习计划不存在")
	}

	questionBank := studyPlan.QuestionBank
	today := time.Now().Truncate(24 * time.Hour)

	// 确定实际使用的题库ID
	actualQuestionBankID := questionBank.ID
	if questionBank.ForkedFrom != nil {
		var localCount int64
		s.db.Model(&models.Question{}).Where("question_bank_id = ?", questionBank.ID).Count(&localCount)
		if localCount == 0 {
			actualQuestionBankID = *questionBank.ForkedFrom
		}
	}

	// 检查今天是否已有缓存的学习计划
	var cache models.DailyStudyPlanCache
	err := s.db.Where("user_id = ? AND study_plan_id = ? AND DATE(cache_date) = DATE(?)",
		userID, studyPlanID, today).First(&cache).Error

	var questionIDs []uint

	if err == gorm.ErrRecordNotFound {
		// 今天第一次学习，生成新的学习计划并缓存
		questions, err := s.generateDailyQuestions(userID, studyPlanID, actualQuestionBankID, studyPlan.DailyCount)
		if err != nil {
			return nil, err
		}

		// 提取题目ID
		for _, q := range questions {
			questionIDs = append(questionIDs, q.Question.ID)
		}

		// 缓存题目ID列表
		if err := s.cacheDailyQuestions(userID, studyPlanID, questionIDs, today); err != nil {
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
	questions, err := s.getQuestionsByIDs(questionIDs, actualQuestionBankID)
	if err != nil {
		return nil, err
	}

	// 计算 start 位置（今天已学习的题目数量）
	start, err := s.calculateStartPosition(userID, questionIDs, today)
	if err != nil {
		return nil, err
	}

	return &DailyQuestionsResponse{
		Questions: questions,
		Start:     start,
		Total:     len(questions),
	}, nil
}

// getReviewQuestions 获取需要复习的题目
func (s *EbbinghausService) getReviewQuestions(userID, questionBankID uint, today time.Time, reviewQuestions *[]QuestionWithProgress) error {
	var progresses []models.UserQuestionProgress

	// 查找到了复习时间的题目
	if err := s.db.Where("user_id = ? AND next_review_date <= ? AND is_completed = ?",
		userID, today, false).
		Preload("Question", "question_bank_id = ?", questionBankID).
		Find(&progresses).Error; err != nil {
		return err
	}

	for _, progress := range progresses {
		// 只处理指定题库的题目
		if progress.Question.QuestionBankID == questionBankID {
			*reviewQuestions = append(*reviewQuestions, QuestionWithProgress{
				Question: progress.Question,
				Progress: progress,
				IsReview: true,
			})
		}
	}

	return nil
}

// getNewQuestions 获取新题目
func (s *EbbinghausService) getNewQuestions(userID, questionBankID uint, count int) ([]QuestionWithProgress, error) {
	var questions []models.Question

	// 获取用户没有学习过的题目
	if err := s.db.Table("questions").
		Where("question_bank_id = ?", questionBankID).
		Where("id NOT IN (SELECT question_id FROM user_question_progresses WHERE user_id = ?)", userID).
		Limit(count).Find(&questions).Error; err != nil {
		return nil, err
	}

	var result []QuestionWithProgress
	for _, question := range questions {
		result = append(result, QuestionWithProgress{
			Question: question,
			Progress: models.UserQuestionProgress{}, // 新题目没有进度
			IsReview: false,
		})
	}

	return result, nil
}

// getMasteredQuestions 获取已掌握的题目供重新学习
func (s *EbbinghausService) getMasteredQuestions(userID, questionBankID uint, count int) ([]QuestionWithProgress, error) {
	var progresses []models.UserQuestionProgress

	// 获取已掌握的题目，按最后复习时间排序（最久没复习的优先）
	if err := s.db.Where("user_id = ? AND is_completed = ?", userID, true).
		Preload("Question", "question_bank_id = ?", questionBankID).
		Order("last_review_date ASC").
		Limit(count).Find(&progresses).Error; err != nil {
		return nil, err
	}

	var result []QuestionWithProgress
	for _, progress := range progresses {
		// 只处理指定题库的题目
		if progress.Question.QuestionBankID == questionBankID {
			result = append(result, QuestionWithProgress{
				Question: progress.Question,
				Progress: progress,
				IsReview: true, // 标记为复习题目
			})
		}
	}

	return result, nil
}

// CompleteQuestion 完成题目学习
// 对于fork题库，直接使用原题库的题目ID记录学习进度，避免不必要的数据复制
func (s *EbbinghausService) CompleteQuestion(userID, questionID uint, resultType string) error {
	// 查找或创建学习进度记录
	// 注意：这里直接使用传入的questionID，不需要进行写时复制
	var progress models.UserQuestionProgress
	err := s.db.Where("user_id = ? AND question_id = ?", userID, questionID).First(&progress).Error

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

		if err := s.db.Create(&progress).Error; err != nil {
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

		if err := s.db.Save(&progress).Error; err != nil {
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

// QuestionWithProgress 带学习进度的题目
type QuestionWithProgress struct {
	Question models.Question             `json:"question"`
	Progress models.UserQuestionProgress `json:"progress"`
	IsReview bool                        `json:"is_review"` // 是否是复习题目
}

// GetStudyStats 获取学习统计信息
func (s *EbbinghausService) GetStudyStats(userID uint) (*StudyStats, error) {
	var stats StudyStats

	// 总学习题目数
	s.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ?", userID).Count(&stats.TotalStudied)

	// 已完成题目数
	s.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ? AND is_completed = ?", userID, true).Count(&stats.Completed)

	// 今日需复习题目数
	today := time.Now().Truncate(24 * time.Hour)
	s.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ? AND next_review_date <= ? AND is_completed = ?",
			userID, today, false).Count(&stats.TodayReview)

	return &stats, nil
}

// StudyStats 学习统计
type StudyStats struct {
	TotalStudied int64 `json:"total_studied"`
	Completed    int64 `json:"completed"`
	TodayReview  int64 `json:"today_review"`
}

// QuestionBankProgress 题库进度统计
type QuestionBankProgress struct {
	QuestionBankID uint  `json:"question_bank_id"`
	TotalQuestions int64 `json:"total_questions"` // 题库总题目数
	StudiedCount   int64 `json:"studied_count"`   // 已学习题目数
	CompletedCount int64 `json:"completed_count"` // 已掌握题目数
	ReviewCount    int64 `json:"review_count"`    // 待复习题目数
	ProgressRate   int   `json:"progress_rate"`   // 学习进度百分比
	MasteryRate    int   `json:"mastery_rate"`    // 掌握率百分比
}

// GetQuestionBankProgress 获取用户在特定题库的学习进度
func (s *EbbinghausService) GetQuestionBankProgress(userID, questionBankID uint) (*QuestionBankProgress, error) {
	var progress QuestionBankProgress
	progress.QuestionBankID = questionBankID

	// 获取题库信息，判断是否是fork题库
	var questionBank models.QuestionBank
	if err := s.db.Where("id = ?", questionBankID).First(&questionBank).Error; err != nil {
		return nil, err
	}

	// 确定实际统计的题库ID
	actualQuestionBankID := questionBankID
	if questionBank.ForkedFrom != nil {
		var localCount int64
		s.db.Model(&models.Question{}).Where("question_bank_id = ?", questionBank.ID).Count(&localCount)
		if localCount == 0 {
			// 还没有发生写时复制，使用原题库ID进行统计
			actualQuestionBankID = *questionBank.ForkedFrom
		}
	}

	// 获取题库总题目数
	if err := s.db.Model(&models.Question{}).
		Where("question_bank_id = ?", actualQuestionBankID).
		Count(&progress.TotalQuestions).Error; err != nil {
		return nil, err
	}

	// 获取已学习题目数（有学习记录的）
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ?", userID, actualQuestionBankID).
		Count(&progress.StudiedCount).Error; err != nil {
		return nil, err
	}

	// 获取已掌握题目数
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ?",
			userID, actualQuestionBankID, true).
		Count(&progress.CompletedCount).Error; err != nil {
		return nil, err
	}

	// 获取待复习题目数（未完成且到了复习时间）
	today := time.Now().Truncate(24 * time.Hour)
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ? AND user_question_progresses.next_review_date <= ?",
			userID, actualQuestionBankID, false, today).
		Count(&progress.ReviewCount).Error; err != nil {
		return nil, err
	}

	// 计算进度百分比
	if progress.TotalQuestions > 0 {
		progress.ProgressRate = int((progress.StudiedCount * 100) / progress.TotalQuestions)
		progress.MasteryRate = int((progress.CompletedCount * 100) / progress.TotalQuestions)
	}

	return &progress, nil
}

// GetAllQuestionBanksProgress 获取用户在所有题库的学习进度
func (s *EbbinghausService) GetAllQuestionBanksProgress(userID uint) ([]QuestionBankProgress, error) {
	// 获取所有题库
	var questionBanks []models.QuestionBank
	if err := s.db.Find(&questionBanks).Error; err != nil {
		return nil, err
	}

	var progressList []QuestionBankProgress
	for _, bank := range questionBanks {
		progress, err := s.GetQuestionBankProgress(userID, bank.ID)
		if err != nil {
			return nil, err
		}
		progressList = append(progressList, *progress)
	}

	return progressList, nil
}

// CheckInToday 今日打卡
func (s *EbbinghausService) CheckInToday(userID uint) error {
	// 明确使用UTC时间，避免时区问题
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// 检查今日是否已打卡
	var existingCheckIn models.UserCheckIn
	err := s.db.Where("user_id = ? AND check_date = ?", userID, today).First(&existingCheckIn).Error
	if err == nil {
		return errors.New("今日已打卡")
	}

	// 获取昨天的打卡记录来计算连续天数和最长连续天数
	yesterday := today.AddDate(0, 0, -1)
	var yesterdayCheckIn models.UserCheckIn
	consecutiveDays := 1 // 默认为1（今天是第一天）
	bestStreak := 1      // 默认为1

	err = s.db.Where("user_id = ? AND check_date = ?", userID, yesterday).First(&yesterdayCheckIn).Error
	if err == nil {
		// 昨天有打卡，连续天数 = 昨天的连续天数 + 1
		consecutiveDays = yesterdayCheckIn.ConsecutiveDays + 1
		// 最长连续天数 = max(当前连续天数, 昨天的最长连续天数)
		bestStreak = max(consecutiveDays, yesterdayCheckIn.BestStreak)
	} else {
		// 昨天没打卡，获取最近的一条记录来获取历史最长连续天数
		var latestCheckIn models.UserCheckIn
		err = s.db.Where("user_id = ?", userID).
			Order("check_date DESC").
			First(&latestCheckIn).Error
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

	return s.db.Create(&checkIn).Error
}

// GetCheckInStats 获取打卡统计
func (s *EbbinghausService) GetCheckInStats(userID uint) (*CheckInStats, error) {
	var stats CheckInStats
	// 明确使用UTC时间，避免时区问题
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// 检查今日是否已打卡，同时获取连续天数和最长连续天数
	var todayCheckIn models.UserCheckIn
	err := s.db.Where("user_id = ? AND check_date = ?", userID, today).First(&todayCheckIn).Error
	if err == nil {
		stats.CheckedInToday = true
		stats.ConsecutiveDays = int64(todayCheckIn.ConsecutiveDays)
		stats.BestStreak = int64(todayCheckIn.BestStreak)
	} else {
		stats.CheckedInToday = false

		// 获取昨天的打卡记录
		yesterday := today.AddDate(0, 0, -1)
		var yesterdayCheckIn models.UserCheckIn
		err = s.db.Where("user_id = ? AND check_date = ?", userID, yesterday).First(&yesterdayCheckIn).Error
		if err == nil {
			// 昨天有打卡，显示昨天的连续天数和最长连续天数
			stats.ConsecutiveDays = int64(yesterdayCheckIn.ConsecutiveDays)
			stats.BestStreak = int64(yesterdayCheckIn.BestStreak)
		} else {
			// 昨天没打卡，连续天数归零，但仍需获取历史最长连续天数
			stats.ConsecutiveDays = 0

			// 获取最近的一条记录来获取历史最长连续天数
			var latestCheckIn models.UserCheckIn
			err = s.db.Where("user_id = ?", userID).
				Order("check_date DESC").
				First(&latestCheckIn).Error
			if err == nil {
				stats.BestStreak = int64(latestCheckIn.BestStreak)
			} else {
				stats.BestStreak = 0 // 没有任何历史记录
			}
		}
	}

	// 计算总打卡天数
	s.db.Model(&models.UserCheckIn{}).Where("user_id = ?", userID).Count(&stats.TotalCheckInDays)

	return &stats, nil
}

// CheckInStats 打卡统计
type CheckInStats struct {
	CheckedInToday   bool  `json:"checked_in_today"`    // 今日是否已打卡
	TotalCheckInDays int64 `json:"total_check_in_days"` // 总打卡天数
	ConsecutiveDays  int64 `json:"consecutive_days"`    // 当前连续打卡天数
	BestStreak       int64 `json:"best_streak"`         // 历史最长连续天数
}

// StudyPlanProgress 学习计划进度统计
type StudyPlanProgress struct {
	StudyPlanID    uint  `json:"study_plan_id"`
	TotalQuestions int64 `json:"total_questions"` // 题库总题目数
	StudiedCount   int64 `json:"studied_count"`   // 已学习题目数
	CompletedCount int64 `json:"completed_count"` // 已掌握题目数
	ReviewCount    int64 `json:"review_count"`    // 待复习题目数
	ProgressRate   int   `json:"progress_rate"`   // 学习进度百分比
	MasteryRate    int   `json:"mastery_rate"`    // 掌握率百分比
}

// GetStudyPlanProgress 获取学习计划的学习进度
func (s *EbbinghausService) GetStudyPlanProgress(userID, studyPlanID uint) (*StudyPlanProgress, error) {
	// 获取学习计划
	var studyPlan models.UserStudyPlan
	if err := s.db.Where("id = ? AND user_id = ?", studyPlanID, userID).
		Preload("QuestionBank").First(&studyPlan).Error; err != nil {
		return nil, errors.New("学习计划不存在")
	}

	var progress StudyPlanProgress
	progress.StudyPlanID = studyPlanID

	// 确定实际统计的题库ID
	actualQuestionBankID := studyPlan.QuestionBankID
	questionBank := studyPlan.QuestionBank
	if questionBank.ForkedFrom != nil {
		var localCount int64
		s.db.Model(&models.Question{}).Where("question_bank_id = ?", questionBank.ID).Count(&localCount)
		if localCount == 0 {
			// 还没有发生写时复制，使用原题库ID进行统计
			actualQuestionBankID = *questionBank.ForkedFrom
		}
	}

	// 获取题库总题目数
	if err := s.db.Model(&models.Question{}).
		Where("question_bank_id = ?", actualQuestionBankID).
		Count(&progress.TotalQuestions).Error; err != nil {
		return nil, err
	}

	// 获取已学习题目数（有学习记录的）
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ?", userID, actualQuestionBankID).
		Count(&progress.StudiedCount).Error; err != nil {
		return nil, err
	}

	// 获取已掌握题目数
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ?",
			userID, actualQuestionBankID, true).
		Count(&progress.CompletedCount).Error; err != nil {
		return nil, err
	}

	// 获取待复习题目数（未完成且到了复习时间）
	today := time.Now().Truncate(24 * time.Hour)
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ? AND user_question_progresses.next_review_date <= ?",
			userID, actualQuestionBankID, false, today).
		Count(&progress.ReviewCount).Error; err != nil {
		return nil, err
	}

	// 计算进度百分比
	if progress.TotalQuestions > 0 {
		progress.ProgressRate = int((progress.StudiedCount * 100) / progress.TotalQuestions)
		progress.MasteryRate = int((progress.CompletedCount * 100) / progress.TotalQuestions)
	}

	return &progress, nil
}

// DeleteStudyPlanWithProgress 删除学习计划并清空相关的学习进度
func (s *EbbinghausService) DeleteStudyPlanWithProgress(userID, studyPlanID uint) error {
	// 先获取学习计划信息
	var studyPlan models.UserStudyPlan
	if err := s.db.Where("id = ? AND user_id = ?", studyPlanID, userID).
		Preload("QuestionBank").First(&studyPlan).Error; err != nil {
		return errors.New("学习计划不存在")
	}

	// 确定实际的题库ID
	actualQuestionBankID := studyPlan.QuestionBankID
	questionBank := studyPlan.QuestionBank
	if questionBank.ForkedFrom != nil {
		var localCount int64
		s.db.Model(&models.Question{}).Where("question_bank_id = ?", questionBank.ID).Count(&localCount)
		if localCount == 0 {
			// 还没有发生写时复制，使用原题库ID
			actualQuestionBankID = *questionBank.ForkedFrom
		}
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除该用户在该题库中的所有学习进度记录
	if err := tx.Exec(`
		DELETE FROM user_question_progresses 
		WHERE user_id = ? AND question_id IN (
			SELECT id FROM questions WHERE question_bank_id = ?
		)
	`, userID, actualQuestionBankID).Error; err != nil {
		tx.Rollback()
		return errors.New("清空学习进度失败")
	}

	// 删除学习计划
	if err := tx.Where("id = ? AND user_id = ?", studyPlanID, userID).
		Delete(&models.UserStudyPlan{}).Error; err != nil {
		tx.Rollback()
		return errors.New("删除学习计划失败")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return errors.New("删除操作提交失败")
	}

	return nil
}

// GetStudyHeatmap 获取学习活动热力图数据
func (s *EbbinghausService) GetStudyHeatmap(userID uint) (*HeatmapResponse, error) {
	// 获取过去一年的日期范围
	endDate := time.Now()
	startDate := endDate.AddDate(-1, 0, 0) // 一年前

	// 查询用户在过去一年的学习记录（每天学习的不同题目数量）
	var dailyStats []struct {
		Date  time.Time `json:"date"`
		Count int64     `json:"count"`
	}

	// 从学习记录表获取每天学习的不同题目数量（去重）
	// 确保包含今天的所有数据，使用明天作为结束日期
	tomorrowDate := endDate.AddDate(0, 0, 1).Format("2006-01-02")
	err := s.db.Table("user_question_progresses").
		Select("DATE(last_review_date) as date, COUNT(DISTINCT question_id) as count").
		Where("user_id = ? AND DATE(last_review_date) >= DATE(?) AND DATE(last_review_date) < DATE(?) AND last_review_date IS NOT NULL", userID, startDate.Format("2006-01-02"), tomorrowDate).
		Group("DATE(last_review_date)").
		Order("date").
		Scan(&dailyStats).Error

	if err != nil {
		return nil, err
	}

	// 构建热力图数据
	heatmapData := make([]HeatmapData, 0)
	currentDate := startDate

	// 创建日期到学习次数的映射
	statsMap := make(map[string]int64)
	for _, stat := range dailyStats {
		dateStr := stat.Date.Format("2006-01-02")
		statsMap[dateStr] = stat.Count
	}

	// 填充过去一年的每一天，包括今天
	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour)
	for currentDate.Before(tomorrow) {
		dateStr := currentDate.Format("2006-01-02")
		count := int(statsMap[dateStr])

		// 根据学习次数确定活跃度级别 (0-4)
		level := 0
		if count > 0 {
			if count >= 10 {
				level = 4
			} else if count >= 7 {
				level = 3
			} else if count >= 4 {
				level = 2
			} else {
				level = 1
			}
		}

		heatmapData = append(heatmapData, HeatmapData{
			Date:  dateStr,
			Count: count,
			Level: level,
		})

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// 计算统计信息
	totalStudyDays := 0     // 总学习天数
	totalStudyCount := 0    // 总学习次数（题目数）
	currentStreak := 0      // 当前连续学习天数
	maxStreak := 0          // 最长连续学习天数
	tempStreak := 0         // 临时连续天数
	thisYearStudyCount := 0 // 今年学习次数

	// 计算总学习天数和总学习次数
	for _, data := range heatmapData {
		if data.Count > 0 {
			totalStudyDays++
			totalStudyCount += data.Count
		}
	}

	// 计算当前连续学习天数（从今天往前推）
	for i := len(heatmapData) - 1; i >= 0; i-- {
		if heatmapData[i].Count > 0 {
			currentStreak++
		} else {
			break // 遇到没有学习的天数就停止
		}
	}

	// 计算最长连续学习天数
	for i := 0; i < len(heatmapData); i++ {
		if heatmapData[i].Count > 0 {
			tempStreak++
			if tempStreak > maxStreak {
				maxStreak = tempStreak
			}
		} else {
			tempStreak = 0 // 重置临时连续天数
		}
	}

	// 计算今年的学习次数
	currentYear := time.Now().Year()
	for _, data := range heatmapData {
		date, _ := time.Parse("2006-01-02", data.Date)
		if date.Year() == currentYear {
			thisYearStudyCount += data.Count
		}
	}

	return &HeatmapResponse{
		Data:          heatmapData,
		TotalCommits:  totalStudyCount,    // 总学习次数（题目数）
		CurrentStreak: currentStreak,      // 当前连续学习天数
		MaxStreak:     maxStreak,          // 最长连续学习天数
		ThisYear:      thisYearStudyCount, // 今年学习次数
	}, nil
}

// generateDailyQuestions 生成每日学习题目
func (s *EbbinghausService) generateDailyQuestions(userID, studyPlanID, questionBankID uint, dailyCount int) ([]QuestionWithProgress, error) {
	today := time.Now().Truncate(24 * time.Hour)

	// 1. 获取需要复习的题目（根据艾宾浩斯遗忘曲线）
	var reviewQuestions []QuestionWithProgress
	if err := s.getReviewQuestions(userID, questionBankID, today, &reviewQuestions); err != nil {
		return nil, err
	}

	// 限制复习题目数量不超过每日学习量
	if len(reviewQuestions) > dailyCount {
		reviewQuestions = reviewQuestions[:dailyCount]
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

	// 3. 如果仍然不足，提供已掌握的题目供重新学习
	currentCount = len(reviewQuestions)
	if currentCount < dailyCount {
		masteredQuestions, err := s.getMasteredQuestions(userID, questionBankID, dailyCount-currentCount)
		if err != nil {
			return nil, err
		}
		reviewQuestions = append(reviewQuestions, masteredQuestions...)
	}

	// 如果还是没有题目，检查题库是否为空
	if len(reviewQuestions) == 0 {
		var questionCount int64
		s.db.Model(&models.Question{}).Where("question_bank_id = ?", questionBankID).Count(&questionCount)
		if questionCount == 0 {
			return nil, errors.New("题库中没有题目，请先添加题目")
		}
	}

	// 确保不超过每日学习量
	if len(reviewQuestions) > dailyCount {
		reviewQuestions = reviewQuestions[:dailyCount]
	}

	return reviewQuestions, nil
}

// cacheDailyQuestions 缓存每日学习题目
func (s *EbbinghausService) cacheDailyQuestions(userID, studyPlanID uint, questionIDs []uint, cacheDate time.Time) error {
	// 序列化题目ID列表
	questionIDsJSON, err := json.Marshal(questionIDs)
	if err != nil {
		return err
	}

	// 创建缓存记录
	cache := models.DailyStudyPlanCache{
		UserID:      userID,
		StudyPlanID: studyPlanID,
		CacheDate:   cacheDate,
		QuestionIDs: string(questionIDsJSON),
	}

	return s.db.Create(&cache).Error
}

// getQuestionsByIDs 根据题目ID列表获取完整的题目信息
func (s *EbbinghausService) getQuestionsByIDs(questionIDs []uint, questionBankID uint) ([]QuestionWithProgress, error) {
	if len(questionIDs) == 0 {
		return []QuestionWithProgress{}, nil
	}

	var questions []models.Question
	if err := s.db.Where("id IN ? AND question_bank_id = ?", questionIDs, questionBankID).Find(&questions).Error; err != nil {
		return nil, err
	}

	// 创建ID到题目的映射，保持原有顺序
	questionMap := make(map[uint]models.Question)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	var result []QuestionWithProgress
	for _, questionID := range questionIDs {
		if question, exists := questionMap[questionID]; exists {
			// 获取学习进度（如果有的话）
			var progress models.UserQuestionProgress
			s.db.Where("question_id = ?", questionID).First(&progress)

			result = append(result, QuestionWithProgress{
				Question: question,
				Progress: progress,
				IsReview: progress.ID != 0, // 如果有进度记录，说明是复习题目
			})
		}
	}

	return result, nil
}

// calculateStartPosition 计算开始位置（今天已学习的题目数量）
func (s *EbbinghausService) calculateStartPosition(userID uint, questionIDs []uint, today time.Time) (int, error) {
	if len(questionIDs) == 0 {
		return 0, nil
	}

	// 统计今天已经学习过的题目数量
	var studiedCount int64
	err := s.db.Model(&models.UserQuestionProgress{}).
		Where("user_id = ? AND question_id IN ? AND DATE(last_review_date) = DATE(?)",
			userID, questionIDs, today).
		Count(&studiedCount).Error

	if err != nil {
		return 0, err
	}

	return int(studiedCount), nil
}
