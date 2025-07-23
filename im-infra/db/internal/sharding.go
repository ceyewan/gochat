package internal

import (
	"fmt"
	"strconv"

	"github.com/ceyewan/gochat/im-infra/clog"
	"gorm.io/gorm"
	"gorm.io/sharding"
)

// configureSharding 配置分库分表
func configureSharding(db *gorm.DB, cfg *ShardingConfig) error {
	logger := clog.Module("db.sharding")

	logger.Info("开始配置分库分表",
		clog.String("shardingKey", cfg.ShardingKey),
		clog.Int("numberOfShards", cfg.NumberOfShards),
	)

	// 创建分片配置
	shardingConfig := sharding.Config{
		ShardingKey:         cfg.ShardingKey,
		NumberOfShards:      uint(cfg.NumberOfShards),
		PrimaryKeyGenerator: sharding.PKSnowflake,
	}

	// 根据 gorm.io/sharding 的实际 API，Register 函数接受配置和表名/结构体列表
	var err error
	if len(cfg.Tables) > 0 {
		// 收集需要分片的表名
		tables := make([]interface{}, 0, len(cfg.Tables))
		for tableName := range cfg.Tables {
			tables = append(tables, tableName)
			logger.Info("添加分片表",
				clog.String("table", tableName),
			)
		}
		err = db.Use(sharding.Register(shardingConfig, tables...))
	} else {
		// 如果没有指定表，则对所有表应用分片规则
		err = db.Use(sharding.Register(shardingConfig))
	}

	if err != nil {
		logger.Error("注册分片插件失败", clog.Err(err))
		return fmt.Errorf("failed to register sharding plugin: %w", err)
	}

	logger.Info("分库分表配置完成")
	return nil
}

// ShardingHelper 分片辅助工具
type ShardingHelper struct {
	config *ShardingConfig
	logger clog.Logger
}

// NewShardingHelper 创建分片辅助工具
func NewShardingHelper(config *ShardingConfig) *ShardingHelper {
	return &ShardingHelper{
		config: config,
		logger: clog.Module("db.sharding.helper"),
	}
}

// GetShardSuffix 根据分片键值获取分片后缀
// 这是一个简化的实现，实际的分片逻辑由 gorm.io/sharding 库处理
func (h *ShardingHelper) GetShardSuffix(value interface{}) (string, error) {
	var intValue int64
	var err error

	switch v := value.(type) {
	case int:
		intValue = int64(v)
	case int32:
		intValue = int64(v)
	case int64:
		intValue = v
	case uint:
		intValue = int64(v)
	case uint32:
		intValue = int64(v)
	case uint64:
		intValue = int64(v)
	case string:
		intValue, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			// 对于字符串，使用简单的哈希
			hash := int64(0)
			for _, c := range v {
				hash = hash*31 + int64(c)
			}
			intValue = hash
		}
	default:
		return "", fmt.Errorf("unsupported sharding key type: %T", value)
	}

	if intValue < 0 {
		intValue = -intValue
	}

	shardIndex := intValue % int64(h.config.NumberOfShards)
	return fmt.Sprintf("_%02d", shardIndex), nil
}
