package client

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	clientv3 "go.etcd.io/etcd/client/v3"
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

// EtcdClient etcd 客户端封装
type EtcdClient struct {
	client      *clientv3.Client
	retryConfig *RetryConfig
	logger      clog.Logger
}

// NewEtcdClient 创建新的 etcd 客户端
func NewEtcdClient(opts CoordinatorOptions) (*EtcdClient, error) {
	// 验证选项
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// 创建 etcd 客户端配置
	config := clientv3.Config{
		Endpoints:   opts.Endpoints,
		DialTimeout: opts.Timeout,
		Username:    opts.Username,
		Password:    opts.Password,
	}

	// 创建 etcd 客户端
	client, err := clientv3.New(config)
	if err != nil {
		return nil, NewCoordinationError(
			ErrCodeConnection,
			"failed to create etcd client",
			err,
		)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	if _, err := client.Status(ctx, opts.Endpoints[0]); err != nil {
		client.Close()
		return nil, NewCoordinationError(
			ErrCodeConnection,
			"failed to connect to etcd",
			err,
		)
	}

	logger := clog.Module("coordination.client")
	logger.Info("etcd client created successfully",
		clog.Strings("endpoints", opts.Endpoints))

	return &EtcdClient{
		client:      client,
		retryConfig: opts.RetryConfig,
		logger:      logger,
	}, nil
}

// Client 获取原始的 etcd 客户端
func (c *EtcdClient) Client() *clientv3.Client {
	return c.client
}

// Close 关闭客户端
func (c *EtcdClient) Close() error {
	if c.client != nil {
		err := c.client.Close()
		if err != nil {
			c.logger.Error("failed to close etcd client", clog.Err(err))
			return NewCoordinationError(
				ErrCodeConnection,
				"failed to close etcd client",
				err,
			)
		}
		c.logger.Info("etcd client closed successfully")
	}
	return nil
}

// Ping 检查连接状态
func (c *EtcdClient) Ping(ctx context.Context) error {
	return c.retryOperation(ctx, func() error {
		_, err := c.client.Status(ctx, c.client.Endpoints()[0])
		if err != nil {
			return NewCoordinationError(
				ErrCodeConnection,
				"etcd ping failed",
				err,
			)
		}
		return nil
	})
}

// retryOperation 执行重试操作
func (c *EtcdClient) retryOperation(ctx context.Context, operation func() error) error {
	if c.retryConfig == nil || c.retryConfig.MaxAttempts <= 1 {
		return operation()
	}

	var lastErr error
	delay := c.retryConfig.InitialDelay

	for attempt := 0; attempt < c.retryConfig.MaxAttempts; attempt++ {
		if err := operation(); err == nil {
			if attempt > 0 {
				c.logger.Info("operation succeeded after retry",
					clog.Int("attempt", attempt+1))
			}
			return nil
		} else {
			lastErr = err
			c.logger.Warn("operation failed, will retry",
				clog.Int("attempt", attempt+1),
				clog.Int("max_attempts", c.retryConfig.MaxAttempts),
				clog.Duration("delay", delay),
				clog.Err(err))
		}

		// 如果不是最后一次尝试，则等待后重试
		if attempt < c.retryConfig.MaxAttempts-1 {
			select {
			case <-ctx.Done():
				return NewCoordinationError(
					ErrCodeTimeout,
					"context cancelled during retry",
					ctx.Err(),
				)
			case <-time.After(delay):
			}

			// 计算下一次延迟时间
			delay = time.Duration(float64(delay) * c.retryConfig.Multiplier)
			if delay > c.retryConfig.MaxDelay {
				delay = c.retryConfig.MaxDelay
			}
		}
	}

	c.logger.Error("operation failed after all retries",
		clog.Int("max_attempts", c.retryConfig.MaxAttempts),
		clog.Err(lastErr))

	return lastErr
}

// Put 设置键值对
func (c *EtcdClient) Put(ctx context.Context, key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	var resp *clientv3.PutResponse
	err := c.retryOperation(ctx, func() error {
		var err error
		resp, err = c.client.Put(ctx, key, value, opts...)
		if err != nil {
			return NewCoordinationError(
				ErrCodeConnection,
				"etcd put operation failed",
				err,
			)
		}
		return nil
	})
	return resp, err
}

// Get 获取键值对
func (c *EtcdClient) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	var resp *clientv3.GetResponse
	err := c.retryOperation(ctx, func() error {
		var err error
		resp, err = c.client.Get(ctx, key, opts...)
		if err != nil {
			return NewCoordinationError(
				ErrCodeConnection,
				"etcd get operation failed",
				err,
			)
		}
		return nil
	})
	return resp, err
}

// Delete 删除键值对
func (c *EtcdClient) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	var resp *clientv3.DeleteResponse
	err := c.retryOperation(ctx, func() error {
		var err error
		resp, err = c.client.Delete(ctx, key, opts...)
		if err != nil {
			return NewCoordinationError(
				ErrCodeConnection,
				"etcd delete operation failed",
				err,
			)
		}
		return nil
	})
	return resp, err
}

// Watch 监听键变化
func (c *EtcdClient) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return c.client.Watch(ctx, key, opts...)
}

// Grant 创建租约
func (c *EtcdClient) Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error) {
	var resp *clientv3.LeaseGrantResponse
	err := c.retryOperation(ctx, func() error {
		var err error
		resp, err = c.client.Grant(ctx, ttl)
		if err != nil {
			return NewCoordinationError(
				ErrCodeConnection,
				"etcd grant operation failed",
				err,
			)
		}
		return nil
	})
	return resp, err
}

// KeepAlive 保持租约活跃
func (c *EtcdClient) KeepAlive(ctx context.Context, id clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	ch, err := c.client.KeepAlive(ctx, id)
	if err != nil {
		return nil, NewCoordinationError(
			ErrCodeConnection,
			"etcd keep alive failed",
			err,
		)
	}
	return ch, nil
}

// Revoke 撤销租约
func (c *EtcdClient) Revoke(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
	var resp *clientv3.LeaseRevokeResponse
	err := c.retryOperation(ctx, func() error {
		var err error
		resp, err = c.client.Revoke(ctx, id)
		if err != nil {
			return NewCoordinationError(
				ErrCodeConnection,
				"etcd revoke operation failed",
				err,
			)
		}
		return nil
	})
	return resp, err
}

// Txn 创建事务
func (c *EtcdClient) Txn(ctx context.Context) clientv3.Txn {
	return c.client.Txn(ctx)
}
