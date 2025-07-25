package internal

import (
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// NewIDGenerator 根据配置创建 ID 生成器
func NewIDGenerator(cfg Config) (IDGenerator, error) {
	logger := clog.Module("idgen")
	
	if err := cfg.Validate(); err != nil {
		logger.Error("配置验证失败", clog.Err(err))
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger.Info("创建 ID 生成器", clog.String("type", cfg.Type.String()))

	switch cfg.Type {
	case SnowflakeType:
		return NewSnowflakeGenerator(*cfg.Snowflake)
		
	case UUIDType:
		return NewUUIDGenerator(*cfg.UUID)
		
	case RedisType:
		return NewRedisIDGenerator(*cfg.Redis)
		
	default:
		logger.Error("不支持的生成器类型", clog.String("type", cfg.Type.String()))
		return nil, fmt.Errorf("unsupported generator type: %s", cfg.Type)
	}
}

// NewSnowflakeIDGenerator 创建雪花算法 ID 生成器的便捷方法
func NewSnowflakeIDGenerator(config *SnowflakeConfig) (SnowflakeGenerator, error) {
	if config == nil {
		config = &SnowflakeConfig{
			NodeID:     0,
			AutoNodeID: true,
			Epoch:      1288834974657, // Twitter 雪花算法起始时间
		}
	}
	return NewSnowflakeGenerator(*config)
}

// NewUUIDIDGenerator 创建 UUID ID 生成器的便捷方法
func NewUUIDIDGenerator(config *UUIDConfig) (UUIDGenerator, error) {
	if config == nil {
		config = &UUIDConfig{
			Version:   4,
			Format:    "standard",
			UpperCase: false,
		}
	}
	return NewUUIDGenerator(*config)
}

// NewRedisIDIDGenerator 创建 Redis 自增 ID 生成器的便捷方法
func NewRedisIDIDGenerator(config *RedisConfig) (RedisIDGenerator, error) {
	if config == nil {
		return nil, fmt.Errorf("redis config is required")
	}
	return NewRedisIDGenerator(*config)
}

// CreateDefaultSnowflakeGenerator 创建默认配置的雪花算法生成器
func CreateDefaultSnowflakeGenerator() (SnowflakeGenerator, error) {
	config := SnowflakeConfig{
		NodeID:     0,
		AutoNodeID: true,
		Epoch:      1288834974657,
	}
	return NewSnowflakeGenerator(config)
}

// CreateDefaultUUIDGenerator 创建默认配置的 UUID 生成器
func CreateDefaultUUIDGenerator() (UUIDGenerator, error) {
	config := UUIDConfig{
		Version:   4,
		Format:    "standard",
		UpperCase: false,
	}
	return NewUUIDGenerator(config)
}

// CreateDefaultRedisGenerator 创建默认配置的 Redis 生成器
func CreateDefaultRedisGenerator() (RedisIDGenerator, error) {
	// Redis 生成器需要缓存配置，不能使用完全默认的配置
	return nil, fmt.Errorf("redis generator requires explicit cache configuration")
}
