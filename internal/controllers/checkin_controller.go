package controllers

import (
	"ggcode/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CheckInController struct {
	checkInService services.CheckInServiceInterface
}

func NewCheckInController(checkInService services.CheckInServiceInterface) *CheckInController {
	return &CheckInController{checkInService: checkInService}
}

// @Summary      今日打卡
// @Description  用户手动打卡，记录当天学习情况
// @Tags         打卡
// @Produce      json
// @Success      200  {object}  map[string]string  "打卡成功"
// @Failure      500  {object}  map[string]string  "打卡失败"
// @Router       /api/checkin [post]
func (ctrl *CheckInController) CheckInToday(c *gin.Context) {
	userID := c.GetUint("user_id")

	err := ctrl.checkInService.CheckInToday(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打卡失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "打卡成功"})
}

// @Summary      获取打卡统计
// @Description  获取用户的打卡天数、连续天数等统计信息
// @Tags         打卡
// @Produce      json
// @Success      200  {object}  models.CheckInStat  "打卡统计信息"
// @Failure      500  {object}  map[string]string   "获取统计失败"
// @Router       /api/checkin-stats [get]
func (ctrl *CheckInController) GetCheckInStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	stats, err := ctrl.checkInService.GetCheckInStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计失败"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// @Summary      获取学习热力图
// @Description  获取用户一年内的学习热力图数据
// @Tags         打卡
// @Produce      json
// @Success      200  {object}  services.HeatmapResponse  "热力图数据"
// @Failure      500  {object}  map[string]string         "获取热力图失败"
// @Router       /api/study-heatmap [get]
func (ctrl *CheckInController) GetStudyHeatmap(c *gin.Context) {
	userID := c.GetUint("user_id")

	heatmap, err := ctrl.checkInService.GetStudyHeatmap(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取热力图失败"})
		return
	}

	c.JSON(http.StatusOK, heatmap)
}
