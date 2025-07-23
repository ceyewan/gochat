package internal

import (
	"fmt"
	"time"
)

// Config 是 coordination 的主配置结构体。
// 用于声明式地定义分布式协调组件的行为和连接参数。
type Config struct {
	// Endpoints etcd 服务器地址列表
	Endpoints []string `json:"endpoints" yaml:"endpoints"`

	// DialTimeout 连接超时时间
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dial_timeout"`

	// Username etcd 用户名（可选）
	Username string `json:"username,omitempty" yaml:"username,omitempty"`

	// Password etcd 密码（可选）
	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	// TLS TLS 配置（可选）
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`

	// ServiceRegistry 服务注册与发现配置
	ServiceRegistry ServiceRegistryConfig `json:"service_registry" yaml:"service_registry"`

	// DistributedLock 分布式锁配置
	DistributedLock DistributedLockConfig `json:"distributed_lock" yaml:"distributed_lock"`

	// ConfigCenter 配置中心配置
	ConfigCenter ConfigCenterConfig `json:"config_center" yaml:"config_center"`

	// Retry 重试策略配置
	Retry *RetryConfig `json:"retry,omitempty" yaml:"retry,omitempty"`

	// LogLevel 日志级别
	LogLevel string `json:"log_level" yaml:"log_level"`

	// EnableMetrics 是否启用指标收集
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`

	// EnableTracing 是否启用链路追踪
	EnableTracing bool `json:"enable_tracing" yaml:"enable_tracing"`
}

// TLSConfig 定义 TLS 配置。
type TLSConfig struct {
	// CertFile 客户端证书文件路径
	CertFile string `json:"cert_file" yaml:"cert_file"`

	// KeyFile 客户端私钥文件路径
	KeyFile string `json:"key_file" yaml:"key_file"`

	// CAFile CA 证书文件路径
	CAFile string `json:"ca_file" yaml:"ca_file"`

	// InsecureSkipVerify 是否跳过证书验证
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

// ServiceRegistryConfig 定义服务注册与发现的配置。
type ServiceRegistryConfig struct {
	// KeyPrefix 服务注册的键前缀
	KeyPrefix string `json:"key_prefix" yaml:"key_prefix"`

	// TTL 服务注册的生存时间
	TTL time.Duration `json:"ttl" yaml:"ttl"`

	// HealthCheckInterval 健康检查间隔
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"`

	// EnableHealthCheck 是否启用健康检查
	EnableHealthCheck bool `json:"enable_health_check" yaml:"enable_health_check"`

	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// RetryInterval 重试间隔
	RetryInterval time.Duration `json:"retry_interval" yaml:"retry_interval"`
}

// DistributedLockConfig 定义分布式锁的配置。
type DistributedLockConfig struct {
	// KeyPrefix 锁的键前缀
	KeyPrefix string `json:"key_prefix" yaml:"key_prefix"`

	// DefaultTTL 默认锁的生存时间
	DefaultTTL time.Duration `json:"default_ttl" yaml:"default_ttl"`

	// RenewInterval 锁续期间隔
	RenewInterval time.Duration `json:"renew_interval" yaml:"renew_interval"`

	// EnableReentrant 是否启用可重入锁
	EnableReentrant bool `json:"enable_reentrant" yaml:"enable_reentrant"`

	// SessionTTL 会话生存时间
	SessionTTL time.Duration `json:"session_ttl" yaml:"session_ttl"`

	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// RetryInterval 重试间隔
	RetryInterval time.Duration `json:"retry_interval" yaml:"retry_interval"`
}

// ConfigCenterConfig 定义配置中心的配置。
type ConfigCenterConfig struct {
	// KeyPrefix 配置的键前缀
	KeyPrefix string `json:"key_prefix" yaml:"key_prefix"`

	// EnableVersioning 是否启用版本控制
	EnableVersioning bool `json:"enable_versioning" yaml:"enable_versioning"`

	// MaxVersionHistory 最大版本历史数量
	MaxVersionHistory int `json:"max_version_history" yaml:"max_version_history"`

	// EnableValidation 是否启用配置验证
	EnableValidation bool `json:"enable_validation" yaml:"enable_validation"`

	// WatchBufferSize 监听缓冲区大小
	WatchBufferSize int `json:"watch_buffer_size" yaml:"watch_buffer_size"`

	// EnableCompression 是否启用压缩
	EnableCompression bool `json:"enable_compression" yaml:"enable_compression"`
}

// RetryConfig 定义重试策略配置。
type RetryConfig struct {
	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// InitialInterval 初始重试间隔
	InitialInterval time.Duration `json:"initial_interval" yaml:"initial_interval"`

	// MaxInterval 最大重试间隔
	MaxInterval time.Duration `json:"max_interval" yaml:"max_interval"`

	// Multiplier 重试间隔倍数
	Multiplier float64 `json:"multiplier" yaml:"multiplier"`

	// RandomizationFactor 随机化因子
	RandomizationFactor float64 `json:"randomization_factor" yaml:"randomization_factor"`
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("endpoints cannot be empty")
	}

	if c.DialTimeout <= 0 {
		return fmt.Errorf("dial_timeout must be positive")
	}

	if err := c.ServiceRegistry.Validate(); err != nil {
		return fmt.Errorf("service_registry config invalid: %w", err)
	}

	if err := c.DistributedLock.Validate(); err != nil {
		return fmt.Errorf("distributed_lock config invalid: %w", err)
	}

	if err := c.ConfigCenter.Validate(); err != nil {
		return fmt.Errorf("config_center config invalid: %w", err)
	}

	if c.Retry != nil {
		if err := c.Retry.Validate(); err != nil {
			return fmt.Errorf("retry config invalid: %w", err)
		}
	}

	return nil
}

