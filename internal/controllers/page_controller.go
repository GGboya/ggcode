package controllers

import (
	"net/http"
	"os"
	"strings"

	"ggcode/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type PageController struct {
	// 可以添加需要的服务依赖
}

func NewPageController() *PageController {
	return &PageController{}
}

// isValidToken 验证token的有效性
func (ctrl *PageController) isValidToken(tokenString string) bool {
	// getJWTSecret 从环境变量获取JWT密钥，如果没有则使用默认值
	getJWTSecret := func() []byte {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "your-secret-key" // 默认密钥，生产环境应该设置环境变量
		}
		return []byte(secret)
	}

	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	return err == nil && token.Valid
}

// HomePage 首页处理器
func (ctrl *PageController) HomePage(c *gin.Context) {
	// 检查用户是否已认证
	// 先尝试从Authorization header获取token
	authHeader := c.GetHeader("Authorization")
	var tokenString string

	if authHeader != "" {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		// 如果没有Authorization header，尝试从cookie获取token
		cookie, err := c.Cookie("token")
		if err == nil {
			tokenString = cookie
		}
	}

	// 如果有token，验证其有效性
	if tokenString != "" {
		// 验证token有效性
		if ctrl.isValidToken(tokenString) {
			// token有效，跳转到仪表盘
			c.Redirect(http.StatusFound, "/dashboard")
			return
		}
	}

	// 没有有效token，跳转到登录页面
	c.Redirect(http.StatusFound, "/login")
}

// LoginPage 登录页面
func (ctrl *PageController) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":    "登录",
		"pageType": "login",
	})
}

// RegisterPage 注册页面
func (ctrl *PageController) RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"title":    "注册",
		"pageType": "register",
	})
}

// Dashboard 仪表盘页面
func (ctrl *PageController) Dashboard(c *gin.Context) {
	userID := c.GetUint("user_id")
	username := c.GetString("username")

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":    "仪表板",
		"userID":   userID,
		"username": username,
		"pageType": "dashboard",
	})
}

// QuestionBanksPage 题库管理页面
func (ctrl *PageController) QuestionBanksPage(c *gin.Context) {
	username := c.GetString("username")
	userID := c.GetUint("user_id")
	c.HTML(http.StatusOK, "questionbanks.html", gin.H{
		"title":    "题库管理",
		"username": username,
		"userID":   userID,
	})
}

// StudyPlansPage 学习计划页面
func (ctrl *PageController) StudyPlansPage(c *gin.Context) {
	username := c.GetString("username")
	userID := c.GetUint("user_id")
	c.HTML(http.StatusOK, "study-plans.html", gin.H{
		"title":    "学习计划管理",
		"username": username,
		"userID":   userID,
		"pageType": "study-plans",
	})
}

// StudyPage 学习页面
func (ctrl *PageController) StudyPage(c *gin.Context) {
	username := c.GetString("username")
	userID := c.GetUint("user_id")
	c.HTML(http.StatusOK, "study.html", gin.H{
		"title":    "开始学习",
		"username": username,
		"userID":   userID,
		"pageType": "study",
	})
}

// InterviewIslandPage 面试岛地图页面
func (ctrl *PageController) InterviewIslandPage(c *gin.Context) {
	username := c.GetString("username")
	userID := c.GetUint("user_id")
	c.HTML(http.StatusOK, "interview-island.html", gin.H{
		"title":    "面试岛",
		"username": username,
		"userID":   userID,
		"pageType": "interview-island",
	})
}

// LevelPage 关卡详情页面
func (ctrl *PageController) LevelPage(c *gin.Context) {
	username := c.GetString("username")
	userID := c.GetUint("user_id")
	levelID := c.Param("levelId")

	c.HTML(http.StatusOK, "base.html", gin.H{
		"title":    "关卡挑战",
		"username": username,
		"userID":   userID,
		"levelID":  levelID,
		"pageType": "level",
	})
}
