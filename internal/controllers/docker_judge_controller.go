package controllers

import (
	"fmt"
	"ggcode/internal/models"
	"ggcode/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DockerJudgeController struct {
	interviewService services.InterviewService
	dockerService    *services.DockerJudgeService
	containerPool    *services.SimpleContainerPool
}

func NewDockerJudgeController(interviewService services.InterviewService, dockerService *services.DockerJudgeService, containerPool *services.SimpleContainerPool) *DockerJudgeController {
	return &DockerJudgeController{
		interviewService: interviewService,
		dockerService:    dockerService,
		containerPool:    containerPool,
	}
}

// TestCode 使用容器池测试代码
func (ctrl *DockerJudgeController) TestCode(c *gin.Context) {
	levelIdStr := c.Param("levelId")
	levelId, err := strconv.Atoi(levelIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的关卡ID"})
		return
	}

	var req struct {
		Code     string `json:"code" binding:"required"`
		Language string `json:"language" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 获取关卡信息
	level, err := ctrl.interviewService.GetLevelDetail(userID.(uint), uint(levelId))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "关卡不存在"})
		return
	}

	// 使用容器池进行评测
	result, err := ctrl.runContainerPoolJudge(req.Code, req.Language, &level.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评测失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "测试完成",
		"data":    result,
	})
}

// SubmitCode 使用容器池提交代码
func (ctrl *DockerJudgeController) SubmitCode(c *gin.Context) {
	levelIdStr := c.Param("levelId")
	levelId, err := strconv.Atoi(levelIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的关卡ID"})
		return
	}

	var req struct {
		Code       string `json:"code" binding:"required"`
		Language   string `json:"language" binding:"required"`
		SubmitTime int    `json:"submit_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 获取关卡信息
	level, err := ctrl.interviewService.GetLevelDetail(userID.(uint), uint(levelId))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "关卡不存在"})
		return
	}

	// 使用容器池进行评测
	result, err := ctrl.runContainerPoolJudge(req.Code, req.Language, &level.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评测失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "提交完成",
		"data":    result,
	})
}

// CustomTest 自定义测试用例
func (ctrl *DockerJudgeController) CustomTest(c *gin.Context) {
	var req struct {
		Code     string `json:"code" binding:"required"`
		Language string `json:"language" binding:"required"`
		Input    string `json:"input"`
		Expected string `json:"expected"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 使用自定义测试用例进行评测
	dockerReq := &services.DockerJudgeRequest{
		Code:        req.Code,
		Language:    req.Language,
		Input:       req.Input,
		Expected:    req.Expected,
		TimeLimit:   5,   // 5秒超时
		MemoryLimit: 128, // 128MB内存限制
	}

	// 运行评测
	dockerResult, err := ctrl.dockerService.RunJudge(dockerReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评测失败: " + err.Error()})
		return
	}

	// 返回详细结果
	result := &DockerJudgeResult{
		Status:         dockerResult.Status,
		Score:          0,
		MaxScore:       100,
		TimeUsed:       dockerResult.TimeUsed,
		MemoryUsed:     dockerResult.MemoryUsed,
		CompileMessage: dockerResult.CompileMessage,
		Error:          dockerResult.RuntimeMessage,
		TestCases: []TestCaseDetail{
			{
				Input:    req.Input,
				Expected: req.Expected,
				Output:   dockerResult.ActualOutput,
				Status:   dockerResult.Status,
				IsSample: false,
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "自定义测试完成",
		"data":    result,
	})
}

// GetSystemInfo 获取容器池系统信息
func (ctrl *DockerJudgeController) GetSystemInfo(c *gin.Context) {
	stats := ctrl.containerPool.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"message": "系统信息",
		"data": gin.H{
			"judge_system":        "Docker容器池",
			"version":             "1.0.0",
			"container_stats":     stats,
			"supported_languages": []string{"cpp", "python", "java", "go"},
		},
	})
}

// runContainerPoolJudge 使用容器池运行评测
func (ctrl *DockerJudgeController) runContainerPoolJudge(code, language string, level *models.InterviewLevel) (*DockerJudgeResult, error) {
	// 获取关卡的样例测试用例（简化版本，只用样例）
	// 通过InterviewService的TestCode方法来获取样例测试用例
	testResult, err := ctrl.interviewService.TestCode(0, level.ID, code, language)
	if err != nil {
		return nil, fmt.Errorf("无法获取测试用例: %v", err)
	}

	if len(testResult.TestCases) == 0 {
		return nil, fmt.Errorf("关卡 %d 没有配置测试用例", level.ID)
	}

	// 使用第一个测试用例
	firstTestCase := testResult.TestCases[0]

	dockerReq := &services.DockerJudgeRequest{
		Code:        code,
		Language:    language,
		Input:       firstTestCase.Input,
		Expected:    firstTestCase.Expected,
		TimeLimit:   level.TimeLimit / 60, // 转换为秒，如果TimeLimit是秒则不需要除法
		MemoryLimit: 128,                  // 128MB内存限制
	}

	// 运行评测
	dockerResult, err := ctrl.dockerService.RunJudge(dockerReq)
	if err != nil {
		return nil, err
	}

	// 转换为前端期望的格式
	result := &DockerJudgeResult{
		Status:         dockerResult.Status,
		Score:          0,
		MaxScore:       100,
		TimeUsed:       dockerResult.TimeUsed,
		MemoryUsed:     dockerResult.MemoryUsed,
		CompileMessage: dockerResult.CompileMessage,
		Error:          dockerResult.RuntimeMessage,
		TestCases: []TestCaseDetail{
			{
				Input:    firstTestCase.Input,
				Expected: firstTestCase.Expected,
				Output:   dockerResult.ActualOutput,
				Status:   dockerResult.Status,
				IsSample: true,
			},
		},
	}

	// 计算得分
	if dockerResult.Status == "AC" {
		result.Score = 100
		result.Stars = 3 // 满分3星
	} else if dockerResult.Status == "WA" {
		result.Score = 0
	} else if dockerResult.Status == "TLE" {
		result.Score = 20
	} else if dockerResult.Status == "MLE" {
		result.Score = 10
	}

	return result, nil
}

// DockerJudgeResult 前端期望的评测结果格式
type DockerJudgeResult struct {
	Status         string           `json:"status"`
	Score          int              `json:"score"`
	MaxScore       int              `json:"max_score"`
	Stars          int              `json:"stars"`
	TimeUsed       int              `json:"time_used"`
	MemoryUsed     int              `json:"memory_used"`
	CompileMessage string           `json:"compile_message,omitempty"`
	Error          string           `json:"error,omitempty"`
	TestCases      []TestCaseDetail `json:"test_cases,omitempty"`
}

// TestCaseDetail 测试用例详情
type TestCaseDetail struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
	Output   string `json:"output"`
	Status   string `json:"status"`
	IsSample bool   `json:"is_sample"`
}
