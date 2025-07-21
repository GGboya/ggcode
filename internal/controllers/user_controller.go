package controllers

import (
	"ggcode/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService services.UserServiceInterface
}

func NewUserController(userService services.UserServiceInterface) *UserController {
	return &UserController{userService: userService}
}

// @Summary      用户登录
// @Description  用户登录，返回 token
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        data  body     object  true  "登录参数"
// @Success      200   {object}  map[string]interface{}  "登录成功"
// @Failure      400   {object}  map[string]string       "参数错误"
// @Failure      401   {object}  map[string]string       "认证失败"
// @Router       /api/login [post]
func (ctrl *UserController) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := ctrl.userService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 设置 Cookie，便于页面跳转自动携带认证
	c.SetCookie("token", token, 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// @Summary      用户注册
// @Description  用户注册，返回 token
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        data  body     object  true  "注册参数"
// @Success      201   {object}  map[string]interface{}  "注册成功"
// @Failure      400   {object}  map[string]string       "参数错误"
// @Failure      500   {object}  map[string]string       "注册失败"
// @Router       /api/register [post]
func (ctrl *UserController) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转发给service层

	user, token, err := ctrl.userService.Register(req.Username, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 设置 Cookie
	c.SetCookie("token", token, 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user":  user,
	})
}

// @Summary      用户登出
// @Description  清除登录状态
// @Tags         用户
// @Produce      json
// @Success      200   {object}  map[string]string  "登出成功"
// @Router       /api/logout [post]
func (ctrl *UserController) Logout(c *gin.Context) {
	// 清除服务端cookie
	c.SetCookie("token", "", -1, "/", "", false, true)

	// 这里可以实现token黑名单机制（如果需要的话）
	// 目前由于JWT是无状态的，主要依赖客户端清除token和token自然过期

	c.JSON(http.StatusOK, gin.H{
		"message": "退出登录成功",
	})
}

// @Summary      校验 token
// @Description  校验当前 token 是否有效，返回用户信息
// @Tags         用户
// @Produce      json
// @Success      200   {object}  map[string]interface{}  "token 有效"
// @Router       /api/verify-token [get]
func (ctrl *UserController) VerifyToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
	})
}
