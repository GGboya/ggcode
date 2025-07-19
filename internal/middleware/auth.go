package middleware

import (
	"ggcode/internal/config"
	"ggcode/internal/pkg/errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthMiddleware 认证中间件
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
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
			// 清除可能存在的无效Token Cookie，避免前端陷入跳转循环
			c.SetCookie("token", "", -1, "/", "", false, true)

			// 对于API请求，返回JSON错误
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				appErr := errors.NewWithDetails(
					errors.ErrTokenMissing,
					"缺少认证令牌",
					"请先登录",
				)
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   appErr,
					"message": appErr.Message,
				})
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
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			// 清除无效的Token Cookie
			c.SetCookie("token", "", -1, "/", "", false, true)

			// 对于API请求，返回JSON错误
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				var appErr *errors.AppError
				if err != nil {
					if strings.Contains(err.Error(), "expired") {
						appErr = errors.NewWithDetails(
							errors.ErrTokenExpired,
							"令牌已过期",
							"请重新登录",
						)
					} else {
						appErr = errors.NewWithDetails(
							errors.ErrTokenInvalid,
							"无效的令牌",
							"请重新登录",
						)
					}
				} else {
					appErr = errors.NewWithDetails(
						errors.ErrTokenInvalid,
						"无效的令牌",
						"请重新登录",
					)
				}
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   appErr,
					"message": appErr.Message,
				})
				c.Abort()
				return
			}
			// 对于页面请求，重定向到登录页
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 检查令牌是否过期
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			// 清除过期的Token Cookie
			c.SetCookie("token", "", -1, "/", "", false, true)

			// 对于API请求，返回JSON错误
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				appErr := errors.NewWithDetails(
					errors.ErrTokenExpired,
					"令牌已过期",
					"请重新登录",
				)
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   appErr,
					"message": appErr.Message,
				})
				c.Abort()
				return
			}
			// 对于页面请求，重定向到登录页
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

// GenerateToken 生成JWT令牌
func GenerateToken(userID uint, username string, cfg *config.Config) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.JWT.Expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.JWT.Issuer,
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

// GetUserID 从上下文中获取用户ID
func GetUserID(c *gin.Context) uint {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return userID.(uint)
}

// GetUsername 从上下文中获取用户名
func GetUsername(c *gin.Context) string {
	username, exists := c.Get("username")
	if !exists {
		return ""
	}
	return username.(string)
}
