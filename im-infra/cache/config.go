package cache

import (
	"fmt"
	"time"
)

// Config 是 cache 的主配置结构体
type Config struct {
	// Addr Redis 服务器地址，格式为 "host:port"
	Addr string `json:"addr" yaml:"addr"`

	// Password Redis 服务器密码
	Password string `json:"password" yaml:"password"`

	// DB Redis 数据库编号
	DB int `json:"db" yaml:"db"`

	// PoolSize 连接池大小
	PoolSize int `json:"poolSize" yaml:"poolSize"`

	// MinIdleConns 最小空闲连接数
	MinIdleConns int `json:"minIdleConns" yaml:"minIdleConns"`

	// MaxIdleConns 最大空闲连接数
	MaxIdleConns int `json:"maxIdleConns" yaml:"maxIdleConns"`

	// ConnMaxIdleTime 连接最大空闲时间
	ConnMaxIdleTime time.Duration `json:"connMaxIdleTime" yaml:"connMaxIdleTime"`

	// ConnMaxLifetime 连接最大生存时间
	ConnMaxLifetime time.Duration `json:"connMaxLifetime" yaml:"connMaxLifetime"`

	// DialTimeout 连接超时时间
	DialTimeout time.Duration `json:"dialTimeout" yaml:"dialTimeout"`

	// ReadTimeout 读取超时时间
	ReadTimeout time.Duration `json:"readTimeout" yaml:"readTimeout"`

	// WriteTimeout 写入超时时间
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`

	// PoolTimeout 从连接池获取连接的超时时间
	PoolTimeout time.Duration `json:"poolTimeout" yaml:"poolTimeout"`

	// MaxRetries 最大重试次数
	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`

	// MinRetryBackoff 最小重试间隔
	MinRetryBackoff time.Duration `json:"minRetryBackoff" yaml:"minRetryBackoff"`

	// MaxRetryBackoff 最大重试间隔
	MaxRetryBackoff time.Duration `json:"maxRetryBackoff" yaml:"maxRetryBackoff"`

	// KeyPrefix 键名前缀，用于命名空间隔离
	KeyPrefix string `json:"keyPrefix" yaml:"keyPrefix"`
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证 Redis 地址
	if c.Addr == "" {
		return fmt.Errorf("redis address cannot be empty")
	}

	// 验证连接池配置
	if c.PoolSize <= 0 {
		return fmt.Errorf("pool size must be positive, got: %d", c.PoolSize)
	}

	if c.MinIdleConns < 0 {
		return fmt.Errorf("min idle connections cannot be negative, got: %d", c.MinIdleConns)
	}

	if c.MaxIdleConns < 0 {
		return fmt.Errorf("max idle connections cannot be negative, got: %d", c.MaxIdleConns)
	}

	if c.MaxIdleConns > c.PoolSize {
		return fmt.Errorf("max idle connections (%d) cannot exceed pool size (%d)", c.MaxIdleConns, c.PoolSize)
	}

	// 验证超时配置
	if c.DialTimeout <= 0 {
		return fmt.Errorf("dial timeout must be positive, got: %v", c.DialTimeout)
	}

	if c.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive, got: %v", c.ReadTimeout)
	}

	if c.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive, got: %v", c.WriteTimeout)
	}

	if c.PoolTimeout <= 0 {
		return fmt.Errorf("pool timeout must be positive, got: %v", c.PoolTimeout)
	}

	// 验证重试配置
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative, got: %d", c.MaxRetries)
	}

	if c.MinRetryBackoff < 0 {
		return fmt.Errorf("min retry backoff cannot be negative, got: %v", c.MinRetryBackoff)
	}

	if c.MaxRetryBackoff < 0 {
		return fmt.Errorf("max retry backoff cannot be negative, got: %v", c.MaxRetryBackoff)
	}

	if c.MaxRetryBackoff < c.MinRetryBackoff {
		return fmt.Errorf("max retry backoff (%v) cannot be less than min retry backoff (%v)", c.MaxRetryBackoff, c.MinRetryBackoff)
	}

	return nil
}
