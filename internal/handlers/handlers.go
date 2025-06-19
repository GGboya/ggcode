package handlers

import (
	"ggcode/internal/database"
	"ggcode/internal/middleware"
	"ggcode/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	db                *gorm.DB
	ebbinghausService *services.EbbinghausService
}

func New(db *gorm.DB) *Handler {
	return &Handler{
		db:                db,
		ebbinghausService: services.NewEbbinghausService(db),
	}
}

// 页面处理器
func (h *Handler) HomePage(c *gin.Context) {
	// 首页不强制要求认证，但如果有token则获取用户信息
	username := c.GetString("username")
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":    "GGCode - 算法学习平台",
		"username": username,
	})
}

func (h *Handler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":    "登录",
		"pageType": "login",
	})
}

func (h *Handler) RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"title":    "注册",
		"pageType": "register",
	})
}

func (h *Handler) Dashboard(c *gin.Context) {
	userID := c.GetUint("user_id")
	username := c.GetString("username")

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":    "仪表板",
		"userID":   userID,
		"username": username,
		"pageType": "dashboard",
	})
}

func (h *Handler) QuestionBanksPage(c *gin.Context) {
	username := c.GetString("username")
	userID := c.GetUint("user_id")
	c.HTML(http.StatusOK, "questionbanks.html", gin.H{
		"title":    "题库管理",
		"username": username,
		"userID":   userID,
	})
}

func (h *Handler) StudyPlansPage(c *gin.Context) {
	username := c.GetString("username")
	userID := c.GetUint("user_id")
	c.HTML(http.StatusOK, "study-plans.html", gin.H{
		"title":    "学习计划管理",
		"username": username,
		"userID":   userID,
		"pageType": "study-plans",
	})
}

func (h *Handler) StudyPage(c *gin.Context) {
	username := c.GetString("username")
	userID := c.GetUint("user_id")
	c.HTML(http.StatusOK, "study.html", gin.H{
		"title":    "开始学习",
		"username": username,
		"userID":   userID,
		"pageType": "study",
	})
}

// API 处理器
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user database.User
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	// 设置 Cookie，便于页面跳转自动携带认证
	c.SetCookie("token", token, 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户名是否已存在
	var existingUser database.User
	if err := h.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名或邮箱已存在"})
		return
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	user := database.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	// 设置 Cookie
	c.SetCookie("token", token, 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *Handler) GetQuestionBanks(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankType := c.Query("type") // "shared", "personal", "all"
	sortBy := c.Query("sort")   // "star_count", "fork_count", "created_at"

	var questionBanks []database.QuestionBank
	query := h.db.Model(&database.QuestionBank{})

	switch bankType {
	case "shared":
		// 获取共享题库，包括官方题库和其他用户分享的题库
		query = query.Where("is_official = ? OR is_shared = ?", true, true)
	case "personal":
		// 获取个人题库（包括自己创建的和Fork的）
		query = query.Where("created_by = ?", userID)
	default:
		// 默认获取所有可见题库：官方题库、共享题库、用户创建的题库
		query = query.Where("is_official = ? OR is_shared = ? OR created_by = ?", true, true, userID)
	}

	// 排序
	switch sortBy {
	case "star_count":
		query = query.Order("star_count DESC")
	case "fork_count":
		query = query.Order("fork_count DESC")
	default:
		query = query.Order("created_at DESC")
	}

	if err := query.Preload("Creator").Preload("OriginalBank").Find(&questionBanks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题库失败"})
		return
	}

	// 查询当前用户是否已经Star了这些题库
	var starredBankIDs []uint
	if bankType != "personal" {
		h.db.Model(&database.QuestionBankStar{}).
			Where("user_id = ?", userID).
			Pluck("question_bank_id", &starredBankIDs)
	}

	// 为每个题库添加是否被当前用户Star的标记
	for i := range questionBanks {
		questionBanks[i].ID = questionBanks[i].ID // 确保ID被包含
		// 检查是否被当前用户Star
		for _, starredID := range starredBankIDs {
			if questionBanks[i].ID == starredID {
				// 这里可以添加一个字段表示已Star，暂时通过前端处理
				break
			}
		}
	}

	c.JSON(http.StatusOK, questionBanks)
}

func (h *Handler) CreateQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	questionBank := database.QuestionBank{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   &userID, // 使用指针
		IsOfficial:  false,
	}

	if err := h.db.Create(&questionBank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建题库失败"})
		return
	}

	c.JSON(http.StatusCreated, questionBank)
}

