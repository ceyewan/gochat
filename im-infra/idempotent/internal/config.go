package internal

import (
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
)

// Config 是 idempotent 的主配置结构体。
// 用于声明式地定义幂等组件的行为和参数。
type Config struct {
	// CacheConfig Redis 连接配置，复用 cache 组件的配置
	CacheConfig cache.Config `json:"cache_config" yaml:"cache_config"`
	
	// KeyPrefix 键前缀，用于业务隔离
	KeyPrefix string `json:"key_prefix" yaml:"key_prefix"`
	
	// DefaultTTL 默认过期时间，0 表示不过期
	DefaultTTL time.Duration `json:"default_ttl" yaml:"default_ttl"`
	
	// Serializer 序列化方式，支持 "json", "msgpack", "gob"
	Serializer string `json:"serializer" yaml:"serializer"`
	
	// EnableCompression 是否启用压缩
	EnableCompression bool `json:"enable_compression" yaml:"enable_compression"`
	
	// MaxKeyLength 最大键长度限制
	MaxKeyLength int `json:"max_key_length" yaml:"max_key_length"`
	
	// KeyValidator 键名验证器
	KeyValidator string `json:"key_validator" yaml:"key_validator"`
	
	// EnableMetrics 是否启用指标收集
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`
	
	// EnableTracing 是否启用链路追踪
	EnableTracing bool `json:"enable_tracing" yaml:"enable_tracing"`
	
	// RetryConfig 重试配置
	RetryConfig *RetryConfig `json:"retry_config,omitempty" yaml:"retry_config,omitempty"`
}

// RetryConfig 重试配置
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

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	if c.KeyPrefix == "" {
		return fmt.Errorf("key_prefix cannot be empty")
	}
	
	if c.DefaultTTL < 0 {
		return fmt.Errorf("default_ttl cannot be negative")
	}
	
	if c.MaxKeyLength <= 0 {
		c.MaxKeyLength = 250 // Redis 键名默认最大长度
	}
	
	if c.Serializer == "" {
		c.Serializer = "json"
	}
	
	// 验证序列化器
	switch c.Serializer {
	case "json", "msgpack", "gob":
		// 支持的序列化器
	default:
		return fmt.Errorf("unsupported serializer: %s", c.Serializer)
	}
	
	// 验证键名验证器
	switch c.KeyValidator {
	case "", "default", "strict", "loose":
		// 支持的验证器
	default:
		return fmt.Errorf("unsupported key_validator: %s", c.KeyValidator)
	}
	
	// 验证重试配置
	if c.RetryConfig != nil {
		if err := c.RetryConfig.Validate(); err != nil {
			return fmt.Errorf("invalid retry_config: %w", err)
		}
	}
	
	return nil
}

// Validate 验证重试配置
func (r *RetryConfig) Validate() error {
	if r.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	
	if r.InitialInterval < 0 {
		return fmt.Errorf("initial_interval cannot be negative")
	}
	
	if r.MaxInterval < 0 {
		return fmt.Errorf("max_interval cannot be negative")
	}
	
	if r.Multiplier <= 0 {
		return fmt.Errorf("multiplier must be positive")
	}
	
	if r.RandomizationFactor < 0 || r.RandomizationFactor > 1 {
		return fmt.Errorf("randomization_factor must be between 0 and 1")
	}
	
	return nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		CacheConfig:       cache.DevelopmentConfig(),
		KeyPrefix:         "idempotent",
		DefaultTTL:        time.Hour,
		Serializer:        "json",
		EnableCompression: false,
		MaxKeyLength:      250,
		KeyValidator:      "default",
		EnableMetrics:     false,
		EnableTracing:     false,
		RetryConfig:       DefaultRetryConfig(),
	}
}

// DevelopmentConfig 返回适用于开发环境的配置
func DevelopmentConfig() Config {
	return Config{
		CacheConfig:       cache.DevelopmentConfig(),
		KeyPrefix:         "dev:idempotent",
		DefaultTTL:        30 * time.Minute,
		Serializer:        "json",
		EnableCompression: false,
		MaxKeyLength:      250,
		KeyValidator:      "loose",
		EnableMetrics:     false,
		EnableTracing:     false,
		RetryConfig:       DefaultRetryConfig(),
	}
}

// ProductionConfig 返回适用于生产环境的配置
func ProductionConfig() Config {
	return Config{
		CacheConfig:       cache.ProductionConfig(),
		KeyPrefix:         "prod:idempotent",
		DefaultTTL:        2 * time.Hour,
		Serializer:        "json",
		EnableCompression: true,
		MaxKeyLength:      250,
		KeyValidator:      "strict",
		EnableMetrics:     true,
		EnableTracing:     true,
		RetryConfig:       ProductionRetryConfig(),
	}
}

// TestConfig 返回适用于测试环境的配置
func TestConfig() Config {
	return Config{
		CacheConfig:       cache.TestConfig(),
		KeyPrefix:         "test:idempotent",
		DefaultTTL:        5 * time.Minute,
		Serializer:        "json",
		EnableCompression: false,
		MaxKeyLength:      250,
		KeyValidator:      "loose",
		EnableMetrics:     false,
		EnableTracing:     false,
		RetryConfig:       nil, // 测试环境不重试
	}
}

// DefaultRetryConfig 返回默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:          3,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         5 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	}
}

// ProductionRetryConfig 返回生产环境重试配置
func ProductionRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:          5,
		InitialInterval:     50 * time.Millisecond,
		MaxInterval:         10 * time.Second,
		Multiplier:          1.5,
		RandomizationFactor: 0.2,
	}
}

// ConfigBuilder 配置构建器，提供链式调用方式构建配置
type ConfigBuilder struct {
	config Config
}

// NewConfigBuilder 创建新的配置构建器
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: DefaultConfig(),
	}
}

// CacheConfig 设置缓存配置
func (b *ConfigBuilder) CacheConfig(config cache.Config) *ConfigBuilder {
	b.config.CacheConfig = config
	return b
}

// KeyPrefix 设置键前缀
func (b *ConfigBuilder) KeyPrefix(prefix string) *ConfigBuilder {
	b.config.KeyPrefix = prefix
	return b
}

// DefaultTTL 设置默认过期时间
func (b *ConfigBuilder) DefaultTTL(ttl time.Duration) *ConfigBuilder {
	b.config.DefaultTTL = ttl
	return b
}

// Serializer 设置序列化器
func (b *ConfigBuilder) Serializer(serializer string) *ConfigBuilder {
	b.config.Serializer = serializer
	return b
}

// EnableCompression 启用压缩
func (b *ConfigBuilder) EnableCompression() *ConfigBuilder {
	b.config.EnableCompression = true
	return b
}

// DisableCompression 禁用压缩
func (b *ConfigBuilder) DisableCompression() *ConfigBuilder {
	b.config.EnableCompression = false
	return b
}

// MaxKeyLength 设置最大键长度
func (b *ConfigBuilder) MaxKeyLength(length int) *ConfigBuilder {
	b.config.MaxKeyLength = length
	return b
}

// KeyValidator 设置键名验证器
func (b *ConfigBuilder) KeyValidator(validator string) *ConfigBuilder {
	b.config.KeyValidator = validator
	return b
}

// EnableMetrics 启用指标收集
func (b *ConfigBuilder) EnableMetrics() *ConfigBuilder {
	b.config.EnableMetrics = true
	return b
}

// EnableTracing 启用链路追踪
func (b *ConfigBuilder) EnableTracing() *ConfigBuilder {
	b.config.EnableTracing = true
	return b
}

// RetryConfig 设置重试配置
func (b *ConfigBuilder) RetryConfig(config *RetryConfig) *ConfigBuilder {
	b.config.RetryConfig = config
	return b
}

// Build 构建配置
func (b *ConfigBuilder) Build() Config {
	return b.config
}
