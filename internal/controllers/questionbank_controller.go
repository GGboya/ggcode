package controllers

import (
	"ggcode/internal/repositories"
	"ggcode/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QuestionBankController struct {
	questionBankService *services.QuestionBankService
}

func NewQuestionBankController(services *services.Services) *QuestionBankController {
	return &QuestionBankController{questionBankService: services.QuestionBank}
}

// GetQuestionBanks 获取题库列表
func (ctrl *QuestionBankController) GetQuestionBanks(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankType := c.Query("type") // "official", "shared", "personal"
	sortBy := c.Query("sort")   // "star_count", "fork_count", "created_at"

	// 解析分页参数
	page := 1
	limit := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// 调用服务层获取题库列表
	response, err := ctrl.questionBankService.GetQuestionBanks(userID, bankType, sortBy, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题库失败"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CreateQuestionBank 创建题库
func (ctrl *QuestionBankController) CreateQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Source      string `json:"source"`
		MinScore    *int   `json:"min_score"`
		MaxScore    *int   `json:"max_score"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	minScore := 0
	if req.MinScore != nil {
		minScore = *req.MinScore
	}
	maxScore := 0
	if req.MaxScore != nil {
		maxScore = *req.MaxScore
	}

	questionBank, err := ctrl.questionBankService.CreateQuestionBankWithImport(req.Name, req.Description, userID, req.Source, minScore, maxScore)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建题库失败"})
		return
	}

	c.JSON(http.StatusCreated, questionBank)
}

// UpdateQuestionBank 更新题库
func (ctrl *QuestionBankController) UpdateQuestionBank(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = ctrl.questionBankService.UpdateQuestionBank(uint(bankID), userID, repositories.QuestionBankUpdateData{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新题库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题库更新成功"})
}

// DeleteQuestionBank 删除题库
func (ctrl *QuestionBankController) DeleteQuestionBank(c *gin.Context) {
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = ctrl.questionBankService.DeleteQuestionBank(uint(bankID), userID)
	if err != nil {
		// 根据错误类型返回不同的HTTP状态码
		switch err.Error() {
		case "题库不存在或无权限删除":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "该题库正在被学习计划使用，无法删除":
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题库删除成功"})
}

// GetQuestionBankProgress 获取特定题库的学习进度
func (ctrl *QuestionBankController) GetQuestionBankProgress(c *gin.Context) {
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	progress, err := ctrl.questionBankService.GetQuestionBankProgress(userID, uint(bankID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取进度失败"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetAllQuestionBanksProgress 获取所有题库的学习进度
func (ctrl *QuestionBankController) GetAllQuestionBanksProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	progresses, err := ctrl.questionBankService.GetAllQuestionBanksProgress(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取进度失败"})
		return
	}

	c.JSON(http.StatusOK, progresses)
}
