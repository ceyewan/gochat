package internal

import (
	"time"
)

// Config 是 cache 的主配置结构体。
// 用于声明式地定义缓存行为和 Redis 连接参数。
type Config struct {
	// Addr Redis 服务器地址，格式为 "host:port"
	// 默认："localhost:6379"
	Addr string `json:"addr" yaml:"addr"`

	// Password Redis 服务器密码
	// 默认：""（无密码）
	Password string `json:"password" yaml:"password"`

	// DB Redis 数据库编号
	// 默认：0
	DB int `json:"db" yaml:"db"`

	// PoolSize 连接池大小
	// 默认：10
	PoolSize int `json:"poolSize" yaml:"poolSize"`

	// MinIdleConns 最小空闲连接数
	// 默认：5
	MinIdleConns int `json:"minIdleConns" yaml:"minIdleConns"`

	// MaxIdleConns 最大空闲连接数
	// 默认：10
	MaxIdleConns int `json:"maxIdleConns" yaml:"maxIdleConns"`

	// ConnMaxIdleTime 连接最大空闲时间
	// 默认：30分钟
	ConnMaxIdleTime time.Duration `json:"connMaxIdleTime" yaml:"connMaxIdleTime"`

	// ConnMaxLifetime 连接最大生存时间
	// 默认：1小时
	ConnMaxLifetime time.Duration `json:"connMaxLifetime" yaml:"connMaxLifetime"`

	// DialTimeout 连接超时时间
	// 默认：5秒
	DialTimeout time.Duration `json:"dialTimeout" yaml:"dialTimeout"`

	// ReadTimeout 读取超时时间
	// 默认：3秒
	ReadTimeout time.Duration `json:"readTimeout" yaml:"readTimeout"`

	// WriteTimeout 写入超时时间
	// 默认：3秒
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`

	// PoolTimeout 从连接池获取连接的超时时间
	// 默认：4秒
	PoolTimeout time.Duration `json:"poolTimeout" yaml:"poolTimeout"`

	// MaxRetries 最大重试次数
	// 默认：3
	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`

	// MinRetryBackoff 最小重试间隔
	// 默认：8毫秒
	MinRetryBackoff time.Duration `json:"minRetryBackoff" yaml:"minRetryBackoff"`

	// MaxRetryBackoff 最大重试间隔
	// 默认：512毫秒
	MaxRetryBackoff time.Duration `json:"maxRetryBackoff" yaml:"maxRetryBackoff"`

	// EnableTracing 是否启用链路追踪
	// 默认：false
	EnableTracing bool `json:"enableTracing" yaml:"enableTracing"`

	// EnableMetrics 是否启用指标收集
	// 默认：false
	EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics"`

	// KeyPrefix 键名前缀，用于命名空间隔离
	// 默认：""
	KeyPrefix string `json:"keyPrefix" yaml:"keyPrefix"`

	// Serializer 序列化器类型
	// 支持："json"、"msgpack"、"gob"
	// 默认："json"
	Serializer string `json:"serializer" yaml:"serializer"`

	// Compression 是否启用压缩
	// 默认：false
	Compression bool `json:"compression" yaml:"compression"`
}

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置适用于大多数开发和测试场景。
func DefaultConfig() Config {
	return Config{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		PoolSize:        10,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: time.Hour,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		EnableTracing:   false,
		EnableMetrics:   false,
		KeyPrefix:       "",
		Serializer:      "json",
		Compression:     false,
	}
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	if c.Addr == "" {
		c.Addr = "localhost:6379"
	}
	
	if c.PoolSize <= 0 {
		c.PoolSize = 10
	}
	
	if c.MinIdleConns < 0 {
		c.MinIdleConns = 0
	}
	
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = c.PoolSize
	}
	
	if c.DialTimeout <= 0 {
		c.DialTimeout = 5 * time.Second
	}
	
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 3 * time.Second
	}
	
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 3 * time.Second
	}
	
	if c.PoolTimeout <= 0 {
		c.PoolTimeout = 4 * time.Second
	}
	
	if c.MaxRetries < 0 {
		c.MaxRetries = 3
	}
	
	if c.Serializer == "" {
		c.Serializer = "json"
	}
	
	return nil
}

// LockConfig 分布式锁的配置
type LockConfig struct {
	// DefaultExpiration 默认锁过期时间
	// 默认：30秒
	DefaultExpiration time.Duration `json:"defaultExpiration" yaml:"defaultExpiration"`

	// RefreshInterval 锁续期间隔
	// 默认：10秒
	RefreshInterval time.Duration `json:"refreshInterval" yaml:"refreshInterval"`

	// RetryDelay 获取锁失败时的重试间隔
	// 默认：100毫秒
	RetryDelay time.Duration `json:"retryDelay" yaml:"retryDelay"`

	// MaxRetries 获取锁的最大重试次数
	// 默认：10
	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`
}

// DefaultLockConfig 返回默认的锁配置
func DefaultLockConfig() LockConfig {
	return LockConfig{
		DefaultExpiration: 30 * time.Second,
		RefreshInterval:   10 * time.Second,
		RetryDelay:        100 * time.Millisecond,
		MaxRetries:        10,
	}
}

// BloomConfig 布隆过滤器的配置
type BloomConfig struct {
	// DefaultCapacity 默认容量
	// 默认：1000000
	DefaultCapacity uint64 `json:"defaultCapacity" yaml:"defaultCapacity"`

	// DefaultErrorRate 默认错误率
	// 默认：0.01 (1%)
	DefaultErrorRate float64 `json:"defaultErrorRate" yaml:"defaultErrorRate"`

	// HashFunctions 哈希函数数量
	// 默认：7
	HashFunctions int `json:"hashFunctions" yaml:"hashFunctions"`
}

// DefaultBloomConfig 返回默认的布隆过滤器配置
func DefaultBloomConfig() BloomConfig {
	return BloomConfig{
		DefaultCapacity:  1000000,
		DefaultErrorRate: 0.01,
		HashFunctions:    7,
	}
}
