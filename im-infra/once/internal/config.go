package internal

import (
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
)

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

// Config 是 idempotent 的主配置结构体。
// 用于声明式地定义幂等组件的行为和参数。
type Config struct {
	// CacheConfig Redis 连接配置，复用 cache 组件的配置
	CacheConfig cache.Config `json:"cache_config" yaml:"cache_config"`
	// KeyPrefix 键前缀，用于业务隔离
	KeyPrefix string `json:"key_prefix" yaml:"key_prefix"`
	// DefaultTTL 默认过期时间，0 表示不过期
	DefaultTTL time.Duration `json:"default_ttl" yaml:"default_ttl"`
	// RetryConfig 重试配置
	RetryConfig *RetryConfig `json:"retry_config" yaml:"retry_config"`
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	if c.KeyPrefix == "" {
		return fmt.Errorf("key_prefix cannot be empty")
	}

	if c.DefaultTTL < 0 {
		return fmt.Errorf("default_ttl cannot be negative")
	}

	return nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		CacheConfig: cache.DefaultConfig(),
		KeyPrefix:   "idempotent",
		DefaultTTL:  time.Hour,
	}
}

// DevelopmentConfig 返回适用于开发环境的配置
func DevelopmentConfig() Config {
	return Config{
		CacheConfig: cache.DefaultConfig(),
		KeyPrefix:   "dev:idempotent",
		DefaultTTL:  30 * time.Minute,
	}
}

// ProductionConfig 返回适用于生产环境的配置
func ProductionConfig() Config {
	cfg := cache.DefaultConfig()
	// 生产环境可以设置更大的连接池和更长的超时时间
	cfg.PoolSize = 20
	cfg.MaxIdleConns = 15
	cfg.ConnMaxLifetime = 2 * time.Hour
	return Config{
		CacheConfig: cfg,
		KeyPrefix:   "prod:idempotent",
		DefaultTTL:  2 * time.Hour,
	}
}

// TestConfig 返回适用于测试环境的配置
func TestConfig() Config {
	cfg := cache.DefaultConfig()
	// 测试环境使用更短的超时时间和更小的连接池
	cfg.DialTimeout = 2 * time.Second
	cfg.ReadTimeout = 2 * time.Second
	cfg.WriteTimeout = 2 * time.Second
	cfg.PoolSize = 5
	return Config{
		CacheConfig: cfg,
		KeyPrefix:   "test:idempotent",
		DefaultTTL:  5 * time.Minute,
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

// Build 构建配置
func (b *ConfigBuilder) Build() Config {
	return b.config
}
