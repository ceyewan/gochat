package coordination

import (
	"fmt"
	"time"
)

// CoordinatorOptions 协调器配置选项
type CoordinatorOptions struct {
	// Endpoints etcd 服务器地址列表
	Endpoints []string `json:"endpoints"`

	// Username etcd 用户名（可选）
	Username string `json:"username,omitempty"`

	// Password etcd 密码（可选）
	Password string `json:"password,omitempty"`

	// Timeout 连接超时时间
	Timeout time.Duration `json:"timeout"`

	// RetryConfig 重试配置
	RetryConfig *RetryConfig `json:"retry_config,omitempty"`
}

// RetryConfig 重试机制配置
type RetryConfig struct {
	// MaxAttempts 最大重试次数
	MaxAttempts int `json:"max_attempts"`

	// InitialDelay 初始延迟
	InitialDelay time.Duration `json:"initial_delay"`

	// MaxDelay 最大延迟
	MaxDelay time.Duration `json:"max_delay"`

	// Multiplier 退避倍数
	Multiplier float64 `json:"multiplier"`
}

// CoordinationError 协调器错误类型
type CoordinationError struct {
	// Code 错误码
	Code ErrorCode `json:"code"`

	// Message 错误消息
	Message string `json:"message"`

	// Cause 原始错误
	Cause error `json:"cause,omitempty"`
}

// ErrorCode 错误码定义
type ErrorCode string

const (
	// ErrCodeConnection 连接错误
	ErrCodeConnection ErrorCode = "CONNECTION_ERROR"

	// ErrCodeTimeout 超时错误
	ErrCodeTimeout ErrorCode = "TIMEOUT_ERROR"

	// ErrCodeNotFound 未找到错误
	ErrCodeNotFound ErrorCode = "NOT_FOUND"

	// ErrCodeConflict 冲突错误
	ErrCodeConflict ErrorCode = "CONFLICT"

	// ErrCodeValidation 验证错误
	ErrCodeValidation ErrorCode = "VALIDATION_ERROR"

	// ErrCodeUnavailable 服务不可用错误
	ErrCodeUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// Error 实现 error 接口
func (e *CoordinationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewCoordinationError 创建协调器错误
func NewCoordinationError(code ErrorCode, message string, cause error) *CoordinationError {
	return &CoordinationError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// IsCoordinationError 检查是否为协调器错误
func IsCoordinationError(err error) bool {
	_, ok := err.(*CoordinationError)
	return ok
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) ErrorCode {
	if coordErr, ok := err.(*CoordinationError); ok {
		return coordErr.Code
	}
	return ""
}

// DefaultCoordinatorOptions 返回默认的协调器选项
func DefaultCoordinatorOptions() CoordinatorOptions {
	return CoordinatorOptions{
		Endpoints: []string{"localhost:2379"},
		Timeout:   5 * time.Second,
		RetryConfig: &RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     2 * time.Second,
			Multiplier:   2.0,
		},
	}
}

// Validate 验证选项有效性
func (opts *CoordinatorOptions) Validate() error {
	if len(opts.Endpoints) == 0 {
		return NewCoordinationError(ErrCodeValidation, "endpoints cannot be empty", nil)
	}

	if opts.Timeout <= 0 {
		return NewCoordinationError(ErrCodeValidation, "timeout must be positive", nil)
	}

	if opts.RetryConfig != nil {
		if opts.RetryConfig.MaxAttempts < 0 {
			return NewCoordinationError(ErrCodeValidation, "max_attempts cannot be negative", nil)
		}

		if opts.RetryConfig.InitialDelay <= 0 {
			return NewCoordinationError(ErrCodeValidation, "initial_delay must be positive", nil)
		}

		if opts.RetryConfig.MaxDelay <= 0 {
			return NewCoordinationError(ErrCodeValidation, "max_delay must be positive", nil)
		}

		if opts.RetryConfig.Multiplier <= 1.0 {
			return NewCoordinationError(ErrCodeValidation, "multiplier must be greater than 1.0", nil)
		}
	}

	return nil
}
