package controllers

import (
	"ggcode/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CheckInController struct {
	checkInService *services.CheckInService
}

func NewCheckInController(services *services.Services) *CheckInController {
	return &CheckInController{checkInService: services.CheckIn}
}

// CheckInToday 今日打卡
func (ctrl *CheckInController) CheckInToday(c *gin.Context) {
	userID := c.GetUint("user_id")

	err := ctrl.checkInService.CheckInToday(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打卡失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "打卡成功"})
}

// GetCheckInStats 获取打卡统计
func (ctrl *CheckInController) GetCheckInStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	stats, err := ctrl.checkInService.GetCheckInStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计失败"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetStudyHeatmap 获取学习热力图
func (ctrl *CheckInController) GetStudyHeatmap(c *gin.Context) {
	userID := c.GetUint("user_id")

	heatmap, err := ctrl.checkInService.GetStudyHeatmap(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取热力图失败"})
		return
	}

	c.JSON(http.StatusOK, heatmap)
}