func (h *Handler) GetQuestions(c *gin.Context) {
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	var questions []database.Question
	if err := h.db.Where("question_bank_id = ?", bankID).Find(&questions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题目失败"})
		return
	}

	c.JSON(http.StatusOK, questions)
}

func (h *Handler) CreateQuestion(c *gin.Context) {
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		LeetcodeURL string `json:"leetcode_url" binding:"required"`
		Difficulty  string `json:"difficulty" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	question := database.Question{
		Title:          req.Title,
		LeetcodeURL:    req.LeetcodeURL,
		Difficulty:     req.Difficulty,
		QuestionBankID: uint(bankID),
	}

	if err := h.db.Create(&question).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建题目失败"})
		return
	}

	c.JSON(http.StatusCreated, question)
}

func (h *Handler) CreateStudyPlan(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		QuestionBankID uint `json:"question_bank_id" binding:"required"`
		DailyCount     int  `json:"daily_count" binding:"required,min=1,max=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户是否已经为该题库创建了学习计划
	var existingPlan database.UserStudyPlan
	err := h.db.Where("user_id = ? AND question_bank_id = ?", userID, req.QuestionBankID).First(&existingPlan).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "您已经为该题库创建了学习计划，一个题库只能创建一个学习计划"})
		return
	}

	studyPlan := database.UserStudyPlan{
		UserID:         userID,
		QuestionBankID: req.QuestionBankID,
		DailyCount:     req.DailyCount,
	}

	if err := h.db.Create(&studyPlan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建学习计划失败"})
		return
	}

	// 预加载题库信息
	h.db.Preload("QuestionBank").First(&studyPlan, studyPlan.ID)

	c.JSON(http.StatusCreated, studyPlan)
}

func (h *Handler) GetStudyPlan(c *gin.Context) {
	userID := c.GetUint("user_id")
	planID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	var studyPlan database.UserStudyPlan
	if err := h.db.Where("id = ? AND user_id = ?", planID, userID).
		Preload("QuestionBank").First(&studyPlan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "学习计划不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取学习计划失败"})
		return
	}

	c.JSON(http.StatusOK, studyPlan)
}

