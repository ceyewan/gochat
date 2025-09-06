package internal

import (
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
)

// Config 是 idgen 的主配置结构体。
// 用于声明式地定义 ID 生成器的行为和参数。
type Config struct {
	// Type 指定 ID 生成器类型
	Type GeneratorType `json:"type" yaml:"type"`

	// Snowflake 雪花算法配置
	Snowflake *SnowflakeConfig `json:"snowflake,omitempty" yaml:"snowflake,omitempty"`

	// UUID UUID 生成器配置
	UUID *UUIDConfig `json:"uuid,omitempty" yaml:"uuid,omitempty"`

	// Redis Redis 自增 ID 配置
	Redis *RedisConfig `json:"redis,omitempty" yaml:"redis,omitempty"`
}

// SnowflakeConfig 雪花算法配置
type SnowflakeConfig struct {
	// NodeID 节点 ID，如果为 0 则自动从 IP 地址生成
	NodeID int64 `json:"node_id" yaml:"node_id"`

	// AutoNodeID 是否自动生成节点 ID
	AutoNodeID bool `json:"auto_node_id" yaml:"auto_node_id"`

	// Epoch 自定义起始时间戳（毫秒），默认使用 Twitter 雪花算法的起始时间
	Epoch int64 `json:"epoch" yaml:"epoch"`
}

// UUIDConfig UUID 生成器配置
type UUIDConfig struct {
	// Version UUID 版本，支持 4 和 7
	Version int `json:"version" yaml:"version"`

	// Format 输出格式，支持 "standard"（带连字符）和 "simple"（不带连字符）
	Format string `json:"format" yaml:"format"`

	// UpperCase 是否使用大写字母
	UpperCase bool `json:"upper_case" yaml:"upper_case"`
}

// RedisConfig Redis 自增 ID 配置
type RedisConfig struct {
	// CacheConfig Redis 连接配置
	CacheConfig cache.Config `json:"cache_config" yaml:"cache_config"`

	// KeyPrefix 键前缀
	KeyPrefix string `json:"key_prefix" yaml:"key_prefix"`

	// DefaultKey 默认键名
	DefaultKey string `json:"default_key" yaml:"default_key"`

	// Step 自增步长
	Step int64 `json:"step" yaml:"step"`

	// InitialValue 初始值
	InitialValue int64 `json:"initial_value" yaml:"initial_value"`

	// TTL 键的过期时间，0 表示不过期
	TTL time.Duration `json:"ttl" yaml:"ttl"`
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	if !c.Type.IsValid() {
		return fmt.Errorf("invalid generator type: %s", c.Type)
	}

	switch c.Type {
	case SnowflakeType:
		if c.Snowflake == nil {
			return fmt.Errorf("snowflake configimpl is required for snowflake type")
		}
		return c.Snowflake.Validate()

	case UUIDType:
		if c.UUID == nil {
			return fmt.Errorf("uuid configimpl is required for uuid type")
		}
		return c.UUID.Validate()

	case RedisType:
		if c.Redis == nil {
			return fmt.Errorf("redis configimpl is required for redis type")
		}
		return c.Redis.Validate()

	default:
		return fmt.Errorf("unsupported generator type: %s", c.Type)
	}
}

// Validate 验证雪花算法配置
func (c *SnowflakeConfig) Validate() error {
	if !c.AutoNodeID && (c.NodeID < 0 || c.NodeID > 1023) {
		return fmt.Errorf("node_id must be between 0 and 1023, got: %d", c.NodeID)
	}

	if c.Epoch < 0 {
		return fmt.Errorf("epoch must be non-negative, got: %d", c.Epoch)
	}

	return nil
}

// Validate 验证 UUID 配置
func (c *UUIDConfig) Validate() error {
	if c.Version != 4 && c.Version != 7 {
		return fmt.Errorf("unsupported uuid version: %d, only 4 and 7 are supported", c.Version)
	}

	if c.Format != "" && c.Format != "standard" && c.Format != "simple" {
		return fmt.Errorf("unsupported uuid format: %s, only 'standard' and 'simple' are supported", c.Format)
	}

	return nil
}

// Validate 验证 Redis 配置
func (c *RedisConfig) Validate() error {
	if c.KeyPrefix == "" {
		return fmt.Errorf("key_prefix is required")
	}

	if c.DefaultKey == "" {
		return fmt.Errorf("default_key is required")
	}

	if c.Step <= 0 {
		return fmt.Errorf("step must be positive, got: %d", c.Step)
	}

	if c.InitialValue < 0 {
		return fmt.Errorf("initial_value must be non-negative, got: %d", c.InitialValue)
	}

	if c.TTL < 0 {
		return fmt.Errorf("ttl must be non-negative, got: %v", c.TTL)
	}

	return nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Type: SnowflakeType,
		Snowflake: &SnowflakeConfig{
			NodeID:     0,
			AutoNodeID: true,
			Epoch:      1288834974657, // Twitter 雪花算法起始时间
		},
	}
}

// DefaultSnowflakeConfig 返回默认雪花算法配置
func DefaultSnowflakeConfig() Config {
	return Config{
		Type: SnowflakeType,
		Snowflake: &SnowflakeConfig{
			NodeID:     0,
			AutoNodeID: true,
			Epoch:      1288834974657,
		},
	}
}

// DefaultUUIDConfig 返回默认 UUID 配置
func DefaultUUIDConfig() Config {
	return Config{
		Type: UUIDType,
		UUID: &UUIDConfig{
			Version:   4,
			Format:    "standard",
			UpperCase: false,
		},
	}
}

// DefaultRedisConfig 返回默认 Redis 配置
func DefaultRedisConfig() Config {
	return Config{
		Type: RedisType,
		Redis: &RedisConfig{
			CacheConfig:  cache.DefaultConfig(),
			KeyPrefix:    "idgen",
			DefaultKey:   "default",
			Step:         1,
			InitialValue: 1,
			TTL:          0, // 不过期
		},
	}
}
