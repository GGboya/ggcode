package controllers

import (
	"ggcode/internal/models"
	"ggcode/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QuestionController struct {
	questionService *services.QuestionService
}

func NewQuestionController(services *services.Services) *QuestionController {
	return &QuestionController{questionService: services.Question}
}

// GetQuestions 获取题库下的题目列表
func (ctrl *QuestionController) GetQuestions(c *gin.Context) {
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	// 解析分页参数
	page := 1
	limit := 20
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

	// 调用服务层获取题目列表
	response, err := ctrl.questionService.GetQuestions(uint(bankID), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题目失败"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetAllQuestions 获取所有题目，用于搜索
func (ctrl *QuestionController) GetAllQuestions(c *gin.Context) {
	questions, err := ctrl.questionService.GetAllQuestions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题目列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": questions})
}

// CreateQuestion 在题库中创建题目
func (ctrl *QuestionController) CreateQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
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

	// 调用服务层创建题目
	question, err := ctrl.questionService.CreateQuestion(userID, uint(bankID), req.Title, req.LeetcodeURL, req.Difficulty)
	if err != nil {
		switch err.Error() {
		case "题库不存在或无权限添加题目":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, question)
}

// GetQuestion 获取单个题目
func (ctrl *QuestionController) GetQuestion(c *gin.Context) {
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题目ID"})
		return
	}

	// 调用服务层获取题目
	question, err := ctrl.questionService.GetQuestion(uint(questionID))
	if err != nil {
		switch err.Error() {
		case "题目不存在":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, question)
}

// UpdateQuestion 更新题目信息
func (ctrl *QuestionController) UpdateQuestion(c *gin.Context) {
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
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var question *models.Question
	// 如果包含描述字段，使用包含描述的更新方法
	if req.Description != "" {
		question, err = ctrl.questionService.UpdateQuestionWithDescription(userID, uint(questionID), req.Title, req.LeetcodeURL, req.Difficulty, req.Description)
	} else {
		question, err = ctrl.questionService.UpdateQuestion(userID, uint(questionID), req.Title, req.LeetcodeURL, req.Difficulty)
	}

	if err != nil {
		switch err.Error() {
		case "题目不存在":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "无权限修改此题目":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, question)
}

// DeleteQuestion 删除题目
func (ctrl *QuestionController) DeleteQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题目ID"})
		return
	}

	// 调用服务层删除题目
	err = ctrl.questionService.DeleteQuestion(userID, uint(questionID))
	if err != nil {
		switch err.Error() {
		case "题目不存在":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "无权限删除此题目":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题目删除成功"})
}
