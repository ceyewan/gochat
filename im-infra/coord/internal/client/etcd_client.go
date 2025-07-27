package client

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"errors"
	"github.com/ceyewan/gochat/im-infra/clog"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// ============================================================================
// 配置相关类型定义
// ============================================================================

// Config etcd 客户端配置选项
type Config struct {
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

	// Logger 可选的日志记录器
	Logger clog.Logger `json:"-"`
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

// ============================================================================
// 错误处理相关类型定义
// ============================================================================

// ErrorCode 错误码定义
type ErrorCode string

const (
	ErrCodeConnection  ErrorCode = "CONNECTION_ERROR"
	ErrCodeTimeout     ErrorCode = "TIMEOUT_ERROR"
	ErrCodeNotFound    ErrorCode = "NOT_FOUND"
	ErrCodeConflict    ErrorCode = "CONFLICT"
	ErrCodeValidation  ErrorCode = "VALIDATION_ERROR"
	ErrCodeUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// Error 协调器错误类型
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Cause   error     `json:"cause,omitempty"`
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 支持 Go 1.13+ 的错误包装
func (e *Error) Unwrap() error {
	return e.Cause
}

// NewError 创建协调器错误
func NewError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// ============================================================================
// 配置验证
// ============================================================================

// Validate 验证配置选项有效性
func (cfg *Config) Validate() error {
	if len(cfg.Endpoints) == 0 {
		return NewError(ErrCodeValidation, "endpoints cannot be empty", nil)
	}

	for _, endpoint := range cfg.Endpoints {
		if !isValidEndpoint(endpoint) {
			return NewError(ErrCodeValidation, "invalid endpoint format", nil)
		}
	}

	if cfg.Timeout <= 0 {
		return NewError(ErrCodeValidation, "timeout must be positive", nil)
	}

	if cfg.RetryConfig != nil {
		return cfg.RetryConfig.validate()
	}

	return nil
}

// isValidEndpoint 判断是否为合法的 endpoint 格式，格式为 host:port
func isValidEndpoint(endpoint string) bool {
	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return false
	}
	_ = host
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return false
	}
	if port < 0 || port > 65535 {
		return false
	}
	return true
}

// validate 验证重试配置
func (rc *RetryConfig) validate() error {
	if rc.MaxAttempts < 0 {
		return NewError(ErrCodeValidation, "max_attempts cannot be negative", nil)
	}

	if rc.InitialDelay <= 0 {
		return NewError(ErrCodeValidation, "initial_delay must be positive", nil)
	}

	if rc.MaxDelay <= 0 {
		return NewError(ErrCodeValidation, "max_delay must be positive", nil)
	}

	if rc.Multiplier <= 1.0 {
		return NewError(ErrCodeValidation, "multiplier must be greater than 1.0", nil)
	}

	return nil
}

// ============================================================================
// EtcdClient 主要实现
// ============================================================================

// EtcdClient etcd 客户端封装，提供重试机制和错误处理
type EtcdClient struct {
	client      *clientv3.Client
	retryConfig *RetryConfig
	logger      clog.Logger
}

// New 创建新的 etcd 客户端
func New(cfg Config) (*EtcdClient, error) {
	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 创建 etcd 客户端
	client, err := createEtcdClient(cfg)
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := testConnection(client, cfg); err != nil {
		client.Close()
		return nil, err
	}

	var logger clog.Logger
	if cfg.Logger != nil {
		logger = cfg.Logger
	} else {
		logger = clog.Module("coordination.client")
	}

	logger.Info("etcd client created successfully",
		clog.Strings("endpoints", cfg.Endpoints))

	return &EtcdClient{
		client:      client,
		retryConfig: cfg.RetryConfig,
		logger:      logger,
	}, nil
}

// createEtcdClient 创建原始的 etcd 客户端
func createEtcdClient(cfg Config) (*clientv3.Client, error) {
	config := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.Timeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	}

	client, err := clientv3.New(config)
	if err != nil {
		return nil, NewError(ErrCodeConnection, "failed to create etcd client", err)
	}

	return client, nil
}

// testConnection 测试 etcd 连接
func testConnection(client *clientv3.Client, cfg Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if _, err := client.Status(ctx, cfg.Endpoints[0]); err != nil {
		return NewError(ErrCodeConnection, "failed to connect to etcd", err)
	}

	return nil
}

// ============================================================================
// 客户端基础方法
// ============================================================================

// Client 获取原始的 etcd 客户端
func (c *EtcdClient) Client() *clientv3.Client {
	return c.client
}

// Close 关闭客户端连接
func (c *EtcdClient) Close() error {
	if c.client == nil {
		return nil
	}

	if err := c.client.Close(); err != nil {
		c.logger.Error("failed to close etcd client", clog.Err(err))
		return NewError(ErrCodeConnection, "failed to close etcd client", err)
	}

	c.logger.Info("etcd client closed successfully")
	return nil
}

// Ping 检查 etcd 连接状态
func (c *EtcdClient) Ping(ctx context.Context) error {
	return c.executeWithRetry(ctx, func() error {
		// client.Sync() 会与集群的一个健康节点同步 revision，是更可靠的健康检查
		if err := c.client.Sync(ctx); err != nil {
			return NewError(ErrCodeConnection, "etcd ping failed", err)
		}
		return nil
	})
}

