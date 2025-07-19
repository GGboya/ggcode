package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 错误码类型
type ErrorCode int

// 系统级错误码 (1000-1999)
const (
	// 通用错误
	ErrInternalServer   ErrorCode = 1000
	ErrInvalidRequest   ErrorCode = 1001
	ErrUnauthorized     ErrorCode = 1002
	ErrForbidden        ErrorCode = 1003
	ErrNotFound         ErrorCode = 1004
	ErrMethodNotAllowed ErrorCode = 1005
	ErrRequestTimeout   ErrorCode = 1006
	ErrTooManyRequests  ErrorCode = 1007
	ErrRequestTooLarge  ErrorCode = 1008
	ErrValidationFailed ErrorCode = 1009
	ErrDatabaseError    ErrorCode = 1010
	ErrConfigError      ErrorCode = 1011

	// 认证相关错误 (1100-1199)
	ErrInvalidCredentials ErrorCode = 1100
	ErrTokenExpired       ErrorCode = 1101
	ErrTokenInvalid       ErrorCode = 1102
	ErrTokenMissing       ErrorCode = 1103
	ErrUserNotFound       ErrorCode = 1104
	ErrUserAlreadyExists  ErrorCode = 1105
	ErrPasswordTooWeak    ErrorCode = 1106

	// 业务逻辑错误 (2000-2999)
	// 题库相关 (2000-2099)
	ErrQuestionBankNotFound   ErrorCode = 2000
	ErrQuestionBankExists     ErrorCode = 2001
	ErrQuestionNotFound       ErrorCode = 2002
	ErrQuestionExists         ErrorCode = 2003
	ErrInvalidQuestionData    ErrorCode = 2004
	ErrQuestionBankPermission ErrorCode = 2005

	// 学习计划相关 (2100-2199)
	ErrStudyPlanNotFound    ErrorCode = 2100
	ErrStudyPlanExists      ErrorCode = 2101
	ErrInvalidStudyPlanData ErrorCode = 2102
	ErrStudyPlanPermission  ErrorCode = 2103

	// 评测系统相关 (2200-2299)
	ErrJudgeSystemError        ErrorCode = 2200
	ErrCodeCompilationFailed   ErrorCode = 2201
	ErrCodeExecutionTimeout    ErrorCode = 2202
	ErrCodeMemoryLimitExceeded ErrorCode = 2203
	ErrCodeRuntimeError        ErrorCode = 2204
	ErrInvalidLanguage         ErrorCode = 2205
	ErrCodeTooLong             ErrorCode = 2206
	ErrTestCaseNotFound        ErrorCode = 2207
	ErrTestCaseInvalid         ErrorCode = 2208

	// 面试岛相关 (2300-2399)
	ErrInterviewIslandNotFound ErrorCode = 2300
	ErrLevelNotFound           ErrorCode = 2301
	ErrLevelNotUnlocked        ErrorCode = 2302
	ErrSubmissionFailed        ErrorCode = 2303
)

