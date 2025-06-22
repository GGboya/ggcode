package controllers

import (
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
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: 实现创建题库的服务层方法
	c.JSON(http.StatusNotImplemented, gin.H{"error": "创建题库功能待实现"})
}