// ============================================================================
// 重试机制实现
// ============================================================================

// executeWithRetry 执行带重试的操作
func (c *EtcdClient) executeWithRetry(ctx context.Context, operation func() error) error {
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

			// 检查是否为不应该重试的错误
			if c.shouldNotRetry(err) {
				return err
			}

			c.logger.Warn("operation failed, will retry",
				clog.Int("attempt", attempt+1),
				clog.Int("max_attempts", c.retryConfig.MaxAttempts),
				clog.Duration("delay", delay),
				clog.Err(err))
		}

		// 如果不是最后一次尝试，则等待后重试
		if attempt < c.retryConfig.MaxAttempts-1 {
			if err := c.waitForRetry(ctx, delay); err != nil {
				return err
			}

			// 计算下一次延迟时间（指数退避）
			delay = c.calculateNextDelay(delay)
		}
	}

	c.logger.Error("operation failed after all retries",
		clog.Int("max_attempts", c.retryConfig.MaxAttempts),
		clog.Err(lastErr))

	return lastErr
}

// waitForRetry 等待重试延迟
func (c *EtcdClient) waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return NewError(ErrCodeTimeout, "context cancelled during retry", ctx.Err())
	case <-timer.C:
		return nil
	}
}

// calculateNextDelay 计算下一次重试的延迟时间
func (c *EtcdClient) calculateNextDelay(currentDelay time.Duration) time.Duration {
	nextDelay := time.Duration(float64(currentDelay) * c.retryConfig.Multiplier)
	if nextDelay > c.retryConfig.MaxDelay {
		return c.retryConfig.MaxDelay
	}
	return nextDelay
}

// shouldNotRetry 检查是否不应该重试的错误
func (c *EtcdClient) shouldNotRetry(err error) bool {
	if coordErr, ok := err.(*Error); ok {
		// 对于 NotFound 和 Validation 错误，不应该重试
		return coordErr.Code == ErrCodeNotFound || coordErr.Code == ErrCodeValidation
	}
	return false
}

// ============================================================================
// etcd 基础操作封装
// ============================================================================

// Put 设置键值对
func (c *EtcdClient) Put(ctx context.Context, key, value string, cfg ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	var resp *clientv3.PutResponse
	err := c.executeWithRetry(ctx, func() error {
		var err error
		resp, err = c.client.Put(ctx, key, value, cfg...)
		if err != nil {
			return NewError(ErrCodeConnection, "etcd put operation failed", err)
		}
		return nil
	})
	return resp, err
}

// Get 获取键值对
func (c *EtcdClient) Get(ctx context.Context, key string, cfg ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	var resp *clientv3.GetResponse
	err := c.executeWithRetry(ctx, func() error {
		var err error
		resp, err = c.client.Get(ctx, key, cfg...)
		if err != nil {
			return NewError(ErrCodeConnection, "etcd get operation failed", err)
		}
		return nil
	})
	return resp, err
}

// Delete 删除键值对
func (c *EtcdClient) Delete(ctx context.Context, key string, cfg ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	var resp *clientv3.DeleteResponse
	err := c.executeWithRetry(ctx, func() error {
		var err error
		resp, err = c.client.Delete(ctx, key, cfg...)
		if err != nil {
			return NewError(ErrCodeConnection, "etcd delete operation failed", err)
		}
		return nil
	})
	return resp, err
}

// Watch 监听键变化（不需要重试机制）
func (c *EtcdClient) Watch(ctx context.Context, key string, cfg ...clientv3.OpOption) clientv3.WatchChan {
	return c.client.Watch(ctx, key, cfg...)
}

// ============================================================================
// 租约操作封装
// ============================================================================

// Grant 创建租约
func (c *EtcdClient) Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error) {
	var resp *clientv3.LeaseGrantResponse
	err := c.executeWithRetry(ctx, func() error {
		var err error
		resp, err = c.client.Grant(ctx, ttl)
		if err != nil {
			return NewError(ErrCodeConnection, "etcd grant operation failed", err)
		}
		return nil
	})
	return resp, err
}

// KeepAlive 保持租约活跃（不需要重试机制）
func (c *EtcdClient) KeepAlive(ctx context.Context, id clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	ch, err := c.client.KeepAlive(ctx, id)
	if err != nil {
		return nil, NewError(ErrCodeConnection, "etcd keep alive failed", err)
	}
	return ch, nil
}

// Revoke 撤销租约
func (c *EtcdClient) Revoke(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
	var resp *clientv3.LeaseRevokeResponse
	err := c.executeWithRetry(ctx, func() error {
		var err error
		resp, err = c.client.Revoke(ctx, id)
		if err != nil {
			// 如果租约不存在，这是正常情况，不需要重试
			if c.isLeaseNotFoundError(err) {
				return NewError(ErrCodeNotFound, "lease not found (already expired)", err)
			}
			return NewError(ErrCodeConnection, "etcd revoke operation failed", err)
		}
		return nil
	})
	return resp, err
}

// isLeaseNotFoundError 检查是否为租约未找到错误
func (c *EtcdClient) isLeaseNotFoundError(err error) bool {
	return errors.Is(err, rpctypes.ErrLeaseNotFound)
}

// Txn 创建事务（不需要重试机制）
func (c *EtcdClient) Txn(ctx context.Context) clientv3.Txn {
	return c.client.Txn(ctx)
}
