package controllers

import (
	"ggcode/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QuestionController struct {
	questionService services.QuestionServiceInterface
}

func NewQuestionController(questionService services.QuestionServiceInterface) *QuestionController {
	return &QuestionController{questionService: questionService}
}

// @Summary      获取题库下的题目列表
// @Description  分页获取指定题库的题目列表
// @Tags         题目
// @Produce      json
// @Param        id     path     int  true  "题库ID"
// @Param        page   query    int  false "页码"
// @Param        limit  query    int  false "每页数量"
// @Success      200    {object}  map[string]interface{}  "题目列表"
// @Failure      400    {object}  map[string]string       "参数错误"
// @Failure      500    {object}  map[string]string       "获取失败"
// @Router       /api/questionbanks/{id}/questions [get]
func (ctrl *QuestionController) GetQuestions(c *gin.Context) {
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	// 解析分页参数
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

	// 调用服务层获取题目列表
	response, err := ctrl.questionService.GetQuestions(uint(bankID), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题目失败"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      获取所有题目（用于搜索）
// @Description  获取所有题目列表
// @Tags         题目
// @Produce      json
// @Success      200    {object}  map[string]interface{}  "题目列表"
// @Failure      500    {object}  map[string]string       "获取失败"
// @Router       /api/questions [get]
func (ctrl *QuestionController) GetAllQuestions(c *gin.Context) {
	questions, err := ctrl.questionService.GetAllQuestions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题目列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": questions})
}

// @Summary      在题库中创建题目
// @Description  在指定题库中创建新题目
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id   path     int     true  "题库ID"
// @Param        data body     object  true  "题目信息"
// @Success      201  {object}  map[string]interface{}  "创建成功"
// @Failure      400  {object}  map[string]string       "参数错误"
// @Failure      404  {object}  map[string]string       "题库不存在"
// @Failure      500  {object}  map[string]string       "创建失败"
// @Router       /api/questionbanks/{id}/questions [post]
func (ctrl *QuestionController) CreateQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	var req struct {
		Title      string  `json:"title" binding:"required"`
		URL        string  `json:"url" binding:"required"`
		Difficulty string  `json:"difficulty" binding:"required"`
		Score      float64 `json:"score"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用服务层创建题目
	question, err := ctrl.questionService.CreateQuestion(userID, uint(bankID), req.Title, req.URL, req.Difficulty, req.Score)
	if err != nil {
		switch err.Error() {
		case "题库不存在或无权限添加题目":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, question)
}

// @Summary      获取单个题目
// @Description  获取指定ID的题目信息
// @Tags         题目
// @Produce      json
// @Param        id   path     int  true  "题目ID"
// @Success      200  {object}  map[string]interface{}  "题目信息"
// @Failure      400  {object}  map[string]string       "参数错误"
// @Failure      404  {object}  map[string]string       "题目不存在"
// @Failure      500  {object}  map[string]string       "获取失败"
// @Router       /api/questions/{id} [get]
func (ctrl *QuestionController) GetQuestion(c *gin.Context) {
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题目ID"})
		return
	}

	// 调用服务层获取题目
	question, err := ctrl.questionService.GetQuestion(uint(questionID))
	if err != nil {
		switch err.Error() {
		case "题目不存在":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, question)
}

// @Summary      更新题目信息
// @Description  更新指定ID的题目信息
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id   path     int     true  "题目ID"
// @Param        data body     object  true  "题目信息"
// @Success      200  {object}  map[string]interface{}  "更新成功"
// @Failure      400  {object}  map[string]string       "参数错误"
// @Failure      403  {object}  map[string]string       "无权限"
// @Failure      404  {object}  map[string]string       "题目不存在"
// @Failure      500  {object}  map[string]string       "更新失败"
// @Router       /api/questions/{id} [put]
func (ctrl *QuestionController) UpdateQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题目ID"})
		return
	}

	var req struct {
		Title          string `json:"title" binding:"required"`
		URL            string `json:"url" binding:"required"`
		Difficulty     string `json:"difficulty" binding:"required"`
		QuestionBankID uint   `json:"question_bank_id"` // 添加题库ID字段
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新题目信息
	question, err := ctrl.questionService.UpdateQuestion(userID, uint(questionID), req.QuestionBankID, req.Title, req.URL, req.Difficulty)

	if err != nil {
		switch err.Error() {
		case "题目不存在":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "无权限修改此题目":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, question)
}

// @Summary      删除题目
// @Description  删除指定ID的题目
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id   path     int     true  "题目ID"
// @Param        data body     object  true  "题库ID"
// @Success      200  {object}  map[string]string       "删除成功"
// @Failure      400  {object}  map[string]string       "参数错误"
// @Failure      403  {object}  map[string]string       "无权限"
// @Failure      404  {object}  map[string]string       "题目不存在"
// @Failure      500  {object}  map[string]string       "删除失败"
// @Router       /api/questions/{id} [delete]
func (ctrl *QuestionController) DeleteQuestion(c *gin.Context) {
	userID := c.GetUint("user_id")
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题目ID"})
		return
	}

	// 从请求体中获取题库ID
	var req struct {
		QuestionBankID uint `json:"question_bank_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供题库ID"})
		return
	}

	// 调用服务层删除题目
	err = ctrl.questionService.DeleteQuestion(userID, uint(questionID), req.QuestionBankID)
	if err != nil {
		switch err.Error() {
		case "题目不存在":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "无权限删除此题目":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case "题库不存在或无权限删除题目":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题目删除成功"})
}
