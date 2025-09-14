package internal

import (
	"fmt"
	"time"
)

// Config 是 db 的主配置结构体。
// 通常从配置中心获取，也可以手动构建。
type Config struct {
	// DSN 数据库连接字符串
	// MySQL 示例: "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
	DSN string `json:"dsn" yaml:"dsn"`

	// Driver 数据库驱动类型
	// 仅支持: "mysql"
	// 默认: "mysql"
	Driver string `json:"driver" yaml:"driver"`

	// MaxOpenConns 最大打开连接数
	// 默认: 25
	MaxOpenConns int `json:"maxOpenConns" yaml:"maxOpenConns"`

	// MaxIdleConns 最大空闲连接数
	// 默认: 10
	MaxIdleConns int `json:"maxIdleConns" yaml:"maxIdleConns"`

	// ConnMaxLifetime 连接最大生存时间
	// 默认: 1小时
	ConnMaxLifetime time.Duration `json:"connMaxLifetime" yaml:"connMaxLifetime"`

	// ConnMaxIdleTime 连接最大空闲时间
	// 默认: 30分钟
	ConnMaxIdleTime time.Duration `json:"connMaxIdleTime" yaml:"connMaxIdleTime"`

	// LogLevel GORM 日志级别
	// 支持: "silent", "error", "warn", "info"
	// 默认: "warn"
	LogLevel string `json:"logLevel" yaml:"logLevel"`

	// SlowThreshold 慢查询阈值
	// 默认: 200毫秒
	SlowThreshold time.Duration `json:"slowThreshold" yaml:"slowThreshold"`

	// EnableMetrics 是否启用指标收集
	// 默认: false
	EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics"`

	// EnableTracing 是否启用链路追踪
	// 默认: false
	EnableTracing bool `json:"enableTracing" yaml:"enableTracing"`

	// TablePrefix 表名前缀
	// 默认: ""
	TablePrefix string `json:"tablePrefix" yaml:"tablePrefix"`

	// DisableForeignKeyConstraintWhenMigrating 迁移时是否禁用外键约束
	// 默认: false
	DisableForeignKeyConstraintWhenMigrating bool `json:"disableForeignKeyConstraintWhenMigrating" yaml:"disableForeignKeyConstraintWhenMigrating"`

	// AutoCreateDatabase 是否自动创建数据库（如果不存在）
	// 当连接数据库失败且错误是"数据库不存在"时，自动尝试创建数据库
	// 默认: true
	AutoCreateDatabase bool `json:"autoCreateDatabase" yaml:"autoCreateDatabase"`

	// Sharding 分库分表配置（可选）
	Sharding *ShardingConfig `json:"sharding,omitempty" yaml:"sharding,omitempty"`
}

// ShardingConfig 分库分表配置
type ShardingConfig struct {
	// ShardingKey 分片键字段名
	ShardingKey string `json:"shardingKey" yaml:"shardingKey"`

	// NumberOfShards 分片数量
	NumberOfShards int `json:"numberOfShards" yaml:"numberOfShards"`

	// Tables 需要分片的表配置
	Tables map[string]*TableShardingConfig `json:"tables" yaml:"tables"`
}

// TableShardingConfig 表分片配置
type TableShardingConfig struct {
	// ShardingKey 该表的分片键（如果与全局不同）
	ShardingKey string `json:"shardingKey,omitempty" yaml:"shardingKey,omitempty"`

	// NumberOfShards 该表的分片数量（如果与全局不同）
	NumberOfShards int `json:"numberOfShards,omitempty" yaml:"numberOfShards,omitempty"`
}

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置适用于大多数开发和测试场景。
func DefaultConfig() Config {
	return GetDefaultConfig("development")
}

// GetDefaultConfig 返回默认的数据库配置。
// 开发环境：较少连接数，较详细的日志级别，较短的超时时间
// 生产环境：较多连接数，较少的日志输出，较长的连接生命周期
func GetDefaultConfig(env string) Config {
	switch env {
	case "development":
		return Config{
			DSN:                                      "root:mysql@tcp(localhost:3306)/gochat?charset=utf8mb4&parseTime=True&loc=Local",
			Driver:                                   "mysql",
			MaxOpenConns:                             25,
			MaxIdleConns:                             5,
			ConnMaxLifetime:                          5 * time.Minute,
			ConnMaxIdleTime:                          30 * time.Minute,
			LogLevel:                                 "info",
			SlowThreshold:                            100 * time.Millisecond,
			EnableMetrics:                            false,
			EnableTracing:                            false,
			TablePrefix:                              "",
			DisableForeignKeyConstraintWhenMigrating: false,
			AutoCreateDatabase:                       true,
		}
	case "production":
		return Config{
			DSN:                                      "",
			Driver:                                   "mysql",
			MaxOpenConns:                             100,
			MaxIdleConns:                             10,
			ConnMaxLifetime:                          time.Hour,
			ConnMaxIdleTime:                          30 * time.Minute,
			LogLevel:                                 "warn",
			SlowThreshold:                            500 * time.Millisecond,
			EnableMetrics:                            true,
			EnableTracing:                            true,
			TablePrefix:                              "",
			DisableForeignKeyConstraintWhenMigrating: false,
			AutoCreateDatabase:                       false,
		}
	default:
		return DefaultConfig()
	}
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	if c.DSN == "" {
		return fmt.Errorf("DSN cannot be empty")
	}

	if c.Driver == "" {
		c.Driver = "mysql"
	}

	if c.Driver != "mysql" {
		return fmt.Errorf("unsupported driver: %s, only mysql is supported", c.Driver)
	}

	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = 25
	}

	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 10
	}

	if c.MaxIdleConns > c.MaxOpenConns {
		c.MaxIdleConns = c.MaxOpenConns
	}

	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = time.Hour
	}

	if c.ConnMaxIdleTime <= 0 {
		c.ConnMaxIdleTime = 30 * time.Minute
	}

	if c.LogLevel == "" {
		c.LogLevel = "warn"
	}

	if c.SlowThreshold <= 0 {
		c.SlowThreshold = 200 * time.Millisecond
	}

	// 验证分库分表配置
	if c.Sharding != nil {
		if err := c.validateShardingConfig(); err != nil {
			return fmt.Errorf("invalid sharding configimpl: %w", err)
		}
	}

	return nil
}

// validateShardingConfig 验证分库分表配置
func (c *Config) validateShardingConfig() error {
	if c.Sharding.ShardingKey == "" {
		return fmt.Errorf("sharding key cannot be empty")
	}

	if c.Sharding.NumberOfShards <= 0 {
		return fmt.Errorf("number of shards must be greater than 0")
	}

	return nil
}

// ValidateConfig 验证配置的完整性和合理性（导出函数）
func ValidateConfig(cfg *Config) error {
	return cfg.Validate()
}