// Validate 验证服务注册配置的有效性
func (c *ServiceRegistryConfig) Validate() error {
	if c.KeyPrefix == "" {
		return fmt.Errorf("key_prefix cannot be empty")
	}

	if c.TTL <= 0 {
		return fmt.Errorf("ttl must be positive")
	}

	if c.EnableHealthCheck && c.HealthCheckInterval <= 0 {
		return fmt.Errorf("health_check_interval must be positive when health check is enabled")
	}

	return nil
}

// Validate 验证分布式锁配置的有效性
func (c *DistributedLockConfig) Validate() error {
	if c.KeyPrefix == "" {
		return fmt.Errorf("key_prefix cannot be empty")
	}

	if c.DefaultTTL <= 0 {
		return fmt.Errorf("default_ttl must be positive")
	}

	if c.RenewInterval <= 0 {
		return fmt.Errorf("renew_interval must be positive")
	}

	if c.SessionTTL <= 0 {
		return fmt.Errorf("session_ttl must be positive")
	}

	return nil
}

// Validate 验证配置中心配置的有效性
func (c *ConfigCenterConfig) Validate() error {
	if c.KeyPrefix == "" {
		return fmt.Errorf("key_prefix cannot be empty")
	}

	if c.EnableVersioning && c.MaxVersionHistory <= 0 {
		return fmt.Errorf("max_version_history must be positive when versioning is enabled")
	}

	if c.WatchBufferSize <= 0 {
		return fmt.Errorf("watch_buffer_size must be positive")
	}

	return nil
}

// Validate 验证重试配置的有效性
func (c *RetryConfig) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}

	if c.InitialInterval <= 0 {
		return fmt.Errorf("initial_interval must be positive")
	}

	if c.MaxInterval <= 0 {
		return fmt.Errorf("max_interval must be positive")
	}

	if c.Multiplier <= 1.0 {
		return fmt.Errorf("multiplier must be greater than 1.0")
	}

	if c.RandomizationFactor < 0 || c.RandomizationFactor > 1 {
		return fmt.Errorf("randomization_factor must be between 0 and 1")
	}

	return nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Endpoints:   []string{"localhost:23791"},
		DialTimeout: 3 * time.Second, // 减少连接超时时间，快速失败
		ServiceRegistry: ServiceRegistryConfig{
			KeyPrefix:           "/services",
			TTL:                 30 * time.Second,
			HealthCheckInterval: 10 * time.Second,
			EnableHealthCheck:   true,
			MaxRetries:          3,
			RetryInterval:       time.Second,
		},
		DistributedLock: DistributedLockConfig{
			KeyPrefix:       "/locks",
			DefaultTTL:      30 * time.Second,
			RenewInterval:   10 * time.Second,
			EnableReentrant: true,
			SessionTTL:      60 * time.Second,
			MaxRetries:      3,
			RetryInterval:   time.Second,
		},
		ConfigCenter: ConfigCenterConfig{
			KeyPrefix:         "/config",
			EnableVersioning:  true,
			MaxVersionHistory: 100,
			EnableValidation:  true,
			WatchBufferSize:   100,
			EnableCompression: false,
		},
		Retry: &RetryConfig{
			MaxRetries:          2, // 减少重试次数，快速失败
			InitialInterval:     100 * time.Millisecond,
			MaxInterval:         2 * time.Second, // 减少最大重试间隔
			Multiplier:          1.5,             // 减少倍数
			RandomizationFactor: 0.1,
		},
		LogLevel:      "info",
		EnableMetrics: false,
		EnableTracing: false,
	}
}

// DevelopmentConfig 返回适用于开发环境的配置
func DevelopmentConfig() Config {
	cfg := DefaultConfig()
	cfg.LogLevel = "debug"
	cfg.ServiceRegistry.TTL = 15 * time.Second
	cfg.ServiceRegistry.HealthCheckInterval = 5 * time.Second
	cfg.DistributedLock.DefaultTTL = 15 * time.Second
	cfg.DistributedLock.RenewInterval = 5 * time.Second
	cfg.ConfigCenter.MaxVersionHistory = 50
	return cfg
}

