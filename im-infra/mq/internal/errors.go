package internal

import (
	"errors"
	"fmt"
)

// 预定义的错误类型
var (
	// ErrInvalidConfig 配置无效错误
	ErrInvalidConfig = errors.New("配置无效")

	// ErrConnectionFailed 连接失败错误
	ErrConnectionFailed = errors.New("连接失败")

	// ErrConnectionClosed 连接已关闭错误
	ErrConnectionClosed = errors.New("连接已关闭")

	// ErrConnectionPoolExhausted 连接池耗尽错误
	ErrConnectionPoolExhausted = errors.New("连接池耗尽")

	// ErrProducerClosed 生产者已关闭错误
	ErrProducerClosed = errors.New("生产者已关闭")

	// ErrConsumerClosed 消费者已关闭错误
	ErrConsumerClosed = errors.New("消费者已关闭")

	// ErrMessageTooLarge 消息过大错误
	ErrMessageTooLarge = errors.New("消息过大")

	// ErrInvalidTopic 无效主题错误
	ErrInvalidTopic = errors.New("无效主题")

	// ErrInvalidPartition 无效分区错误
	ErrInvalidPartition = errors.New("无效分区")

	// ErrInvalidOffset 无效偏移量错误
	ErrInvalidOffset = errors.New("无效偏移量")

	// ErrTimeout 超时错误
	ErrTimeout = errors.New("操作超时")

	// ErrRetryExhausted 重试次数耗尽错误
	ErrRetryExhausted = errors.New("重试次数耗尽")

	// ErrSerializationFailed 序列化失败错误
	ErrSerializationFailed = errors.New("序列化失败")

	// ErrDeserializationFailed 反序列化失败错误
	ErrDeserializationFailed = errors.New("反序列化失败")

	// ErrCompressionFailed 压缩失败错误
	ErrCompressionFailed = errors.New("压缩失败")

	// ErrDecompressionFailed 解压失败错误
	ErrDecompressionFailed = errors.New("解压失败")

	// ErrOffsetCommitFailed 偏移量提交失败错误
	ErrOffsetCommitFailed = errors.New("偏移量提交失败")

	// ErrRebalanceInProgress 重平衡进行中错误
	ErrRebalanceInProgress = errors.New("重平衡进行中")

	// ErrGroupCoordinatorNotFound 组协调器未找到错误
	ErrGroupCoordinatorNotFound = errors.New("组协调器未找到")

	// ErrUnknownMemberId 未知成员ID错误
	ErrUnknownMemberId = errors.New("未知成员ID")

	// ErrIllegalGeneration 非法代数错误
	ErrIllegalGeneration = errors.New("非法代数")

	// ErrInconsistentGroupProtocol 不一致的组协议错误
	ErrInconsistentGroupProtocol = errors.New("不一致的组协议")

	// ErrInvalidSessionTimeout 无效会话超时错误
	ErrInvalidSessionTimeout = errors.New("无效会话超时")

	// ErrAuthenticationFailed 认证失败错误
	ErrAuthenticationFailed = errors.New("认证失败")

	// ErrAuthorizationFailed 授权失败错误
	ErrAuthorizationFailed = errors.New("授权失败")
)

// MQError MQ模块的自定义错误类型
type MQError struct {
	// Code 错误代码
	Code string

	// Message 错误消息
	Message string

	// Cause 原始错误
	Cause error

	// Context 错误上下文信息
	Context map[string]interface{}
}

