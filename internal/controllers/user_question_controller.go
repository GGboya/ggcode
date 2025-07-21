package controllers

import (
	"ggcode/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserQuestionController struct {
	userQuestionService services.UserQuestionServiceInterface
}

func NewUserQuestionController(userQuestionService services.UserQuestionServiceInterface) *UserQuestionController {
	return &UserQuestionController{userQuestionService: userQuestionService}
}

// CompleteQuestion 用户完成题目学习
func (ctrl *UserQuestionController) CompleteQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID, err := strconv.ParseUint(c.Param("question_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题目ID"})
		return
	}
	var req struct {
		ResultType string `json:"result_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ctrl.userQuestionService.CompleteQuestion(userID, uint(questionID), req.ResultType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "完成学习记录"})
}

// GetStudyStats 获取用户学习统计
func (ctrl *UserQuestionController) GetStudyStats(c *gin.Context) {
	userID := c.GetUint("user_id")
	stats, err := ctrl.userQuestionService.GetStudyStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
