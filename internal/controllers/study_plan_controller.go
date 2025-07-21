package controllers

import (
	"net/http"
	"strconv"

	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
)

type StudyPlanController struct {
	studyPlanService services.StudyPlanServiceInterface
}

func NewStudyPlanController(studyPlanService services.StudyPlanServiceInterface) *StudyPlanController {
	return &StudyPlanController{
		studyPlanService: studyPlanService,
	}
}

// @Summary      创建学习计划
// @Description  创建新的学习计划
// @Tags         学习计划
// @Accept       json
// @Produce      json
// @Param        data  body     object  true  "学习计划信息"
// @Success      201   {object}  map[string]interface{}  "创建成功"
// @Failure      400   {object}  map[string]string       "参数错误"
// @Failure      409   {object}  map[string]string       "已存在"
// @Failure      500   {object}  map[string]string       "创建失败"
// @Router       /api/study-plan [post]
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

// @Summary      获取单个学习计划
// @Description  获取指定ID的学习计划信息
// @Tags         学习计划
// @Produce      json
// @Param        id   path     int  true  "学习计划ID"
// @Success      200  {object}  map[string]interface{}  "学习计划信息"
// @Failure      400  {object}  map[string]string       "参数错误"
// @Failure      404  {object}  map[string]string       "不存在"
// @Router       /api/study-plan/{id} [get]
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

// @Summary      更新学习计划
// @Description  更新指定ID的学习计划
// @Tags         学习计划
// @Accept       json
// @Produce      json
// @Param        id    path     int     true  "学习计划ID"
// @Param        data  body     object  true  "学习计划信息"
// @Success      200   {object}  map[string]string       "更新成功"
// @Failure      400   {object}  map[string]string       "参数错误"
// @Failure      500   {object}  map[string]string       "更新失败"
// @Router       /api/study-plan/{id} [put]
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

// @Summary      删除学习计划
// @Description  删除指定ID的学习计划
// @Tags         学习计划
// @Produce      json
// @Param        id    path     int  true  "学习计划ID"
// @Success      200   {object}  map[string]string       "删除成功"
// @Failure      400   {object}  map[string]string       "参数错误"
// @Failure      404   {object}  map[string]string       "不存在"
// @Failure      500   {object}  map[string]string       "删除失败"
// @Router       /api/study-plan/{id} [delete]
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

// @Summary      获取所有学习计划
// @Description  分页获取当前用户的所有学习计划
// @Tags         学习计划
// @Produce      json
// @Param        page   query    int  false "页码"
// @Param        limit  query    int  false "每页数量"
// @Success      200    {object}  map[string]interface{}  "学习计划列表"
// @Failure      500    {object}  map[string]string       "获取失败"
// @Router       /api/study-plans [get]
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

// @Summary      获取学习计划进度
// @Description  获取指定学习计划的进度信息
// @Tags         学习计划
// @Produce      json
// @Param        id   path     int  true  "学习计划ID"
// @Success      200  {object}  map[string]interface{}  "进度信息"
// @Failure      400  {object}  map[string]string       "参数错误"
// @Failure      500  {object}  map[string]string       "获取失败"
// @Router       /api/study-plan/{id}/progress [get]
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

// @Summary      获取每日学习任务
// @Description  获取指定学习计划的每日学习题目
// @Tags         学习计划
// @Produce      json
// @Param        id   path     int  true  "学习计划ID"
// @Success      200  {object}  map[string]interface{}  "每日任务"
// @Failure      400  {object}  map[string]string       "参数错误"
// @Failure      500  {object}  map[string]string       "获取失败"
// @Router       /api/study-plan/{id}/daily-questions [get]
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