func (h *Handler) UpdateStudyPlan(c *gin.Context) {
	userID := c.GetUint("user_id")
	planID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	var req struct {
		DailyCount int `json:"daily_count" binding:"required,min=1,max=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Model(&database.UserStudyPlan{}).
		Where("id = ? AND user_id = ?", planID, userID).
		Update("daily_count", req.DailyCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新学习计划失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "学习计划已更新"})
}

func (h *Handler) GetDailyQuestions(c *gin.Context) {
	userID := c.GetUint("user_id")
	planID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	questions, err := h.ebbinghausService.GetDailyQuestions(userID, uint(planID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, questions)
}

func (h *Handler) CompleteQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		QuestionID uint   `json:"question_id" binding:"required"`
		ResultType string `json:"result_type" binding:"required"` // "ac" 或 "failed"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证结果类型
	if req.ResultType != "ac" && req.ResultType != "failed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结果类型"})
		return
	}

	if err := h.ebbinghausService.CompleteQuestion(userID, req.QuestionID, req.ResultType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 根据结果类型返回不同的消息
	message := "题目完成"
	if req.ResultType == "ac" {
		message = "恭喜独立AC！已自动打卡"
	} else {
		message = "学习记录已保存，继续加油！已自动打卡"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

func (h *Handler) GetStudyStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	stats, err := h.ebbinghausService.GetStudyStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetQuestionBankProgress 获取特定题库的学习进度
func (h *Handler) GetQuestionBankProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	progress, err := h.ebbinghausService.GetQuestionBankProgress(userID, uint(bankID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetAllQuestionBanksProgress 获取所有题库的学习进度
func (h *Handler) GetAllQuestionBanksProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	progressList, err := h.ebbinghausService.GetAllQuestionBanksProgress(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progressList)
}

// CheckInToday 今日打卡
func (h *Handler) CheckInToday(c *gin.Context) {
	userID := c.GetUint("user_id")

	err := h.ebbinghausService.CheckInToday(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "打卡成功！"})
}

// GetCheckInStats 获取打卡统计
func (h *Handler) GetCheckInStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	stats, err := h.ebbinghausService.GetCheckInStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// DeleteStudyPlan 删除学习计划
func (h *Handler) DeleteStudyPlan(c *gin.Context) {
	userID := c.GetUint("user_id")
	planID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	// 使用服务层的删除方法
	if err := h.ebbinghausService.DeleteStudyPlanWithProgress(userID, uint(planID)); err != nil {
		if err.Error() == "学习计划不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "学习计划和对应的学习进度已删除"})
}

// GetAllStudyPlans 获取用户所有学习计划
func (h *Handler) GetAllStudyPlans(c *gin.Context) {
	userID := c.GetUint("user_id")

	var studyPlans []database.UserStudyPlan
	if err := h.db.Where("user_id = ?", userID).
		Preload("QuestionBank").
		Order("created_at DESC").
		Find(&studyPlans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, studyPlans)
}

// GetStudyPlanProgress 获取学习计划进度
func (h *Handler) GetStudyPlanProgress(c *gin.Context) {
	userID := c.GetUint("user_id")
	planID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	progress, err := h.ebbinghausService.GetStudyPlanProgress(userID, uint(planID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetRandomMasteredQuestions 获取随机的已掌握题目供重复学习
func (h *Handler) GetRandomMasteredQuestions(c *gin.Context) {
	userID := c.GetUint("user_id")
	planID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	// 获取指定的学习计划
	var studyPlan database.UserStudyPlan
	err = h.db.Where("id = ? AND user_id = ?", planID, userID).
		Preload("QuestionBank").First(&studyPlan).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学习计划不存在"})
		return
	}

	// 获取已掌握的题目，随机排序
	var progresses []database.UserQuestionProgress
	if err := h.db.Where("user_id = ? AND is_completed = ?", userID, true).
		Preload("Question", "question_bank_id = ?", studyPlan.QuestionBankID).
		Order("RANDOM()").
		Limit(studyPlan.DailyCount).
		Find(&progresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为QuestionWithProgress格式
	var questions []services.QuestionWithProgress
	for _, progress := range progresses {
		if progress.Question.QuestionBankID == studyPlan.QuestionBankID {
			questions = append(questions, services.QuestionWithProgress{
				Question: progress.Question,
				Progress: progress,
				IsReview: true, // 标记为复习题目
			})
		}
	}

	c.JSON(http.StatusOK, questions)
}

// UpdateQuestionBank 更新题库信息
func (h *Handler) UpdateQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查题库是否存在且属于当前用户
	var questionBank database.QuestionBank
	if err := h.db.Where("id = ? AND created_by = ?", bankID, userID).First(&questionBank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "题库不存在或无权限修改"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询题库失败"})
		return
	}

	// 更新题库信息
	questionBank.Name = req.Name
	questionBank.Description = req.Description

	if err := h.db.Save(&questionBank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新题库失败"})
		return
	}

	c.JSON(http.StatusOK, questionBank)
}

// DeleteQuestionBank 删除题库
func (h *Handler) DeleteQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	// 检查题库是否存在且属于当前用户
	var questionBank database.QuestionBank
	if err := h.db.Where("id = ? AND created_by = ?", bankID, userID).First(&questionBank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "题库不存在或无权限删除"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询题库失败"})
		return
	}

	// 检查是否有用户正在使用此题库的学习计划
	var studyPlanCount int64
	h.db.Model(&database.UserStudyPlan{}).Where("question_bank_id = ?", bankID).Count(&studyPlanCount)
	if studyPlanCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该题库正在被学习计划使用，无法删除"})
		return
	}

	// 开始事务删除题库及其题目
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除题库中的所有题目
	if err := tx.Where("question_bank_id = ?", bankID).Delete(&database.Question{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除题目失败"})
		return
	}

	// 删除题库
	if err := tx.Delete(&questionBank).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除题库失败"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "题库删除成功"})
}

// UpdateQuestion 更新题目信息
func (h *Handler) UpdateQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题目ID"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		LeetcodeURL string `json:"leetcode_url" binding:"required"`
		Difficulty  string `json:"difficulty" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查题目是否存在且属于用户创建的题库
	var question database.Question
	if err := h.db.Preload("QuestionBank").Where("id = ?", questionID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询题目失败"})
		return
	}

	// 检查权限：只能编辑自己创建的题库中的题目
	if question.QuestionBank.CreatedBy == nil || *question.QuestionBank.CreatedBy != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限修改此题目"})
		return
	}

	// 更新题目信息
	question.Title = req.Title
	question.LeetcodeURL = req.LeetcodeURL
	question.Difficulty = req.Difficulty

	if err := h.db.Save(&question).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新题目失败"})
		return
	}

	c.JSON(http.StatusOK, question)
}

