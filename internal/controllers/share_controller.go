package controllers

import (
	"net/http"
	"strconv"

	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
)

type ShareController struct {
	shareService services.ShareServiceInterface
}

func NewShareController(shareService services.ShareServiceInterface) *ShareController {
	return &ShareController{
		shareService: shareService,
	}
}

// @Summary      题库分享
// @Description  将指定题库设置为分享状态
// @Tags         题库分享
// @Produce      json
// @Param        id    path     int  true  "题库ID"
// @Success      200   {object}  map[string]string       "设置为分享成功"
// @Failure      400   {object}  map[string]string       "无效题库ID"
// @Failure      404   {object}  map[string]string       "题库不存在或无权限"
// @Failure      500   {object}  map[string]string       "设置失败"
// @Router       /api/questionbanks/{id}/share [post]
func (ctrl *ShareController) ShareQuestionBank(c *gin.Context) {
	bankIDStr := c.Param("id")
	bankID, err := strconv.ParseUint(bankIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = ctrl.shareService.ShareQuestionBank(uint(bankID), userID)
	if err != nil {
		if err.Error() == "题库不存在或无权限操作" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置共享失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题库已设为共享"})
}

// @Summary      取消题库分享
// @Description  取消指定题库的分享状态
// @Tags         题库分享
// @Produce      json
// @Param        id    path     int  true  "题库ID"
// @Success      200   {object}  map[string]string       "取消分享成功"
// @Failure      400   {object}  map[string]string       "无效题库ID"
// @Failure      404   {object}  map[string]string       "题库不存在或无权限"
// @Failure      500   {object}  map[string]string       "取消失败"
// @Router       /api/questionbanks/{id}/unshare [post]
func (ctrl *ShareController) UnshareQuestionBank(c *gin.Context) {
	bankIDStr := c.Param("id")
	bankID, err := strconv.ParseUint(bankIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = ctrl.shareService.UnshareQuestionBank(uint(bankID), userID)
	if err != nil {
		if err.Error() == "题库不存在或无权限操作" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消共享失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "题库已取消共享"})
}

// @Summary      收藏题库
// @Description  收藏指定题库
// @Tags         题库分享
// @Produce      json
// @Param        id    path     int  true  "题库ID"
// @Success      200   {object}  map[string]string       "收藏成功"
// @Failure      400   {object}  map[string]string       "无效题库ID"
// @Failure      404   {object}  map[string]string       "题库不存在或未分享"
// @Failure      409   {object}  map[string]string       "已收藏"
// @Failure      500   {object}  map[string]string       "收藏失败"
// @Router       /api/questionbanks/{id}/star [post]
func (ctrl *ShareController) StarQuestionBank(c *gin.Context) {
	bankIDStr := c.Param("id")
	bankID, err := strconv.ParseUint(bankIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = ctrl.shareService.StarQuestionBank(uint(bankID), userID)
	if err != nil {
		if err.Error() == "题库不存在或未共享" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "已经Star过这个题库" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Star失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Star成功"})
}

// @Summary      取消收藏题库
// @Description  取消收藏指定题库
// @Tags         题库分享
// @Produce      json
// @Param        id    path     int  true  "题库ID"
// @Success      200   {object}  map[string]string       "取消收藏成功"
// @Failure      400   {object}  map[string]string       "无效题库ID"
// @Failure      404   {object}  map[string]string       "尚未收藏该题库"
// @Failure      500   {object}  map[string]string       "取消收藏失败"
// @Router       /api/questionbanks/{id}/unstar [post]
func (ctrl *ShareController) UnstarQuestionBank(c *gin.Context) {
	bankIDStr := c.Param("id")
	bankID, err := strconv.ParseUint(bankIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	err = ctrl.shareService.UnstarQuestionBank(uint(bankID), userID)
	if err != nil {
		if err.Error() == "尚未Star该题库" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消Star失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "取消Star成功"})
}

// @Summary      Fork题库
// @Description  Fork指定题库到个人空间
// @Tags         题库分享
// @Produce      json
// @Param        id    path     int  true  "题库ID"
// @Success      201   {object}  map[string]interface{}  "Fork成功"
// @Failure      400   {object}  map[string]string       "无效题库ID"
// @Failure      404   {object}  map[string]string       "题库不存在或未分享"
// @Failure      409   {object}  map[string]string       "已Fork"
// @Failure      500   {object}  map[string]string       "Fork失败"
// @Router       /api/questionbanks/{id}/fork [post]
func (ctrl *ShareController) ForkQuestionBank(c *gin.Context) {
	bankIDStr := c.Param("id")
	bankID, err := strconv.ParseUint(bankIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的题库ID"})
		return
	}

	userID := c.GetUint("user_id")

	forkedBank, err := ctrl.shareService.ForkQuestionBank(uint(bankID), userID)
	if err != nil {
		if err.Error() == "题库不存在或未共享" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "已经Fork过这个题库" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fork失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Fork成功",
		"forked_bank": forkedBank,
	})
}

// @Summary      获取用户收藏的题库
// @Description  分页获取当前用户收藏的题库列表
// @Tags         题库分享
// @Produce      json
// @Param        page   query    int  false "页码"
// @Param        limit  query    int  false "每页数量"
// @Success      200    {object}  map[string]interface{}  "收藏题库列表"
// @Failure      500    {object}  map[string]string       "获取失败"
// @Router       /api/starred-questionbanks [get]
func (ctrl *ShareController) GetUserStarredBanks(c *gin.Context) {
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

	starredBanks, total, err := ctrl.shareService.GetUserStarredBanks(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取Star题库失败"})
		return
	}

	// 计算分页信息
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, gin.H{
		"data": starredBanks,
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
