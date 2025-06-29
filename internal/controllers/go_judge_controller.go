package controllers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
)

// GoJudgeController go-judge 控制器
type GoJudgeController struct {
	goJudgeService   *services.GoJudgeService
	interviewService services.InterviewService
}

// NewGoJudgeController 创建 go-judge 控制器
func NewGoJudgeController(goJudgeService *services.GoJudgeService, interviewService services.InterviewService) *GoJudgeController {
	return &GoJudgeController{
		goJudgeService:   goJudgeService,
		interviewService: interviewService,
	}
}

// ExecuteCode 执行代码
func (c *GoJudgeController) ExecuteCode(ctx *gin.Context) {
	var req services.GoJudgeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "请求格式错误: " + err.Error(),
		})
		return
	}

	// 设置默认值
	if req.TimeLimit == 0 {
		req.TimeLimit = 5000 // 默认 5 秒
	}
	if req.MemoryLimit == 0 {
		req.MemoryLimit = 128 * 1024 // 默认 128MB
	}

	// 执行代码
	result, err := c.goJudgeService.ExecuteCode(&req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "执行失败: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// HealthCheck 健康检查
func (c *GoJudgeController) HealthCheck(ctx *gin.Context) {
	if err := c.goJudgeService.HealthCheck(); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// GetSupportedLanguages 获取支持的编程语言列表
func (c *GoJudgeController) GetSupportedLanguages(ctx *gin.Context) {
	languages := []map[string]interface{}{
		{
			"id":          "cpp",
			"name":        "C++",
			"description": "C++17 with g++",
			"extensions":  []string{".cpp", ".cc", ".cxx"},
		},
		{
			"id":          "c",
			"name":        "C",
			"description": "C with gcc",
			"extensions":  []string{".c"},
		},
		{
			"id":          "python",
			"name":        "Python",
			"description": "Python 3",
			"extensions":  []string{".py"},
		},
		{
			"id":          "java",
			"name":        "Java",
			"description": "Java with OpenJDK",
			"extensions":  []string{".java"},
		},
		{
			"id":          "go",
			"name":        "Go",
			"description": "Go programming language",
			"extensions":  []string{".go"},
		},
		{
			"id":          "javascript",
			"name":        "JavaScript",
			"description": "JavaScript with Node.js",
			"extensions":  []string{".js"},
		},
	}

	ctx.JSON(http.StatusOK, gin.H{
		"languages": languages,
	})
}

// ExecuteCodeSimple 简化的代码执行接口（通过 query 参数）
func (c *GoJudgeController) ExecuteCodeSimple(ctx *gin.Context) {
	language := ctx.Query("language")
	code := ctx.Query("code")
	input := ctx.Query("input")

	if language == "" || code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少必要参数: language 和 code",
		})
		return
	}

	// 解析可选参数
	var timeLimit int64 = 5000         // 默认 5 秒
	var memoryLimit int64 = 128 * 1024 // 默认 128MB

	if tl := ctx.Query("timeLimit"); tl != "" {
		if parsed, err := strconv.ParseInt(tl, 10, 64); err == nil {
			timeLimit = parsed
		}
	}

	if ml := ctx.Query("memoryLimit"); ml != "" {
		if parsed, err := strconv.ParseInt(ml, 10, 64); err == nil {
			memoryLimit = parsed
		}
	}

	req := &services.GoJudgeRequest{
		Language:    language,
		Code:        code,
		Input:       input,
		TimeLimit:   timeLimit,
		MemoryLimit: memoryLimit,
	}

	// 执行代码
	result, err := c.goJudgeService.ExecuteCode(req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "执行失败: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// TestCode 测试代码（用于关卡模式）
func (c *GoJudgeController) TestCode(ctx *gin.Context) {
	// 获取关卡ID
	levelIDStr := ctx.Param("levelId")
	levelID, err := strconv.ParseUint(levelIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的关卡ID"})
		return
	}

	var req struct {
		Code     string `json:"code" binding:"required"`
		Language string `json:"language" binding:"required"`
		Input    string `json:"input"`    // 可选自定义输入
		Expected string `json:"expected"` // 可选期望输出
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 获取用户ID（验证用户认证）
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	log.Printf("[GoJudge] 测试代码 - 用户ID: %v, 关卡ID: %d, 语言: %s", userID, levelID, req.Language)

	// 使用面试岛服务获取关卡详情和样例测试用例
	levelDetail, err := c.interviewService.GetLevelDetail(userID.(uint), uint(levelID))
	if err != nil {
		log.Printf("[GoJudge] 获取关卡详情失败: %v", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "关卡不存在或未解锁: " + err.Error()})
		return
	}

	if len(levelDetail.SampleCases) == 0 {
		log.Printf("[GoJudge] 关卡 %d 没有样例测试用例", levelID)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "关卡没有配置测试用例"})
		return
	}

	// 判断是否提供自定义输入
	var inputStr, expectedStr string
	if strings.TrimSpace(req.Input) != "" {
		inputStr = req.Input
		expectedStr = req.Expected
	} else {
		// 使用第一个样例测试用例
		testCase := levelDetail.SampleCases[0]
		inputStr = testCase.Input
		expectedStr = testCase.Output
	}

	log.Printf("[GoJudge] 使用测试用例 - 输入: %q, 期望输出: %q", inputStr, expectedStr)

	judgeReq := &services.GoJudgeRequest{
		Language:    req.Language,
		Code:        req.Code,
		Input:       inputStr,
		TimeLimit:   5000,
		MemoryLimit: 128 * 1024,
	}

	result, err := c.goJudgeService.ExecuteCode(judgeReq)
	if err != nil {
		log.Printf("[GoJudge] 执行代码失败: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "测试失败: " + err.Error()})
		return
	}

	// 增强结果处理 - 添加输出比较
	enhancedResult := c.enhanceGoJudgeResult(result, inputStr, expectedStr, false)
	log.Printf("[GoJudge] 测试结果: %+v", enhancedResult)

	ctx.JSON(http.StatusOK, gin.H{
		"message": "测试完成",
		"levelId": levelID,
		"result":  enhancedResult,
	})
}