// DeleteQuestion 删除题目
func (h *Handler) DeleteQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题目ID"})
		return
	}

	// 检查题目是否存在且属于用户创建的题库
	var question database.Question
	if err := h.db.Preload("QuestionBank").Where("id = ?", questionID).First(&question).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询题目失败"})
		return
	}

	// 检查权限：只能删除自己创建的题库中的题目
	if question.QuestionBank.CreatedBy == nil || *question.QuestionBank.CreatedBy != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限删除此题目"})
		return
	}

	// 开始事务删除题目及相关学习进度
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除该题目的所有学习进度记录
	if err := tx.Where("question_id = ?", questionID).Delete(&database.UserQuestionProgress{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除学习进度失败"})
		return
	}

	// 删除题目
	if err := tx.Delete(&question).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除题目失败"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "题目删除成功"})
}

// 共享题库相关功能

// ShareQuestionBank 将个人题库设为共享
func (h *Handler) ShareQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	// 检查题库是否存在且属于当前用户
	var questionBank database.QuestionBank
	if err := h.db.Where("id = ? AND created_by = ?", bankID, userID).First(&questionBank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "题库不存在或无权限操作"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询题库失败"})
		return
	}

	// 更新为共享状态
	questionBank.IsShared = true
	if err := h.db.Save(&questionBank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置共享失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题库已设为共享"})
}

// UnshareQuestionBank 取消共享题库
func (h *Handler) UnshareQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	// 检查题库是否存在且属于当前用户
	var questionBank database.QuestionBank
	if err := h.db.Where("id = ? AND created_by = ?", bankID, userID).First(&questionBank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "题库不存在或无权限操作"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询题库失败"})
		return
	}

	// 取消共享状态
	questionBank.IsShared = false
	if err := h.db.Save(&questionBank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消共享失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题库已取消共享"})
}

