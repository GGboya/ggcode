package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// getJWTSecret 从环境变量获取JWT密钥，如果没有则使用默认值
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key" // 默认密钥，生产环境应该设置环境变量
	}
	return []byte(secret)
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先尝试从Authorization header获取token
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

		if tokenString == "" {
			// 对于API请求，返回JSON错误
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
				c.Abort()
				return
			}
			// 对于页面请求，重定向到登录页
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			// 对于API请求，返回JSON错误
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}
			// 对于页面请求，重定向到登录页
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 如果是页面请求且token有效，设置cookie以便后续使用
		if !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.SetCookie("token", tokenString, 3600*24*7, "/", "", false, true) // 7天有效期
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func GenerateToken(userID uint, username string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "ggcode",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}
