package ratelimit

import "errors"

// 预定义错误
var (
	// ErrInvalidRate 无效的速率配置
	ErrInvalidRate = errors.New("rate must be positive")

	// ErrInvalidCapacity 无效的容量配置
	ErrInvalidCapacity = errors.New("capacity must be positive")

	// ErrInvalidRuleName 无效的规则名称
	ErrInvalidRuleName = errors.New("rule name cannot be empty")

	// ErrInvalidResource 无效的资源标识
	ErrInvalidResource = errors.New("resource identifier cannot be empty")

	// ErrRuleNotFound 规则未找到
	ErrRuleNotFound = errors.New("rate limit rule not found")

	// ErrCacheUnavailable 缓存不可用
	ErrCacheUnavailable = errors.New("cache service unavailable")

	// ErrConfigUnavailable 配置中心不可用
	ErrConfigUnavailable = errors.New("configuration center unavailable")

	// ErrScriptExecutionFailed Lua脚本执行失败
	ErrScriptExecutionFailed = errors.New("lua script execution failed")

	// ErrInvalidCount 无效的请求数量
	ErrInvalidCount = errors.New("request count must be positive")

	// ErrLimiterClosed 限流器已关闭
	ErrLimiterClosed = errors.New("rate limiter is closed")

	// ErrInvalidConfiguration 无效的配置
	ErrInvalidConfiguration = errors.New("invalid rate limiter configuration")

	// ErrServiceNameEmpty 服务名为空
	ErrServiceNameEmpty = errors.New("service name cannot be empty")

	// ErrBatchSizeExceeded 批处理大小超限
	ErrBatchSizeExceeded = errors.New("batch size exceeded maximum limit")

	// ErrStatisticsDisabled 统计功能未启用
	ErrStatisticsDisabled = errors.New("statistics collection is disabled")

	// ErrMetricsDisabled 指标收集未启用
	ErrMetricsDisabled = errors.New("metrics collection is disabled")

	// ErrRuleValidationFailed 规则验证失败
	ErrRuleValidationFailed = errors.New("rate limit rule validation failed")

	// ErrContextCanceled 上下文已取消
	ErrContextCanceled = errors.New("context canceled")

	// ErrTimeout 操作超时
	ErrTimeout = errors.New("operation timeout")

	// ErrRateLimited 请求被限流
	ErrRateLimited = errors.New("request rate limited")
)

// RateLimitError 限流错误类型
type RateLimitError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Cause   error                  `json:"-"`
}

// Error 实现 error 接口
func (e *RateLimitError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap 支持错误链
func (e *RateLimitError) Unwrap() error {
	return e.Cause
}

// 错误码常量
const (
	ErrorCodeInvalidInput       = "INVALID_INPUT"
	ErrorCodeRuleNotFound       = "RULE_NOT_FOUND"
	ErrorCodeCacheError         = "CACHE_ERROR"
	ErrorCodeConfigError        = "CONFIG_ERROR"
	ErrorCodeScriptError        = "SCRIPT_ERROR"
	ErrorCodeRateLimited        = "RATE_LIMITED"
	ErrorCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrorCodeTimeout            = "TIMEOUT"
	ErrorCodeInternalError      = "INTERNAL_ERROR"
)

// NewRateLimitError 创建限流错误
func NewRateLimitError(code, message string, cause error) *RateLimitError {
	return &RateLimitError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Details: make(map[string]interface{}),
	}
}

// NewRateLimitErrorWithDetails 创建带详情的限流错误
func NewRateLimitErrorWithDetails(code, message string, details map[string]interface{}, cause error) *RateLimitError {
	return &RateLimitError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   cause,
	}
}

// WithDetail 添加错误详情
func (e *RateLimitError) WithDetail(key string, value interface{}) *RateLimitError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// IsRateLimited 检查是否为限流错误
func IsRateLimited(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrRateLimited) {
		return true
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.Code == ErrorCodeRateLimited
	}

	return false
}

// IsConfigError 检查是否为配置错误
func IsConfigError(err error) bool {
	if err == nil {
		return false
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.Code == ErrorCodeConfigError
	}

	return false
}

// IsCacheError 检查是否为缓存错误
func IsCacheError(err error) bool {
	if err == nil {
		return false
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.Code == ErrorCodeCacheError
	}

	return false
}

// IsTimeoutError 检查是否为超时错误
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrTimeout) {
		return true
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.Code == ErrorCodeTimeout
	}

	return false
}

// IsRetryableError 检查是否为可重试错误
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 缓存错误、超时错误、服务不可用错误通常可以重试
	return IsCacheError(err) || IsTimeoutError(err) || IsServiceUnavailableError(err)
}

// IsServiceUnavailableError 检查是否为服务不可用错误
func IsServiceUnavailableError(err error) bool {
	if err == nil {
		return false
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.Code == ErrorCodeServiceUnavailable
	}

	return false
}

// WrapError 包装错误为限流错误
func WrapError(code, message string, cause error) error {
	return NewRateLimitError(code, message, cause)
}

// WrapErrorWithDetails 包装错误为带详情的限流错误
func WrapErrorWithDetails(code, message string, details map[string]interface{}, cause error) error {
	return NewRateLimitErrorWithDetails(code, message, details, cause)
}
