package etcd

import (
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ManagerOptions 管理器配置选项
type ManagerOptions struct {
	// etcd 连接配置
	Endpoints   []string      `json:"endpoints"`
	DialTimeout time.Duration `json:"dial_timeout"`
	Username    string        `json:"username,omitempty"`
	Password    string        `json:"password,omitempty"`

	// TLS 配置
	TLSConfig *TLSConfig `json:"tls_config,omitempty"`

	// 日志配置
	Logger Logger `json:"-"`

	// 重试配置
	RetryConfig *RetryConfig `json:"retry_config,omitempty"`

	// 服务注册默认配置
	DefaultTTL      int64             `json:"default_ttl"`
	ServicePrefix   string            `json:"service_prefix"`
	LockPrefix      string            `json:"lock_prefix"`
	DefaultMetadata map[string]string `json:"default_metadata,omitempty"`

	// 连接池配置
	MaxIdleConns    int           `json:"max_idle_conns"`
	MaxActiveConns  int           `json:"max_active_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`

	// 健康检查配置
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	CertFile   string `json:"cert_file,omitempty"`
	KeyFile    string `json:"key_file,omitempty"`
	CAFile     string `json:"ca_file,omitempty"`
	ServerName string `json:"server_name,omitempty"`
	Insecure   bool   `json:"insecure"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	InitialInterval time.Duration `json:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval"`
	Multiplier      float64       `json:"multiplier"`
}

// Logger 日志接口
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// DefaultLogger 默认日志实现
type DefaultLogger struct {
	*log.Logger
}

// Debug 实现 Logger 接口
func (l *DefaultLogger) Debug(args ...interface{}) {
	l.Logger.Println(append([]interface{}{"[DEBUG]"}, args...)...)
}

// Info 实现 Logger 接口
func (l *DefaultLogger) Info(args ...interface{}) {
	l.Logger.Println(append([]interface{}{"[INFO]"}, args...)...)
}

// Warn 实现 Logger 接口
func (l *DefaultLogger) Warn(args ...interface{}) {
	l.Logger.Println(append([]interface{}{"[WARN]"}, args...)...)
}

// Error 实现 Logger 接口
func (l *DefaultLogger) Error(args ...interface{}) {
	l.Logger.Println(append([]interface{}{"[ERROR]"}, args...)...)
}

// Debugf 实现 Logger 接口
func (l *DefaultLogger) Debugf(format string, args ...interface{}) {
	l.Logger.Printf("[DEBUG] "+format, args...)
}

// Infof 实现 Logger 接口
func (l *DefaultLogger) Infof(format string, args ...interface{}) {
	l.Logger.Printf("[INFO] "+format, args...)
}

// Warnf 实现 Logger 接口
func (l *DefaultLogger) Warnf(format string, args ...interface{}) {
	l.Logger.Printf("[WARN] "+format, args...)
}

// Errorf 实现 Logger 接口
func (l *DefaultLogger) Errorf(format string, args ...interface{}) {
	l.Logger.Printf("[ERROR] "+format, args...)
}

// DefaultManagerOptions 返回默认的管理器选项
func DefaultManagerOptions() *ManagerOptions {
	// 创建 etcd 模块的 clog 日志器
	etcdLogger := clog.Default().WithGroup("etcd")

	return &ManagerOptions{
		Endpoints:   []string{"localhost:23791", "localhost:23792", "localhost:23793"},
		DialTimeout: 5 * time.Second,
		Logger:      NewClogAdapter(etcdLogger),
		RetryConfig: &RetryConfig{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     3 * time.Second,
			Multiplier:      2.0,
		},
		DefaultTTL:          30,
		ServicePrefix:       "/services",
		LockPrefix:          "/locks",
		MaxIdleConns:        10,
		MaxActiveConns:      100,
		ConnMaxLifetime:     30 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}
}

// ManagerBuilder 管理器建造者
type ManagerBuilder struct {
	options *ManagerOptions
}

// NewManagerBuilder 创建新的管理器建造者
func NewManagerBuilder() *ManagerBuilder {
	return &ManagerBuilder{
		options: DefaultManagerOptions(),
	}
}

// WithEndpoints 设置 etcd 端点
func (b *ManagerBuilder) WithEndpoints(endpoints []string) *ManagerBuilder {
	b.options.Endpoints = endpoints
	return b
}

// WithDialTimeout 设置连接超时
func (b *ManagerBuilder) WithDialTimeout(timeout time.Duration) *ManagerBuilder {
	b.options.DialTimeout = timeout
	return b
}

// WithAuth 设置认证信息
func (b *ManagerBuilder) WithAuth(username, password string) *ManagerBuilder {
	b.options.Username = username
	b.options.Password = password
	return b
}

// WithTLS 设置 TLS 配置
func (b *ManagerBuilder) WithTLS(config *TLSConfig) *ManagerBuilder {
	b.options.TLSConfig = config
	return b
}

// WithLogger 设置日志器
func (b *ManagerBuilder) WithLogger(logger Logger) *ManagerBuilder {
	b.options.Logger = logger
	return b
}

// WithRetryConfig 设置重试配置
func (b *ManagerBuilder) WithRetryConfig(config *RetryConfig) *ManagerBuilder {
	b.options.RetryConfig = config
	return b
}

// WithDefaultTTL 设置默认 TTL
func (b *ManagerBuilder) WithDefaultTTL(ttl int64) *ManagerBuilder {
	b.options.DefaultTTL = ttl
	return b
}

// WithServicePrefix 设置服务前缀
func (b *ManagerBuilder) WithServicePrefix(prefix string) *ManagerBuilder {
	b.options.ServicePrefix = prefix
	return b
}

// WithLockPrefix 设置锁前缀
func (b *ManagerBuilder) WithLockPrefix(prefix string) *ManagerBuilder {
	b.options.LockPrefix = prefix
	return b
}

// WithDefaultMetadata 设置默认元数据
func (b *ManagerBuilder) WithDefaultMetadata(metadata map[string]string) *ManagerBuilder {
	b.options.DefaultMetadata = metadata
	return b
}

// WithConnectionPool 设置连接池配置
func (b *ManagerBuilder) WithConnectionPool(maxIdle, maxActive int, maxLifetime time.Duration) *ManagerBuilder {
	b.options.MaxIdleConns = maxIdle
	b.options.MaxActiveConns = maxActive
	b.options.ConnMaxLifetime = maxLifetime
	return b
}

// WithHealthCheck 设置健康检查配置
func (b *ManagerBuilder) WithHealthCheck(interval, timeout time.Duration) *ManagerBuilder {
	b.options.HealthCheckInterval = interval
	b.options.HealthCheckTimeout = timeout
	return b
}

// Build 构建管理器
func (b *ManagerBuilder) Build() (EtcdManager, error) {
	// 验证配置
	if err := b.validateOptions(); err != nil {
		return nil, WrapConfigurationError(err, "invalid manager options")
	}

	// 创建管理器实例
	return NewEtcdManager(b.options)
}

// validateOptions 验证配置选项
func (b *ManagerBuilder) validateOptions() error {
	if len(b.options.Endpoints) == 0 {
		return ErrMissingEndpoints
	}

	if b.options.DialTimeout <= 0 {
		return ErrInvalidTimeout
	}

	if b.options.DefaultTTL <= 0 {
		b.options.DefaultTTL = 30
	}

	if b.options.RetryConfig != nil && b.options.RetryConfig.MaxRetries < 0 {
		b.options.RetryConfig.MaxRetries = 0
	}

	return nil
}

// 注册选项函数

// WithTTL 设置注册 TTL
func WithTTL(ttl int64) RegisterOption {
	return func(opts *RegisterOptions) {
		opts.TTL = ttl
	}
}

// WithMetadata 设置注册元数据
func WithMetadata(metadata map[string]string) RegisterOption {
	return func(opts *RegisterOptions) {
		opts.Metadata = metadata
	}
}

// WithLeaseID 设置租约 ID
func WithLeaseID(leaseID clientv3.LeaseID) RegisterOption {
	return func(opts *RegisterOptions) {
		opts.LeaseID = leaseID
	}
}

// 发现选项函数

// WithLoadBalancer 设置负载均衡策略
func WithLoadBalancer(lb string) DiscoveryOption {
	return func(opts *DiscoveryOptions) {
		opts.LoadBalancer = lb
	}
}

// WithDiscoveryTimeout 设置发现超时
func WithDiscoveryTimeout(timeout time.Duration) DiscoveryOption {
	return func(opts *DiscoveryOptions) {
		opts.Timeout = timeout
	}
}

// WithDiscoveryMetadata 设置发现元数据过滤
func WithDiscoveryMetadata(metadata map[string]string) DiscoveryOption {
	return func(opts *DiscoveryOptions) {
		opts.Metadata = metadata
	}
}

// ClogAdapter 将 clog.Logger 适配为 etcd.Logger 接口
type ClogAdapter struct {
	logger clog.Logger
}

// NewClogAdapter 创建新的 clog 适配器
func NewClogAdapter(logger clog.Logger) Logger {
	return &ClogAdapter{logger: logger}
}

// Debug 实现 Logger 接口
func (c *ClogAdapter) Debug(args ...interface{}) {
	c.logger.Debug("etcd debug", clog.Any("args", args))
}

// Info 实现 Logger 接口
func (c *ClogAdapter) Info(args ...interface{}) {
	c.logger.Info("etcd info", clog.Any("args", args))
}

// Warn 实现 Logger 接口
func (c *ClogAdapter) Warn(args ...interface{}) {
	c.logger.Warn("etcd warn", clog.Any("args", args))
}

// Error 实现 Logger 接口
func (c *ClogAdapter) Error(args ...interface{}) {
	c.logger.Error("etcd error", clog.Any("args", args))
}
