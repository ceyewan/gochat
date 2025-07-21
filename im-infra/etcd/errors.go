package etcd

import (
	"errors"
	"fmt"
)

// 定义错误类型常量
const (
	ErrTypeConnection    = "CONNECTION_ERROR"
	ErrTypeRegistry      = "REGISTRY_ERROR"
	ErrTypeDiscovery     = "DISCOVERY_ERROR"
	ErrTypeLock          = "LOCK_ERROR"
	ErrTypeLease         = "LEASE_ERROR"
	ErrTypeConfiguration = "CONFIGURATION_ERROR"
	ErrTypeTimeout       = "TIMEOUT_ERROR"
	ErrTypeNotFound      = "NOT_FOUND_ERROR"
	ErrTypeAlreadyExists = "ALREADY_EXISTS_ERROR"
	ErrTypeInvalidState  = "INVALID_STATE_ERROR"
)

// EtcdError 自定义错误类型
type EtcdError struct {
	Type    string `json:"type"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause,omitempty"`
}

// Error 实现 error 接口
func (e *EtcdError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap 支持错误链
func (e *EtcdError) Unwrap() error {
	return e.Cause
}

// Is 支持错误比较
func (e *EtcdError) Is(target error) bool {
	if t, ok := target.(*EtcdError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return false
}

// NewEtcdError 创建新的 EtcdError
func NewEtcdError(errType string, code int, message string, cause error) *EtcdError {
	return &EtcdError{
		Type:    errType,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// 预定义的错误变量
var (
	// 连接相关错误
	ErrConnectionFailed  = NewEtcdError(ErrTypeConnection, 1001, "failed to connect to etcd", nil)
	ErrConnectionLost    = NewEtcdError(ErrTypeConnection, 1002, "connection to etcd lost", nil)
	ErrConnectionTimeout = NewEtcdError(ErrTypeConnection, 1003, "connection timeout", nil)
	ErrNotConnected      = NewEtcdError(ErrTypeConnection, 1004, "not connected to etcd", nil)

	// 注册相关错误
	ErrServiceAlreadyRegistered = NewEtcdError(ErrTypeRegistry, 2001, "service already registered", nil)
	ErrServiceNotRegistered     = NewEtcdError(ErrTypeRegistry, 2002, "service not registered", nil)
	ErrRegistrationFailed       = NewEtcdError(ErrTypeRegistry, 2003, "service registration failed", nil)
	ErrDeregistrationFailed     = NewEtcdError(ErrTypeRegistry, 2004, "service deregistration failed", nil)

	// 发现相关错误
	ErrServiceNotFound      = NewEtcdError(ErrTypeDiscovery, 3001, "service not found", nil)
	ErrNoAvailableInstances = NewEtcdError(ErrTypeDiscovery, 3002, "no available service instances", nil)
	ErrDiscoveryFailed      = NewEtcdError(ErrTypeDiscovery, 3003, "service discovery failed", nil)

	// 锁相关错误
	ErrLockAcquisitionFailed = NewEtcdError(ErrTypeLock, 4001, "failed to acquire lock", nil)
	ErrLockNotHeld           = NewEtcdError(ErrTypeLock, 4002, "lock not held", nil)
	ErrLockAlreadyHeld       = NewEtcdError(ErrTypeLock, 4003, "lock already held", nil)
	ErrLockTimeout           = NewEtcdError(ErrTypeLock, 4004, "lock acquisition timeout", nil)

	// 租约相关错误
	ErrLeaseCreationFailed = NewEtcdError(ErrTypeLease, 5001, "failed to create lease", nil)
	ErrLeaseNotFound       = NewEtcdError(ErrTypeLease, 5002, "lease not found", nil)
	ErrLeaseExpired        = NewEtcdError(ErrTypeLease, 5003, "lease expired", nil)
	ErrLeaseRevokeFailed   = NewEtcdError(ErrTypeLease, 5004, "failed to revoke lease", nil)

	// 配置相关错误
	ErrInvalidConfiguration = NewEtcdError(ErrTypeConfiguration, 6001, "invalid configuration", nil)
	ErrMissingEndpoints     = NewEtcdError(ErrTypeConfiguration, 6002, "missing etcd endpoints", nil)
	ErrInvalidTimeout       = NewEtcdError(ErrTypeConfiguration, 6003, "invalid timeout value", nil)

	// 通用错误
	ErrTimeout      = NewEtcdError(ErrTypeTimeout, 7001, "operation timeout", nil)
	ErrNotFound     = NewEtcdError(ErrTypeNotFound, 7002, "resource not found", nil)
	ErrInvalidState = NewEtcdError(ErrTypeInvalidState, 7003, "invalid state", nil)
)

// 错误包装函数

// WrapConnectionError 包装连接错误
func WrapConnectionError(err error, message string) error {
	return NewEtcdError(ErrTypeConnection, 1000, message, err)
}

// WrapRegistryError 包装注册错误
func WrapRegistryError(err error, message string) error {
	return NewEtcdError(ErrTypeRegistry, 2000, message, err)
}

// WrapDiscoveryError 包装发现错误
func WrapDiscoveryError(err error, message string) error {
	return NewEtcdError(ErrTypeDiscovery, 3000, message, err)
}

// WrapLockError 包装锁错误
func WrapLockError(err error, message string) error {
	return NewEtcdError(ErrTypeLock, 4000, message, err)
}

// WrapLeaseError 包装租约错误
func WrapLeaseError(err error, message string) error {
	return NewEtcdError(ErrTypeLease, 5000, message, err)
}

// WrapConfigurationError 包装配置错误
func WrapConfigurationError(err error, message string) error {
	return NewEtcdError(ErrTypeConfiguration, 6000, message, err)
}

// 错误检查函数

// IsConnectionError 检查是否为连接错误
func IsConnectionError(err error) bool {
	var etcdErr *EtcdError
	return errors.As(err, &etcdErr) && etcdErr.Type == ErrTypeConnection
}

// IsRegistryError 检查是否为注册错误
func IsRegistryError(err error) bool {
	var etcdErr *EtcdError
	return errors.As(err, &etcdErr) && etcdErr.Type == ErrTypeRegistry
}

// IsDiscoveryError 检查是否为发现错误
func IsDiscoveryError(err error) bool {
	var etcdErr *EtcdError
	return errors.As(err, &etcdErr) && etcdErr.Type == ErrTypeDiscovery
}

// IsLockError 检查是否为锁错误
func IsLockError(err error) bool {
	var etcdErr *EtcdError
	return errors.As(err, &etcdErr) && etcdErr.Type == ErrTypeLock
}

// IsLeaseError 检查是否为租约错误
func IsLeaseError(err error) bool {
	var etcdErr *EtcdError
	return errors.As(err, &etcdErr) && etcdErr.Type == ErrTypeLease
}

// IsTimeoutError 检查是否为超时错误
func IsTimeoutError(err error) bool {
	var etcdErr *EtcdError
	return errors.As(err, &etcdErr) && etcdErr.Type == ErrTypeTimeout
}

// IsNotFoundError 检查是否为未找到错误
func IsNotFoundError(err error) bool {
	var etcdErr *EtcdError
	return errors.As(err, &etcdErr) && etcdErr.Type == ErrTypeNotFound
}

// IsConfigurationError 检查是否为配置错误
func IsConfigurationError(err error) bool {
	var etcdErr *EtcdError
	return errors.As(err, &etcdErr) && etcdErr.Type == ErrTypeConfiguration
}

// 错误恢复策略

// IsRetryableError 检查错误是否可重试
func IsRetryableError(err error) bool {
	if IsConnectionError(err) || IsTimeoutError(err) {
		return true
	}

	var etcdErr *EtcdError
	if errors.As(err, &etcdErr) {
		// 某些特定的错误码可以重试
		retryableCodes := []int{1002, 1003, 7001} // 连接丢失、连接超时、操作超时
		for _, code := range retryableCodes {
			if etcdErr.Code == code {
				return true
			}
		}
	}

	return false
}

// GetRetryDelay 获取重试延迟时间（毫秒）
func GetRetryDelay(attempt int) int {
	// 指数退避策略：100ms, 200ms, 400ms, 800ms, 1600ms, 最大3200ms
	delay := 100 * (1 << attempt)
	if delay > 3200 {
		delay = 3200
	}
	return delay
}
