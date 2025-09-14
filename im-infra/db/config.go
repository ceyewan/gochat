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

// ValidateConfig 验证配置的完整性和合理性
func ValidateConfig(cfg *Config) error {
	return internal.ValidateConfig(cfg)
}

// NewShardingConfig 创建新的分库分表配置
func NewShardingConfig(shardingKey string, numberOfShards int) *ShardingConfig {
	return &ShardingConfig{
		ShardingKey:    shardingKey,
		NumberOfShards: numberOfShards,
		Tables:         make(map[string]*TableShardingConfig),
	}
}
