package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message,omitempty"`
	Details   string                 `json:"details,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

// ErrorHandler 全局错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// 处理panic
		handlePanic(c, recovered)
	})
}

// handlePanic 处理panic
func handlePanic(c *gin.Context, recovered interface{}) {
	// 记录panic堆栈信息
	stack := debug.Stack()

	// 创建错误响应
	response := ErrorResponse{
		Success: false,
		Message: "服务器内部错误",
		Details: "系统发生未预期的错误，请稍后重试",
	}

	// 添加调试信息（仅在开发环境）
	if gin.Mode() == gin.DebugMode {
		response.Context = map[string]interface{}{
			"panic": string(recovered.(string)),
			"stack": string(stack),
		}
	}

	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")
	if requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusInternalServerError, response)
}

// HandleError 处理错误的辅助函数
func HandleError(c *gin.Context, err error) {
	response := ErrorResponse{
		Success: false,
		Message: "服务器内部错误",
		Details: err.Error(),
	}

	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")
	if requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusInternalServerError, response)
}

// ValidationError 处理验证错误
func ValidationError(c *gin.Context, field, message string) {
	response := ErrorResponse{
		Success: false,
		Message: "数据验证失败",
		Details: message,
		Context: map[string]interface{}{
			"field": field,
		},
	}

	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")
	if requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusBadRequest, response)
}

// NotFoundError 处理资源未找到错误
func NotFoundError(c *gin.Context, resource string) {
	response := ErrorResponse{
		Success: false,
		Message: "资源未找到",
		Details: resource + "不存在",
	}

	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")
	if requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusNotFound, response)
}

// UnauthorizedError 处理未授权错误
func UnauthorizedError(c *gin.Context, message string) {
	response := ErrorResponse{
		Success: false,
		Message: "未授权访问",
		Details: message,
	}

	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")
	if requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusUnauthorized, response)
}

// ForbiddenError 处理禁止访问错误
func ForbiddenError(c *gin.Context, message string) {
	response := ErrorResponse{
		Success: false,
		Message: "禁止访问",
		Details: message,
	}

	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")
	if requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusForbidden, response)
}

// BadRequestError 处理请求错误
func BadRequestError(c *gin.Context, message string) {
	response := ErrorResponse{
		Success: false,
		Message: "无效的请求",
		Details: message,
	}

	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")
	if requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusBadRequest, response)
}

// DatabaseError 处理数据库错误
func DatabaseError(c *gin.Context, err error) {
	response := ErrorResponse{
		Success: false,
		Message: "数据库操作失败",
		Details: err.Error(),
	}

	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")
	if requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusInternalServerError, response)
}

// ConfigError 处理配置错误
func ConfigError(c *gin.Context, err error) {
	response := ErrorResponse{
		Success: false,
		Message: "配置错误",
		Details: err.Error(),
	}

	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")
	if requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(http.StatusInternalServerError, response)
}
