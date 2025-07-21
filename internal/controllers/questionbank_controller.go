package controllers

import (
	"ggcode/internal/repositories"
	"ggcode/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QuestionBankController struct {
	questionBankService services.QuestionBankServiceInterface
}

func NewQuestionBankController(questionBankService services.QuestionBankServiceInterface) *QuestionBankController {
	return &QuestionBankController{questionBankService: questionBankService}
}

// @Summary      获取题库列表
// @Description  分页获取题库列表，可按类型和排序方式筛选
// @Tags         题库
// @Produce      json
// @Param        type    query    string  false "题库类型(official/shared/personal)"
// @Param        sort    query    string  false "排序方式(star_count/fork_count/created_at)"
// @Param        page    query    int     false "页码"
// @Param        limit   query    int     false "每页数量"
// @Success      200     {object}  map[string]interface{}  "题库列表"
// @Failure      500     {object}  map[string]string       "获取失败"
// @Router       /api/questionbanks [get]
func (ctrl *QuestionBankController) GetQuestionBanks(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankType := c.Query("type") // "official", "shared", "personal"
	sortBy := c.Query("sort")   // "star_count", "fork_count", "created_at"

	// 解析分页参数
	page := 1
	limit := 10
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

	// 调用服务层获取题库列表
	response, err := ctrl.questionBankService.GetQuestionBanks(userID, bankType, sortBy, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题库失败"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      创建题库
// @Description  创建新的题库，可选导入题目
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        data  body     object  true  "题库信息"
// @Success      201   {object}  map[string]interface{}  "创建成功"
// @Failure      400   {object}  map[string]string       "参数错误"
// @Failure      500   {object}  map[string]string       "创建失败"
// @Router       /api/questionbanks [post]
func (ctrl *QuestionBankController) CreateQuestionBank(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Source      string `json:"source"`
		MinScore    *int   `json:"min_score"`
		MaxScore    *int   `json:"max_score"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	minScore := 0
	if req.MinScore != nil {
		minScore = *req.MinScore
	}
	maxScore := 0
	if req.MaxScore != nil {
		maxScore = *req.MaxScore
	}

	questionBank, err := ctrl.questionBankService.CreateQuestionBankWithImport(req.Name, req.Description, userID, req.Source, minScore, maxScore)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建题库失败"})
		return
	}

	c.JSON(http.StatusCreated, questionBank)
}

// @Summary      更新题库
// @Description  更新指定ID的题库信息
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        id    path     int     true  "题库ID"
// @Param        data  body     object  true  "题库信息"
// @Success      200   {object}  map[string]string       "更新成功"
// @Failure      400   {object}  map[string]string       "参数错误"
// @Failure      500   {object}  map[string]string       "更新失败"
// @Router       /api/questionbanks/{id} [put]
func (ctrl *QuestionBankController) UpdateQuestionBank(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = ctrl.questionBankService.UpdateQuestionBank(uint(bankID), userID, repositories.QuestionBankUpdateData{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新题库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题库更新成功"})
}

// @Summary      删除题库
// @Description  删除指定ID的题库
// @Tags         题库
// @Produce      json
// @Param        id    path     int  true  "题库ID"
// @Success      200   {object}  map[string]string       "删除成功"
// @Failure      400   {object}  map[string]string       "参数错误"
// @Failure      404   {object}  map[string]string       "题库不存在或无权限"
// @Failure      409   {object}  map[string]string       "题库被学习计划使用"
// @Failure      500   {object}  map[string]string       "删除失败"
// @Router       /api/questionbanks/{id} [delete]
func (ctrl *QuestionBankController) DeleteQuestionBank(c *gin.Context) {
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = ctrl.questionBankService.DeleteQuestionBank(uint(bankID), userID)
	if err != nil {
		// 根据错误类型返回不同的HTTP状态码
		switch err.Error() {
		case "题库不存在或无权限删除":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "该题库正在被学习计划使用，无法删除":
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题库删除成功"})
}

// @Summary      获取特定题库的学习进度
// @Description  获取指定题库的学习进度信息
// @Tags         题库
// @Produce      json
// @Param        id    path     int  true  "题库ID"
// @Success      200   {object}  map[string]interface{}  "进度信息"
// @Failure      400   {object}  map[string]string       "参数错误"
// @Failure      500   {object}  map[string]string       "获取失败"
// @Router       /api/questionbanks/{id}/progress [get]
func (ctrl *QuestionBankController) GetQuestionBankProgress(c *gin.Context) {
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	progress, err := ctrl.questionBankService.GetQuestionBankProgress(userID, uint(bankID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取进度失败"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// @Summary      获取所有题库的学习进度
// @Description  获取当前用户所有题库的学习进度信息
// @Tags         题库
// @Produce      json
// @Success      200   {object}  map[string]interface{}  "进度信息"
// @Failure      500   {object}  map[string]string       "获取失败"
// @Router       /api/questionbanks-progress [get]
func (ctrl *QuestionBankController) GetAllQuestionBanksProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	progresses, err := ctrl.questionBankService.GetAllQuestionBanksProgress(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取进度失败"})
		return
	}

	c.JSON(http.StatusOK, progresses)
}
