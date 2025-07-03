package services

import (
	"errors"
	"ggcode/internal/models"
	"time"

	"gorm.io/gorm"
)

type EbbinghausService struct {
	db *gorm.DB
}

func NewEbbinghausService(db *gorm.DB) *EbbinghausService {
	return &EbbinghausService{db: db}
}

// 艾宾浩斯遗忘曲线复习间隔（天数）
var reviewIntervals = []int{1, 2, 4, 7, 15, 30, 60}

// GetDailyQuestions 获取指定学习计划的当天需要学习的题目
func (s *EbbinghausService) GetDailyQuestions(userID, studyPlanID uint) ([]QuestionWithProgress, error) {
	// 先确保 Fork 题库在首次使用时完成写时复制（COW）。
	// 如果学习计划关联的题库是 fork 且本地还没有题目，则把原题库的题目复制过来，
	// 这样可以避免后续学习流程因为题库为空而报错。

	// 获取指定的学习计划
	var studyPlan models.UserStudyPlan
	if err := s.db.Where("id = ? AND user_id = ?", studyPlanID, userID).
		Preload("QuestionBank").First(&studyPlan).Error; err != nil {
		return nil, errors.New("学习计划不存在")
	}

	questionBank := studyPlan.QuestionBank
	if questionBank.ForkedFrom != nil {
		var localCount int64
		s.db.Model(&models.Question{}).Where("question_bank_id = ?", questionBank.ID).Count(&localCount)
		if localCount == 0 {
			// 执行写时复制：把原题库题目复制到当前题库
			var originalQuestions []models.Question
			if err := s.db.Where("question_bank_id = ?", *questionBank.ForkedFrom).Find(&originalQuestions).Error; err != nil {
				return nil, err
			}
			for _, q := range originalQuestions {
				newQ := q
				newQ.ID = 0 // 让 GORM 生成新主键
				newQ.QuestionBankID = questionBank.ID
				if err := s.db.Create(&newQ).Error; err != nil {
					return nil, err
				}
			}
			// 解除与原题库的 fork 关联，避免后续重复复制
			_ = s.db.Model(&models.QuestionBank{}).
				Where("id = ?", questionBank.ID).
				Update("forked_from", nil).Error
		}
	}

	// 下面的逻辑保持不变，但由于我们提前重新查询了 studyPlan，需调整变量引用
	studyPlanID = studyPlan.ID // 保持原参数用途

	// 重新定义 today、reviewQuestions 等变量之前，继续向下执行原有逻辑

	today := time.Now().Truncate(24 * time.Hour)

	// 1. 获取需要复习的题目（根据艾宾浩斯遗忘曲线）
	var reviewQuestions []QuestionWithProgress
	if err := s.getReviewQuestions(userID, questionBank.ID, today, &reviewQuestions); err != nil {
		return nil, err
	}

	// 2. 如果复习题目不足每日学习量，添加新题目
	remainingCount := studyPlan.DailyCount - len(reviewQuestions)
	if remainingCount > 0 {
		newQuestions, err := s.getNewQuestions(userID, questionBank.ID, remainingCount)
		if err != nil {
			return nil, err
		}
		reviewQuestions = append(reviewQuestions, newQuestions...)
	}

	// 3. 如果仍然没有足够的题目，提供已掌握的题目供重新学习
	if len(reviewQuestions) < studyPlan.DailyCount {
		remainingCount = studyPlan.DailyCount - len(reviewQuestions)
		masteredQuestions, err := s.getMasteredQuestions(userID, questionBank.ID, remainingCount)
		if err != nil {
			return nil, err
		}
		reviewQuestions = append(reviewQuestions, masteredQuestions...)
	}

	// 4. 如果还是没有题目，说明题库为空，返回空数组而不是错误
	if len(reviewQuestions) == 0 {
		// 检查题库是否有题目
		var questionCount int64
		s.db.Model(&models.Question{}).Where("question_bank_id = ?", questionBank.ID).Count(&questionCount)
		if questionCount == 0 {
			return nil, errors.New("题库中没有题目，请先添加题目")
		}

		// 如果有题目但没有返回，可能是数据问题，返回空数组让用户可以继续操作
		return []QuestionWithProgress{}, nil
	}

	if len(reviewQuestions) > studyPlan.DailyCount {
		reviewQuestions = reviewQuestions[:studyPlan.DailyCount]
	}
	return reviewQuestions, nil
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
func (s *EbbinghausService) CompleteQuestion(userID, questionID uint, resultType string) error {
	// 查找或创建学习进度记录
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

	// 获取题库总题目数
	if err := s.db.Model(&models.Question{}).
		Where("question_bank_id = ?", questionBankID).
		Count(&progress.TotalQuestions).Error; err != nil {
		return nil, err
	}

	// 获取已学习题目数（有学习记录的）
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ?", userID, questionBankID).
		Count(&progress.StudiedCount).Error; err != nil {
		return nil, err
	}

	// 获取已掌握题目数
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ?",
			userID, questionBankID, true).
		Count(&progress.CompletedCount).Error; err != nil {
		return nil, err
	}

	// 获取待复习题目数（未完成且到了复习时间）
	today := time.Now().Truncate(24 * time.Hour)
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ? AND user_question_progresses.next_review_date <= ?",
			userID, questionBankID, false, today).
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

	// 获取题库总题目数
	if err := s.db.Model(&models.Question{}).
		Where("question_bank_id = ?", studyPlan.QuestionBankID).
		Count(&progress.TotalQuestions).Error; err != nil {
		return nil, err
	}

	// 获取已学习题目数（有学习记录的）
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ?", userID, studyPlan.QuestionBankID).
		Count(&progress.StudiedCount).Error; err != nil {
		return nil, err
	}

	// 获取已掌握题目数
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ?",
			userID, studyPlan.QuestionBankID, true).
		Count(&progress.CompletedCount).Error; err != nil {
		return nil, err
	}

	// 获取待复习题目数（未完成且到了复习时间）
	today := time.Now().Truncate(24 * time.Hour)
	if err := s.db.Model(&models.UserQuestionProgress{}).
		Joins("JOIN questions ON questions.id = user_question_progresses.question_id").
		Where("user_question_progresses.user_id = ? AND questions.question_bank_id = ? AND user_question_progresses.is_completed = ? AND user_question_progresses.next_review_date <= ?",
			userID, studyPlan.QuestionBankID, false, today).
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
	if err := s.db.Where("id = ? AND user_id = ?", studyPlanID, userID).First(&studyPlan).Error; err != nil {
		return errors.New("学习计划不存在")
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
	`, userID, studyPlan.QuestionBankID).Error; err != nil {
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
