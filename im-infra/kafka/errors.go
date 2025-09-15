package kafka

import (
	"errors"
	"fmt"
)

// KafkaError 表示 Kafka 操作中的错误
type KafkaError struct {
	Code    string
	Message string
	Err     error
}

func (e *KafkaError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *KafkaError) Unwrap() error {
	return e.Err
}

// 错误代码常量
const (
	ErrCodeConfig      = "CONFIG_ERROR"
	ErrCodeConnection  = "CONNECTION_ERROR"
	ErrCodeProducer    = "PRODUCER_ERROR"
	ErrCodeConsumer    = "CONSUMER_ERROR"
	ErrCodeAdmin       = "ADMIN_ERROR"
	ErrCodeTimeout     = "TIMEOUT_ERROR"
	ErrCodeInvalidArg  = "INVALID_ARGUMENT"
)

// ErrInvalidConfig 创建配置错误
func ErrInvalidConfig(msg string) error {
	return &KafkaError{
		Code:    ErrCodeConfig,
		Message: msg,
	}
}

// ErrConnection 创建连接错误
func ErrConnection(msg string, err error) error {
	return &KafkaError{
		Code:    ErrCodeConnection,
		Message: msg,
		Err:     err,
	}
}

// ErrProducer 创建生产者错误
func ErrProducer(msg string, err error) error {
	return &KafkaError{
		Code:    ErrCodeProducer,
		Message: msg,
		Err:     err,
	}
}

// ErrConsumer 创建消费者错误
func ErrConsumer(msg string, err error) error {
	return &KafkaError{
		Code:    ErrCodeConsumer,
		Message: msg,
		Err:     err,
	}
}

// ErrAdmin 创建管理错误
func ErrAdmin(msg string, err error) error {
	return &KafkaError{
		Code:    ErrCodeAdmin,
		Message: msg,
		Err:     err,
	}
}

// ErrTimeout 创建超时错误
func ErrTimeout(msg string, err error) error {
	return &KafkaError{
		Code:    ErrCodeTimeout,
		Message: msg,
		Err:     err,
	}
}

// ErrInvalidArg 创建无效参数错误
func ErrInvalidArg(msg string) error {
	return &KafkaError{
		Code:    ErrCodeInvalidArg,
		Message: msg,
	}
}

// IsConfigError 检查是否为配置错误
func IsConfigError(err error) bool {
	var kErr *KafkaError
	return err != nil && (errors.As(err, &kErr) && kErr.Code == ErrCodeConfig)
}

// IsConnectionError 检查是否为连接错误
func IsConnectionError(err error) bool {
	var kErr *KafkaError
	return err != nil && (errors.As(err, &kErr) && kErr.Code == ErrCodeConnection)
}

// IsProducerError 检查是否为生产者错误
func IsProducerError(err error) bool {
	var kErr *KafkaError
	return err != nil && (errors.As(err, &kErr) && kErr.Code == ErrCodeProducer)
}

// IsConsumerError 检查是否为消费者错误
func IsConsumerError(err error) bool {
	var kErr *KafkaError
	return err != nil && (errors.As(err, &kErr) && kErr.Code == ErrCodeConsumer)
}

// IsAdminError 检查是否为管理错误
func IsAdminError(err error) bool {
	var kErr *KafkaError
	return err != nil && (errors.As(err, &kErr) && kErr.Code == ErrCodeAdmin)
}

// IsTimeoutError 检查是否为超时错误
func IsTimeoutError(err error) bool {
	var kErr *KafkaError
	return err != nil && (errors.As(err, &kErr) && kErr.Code == ErrCodeTimeout)
}

// IsInvalidArgError 检查是否为无效参数错误
func IsInvalidArgError(err error) bool {
	var kErr *KafkaError
	return err != nil && (errors.As(err, &kErr) && kErr.Code == ErrCodeInvalidArg)
}