// Error 实现error接口
func (e *MQError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
func (e *MQError) Unwrap() error {
	return e.Cause
}

// Is 实现errors.Is接口
func (e *MQError) Is(target error) bool {
	if mqErr, ok := target.(*MQError); ok {
		return e.Code == mqErr.Code
	}
	return errors.Is(e.Cause, target)
}

// WithContext 添加上下文信息
func (e *MQError) WithContext(key string, value interface{}) *MQError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewMQError 创建新的MQ错误
func NewMQError(code, message string, cause error) *MQError {
	return &MQError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// 错误代码常量
const (
	// 连接相关错误代码
	ErrCodeConnectionFailed  = "CONNECTION_FAILED"
	ErrCodeConnectionClosed  = "CONNECTION_CLOSED"
	ErrCodeConnectionTimeout = "CONNECTION_TIMEOUT"
	ErrCodePoolExhausted     = "POOL_EXHAUSTED"

	// 配置相关错误代码
	ErrCodeInvalidConfig = "INVALID_CONFIG"
	ErrCodeMissingConfig = "MISSING_CONFIG"

	// 生产者相关错误代码
	ErrCodeProducerClosed      = "PRODUCER_CLOSED"
	ErrCodeProduceFailed       = "PRODUCE_FAILED"
	ErrCodeMessageTooLarge     = "MESSAGE_TOO_LARGE"
	ErrCodeSerializationFailed = "SERIALIZATION_FAILED"
	ErrCodeCompressionFailed   = "COMPRESSION_FAILED"

	// 消费者相关错误代码
	ErrCodeConsumerClosed        = "CONSUMER_CLOSED"
	ErrCodeConsumeFailed         = "CONSUME_FAILED"
	ErrCodeOffsetCommitFailed    = "OFFSET_COMMIT_FAILED"
	ErrCodeRebalanceInProgress   = "REBALANCE_IN_PROGRESS"
	ErrCodeDeserializationFailed = "DESERIALIZATION_FAILED"
	ErrCodeDecompressionFailed   = "DECOMPRESSION_FAILED"

	// 主题和分区相关错误代码
	ErrCodeInvalidTopic      = "INVALID_TOPIC"
	ErrCodeInvalidPartition  = "INVALID_PARTITION"
	ErrCodeInvalidOffset     = "INVALID_OFFSET"
	ErrCodeTopicNotFound     = "TOPIC_NOT_FOUND"
	ErrCodePartitionNotFound = "PARTITION_NOT_FOUND"

	// 认证和授权相关错误代码
	ErrCodeAuthenticationFailed = "AUTHENTICATION_FAILED"
	ErrCodeAuthorizationFailed  = "AUTHORIZATION_FAILED"

	// 超时和重试相关错误代码
	ErrCodeTimeout        = "TIMEOUT"
	ErrCodeRetryExhausted = "RETRY_EXHAUSTED"

	// 组管理相关错误代码
	ErrCodeGroupCoordinatorNotFound  = "GROUP_COORDINATOR_NOT_FOUND"
	ErrCodeUnknownMemberId           = "UNKNOWN_MEMBER_ID"
	ErrCodeIllegalGeneration         = "ILLEGAL_GENERATION"
	ErrCodeInconsistentGroupProtocol = "INCONSISTENT_GROUP_PROTOCOL"
	ErrCodeInvalidSessionTimeout     = "INVALID_SESSION_TIMEOUT"
)

// 便捷的错误创建函数

// NewConnectionError 创建连接错误
func NewConnectionError(message string, cause error) *MQError {
	return NewMQError(ErrCodeConnectionFailed, message, cause)
}

// NewConfigError 创建配置错误
func NewConfigError(message string, cause error) *MQError {
	return NewMQError(ErrCodeInvalidConfig, message, cause)
}

// NewProducerError 创建生产者错误
func NewProducerError(message string, cause error) *MQError {
	return NewMQError(ErrCodeProduceFailed, message, cause)
}

// NewConsumerError 创建消费者错误
func NewConsumerError(message string, cause error) *MQError {
	return NewMQError(ErrCodeConsumeFailed, message, cause)
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(message string, cause error) *MQError {
	return NewMQError(ErrCodeTimeout, message, cause)
}

// NewSerializationError 创建序列化错误
func NewSerializationError(message string, cause error) *MQError {
	return NewMQError(ErrCodeSerializationFailed, message, cause)
}

// NewDeserializationError 创建反序列化错误
func NewDeserializationError(message string, cause error) *MQError {
	return NewMQError(ErrCodeDeserializationFailed, message, cause)
}

// IsRetryableError 判断错误是否可重试
func IsRetryableError(err error) bool {
	if mqErr, ok := err.(*MQError); ok {
		switch mqErr.Code {
		case ErrCodeConnectionTimeout, ErrCodeConnectionFailed, ErrCodeTimeout:
			return true
		case ErrCodePoolExhausted, ErrCodeRebalanceInProgress:
			return true
		default:
			return false
		}
	}

	// 检查一些常见的可重试错误
	switch err {
	case ErrConnectionFailed, ErrTimeout, ErrConnectionPoolExhausted:
		return true
	case ErrRebalanceInProgress:
		return true
	default:
		return false
	}
}

// IsFatalError 判断错误是否为致命错误（不可恢复）
func IsFatalError(err error) bool {
	if mqErr, ok := err.(*MQError); ok {
		switch mqErr.Code {
		case ErrCodeAuthenticationFailed, ErrCodeAuthorizationFailed:
			return true
		case ErrCodeInvalidConfig, ErrCodeMissingConfig:
			return true
		case ErrCodeInvalidTopic, ErrCodeInvalidPartition:
			return true
		default:
			return false
		}
	}

	// 检查一些常见的致命错误
	switch err {
	case ErrAuthenticationFailed, ErrAuthorizationFailed:
		return true
	case ErrInvalidConfig, ErrInvalidTopic, ErrInvalidPartition:
		return true
	default:
		return false
	}
}
