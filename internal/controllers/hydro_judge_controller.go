package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
)

// HydroJudgeController Hydro评测控制器
type HydroJudgeController struct {
	judgeService services.HydroJudgeService
}

func NewHydroJudgeController(judgeService services.HydroJudgeService) *HydroJudgeController {
	return &HydroJudgeController{
		judgeService: judgeService,
	}
}

// SubmitCode 提交代码进行Hydro风格评测
func (ctrl *HydroJudgeController) SubmitCode(c *gin.Context) {
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
		Priority   int    `json:"priority"`    // 评测优先级，可选
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 生成唯一的提交ID (使用纳秒时间戳的后8位 + 用户ID，确保不超过uint32范围)
	nano := time.Now().UnixNano()
	submissionID := uint(nano%1000000000) + userID // 取纳秒时间戳的后9位，再加用户ID

	// 创建评测提交
	submission := &services.JudgeSubmission{
		ID:         submissionID,
		UserID:     userID,
		LevelID:    uint(levelID),
		Code:       req.Code,
		Language:   req.Language,
		SubmitTime: req.SubmitTime,
		Priority:   req.Priority,
	}

	// 提交评测
	result, err := ctrl.judgeService.SubmitForJudge(submission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "提交评测失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":          result,
		"message":       "评测任务已提交，请稍后查询结果",
		"submission_id": submissionID, // 返回提交ID供轮询使用
	})
}

// GetJudgeResult 获取评测结果
func (ctrl *HydroJudgeController) GetJudgeResult(c *gin.Context) {
	submissionIDStr := c.Param("submissionId")

	// 添加调试日志
	fmt.Printf("DEBUG: 获取评测结果 - 提交ID字符串: '%s'\n", submissionIDStr)

	submissionID, err := strconv.ParseUint(submissionIDStr, 10, 32) // 32位足够了
	if err != nil {
		fmt.Printf("DEBUG: 提交ID解析失败: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的提交ID: " + submissionIDStr,
		})
		return
	}

	fmt.Printf("DEBUG: 解析后的提交ID: %d\n", submissionID)

	result, err := ctrl.judgeService.GetJudgeResult(uint(submissionID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "未找到评测结果",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// GetQueueStatus 获取评测队列状态
func (ctrl *HydroJudgeController) GetQueueStatus(c *gin.Context) {
	status := ctrl.judgeService.GetQueueStatus()
	c.JSON(http.StatusOK, gin.H{
		"data": status,
	})
}

// GetJudgeSystemInfo 获取评测系统信息
func (ctrl *HydroJudgeController) GetJudgeSystemInfo(c *gin.Context) {
	// 获取系统信息
	systemInfo := map[string]interface{}{
		"judge_version": "GGCode Hydro Judge v1.0",
		"supported_languages": []map[string]string{
			{"name": "C++17", "key": "cpp", "version": "GCC 9.4.0"},
			{"name": "Python3", "key": "python", "version": "Python 3.9.2"},
			{"name": "Java", "key": "java", "version": "OpenJDK 11.0.11"},
		},
		"features": []string{
			"多语言支持",
			"沙箱执行",
			"资源限制",
			"详细评测报告",
			"子任务支持",
			"队列管理",
		},
		"queue_status": ctrl.judgeService.GetQueueStatus(),
	}

	c.JSON(http.StatusOK, gin.H{
		"data": systemInfo,
	})
}

// TestCode 测试代码（仅运行样例，快速反馈）
func (ctrl *HydroJudgeController) TestCode(c *gin.Context) {
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
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 生成唯一的测试提交ID
	nano := time.Now().UnixNano()
	submissionID := uint(nano%1000000000) + userID + 100000 // +100000 区分测试和正式提交

	// 创建测试提交（优先级设为最高）
	submission := &services.JudgeSubmission{
		ID:         submissionID,
		UserID:     userID,
		LevelID:    uint(levelID),
		Code:       req.Code,
		Language:   req.Language,
		SubmitTime: 0,
		Priority:   100, // 测试代码优先级最高
	}

	// 提交评测
	result, err := ctrl.judgeService.SubmitForJudge(submission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "代码测试失败",
			"details": err.Error(),
		})
		return
	}

	// 返回提交结果，前端可以通过submission_id轮询获取最终结果
	c.JSON(http.StatusOK, gin.H{
		"data":          result,
		"message":       "代码测试已提交",
		"submission_id": submissionID, // 返回提交ID供轮询使用
	})
}
