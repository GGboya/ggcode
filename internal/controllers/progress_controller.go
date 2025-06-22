package controllers

import (
	"net/http"
	"strconv"

	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
)

type ProgressController struct {
	progressService *services.ProgressService
}

func NewProgressController(services *services.Services) *ProgressController {
	return &ProgressController{
		progressService: services.Progress,
	}
}

// GetQuestionBankProgress 获取特定题库的学习进度
func (ctrl *ProgressController) GetQuestionBankProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	bankIDStr := c.Param("id")
	bankID, err := strconv.ParseUint(bankIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	progress, err := ctrl.progressService.GetQuestionBankProgress(userID, uint(bankID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetAllQuestionBanksProgress 获取所有题库的学习进度
func (ctrl *ProgressController) GetAllQuestionBanksProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	progressList, err := ctrl.progressService.GetAllQuestionBanksProgress(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progressList)
}

// CheckInToday 今日打卡
func (ctrl *ProgressController) CheckInToday(c *gin.Context) {
	userID := c.GetUint("user_id")

	err := ctrl.progressService.CheckInToday(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "打卡成功！"})
}

// GetCheckInStats 获取打卡统计
func (ctrl *ProgressController) GetCheckInStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	stats, err := ctrl.progressService.GetCheckInStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