// SubmitCode 提交代码（用于关卡模式）
func (c *GoJudgeController) SubmitCode(ctx *gin.Context) {
	// 获取关卡ID
	levelIDStr := ctx.Param("levelId")
	levelID, err := strconv.ParseUint(levelIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的关卡ID"})
		return
	}

	var req struct {
		Code       string `json:"code" binding:"required"`
		Language   string `json:"language" binding:"required"`
		SubmitTime int    `json:"submit_time"` // 提交用时（秒）
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 获取用户ID（验证用户认证）
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	log.Printf("[GoJudge] 提交代码 - 用户ID: %v, 关卡ID: %d, 语言: %s, 提交时间: %d秒", userID, levelID, req.Language, req.SubmitTime)

	// 使用面试岛服务获取关卡详情和所有测试用例
	levelDetail, err := c.interviewService.GetLevelDetail(userID.(uint), uint(levelID))
	if err != nil {
		log.Printf("[GoJudge] 获取关卡详情失败: %v", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "关卡不存在或未解锁: " + err.Error()})
		return
	}

	// 对于提交，我们需要运行所有测试用例，但目前先使用样例测试用例
	// TODO: 扩展为运行所有测试用例
	if len(levelDetail.SampleCases) == 0 {
		log.Printf("[GoJudge] 关卡 %d 没有测试用例", levelID)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "关卡没有配置测试用例"})
		return
	}

	// 使用第一个测试用例（样例）
	testCase := levelDetail.SampleCases[0]
	log.Printf("[GoJudge] 使用测试用例 - 输入: %q, 期望输出: %q", testCase.Input, testCase.Output)

	judgeReq := &services.GoJudgeRequest{
		Language:    req.Language,
		Code:        req.Code,
		Input:       testCase.Input,
		TimeLimit:   5000,
		MemoryLimit: 128 * 1024,
	}

	result, err := c.goJudgeService.ExecuteCode(judgeReq)
	if err != nil {
		log.Printf("[GoJudge] 执行代码失败: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "提交失败: " + err.Error()})
		return
	}

	// 增强结果处理 - 添加输出比较
	enhancedResult := c.enhanceGoJudgeResult(result, testCase.Input, testCase.Output, true)
	log.Printf("[GoJudge] 提交结果: %+v", enhancedResult)

	// 如果通过则解锁知识点
	if status, ok := enhancedResult["status"].(string); ok && status == "Accepted" {
		_ = c.interviewService.UnlockTags(userID.(uint), uint(levelID))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":    "提交完成",
		"levelId":    levelID,
		"submitTime": req.SubmitTime,
		"result":     enhancedResult,
	})
}

