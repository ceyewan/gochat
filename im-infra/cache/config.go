package cache

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/cache/internal"
)

// ===== 配置预设 =====

// DevelopmentConfig 返回适用于开发环境的配置
func DevelopmentConfig() Config {
	return Config{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		PoolSize:        5,
		MinIdleConns:    2,
		MaxIdleConns:    5,
		ConnMaxIdleTime: 10 * time.Minute,
		ConnMaxLifetime: 30 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		EnableTracing:   false,
		EnableMetrics:   false,
		KeyPrefix:       "dev",
		Serializer:      "json",
		Compression:     false,
	}
}

// ProductionConfig 返回适用于生产环境的配置
func ProductionConfig() Config {
	return Config{
		Addr:            "redis:6379",
		Password:        "",
		DB:              0,
		PoolSize:        20,
		MinIdleConns:    10,
		MaxIdleConns:    20,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: time.Hour,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		MaxRetries:      5,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		EnableTracing:   true,
		EnableMetrics:   true,
		KeyPrefix:       "prod",
		Serializer:      "json",
		Compression:     true,
	}
}

// TestConfig 返回适用于测试环境的配置
func TestConfig() Config {
	return Config{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              1, // 使用不同的数据库避免冲突
		PoolSize:        3,
		MinIdleConns:    1,
		MaxIdleConns:    3,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 10 * time.Minute,
		DialTimeout:     2 * time.Second,
		ReadTimeout:     1 * time.Second,
		WriteTimeout:    1 * time.Second,
		PoolTimeout:     2 * time.Second,
		MaxRetries:      1,
		MinRetryBackoff: 4 * time.Millisecond,
		MaxRetryBackoff: 256 * time.Millisecond,
		EnableTracing:   false,
		EnableMetrics:   false,
		KeyPrefix:       "test",
		Serializer:      "json",
		Compression:     false,
	}
}

// HighPerformanceConfig 返回适用于高性能场景的配置
func HighPerformanceConfig() Config {
	return Config{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		PoolSize:        50,
		MinIdleConns:    20,
		MaxIdleConns:    50,
		ConnMaxIdleTime: time.Hour,
		ConnMaxLifetime: 2 * time.Hour,
		DialTimeout:     3 * time.Second,
		ReadTimeout:     1 * time.Second,
		WriteTimeout:    1 * time.Second,
		PoolTimeout:     2 * time.Second,
		MaxRetries:      3,
		MinRetryBackoff: 4 * time.Millisecond,
		MaxRetryBackoff: 256 * time.Millisecond,
		EnableTracing:   false,
		EnableMetrics:   true,
		KeyPrefix:       "",
		Serializer:      "json",
		Compression:     false,
	}
}

// ===== 配置构建器 =====

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

// Addr 设置 Redis 地址
func (b *ConfigBuilder) Addr(addr string) *ConfigBuilder {
	b.config.Addr = addr
	return b
}

// Password 设置 Redis 密码
func (b *ConfigBuilder) Password(password string) *ConfigBuilder {
	b.config.Password = password
	return b
}

// DB 设置 Redis 数据库编号
func (b *ConfigBuilder) DB(db int) *ConfigBuilder {
	b.config.DB = db
	return b
}

// PoolSize 设置连接池大小
func (b *ConfigBuilder) PoolSize(size int) *ConfigBuilder {
	b.config.PoolSize = size
	return b
}

// IdleConns 设置空闲连接数
func (b *ConfigBuilder) IdleConns(min, max int) *ConfigBuilder {
	b.config.MinIdleConns = min
	b.config.MaxIdleConns = max
	return b
}

// Timeouts 设置超时时间
func (b *ConfigBuilder) Timeouts(dial, read, write, pool time.Duration) *ConfigBuilder {
	b.config.DialTimeout = dial
	b.config.ReadTimeout = read
	b.config.WriteTimeout = write
	b.config.PoolTimeout = pool
	return b
}

// Retries 设置重试配置
func (b *ConfigBuilder) Retries(maxRetries int, minBackoff, maxBackoff time.Duration) *ConfigBuilder {
	b.config.MaxRetries = maxRetries
	b.config.MinRetryBackoff = minBackoff
	b.config.MaxRetryBackoff = maxBackoff
	return b
}

// KeyPrefix 设置键名前缀
func (b *ConfigBuilder) KeyPrefix(prefix string) *ConfigBuilder {
	b.config.KeyPrefix = prefix
	return b
}

// Serializer 设置序列化器
func (b *ConfigBuilder) Serializer(serializer string) *ConfigBuilder {
	b.config.Serializer = serializer
	return b
}

// EnableTracing 启用链路追踪
func (b *ConfigBuilder) EnableTracing() *ConfigBuilder {
	b.config.EnableTracing = true
	return b
}

// EnableMetrics 启用指标收集
func (b *ConfigBuilder) EnableMetrics() *ConfigBuilder {
	b.config.EnableMetrics = true
	return b
}

// EnableCompression 启用压缩
func (b *ConfigBuilder) EnableCompression() *ConfigBuilder {
	b.config.Compression = true
	return b
}

// Build 构建配置
func (b *ConfigBuilder) Build() Config {
	return b.config
}

// ===== 验证函数 =====

// ValidateConfig 验证配置的完整性和合理性
func ValidateConfig(cfg Config) error {
	return internal.ValidateConfig(cfg)
}

// ===== 默认锁配置 =====

// DefaultLockConfig 返回默认的锁配置
func DefaultLockConfig() LockConfig {
	return internal.DefaultLockConfig()
}

// QuickLockConfig 返回快速锁配置（较短的过期时间和重试间隔）
func QuickLockConfig() LockConfig {
	return LockConfig{
		DefaultExpiration: 10 * time.Second,
		RefreshInterval:   3 * time.Second,
		RetryDelay:        50 * time.Millisecond,
		MaxRetries:        5,
	}
}

// LongLockConfig 返回长时间锁配置（较长的过期时间）
func LongLockConfig() LockConfig {
	return LockConfig{
		DefaultExpiration: 5 * time.Minute,
		RefreshInterval:   time.Minute,
		RetryDelay:        200 * time.Millisecond,
		MaxRetries:        15,
	}
}

// ===== 默认布隆过滤器配置 =====

// DefaultBloomConfig 返回默认的布隆过滤器配置
func DefaultBloomConfig() BloomConfig {
	return internal.DefaultBloomConfig()
}

// SmallBloomConfig 返回小容量布隆过滤器配置
func SmallBloomConfig() BloomConfig {
	return BloomConfig{
		DefaultCapacity:  10000,
		DefaultErrorRate: 0.01,
		HashFunctions:    7,
	}
}

// LargeBloomConfig 返回大容量布隆过滤器配置
func LargeBloomConfig() BloomConfig {
	return BloomConfig{
		DefaultCapacity:  10000000,
		DefaultErrorRate: 0.001,
		HashFunctions:    10,
	}
}

// HighPrecisionBloomConfig 返回高精度布隆过滤器配置
func HighPrecisionBloomConfig() BloomConfig {
	return BloomConfig{
		DefaultCapacity:  1000000,
		DefaultErrorRate: 0.0001,
		HashFunctions:    13,
	}
}
