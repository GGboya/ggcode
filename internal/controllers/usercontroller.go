package controllers

import (
	"ggcode/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService *services.UserService
}

func NewUserController(services *services.Services) *UserController {
	return &UserController{userService: services.User}
}

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

// Logout 用户退出登录
func (ctrl *UserController) Logout(c *gin.Context) {
	// 清除服务端cookie
	c.SetCookie("token", "", -1, "/", "", false, true)

	// 这里可以实现token黑名单机制（如果需要的话）
	// 目前由于JWT是无状态的，主要依赖客户端清除token和token自然过期

	c.JSON(http.StatusOK, gin.H{
		"message": "退出登录成功",
	})
}

// VerifyToken 校验当前 token 是否有效，并返回基础用户信息
func (ctrl *UserController) VerifyToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
	})
}