// StarQuestionBank Star题库
func (h *Handler) StarQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	// 检查题库是否存在且为共享状态
	var questionBank database.QuestionBank
	if err := h.db.Where("id = ? AND (is_official = ? OR is_shared = ?)", bankID, true, true).First(&questionBank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "题库不存在或未共享"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询题库失败"})
		return
	}

	// 检查是否已经Star
	var existingStar database.QuestionBankStar
	if err := h.db.Where("user_id = ? AND question_bank_id = ?", userID, bankID).First(&existingStar).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "已经Star过这个题库"})
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建Star记录
	star := database.QuestionBankStar{
		UserID:         userID,
		QuestionBankID: uint(bankID),
	}
	if err := tx.Create(&star).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Star失败"})
		return
	}

	// 更新题库的Star数量
	if err := tx.Model(&questionBank).Update("star_count", gorm.Expr("star_count + ?", 1)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新Star数量失败"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Star成功"})
}

// UnstarQuestionBank 取消Star题库
func (h *Handler) UnstarQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	// 检查是否已经Star
	var star database.QuestionBankStar
	if err := h.db.Where("user_id = ? AND question_bank_id = ?", userID, bankID).First(&star).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "尚未Star该题库"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询Star记录失败"})
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除Star记录
	if err := tx.Delete(&star).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消Star失败"})
		return
	}

	// 更新题库的Star数量
	var questionBank database.QuestionBank
	if err := tx.Where("id = ?", bankID).First(&questionBank).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询题库失败"})
		return
	}

	if err := tx.Model(&questionBank).Update("star_count", gorm.Expr("star_count - ?", 1)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新Star数量失败"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "取消Star成功"})
}

// ForkQuestionBank Fork题库
func (h *Handler) ForkQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	// 检查题库是否存在且为共享状态
	var originalBank database.QuestionBank
	if err := h.db.Where("id = ? AND (is_official = ? OR is_shared = ?)", bankID, true, true).First(&originalBank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "题库不存在或未共享"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询题库失败"})
		return
	}

	// 检查用户是否已经Fork过这个题库
	var existingFork database.QuestionBank
	if err := h.db.Where("created_by = ? AND forked_from = ?", userID, bankID).First(&existingFork).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "已经Fork过这个题库"})
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建Fork的题库
	forkedBank := database.QuestionBank{
		Name:        originalBank.Name + " (Fork)",
		Description: originalBank.Description,
		CreatedBy:   &userID,
		ForkedFrom:  &originalBank.ID,
		IsOfficial:  false,
		IsShared:    false,
	}

	if err := tx.Create(&forkedBank).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建Fork题库失败"})
		return
	}

	// 复制原题库的所有题目
	var originalQuestions []database.Question
	if err := tx.Where("question_bank_id = ?", bankID).Find(&originalQuestions).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取原题库题目失败"})
		return
	}

	// 批量插入题目
	for _, question := range originalQuestions {
		newQuestion := database.Question{
			Title:          question.Title,
			LeetcodeURL:    question.LeetcodeURL,
			Difficulty:     question.Difficulty,
			QuestionBankID: forkedBank.ID,
		}
		if err := tx.Create(&newQuestion).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "复制题目失败"})
			return
		}
	}

	// 更新原题库的Fork数量
	if err := tx.Model(&originalBank).Update("fork_count", gorm.Expr("fork_count + ?", 1)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新Fork数量失败"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusCreated, gin.H{
		"message":     "Fork成功",
		"forked_bank": forkedBank,
	})
}

// GetUserStarredBanks 获取用户Star的题库列表
func (h *Handler) GetUserStarredBanks(c *gin.Context) {
	userID := c.GetUint("user_id")

	var starredBanks []database.QuestionBank
	if err := h.db.Table("question_banks").
		Joins("JOIN question_bank_stars ON question_banks.id = question_bank_stars.question_bank_id").
		Where("question_bank_stars.user_id = ?", userID).
		Preload("Creator").
		Find(&starredBanks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取Star题库失败"})
		return
	}

	c.JSON(http.StatusOK, starredBanks)
}
