package controllers

import (
	"net/http"
	"strconv"

	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
)

type StudyPlanController struct {
	studyPlanService *services.StudyPlanService
}

func NewStudyPlanController(services *services.Services) *StudyPlanController {
	return &StudyPlanController{
		studyPlanService: services.StudyPlan,
	}
}

// CreateStudyPlan 创建学习计划
func (ctrl *StudyPlanController) CreateStudyPlan(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		QuestionBankID uint `json:"question_bank_id" binding:"required"`
		DailyCount     int  `json:"daily_count" binding:"required,min=1,max=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studyPlan, err := ctrl.studyPlanService.CreateStudyPlan(userID, req.QuestionBankID, req.DailyCount)
	if err != nil {
		if err.Error() == "您已经为该题库创建了学习计划，一个题库只能创建一个学习计划" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建学习计划失败"})
		return
	}

	c.JSON(http.StatusCreated, studyPlan)
}

// GetStudyPlan 获取单个学习计划
func (ctrl *StudyPlanController) GetStudyPlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseUint(planIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	userID := c.GetUint("user_id")

	studyPlan, err := ctrl.studyPlanService.GetStudyPlan(uint(planID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学习计划不存在"})
		return
	}

	c.JSON(http.StatusOK, studyPlan)
}

// UpdateStudyPlan 更新学习计划
func (ctrl *StudyPlanController) UpdateStudyPlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseUint(planIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	userID := c.GetUint("user_id")

	var req struct {
		DailyCount int `json:"daily_count" binding:"required,min=1,max=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = ctrl.studyPlanService.UpdateStudyPlan(uint(planID), userID, req.DailyCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新学习计划失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "学习计划已更新"})
}

// DeleteStudyPlan 删除学习计划
func (ctrl *StudyPlanController) DeleteStudyPlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseUint(planIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = ctrl.studyPlanService.DeleteStudyPlan(uint(planID), userID)
	if err != nil {
		if err.Error() == "学习计划不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "学习计划和对应的学习进度已删除"})
}

// GetAllStudyPlans 获取所有学习计划
func (ctrl *StudyPlanController) GetAllStudyPlans(c *gin.Context) {
	userID := c.GetUint("user_id")

	// 分页参数
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

	studyPlans, total, err := ctrl.studyPlanService.GetAllStudyPlans(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 计算分页信息
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, gin.H{
		"data": studyPlans,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_prev":    page > 1,
			"has_next":    page < totalPages,
		},
	})
}

// GetStudyPlanProgress 获取学习计划进度
func (ctrl *StudyPlanController) GetStudyPlanProgress(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseUint(planIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	userID := c.GetUint("user_id")

	progress, err := ctrl.studyPlanService.GetStudyPlanProgress(userID, uint(planID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetDailyQuestions 获取每日学习题目
func (ctrl *StudyPlanController) GetDailyQuestions(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseUint(planIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	userID := c.GetUint("user_id")

	questions, err := ctrl.studyPlanService.GetDailyQuestions(userID, uint(planID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, questions)
}

// GetRandomMasteredQuestions 获取随机掌握题目
func (ctrl *StudyPlanController) GetRandomMasteredQuestions(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseUint(planIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的学习计划ID"})
		return
	}

	userID := c.GetUint("user_id")

	// 获取指定数量，默认获取学习计划的daily_count数量
	studyPlan, err := ctrl.studyPlanService.GetStudyPlan(uint(planID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "学习计划不存在"})
		return
	}

	questions, err := ctrl.studyPlanService.GetRandomMasteredQuestions(userID, uint(planID), studyPlan.DailyCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, questions)
}

// CompleteQuestion 完成题目
func (ctrl *StudyPlanController) CompleteQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		QuestionID uint   `json:"question_id" binding:"required"`
		ResultType string `json:"result_type" binding:"required"` // "ac" 或 "failed"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ctrl.studyPlanService.CompleteQuestion(userID, req.QuestionID, req.ResultType)
	if err != nil {
		if err.Error() == "无效的结果类型" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 根据结果类型返回不同的消息
	message := "题目完成"
	if req.ResultType == "ac" {
		message = "恭喜独立AC！已自动打卡"
	} else {
		message = "学习记录已保存，继续加油！已自动打卡"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// GetStudyStats 获取学习统计
func (ctrl *StudyPlanController) GetStudyStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	stats, err := ctrl.studyPlanService.GetStudyStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