// ProductionConfig 返回适用于生产环境的配置
func ProductionConfig() Config {
	cfg := DefaultConfig()
	cfg.Endpoints = []string{"etcd-1:2379", "etcd-2:2379", "etcd-3:2379"}
	cfg.LogLevel = "warn"
	cfg.ServiceRegistry.TTL = 60 * time.Second
	cfg.ServiceRegistry.HealthCheckInterval = 20 * time.Second
	cfg.DistributedLock.DefaultTTL = 60 * time.Second
	cfg.DistributedLock.RenewInterval = 20 * time.Second
	cfg.ConfigCenter.MaxVersionHistory = 200
	cfg.ConfigCenter.EnableCompression = true
	cfg.EnableMetrics = true
	cfg.EnableTracing = true
	return cfg
}

// TestConfig 返回适用于测试环境的配置
func TestConfig() Config {
	cfg := DefaultConfig()
	cfg.LogLevel = "debug"
	cfg.DialTimeout = 1 * time.Second // 测试环境快速失败
	cfg.ServiceRegistry.TTL = 5 * time.Second
	cfg.ServiceRegistry.HealthCheckInterval = 2 * time.Second
	cfg.DistributedLock.DefaultTTL = 5 * time.Second
	cfg.DistributedLock.RenewInterval = 2 * time.Second
	cfg.ConfigCenter.MaxVersionHistory = 10
	cfg.ConfigCenter.WatchBufferSize = 10
	cfg.Retry.MaxRetries = 1
	cfg.Retry.InitialInterval = 50 * time.Millisecond
	cfg.Retry.MaxInterval = 500 * time.Millisecond
	return cfg
}

// ExampleConfig 返回适用于示例演示的配置
// 提供快速失败和清晰的错误信息，适合演示和学习
func ExampleConfig() Config {
	cfg := DefaultConfig()
	cfg.LogLevel = "warn"                      // 减少日志噪音
	cfg.DialTimeout = 2 * time.Second          // 快速失败
	cfg.ServiceRegistry.TTL = 15 * time.Second // 较短的 TTL 用于演示
	cfg.ServiceRegistry.HealthCheckInterval = 5 * time.Second
	cfg.DistributedLock.DefaultTTL = 15 * time.Second
	cfg.DistributedLock.RenewInterval = 5 * time.Second
	cfg.ConfigCenter.MaxVersionHistory = 20 // 较少的历史版本
	cfg.Retry.MaxRetries = 1                // 快速失败，不重试
	cfg.Retry.InitialInterval = 100 * time.Millisecond
	cfg.Retry.MaxInterval = 1 * time.Second
	return cfg
}

// NewServiceRegistryConfig 创建服务注册配置
func NewServiceRegistryConfig(keyPrefix string, ttl, healthCheckInterval time.Duration, enableHealthCheck bool) ServiceRegistryConfig {
	return ServiceRegistryConfig{
		KeyPrefix:           keyPrefix,
		TTL:                 ttl,
		HealthCheckInterval: healthCheckInterval,
		EnableHealthCheck:   enableHealthCheck,
		MaxRetries:          3,
		RetryInterval:       time.Second,
	}
}

// NewDistributedLockConfig 创建分布式锁配置
func NewDistributedLockConfig(keyPrefix string, defaultTTL, renewInterval time.Duration, enableReentrant bool) DistributedLockConfig {
	return DistributedLockConfig{
		KeyPrefix:       keyPrefix,
		DefaultTTL:      defaultTTL,
		RenewInterval:   renewInterval,
		EnableReentrant: enableReentrant,
		SessionTTL:      defaultTTL * 2,
		MaxRetries:      3,
		RetryInterval:   time.Second,
	}
}

// NewConfigCenterConfig 创建配置中心配置
func NewConfigCenterConfig(keyPrefix string, enableVersioning bool, maxVersionHistory int, enableValidation bool) ConfigCenterConfig {
	return ConfigCenterConfig{
		KeyPrefix:         keyPrefix,
		EnableVersioning:  enableVersioning,
		MaxVersionHistory: maxVersionHistory,
		EnableValidation:  enableValidation,
		WatchBufferSize:   100,
		EnableCompression: false,
	}
}

// NewRetryConfig 创建重试配置
func NewRetryConfig(maxRetries int, initialInterval, maxInterval time.Duration, multiplier float64) RetryConfig {
	return RetryConfig{
		MaxRetries:          maxRetries,
		InitialInterval:     initialInterval,
		MaxInterval:         maxInterval,
		Multiplier:          multiplier,
		RandomizationFactor: 0.1,
	}
}