// AppError 应用错误结构
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
	Err        error                  `json:"-"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
	}
}

// NewWithDetails 创建带详细信息的错误
func NewWithDetails(code ErrorCode, message, details string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: getHTTPStatus(code),
	}
}

// NewWithError 创建包装原始错误的错误
func NewWithError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
		Err:        err,
	}
}

// NewWithContext 创建带上下文的错误
func NewWithContext(code ErrorCode, message string, context map[string]interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
		Context:    context,
	}
}

// Wrap 包装现有错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
		Err:        err,
	}
}

// getHTTPStatus 根据错误码获取HTTP状态码
func getHTTPStatus(code ErrorCode) int {
	switch {
	case code >= 1000 && code < 1100:
		// 系统级错误
		switch code {
		case ErrInternalServer, ErrDatabaseError, ErrConfigError:
			return http.StatusInternalServerError
		case ErrInvalidRequest, ErrValidationFailed:
			return http.StatusBadRequest
		case ErrUnauthorized:
			return http.StatusUnauthorized
		case ErrForbidden:
			return http.StatusForbidden
		case ErrNotFound:
			return http.StatusNotFound
		case ErrMethodNotAllowed:
			return http.StatusMethodNotAllowed
		case ErrRequestTimeout:
			return http.StatusRequestTimeout
		case ErrTooManyRequests:
			return http.StatusTooManyRequests
		case ErrRequestTooLarge:
			return http.StatusRequestEntityTooLarge
		default:
			return http.StatusInternalServerError
		}
	case code >= 1100 && code < 1200:
		// 认证相关错误
		switch code {
		case ErrInvalidCredentials, ErrTokenInvalid, ErrTokenMissing:
			return http.StatusUnauthorized
		case ErrTokenExpired:
			return http.StatusUnauthorized
		case ErrUserNotFound:
			return http.StatusNotFound
		case ErrUserAlreadyExists:
			return http.StatusConflict
		case ErrPasswordTooWeak:
			return http.StatusBadRequest
		default:
			return http.StatusUnauthorized
		}
	case code >= 2000 && code < 3000:
		// 业务逻辑错误
		switch code {
		case ErrQuestionBankNotFound, ErrQuestionNotFound, ErrStudyPlanNotFound, ErrInterviewIslandNotFound, ErrLevelNotFound:
			return http.StatusNotFound
		case ErrQuestionBankExists, ErrQuestionExists, ErrStudyPlanExists:
			return http.StatusConflict
		case ErrQuestionBankPermission, ErrStudyPlanPermission, ErrLevelNotUnlocked:
			return http.StatusForbidden
		case ErrInvalidQuestionData, ErrInvalidStudyPlanData, ErrInvalidLanguage, ErrTestCaseInvalid:
			return http.StatusBadRequest
		case ErrJudgeSystemError, ErrCodeCompilationFailed, ErrCodeExecutionTimeout, ErrCodeMemoryLimitExceeded, ErrCodeRuntimeError, ErrSubmissionFailed:
			return http.StatusInternalServerError
		case ErrCodeTooLong:
			return http.StatusRequestEntityTooLarge
		default:
			return http.StatusBadRequest
		}
	default:
		return http.StatusInternalServerError
	}
}

// 预定义的错误
var (
	ErrInternalServerError   = New(ErrInternalServer, "内部服务器错误")
	ErrInvalidRequestError   = New(ErrInvalidRequest, "无效的请求")
	ErrUnauthorizedError     = New(ErrUnauthorized, "未授权访问")
	ErrForbiddenError        = New(ErrForbidden, "禁止访问")
	ErrNotFoundError         = New(ErrNotFound, "资源未找到")
	ErrMethodNotAllowedError = New(ErrMethodNotAllowed, "方法不允许")
	ErrRequestTimeoutError   = New(ErrRequestTimeout, "请求超时")
	ErrTooManyRequestsError  = New(ErrTooManyRequests, "请求过于频繁")
	ErrRequestTooLargeError  = New(ErrRequestTooLarge, "请求体过大")
	ErrValidationFailedError = New(ErrValidationFailed, "数据验证失败")
	ErrDatabaseErrorError    = New(ErrDatabaseError, "数据库错误")
	ErrConfigErrorError      = New(ErrConfigError, "配置错误")

	ErrInvalidCredentialsError = New(ErrInvalidCredentials, "用户名或密码错误")
	ErrTokenExpiredError       = New(ErrTokenExpired, "令牌已过期")
	ErrTokenInvalidError       = New(ErrTokenInvalid, "无效的令牌")
	ErrTokenMissingError       = New(ErrTokenMissing, "缺少令牌")
	ErrUserNotFoundError       = New(ErrUserNotFound, "用户不存在")
	ErrUserAlreadyExistsError  = New(ErrUserAlreadyExists, "用户已存在")
	ErrPasswordTooWeakError    = New(ErrPasswordTooWeak, "密码强度不足")

	ErrQuestionBankNotFoundError   = New(ErrQuestionBankNotFound, "题库不存在")
	ErrQuestionBankExistsError     = New(ErrQuestionBankExists, "题库已存在")
	ErrQuestionNotFoundError       = New(ErrQuestionNotFound, "题目不存在")
	ErrQuestionExistsError         = New(ErrQuestionExists, "题目已存在")
	ErrInvalidQuestionDataError    = New(ErrInvalidQuestionData, "题目数据无效")
	ErrQuestionBankPermissionError = New(ErrQuestionBankPermission, "没有题库访问权限")

	ErrStudyPlanNotFoundError    = New(ErrStudyPlanNotFound, "学习计划不存在")
	ErrStudyPlanExistsError      = New(ErrStudyPlanExists, "学习计划已存在")
	ErrInvalidStudyPlanDataError = New(ErrInvalidStudyPlanData, "学习计划数据无效")
	ErrStudyPlanPermissionError  = New(ErrStudyPlanPermission, "没有学习计划访问权限")

	ErrJudgeSystemErrorError        = New(ErrJudgeSystemError, "评测系统错误")
	ErrCodeCompilationFailedError   = New(ErrCodeCompilationFailed, "代码编译失败")
	ErrCodeExecutionTimeoutError    = New(ErrCodeExecutionTimeout, "代码执行超时")
	ErrCodeMemoryLimitExceededError = New(ErrCodeMemoryLimitExceeded, "内存使用超限")
	ErrCodeRuntimeErrorError        = New(ErrCodeRuntimeError, "代码运行时错误")
	ErrInvalidLanguageError         = New(ErrInvalidLanguage, "不支持的编程语言")
	ErrCodeTooLongError             = New(ErrCodeTooLong, "代码长度超限")
	ErrTestCaseNotFoundError        = New(ErrTestCaseNotFound, "测试用例不存在")
	ErrTestCaseInvalidError         = New(ErrTestCaseInvalid, "测试用例数据无效")

	ErrInterviewIslandNotFoundError = New(ErrInterviewIslandNotFound, "面试岛不存在")
	ErrLevelNotFoundError           = New(ErrLevelNotFound, "关卡不存在")
	ErrLevelNotUnlockedError        = New(ErrLevelNotUnlocked, "关卡未解锁")
	ErrSubmissionFailedError        = New(ErrSubmissionFailed, "提交失败")
)

// IsAppError 检查是否为应用错误
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError 获取应用错误
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return nil
}
