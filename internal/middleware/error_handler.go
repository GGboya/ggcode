package middleware

import (
	"ggcode/internal/pkg/errors"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Success   bool                   `json:"success"`
	Error     *errors.AppError       `json:"error,omitempty"`
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

	// 创建内部服务器错误
	appErr := errors.NewWithDetails(
		errors.ErrInternalServer,
		"服务器内部错误",
		"系统发生未预期的错误，请稍后重试",
	)

	// 添加调试信息（仅在开发环境）
	if gin.Mode() == gin.DebugMode {
		appErr.Context = map[string]interface{}{
			"panic": string(recovered.(string)),
			"stack": string(stack),
		}
	}

	// 返回错误响应
	sendErrorResponse(c, appErr)
}

// sendErrorResponse 发送错误响应
func sendErrorResponse(c *gin.Context, appErr *errors.AppError) {
	// 获取请求ID（如果存在）
	requestID := c.GetString("request_id")

	response := ErrorResponse{
		Success:   false,
		Error:     appErr,
		Message:   appErr.Message,
		Details:   appErr.Details,
		Context:   appErr.Context,
		RequestID: requestID,
	}

	// 设置HTTP状态码
	statusCode := appErr.HTTPStatus
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}

	c.JSON(statusCode, response)
}

// HandleError 处理错误的辅助函数
func HandleError(c *gin.Context, err error) {
	// 检查是否为应用错误
	if appErr := errors.GetAppError(err); appErr != nil {
		sendErrorResponse(c, appErr)
		return
	}

	// 处理其他类型的错误
	appErr := errors.NewWithError(
		errors.ErrInternalServer,
		"服务器内部错误",
		err,
	)

	sendErrorResponse(c, appErr)
}

// ValidationError 处理验证错误
func ValidationError(c *gin.Context, field, message string) {
	appErr := errors.NewWithContext(
		errors.ErrValidationFailed,
		"数据验证失败",
		map[string]interface{}{
			"field":   field,
			"message": message,
		},
	)

	sendErrorResponse(c, appErr)
}

// NotFoundError 处理资源未找到错误
func NotFoundError(c *gin.Context, resource string) {
	appErr := errors.NewWithDetails(
		errors.ErrNotFound,
		"资源未找到",
		resource+"不存在",
	)

	sendErrorResponse(c, appErr)
}

// UnauthorizedError 处理未授权错误
func UnauthorizedError(c *gin.Context, message string) {
	appErr := errors.NewWithDetails(
		errors.ErrUnauthorized,
		"未授权访问",
		message,
	)

	sendErrorResponse(c, appErr)
}

// ForbiddenError 处理禁止访问错误
func ForbiddenError(c *gin.Context, message string) {
	appErr := errors.NewWithDetails(
		errors.ErrForbidden,
		"禁止访问",
		message,
	)

	sendErrorResponse(c, appErr)
}

// BadRequestError 处理请求错误
func BadRequestError(c *gin.Context, message string) {
	appErr := errors.NewWithDetails(
		errors.ErrInvalidRequest,
		"无效的请求",
		message,
	)

	sendErrorResponse(c, appErr)
}

// DatabaseError 处理数据库错误
func DatabaseError(c *gin.Context, err error) {
	appErr := errors.NewWithError(
		errors.ErrDatabaseError,
		"数据库操作失败",
		err,
	)

	sendErrorResponse(c, appErr)
}

// ConfigError 处理配置错误
func ConfigError(c *gin.Context, err error) {
	appErr := errors.NewWithError(
		errors.ErrConfigError,
		"配置错误",
		err,
	)

	sendErrorResponse(c, appErr)
}
