package coordination

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/coord/pkg/client"
)

// CoordinatorOptions 协调器配置选项
type CoordinatorOptions = client.CoordinatorOptions

// RetryConfig 重试机制配置
type RetryConfig = client.RetryConfig

// CoordinationError 协调器错误类型
type CoordinationError = client.CoordinationError

// ErrorCode 错误码定义
type ErrorCode = client.ErrorCode

// 错误码常量
const (
	ErrCodeConnection  = client.ErrCodeConnection  // 连接错误
	ErrCodeTimeout     = client.ErrCodeTimeout     // 超时错误
	ErrCodeNotFound    = client.ErrCodeNotFound    // 未找到错误
	ErrCodeConflict    = client.ErrCodeConflict    // 冲突错误
	ErrCodeValidation  = client.ErrCodeValidation  // 验证错误
	ErrCodeUnavailable = client.ErrCodeUnavailable // 服务不可用错误
)

// NewCoordinationError 创建协调器错误
func NewCoordinationError(code ErrorCode, message string, cause error) *CoordinationError {
	return client.NewCoordinationError(code, message, cause)
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
		Endpoints: []string{"localhost:23791", "localhost:23792", "localhost:23793"},
		Timeout:   5 * time.Second,
		RetryConfig: &RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     2 * time.Second,
			Multiplier:   2.0,
		},
	}
}
