package db

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/db/internal"
)

// ===== 配置预设 =====

// DevelopmentConfig 返回适用于开发环境的配置
func DevelopmentConfig() Config {
	return Config{
		DSN:                                      "root:password@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:                                   "mysql",
		MaxOpenConns:                             10,
		MaxIdleConns:                             5,
		ConnMaxLifetime:                          30 * time.Minute,
		ConnMaxIdleTime:                          10 * time.Minute,
		LogLevel:                                 "info",
		SlowThreshold:                            500 * time.Millisecond,
		EnableMetrics:                            false,
		EnableTracing:                            false,
		TablePrefix:                              "dev_",
		DisableForeignKeyConstraintWhenMigrating: false,
	}
}

// ProductionConfig 返回适用于生产环境的配置
func ProductionConfig() Config {
	return Config{
		DSN:                                      "root:password@tcp(mysql:3306)/gochat_prod?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:                                   "mysql",
		MaxOpenConns:                             50,
		MaxIdleConns:                             25,
		ConnMaxLifetime:                          time.Hour,
		ConnMaxIdleTime:                          30 * time.Minute,
		LogLevel:                                 "warn",
		SlowThreshold:                            200 * time.Millisecond,
		EnableMetrics:                            true,
		EnableTracing:                            true,
		TablePrefix:                              "",
		DisableForeignKeyConstraintWhenMigrating: false,
	}
}

// TestConfig 返回适用于测试环境的配置
func TestConfig() Config {
	return Config{
		DSN:                                      "root:password@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:                                   "mysql",
		MaxOpenConns:                             5,
		MaxIdleConns:                             2,
		ConnMaxLifetime:                          10 * time.Minute,
		ConnMaxIdleTime:                          5 * time.Minute,
		LogLevel:                                 "silent",
		SlowThreshold:                            1 * time.Second,
		EnableMetrics:                            false,
		EnableTracing:                            false,
		TablePrefix:                              "test_",
		DisableForeignKeyConstraintWhenMigrating: true,
	}
}

// HighPerformanceConfig 返回适用于高性能场景的配置
func HighPerformanceConfig() Config {
	return Config{
		DSN:                                      "root:password@tcp(localhost:3306)/gochat?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:                                   "mysql",
		MaxOpenConns:                             100,
		MaxIdleConns:                             50,
		ConnMaxLifetime:                          2 * time.Hour,
		ConnMaxIdleTime:                          time.Hour,
		LogLevel:                                 "error",
		SlowThreshold:                            100 * time.Millisecond,
		EnableMetrics:                            true,
		EnableTracing:                            false,
		TablePrefix:                              "",
		DisableForeignKeyConstraintWhenMigrating: false,
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

// DSN 设置数据库连接字符串
func (b *ConfigBuilder) DSN(dsn string) *ConfigBuilder {
	b.config.DSN = dsn
	return b
}

// Driver 设置数据库驱动
func (b *ConfigBuilder) Driver(driver string) *ConfigBuilder {
	b.config.Driver = driver
	return b
}

// MaxOpenConns 设置最大打开连接数
func (b *ConfigBuilder) MaxOpenConns(max int) *ConfigBuilder {
	b.config.MaxOpenConns = max
	return b
}

// MaxIdleConns 设置最大空闲连接数
func (b *ConfigBuilder) MaxIdleConns(max int) *ConfigBuilder {
	b.config.MaxIdleConns = max
	return b
}

// ConnLifetime 设置连接生存时间
func (b *ConfigBuilder) ConnLifetime(lifetime, idleTime time.Duration) *ConfigBuilder {
	b.config.ConnMaxLifetime = lifetime
	b.config.ConnMaxIdleTime = idleTime
	return b
}

// LogLevel 设置日志级别
func (b *ConfigBuilder) LogLevel(level string) *ConfigBuilder {
	b.config.LogLevel = level
	return b
}

// SlowThreshold 设置慢查询阈值
func (b *ConfigBuilder) SlowThreshold(threshold time.Duration) *ConfigBuilder {
	b.config.SlowThreshold = threshold
	return b
}

// TablePrefix 设置表名前缀
func (b *ConfigBuilder) TablePrefix(prefix string) *ConfigBuilder {
	b.config.TablePrefix = prefix
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

// DisableForeignKeyConstraints 禁用外键约束（迁移时）
func (b *ConfigBuilder) DisableForeignKeyConstraints() *ConfigBuilder {
	b.config.DisableForeignKeyConstraintWhenMigrating = true
	return b
}

// Sharding 设置分库分表配置
func (b *ConfigBuilder) Sharding(cfg *ShardingConfig) *ConfigBuilder {
	b.config.Sharding = cfg
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

// ===== 分库分表配置构建器 =====

// ShardingConfigBuilder 分库分表配置构建器
type ShardingConfigBuilder struct {
	config ShardingConfig
}

// NewShardingConfigBuilder 创建新的分库分表配置构建器
func NewShardingConfigBuilder() *ShardingConfigBuilder {
	return &ShardingConfigBuilder{
		config: ShardingConfig{
			ShardingAlgorithm: "hash",
			Tables:            make(map[string]*TableShardingConfig),
		},
	}
}

// ShardingKey 设置分片键
func (b *ShardingConfigBuilder) ShardingKey(key string) *ShardingConfigBuilder {
	b.config.ShardingKey = key
	return b
}

// NumberOfShards 设置分片数量
func (b *ShardingConfigBuilder) NumberOfShards(num int) *ShardingConfigBuilder {
	b.config.NumberOfShards = num
	return b
}

// Algorithm 设置分片算法
func (b *ShardingConfigBuilder) Algorithm(algorithm string) *ShardingConfigBuilder {
	b.config.ShardingAlgorithm = algorithm
	return b
}

// AddTable 添加表分片配置
func (b *ShardingConfigBuilder) AddTable(tableName string, cfg *TableShardingConfig) *ShardingConfigBuilder {
	b.config.Tables[tableName] = cfg
	return b
}

// Build 构建分库分表配置
func (b *ShardingConfigBuilder) Build() *ShardingConfig {
	return &b.config
}

// ===== 便捷方法 =====

// MySQLConfig 创建 MySQL 配置构建器
func MySQLConfig(dsn string) *ConfigBuilder {
	return NewConfigBuilder().Driver("mysql").DSN(dsn)
}

// PostgreSQLConfig 创建 PostgreSQL 配置构建器
func PostgreSQLConfig(dsn string) *ConfigBuilder {
	return NewConfigBuilder().Driver("postgres").DSN(dsn)
}

// SQLiteConfig 创建 SQLite 配置构建器
func SQLiteConfig(dsn string) *ConfigBuilder {
	return NewConfigBuilder().Driver("sqlite").DSN(dsn)
}
