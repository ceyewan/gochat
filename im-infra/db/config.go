package db

import (
	"github.com/ceyewan/gochat/im-infra/db/internal"
)

// ===== 便捷配置构建函数 =====

// MySQLConfig 创建 MySQL 配置
func MySQLConfig(dsn string) Config {
	cfg := DefaultConfig()
	cfg.Driver = "mysql"
	cfg.DSN = dsn
	return cfg
}

// PostgreSQLConfig 创建 PostgreSQL 配置
func PostgreSQLConfig(dsn string) Config {
	cfg := DefaultConfig()
	cfg.Driver = "postgres"
	cfg.DSN = dsn
	return cfg
}

// SQLiteConfig 创建 SQLite 配置
func SQLiteConfig(dsn string) Config {
	cfg := DefaultConfig()
	cfg.Driver = "sqlite"
	cfg.DSN = dsn
	// SQLite 建议使用单连接
	cfg.MaxOpenConns = 1
	cfg.MaxIdleConns = 1
	return cfg
}

// ===== 验证函数 =====

// ValidateConfig 验证配置的完整性和合理性
func ValidateConfig(cfg Config) error {
	return internal.ValidateConfig(cfg)
}

// ===== 分库分表配置构建器 =====

// NewShardingConfig 创建新的分库分表配置
func NewShardingConfig(shardingKey string, numberOfShards int) *ShardingConfig {
	return &ShardingConfig{
		ShardingKey:       shardingKey,
		NumberOfShards:    numberOfShards,
		ShardingAlgorithm: "hash",
		Tables:            make(map[string]*TableShardingConfig),
	}
}
