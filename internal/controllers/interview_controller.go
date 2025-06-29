package controllers

import (
	"net/http"
	"strconv"

	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
)

type InterviewController struct {
	interviewService services.InterviewService
	userService      *services.UserService
}

func NewInterviewController(svcs *services.Services) *InterviewController {
	return &InterviewController{
		interviewService: svcs.Interview,
		userService:      svcs.User,
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

	// 判断管理员
	isAdmin, _ := ctrl.userService.IsAdmin(userID)

	// 计算已解锁的岛屿数量（简化：统计至少有一关已解锁）
	unlockedCount := 0
	for _, info := range islands {
		if info.CompletedCount > 0 || info.TotalCount == 0 {
			unlockedCount++
		} else {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":                  islands,
		"is_admin":              isAdmin,
		"unlocked_island_count": unlockedCount,
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

// ----------------- 管理员岛屿 CRUD -----------------
// CreateIsland 创建面试岛 (管理员)
func (ctrl *InterviewController) CreateIsland(c *gin.Context) {
	userID := c.GetUint("user_id")
	// 校验管理员
	isAdmin, err := ctrl.userService.IsAdmin(userID)
	if err != nil || !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员权限不足"})
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	island, err := ctrl.interviewService.CreateIsland(req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": island})
}

// EditIsland 编辑面试岛信息 (管理员)
func (ctrl *InterviewController) EditIsland(c *gin.Context) {
	userID := c.GetUint("user_id")
	isAdmin, err := ctrl.userService.IsAdmin(userID)
	if err != nil || !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员权限不足"})
		return
	}
	islandID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的岛屿ID"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := ctrl.interviewService.UpdateIsland(uint(islandID), req.Name, req.Description); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteIsland 删除面试岛 (管理员)
func (ctrl *InterviewController) DeleteIsland(c *gin.Context) {
	userID := c.GetUint("user_id")
	isAdmin, err := ctrl.userService.IsAdmin(userID)
	if err != nil || !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员权限不足"})
		return
	}
	islandID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的岛屿ID"})
		return
	}
	if err := ctrl.interviewService.DeleteIsland(uint(islandID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ----------------- 管理员关卡/题目/测试用例 CRUD -----------------

// CreateLevel 创建关卡 (管理员)
func (ctrl *InterviewController) CreateLevel(c *gin.Context) {
	if !ctrl.ensureAdmin(c) {
		return
	}
	var req struct {
		IslandID   uint   `json:"island_id" binding:"required"`
		QuestionID uint   `json:"question_id" binding:"required"`
		Name       string `json:"name" binding:"required"`
		Difficulty string `json:"difficulty" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	level, err := ctrl.interviewService.CreateLevel(req.IslandID, req.QuestionID, req.Name, req.Difficulty)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建关卡失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": level})
}

// ensureAdmin 确保是管理员
func (ctrl *InterviewController) ensureAdmin(c *gin.Context) bool {
	userID := c.GetUint("user_id")
	isAdmin, err := ctrl.userService.IsAdmin(userID)
	if err != nil || !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员权限不足"})
		return false
	}
	return true
}

// GetLevelTestCases 获取关卡的所有测试用例 (管理员)
func (ctrl *InterviewController) GetLevelTestCases(c *gin.Context) {
	if !ctrl.ensureAdmin(c) {
		return
	}
	levelID, _ := strconv.ParseUint(c.Param("levelId"), 10, 32)
	cases, err := ctrl.interviewService.GetTestCases(uint(levelID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cases})
}

// AddTestCase 为关卡新增测试用例 (管理员)
func (ctrl *InterviewController) AddTestCase(c *gin.Context) {
	if !ctrl.ensureAdmin(c) {
		return
	}
	levelID, _ := strconv.ParseUint(c.Param("levelId"), 10, 32)

	var req struct {
		Input    string `json:"input"`
		Output   string `json:"output"`
		IsSample bool   `json:"is_sample"`
		Order    int    `json:"order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	tc, err := ctrl.interviewService.CreateTestCase(uint(levelID), req.Input, req.Output, req.IsSample, req.Order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建测试用例失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tc})
}

// DeleteTestCase 删除测试用例 (管理员)
func (ctrl *InterviewController) DeleteTestCase(c *gin.Context) {
	if !ctrl.ensureAdmin(c) {
		return
	}
	caseID, _ := strconv.ParseUint(c.Param("caseId"), 10, 32)
	if err := ctrl.interviewService.DeleteTestCase(uint(caseID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