// GetSystemInfo 获取系统信息
func (c *GoJudgeController) GetSystemInfo(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "go-judge 评测系统",
		"data": gin.H{
			"judge_system":        "go-judge",
			"version":             "1.9.4",
			"supported_languages": []string{"cpp", "c", "python", "java", "go", "javascript"},
			"description":         "基于 go-judge 的高性能代码评测系统",
			"features": []string{
				"高性能代码执行",
				"多语言支持",
				"安全隔离",
				"资源控制",
			},
		},
	})
}

// enhanceGoJudgeResult 增强 go-judge 结果，添加输出比较和得分计算
func (c *GoJudgeController) enhanceGoJudgeResult(result *services.GoJudgeResponse, input, expected string, isSubmit bool) map[string]interface{} {
	// 转换时间单位：纳秒 -> 毫秒
	timeMs := result.Time / 1000000
	// 转换内存单位：字节 -> KB
	memoryKB := result.Memory / 1024

	enhancedResult := map[string]interface{}{
		"status":     result.Status,
		"exitStatus": result.ExitStatus,
		"time":       timeMs,
		"memory":     memoryKB,
		"stdout":     result.Stdout,
		"stderr":     result.Stderr,
		"error":      result.Error,
		"input":      input,
		"expected":   expected,
	}

	// 首先检查是否是编译错误或运行时错误
	if result.Status == "Accepted" && result.ExitStatus == 0 {
		// 程序正常执行，进行输出比较
		actualOutput := strings.TrimSpace(result.Stdout)
		expectedOutput := strings.TrimSpace(expected)

		if actualOutput == expectedOutput {
			enhancedResult["status"] = "Accepted"
			enhancedResult["score"] = 100
			enhancedResult["max_score"] = 100
			enhancedResult["stars"] = 3
		} else {
			enhancedResult["status"] = "Wrong Answer"
			enhancedResult["score"] = 0
			enhancedResult["max_score"] = 100
			enhancedResult["stars"] = 0
		}
	} else {
		// 程序执行有问题，根据具体情况处理
		enhancedResult["max_score"] = 100

		// 检查是否是编译错误（通常 stderr 有内容且没有 stdout）
		if result.Stderr != "" && result.Stdout == "" {
			enhancedResult["status"] = "Compile Error"
			enhancedResult["score"] = 0
			enhancedResult["stars"] = 0
		} else {
			// 根据 go-judge 返回的状态进行分类
			switch result.Status {
			case "Time Limit Exceeded":
				enhancedResult["status"] = "Time Limit Exceeded"
				enhancedResult["score"] = 20
				enhancedResult["stars"] = 1
			case "Memory Limit Exceeded":
				enhancedResult["status"] = "Memory Limit Exceeded"
				enhancedResult["score"] = 10
				enhancedResult["stars"] = 1
			case "Non Zero Exit Status", "Nonzero Exit Status":
				// 检查是否有输出但退出码非零
				if result.Stdout != "" {
					// 有输出，可能是逻辑错误，按答案错误处理
					actualOutput := strings.TrimSpace(result.Stdout)
					expectedOutput := strings.TrimSpace(expected)
					if actualOutput == expectedOutput {
						enhancedResult["status"] = "Accepted"
						enhancedResult["score"] = 100
						enhancedResult["stars"] = 3
					} else {
						enhancedResult["status"] = "Wrong Answer"
						enhancedResult["score"] = 0
						enhancedResult["stars"] = 0
					}
				} else {
					// 无输出，运行时错误
					enhancedResult["status"] = "Runtime Error"
					enhancedResult["score"] = 0
					enhancedResult["stars"] = 0
				}
			default:
				enhancedResult["status"] = "Runtime Error"
				enhancedResult["score"] = 0
				enhancedResult["stars"] = 0
			}
		}
	}

	return enhancedResult
}
