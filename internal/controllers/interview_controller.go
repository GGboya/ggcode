package controllers

import (
	"net/http"
	"strconv"

	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
)

type InterviewController struct {
	interviewService services.InterviewService
}

func NewInterviewController(interviewService services.InterviewService) *InterviewController {
	return &InterviewController{
		interviewService: interviewService,
	}
}

// GetIslandMap 获取面试岛地图
func (ctrl *InterviewController) GetIslandMap(c *gin.Context) {
	userID := c.GetUint("user_id")

	islands, err := ctrl.interviewService.GetIslandMap(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取面试岛地图失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": islands,
	})
}

// GetLevelDetail 获取关卡详情
func (ctrl *InterviewController) GetLevelDetail(c *gin.Context) {
	userID := c.GetUint("user_id")

	levelIDStr := c.Param("levelId")
	levelID, err := strconv.ParseUint(levelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的关卡ID",
		})
		return
	}

	detail, err := ctrl.interviewService.GetLevelDetail(userID, uint(levelID))
	if err != nil {
		if err.Error() == "关卡未解锁" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "关卡未解锁",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取关卡详情失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": detail,
	})
}

// TestCode 测试代码
func (ctrl *InterviewController) TestCode(c *gin.Context) {
	userID := c.GetUint("user_id")

	levelIDStr := c.Param("levelId")
	levelID, err := strconv.ParseUint(levelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的关卡ID",
		})
		return
	}

	var req struct {
		Code     string `json:"code" binding:"required"`
		Language string `json:"language" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误",
		})
		return
	}

	result, err := ctrl.interviewService.TestCode(userID, uint(levelID), req.Code, req.Language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "代码测试失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// SubmitCode 提交代码
func (ctrl *InterviewController) SubmitCode(c *gin.Context) {
	userID := c.GetUint("user_id")

	levelIDStr := c.Param("levelId")
	levelID, err := strconv.ParseUint(levelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的关卡ID",
		})
		return
	}

	var req struct {
		Code       string `json:"code" binding:"required"`
		Language   string `json:"language" binding:"required"`
		SubmitTime int    `json:"submit_time"` // 提交时的时间（从开始计时到现在的秒数）
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误",
		})
		return
	}

	result, err := ctrl.interviewService.SubmitCode(userID, uint(levelID), req.Code, req.Language, req.SubmitTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "代码提交失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// GetUserProgress 获取用户进度总结
func (ctrl *InterviewController) GetUserProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	progress, err := ctrl.interviewService.GetUserProgress(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取用户进度失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": progress,
	})
}
