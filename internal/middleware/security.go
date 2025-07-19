package middleware

import (
	"ggcode/internal/config"
	"ggcode/internal/pkg/errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// SecurityMiddleware 安全中间件
type SecurityMiddleware struct {
	config *config.Config
	// 速率限制器映射
	rateLimiters map[string]*rate.Limiter
	mu           sync.RWMutex
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware(cfg *config.Config) *SecurityMiddleware {
	return &SecurityMiddleware{
		config:       cfg,
		rateLimiters: make(map[string]*rate.Limiter),
	}
}

// CORS 跨域资源共享中间件
func (sm *SecurityMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.Security.EnableCORS {
			c.Next()
			return
		}

		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			// 检查是否在允许的源列表中
			allowed := false
			for _, allowedOrigin := range sm.config.Security.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			if allowed {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestSizeLimit 请求大小限制中间件
func (sm *SecurityMiddleware) RequestSizeLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置请求体大小限制
		c.Request.Body = http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			sm.config.Security.MaxRequestSize,
		)

		c.Next()
	}
}

// RateLimit 速率限制中间件
func (sm *SecurityMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端标识（IP地址）
		clientIP := c.ClientIP()

		// 获取或创建速率限制器
		limiter := sm.getRateLimiter(clientIP)

		// 检查是否超过限制
		if !limiter.Allow() {
			appErr := errors.NewWithDetails(
				errors.ErrTooManyRequests,
				"请求过于频繁",
				"请稍后再试",
			)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   appErr,
				"message": appErr.Message,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getRateLimiter 获取或创建速率限制器
func (sm *SecurityMiddleware) getRateLimiter(clientIP string) *rate.Limiter {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	limiter, exists := sm.rateLimiters[clientIP]
	if !exists {
		// 创建新的速率限制器
		// 将时间窗口转换为每秒请求数
		requestsPerSecond := float64(sm.config.Security.RateLimitRequests) / sm.config.Security.RateLimitWindow.Seconds()
		limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), sm.config.Security.RateLimitRequests)
		sm.rateLimiters[clientIP] = limiter
	}

	return limiter
}

// SecurityHeaders 安全响应头中间件
func (sm *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// 防止MIME类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS保护
		c.Header("X-XSS-Protection", "1; mode=block")

		// 引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 内容安全策略（CSP）
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none';")

		// 权限策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// NoCache 禁用缓存中间件
func (sm *SecurityMiddleware) NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}

// RequestID 请求ID中间件
func (sm *SecurityMiddleware) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// CleanupRateLimiters 清理过期的速率限制器
func (sm *SecurityMiddleware) CleanupRateLimiters() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			sm.mu.Lock()
			// 这里可以实现更复杂的清理逻辑
			// 比如基于最后访问时间的清理
			sm.mu.Unlock()
		}
	}()
}
