package idgen

import (
	"context"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/idgen/internal"
)

// IDGenerator 定义 ID 生成器的核心接口
type IDGenerator = internal.IDGenerator

// SnowflakeGenerator 定义雪花算法 ID 生成器的接口
type SnowflakeGenerator = internal.SnowflakeGenerator

// UUIDGenerator 定义 UUID 生成器的接口
type UUIDGenerator = internal.UUIDGenerator

// RedisIDGenerator 定义 Redis 自增 ID 生成器的接口
type RedisIDGenerator = internal.RedisIDGenerator

// Config 是 idgen 的主配置结构体
type Config = internal.Config

// SnowflakeConfig 雪花算法配置
type SnowflakeConfig = internal.SnowflakeConfig

// UUIDConfig UUID 生成器配置
type UUIDConfig = internal.UUIDConfig

// RedisConfig Redis 自增 ID 配置
type RedisConfig = internal.RedisConfig

// GeneratorType 定义 ID 生成器类型
type GeneratorType = internal.GeneratorType

// 生成器类型常量
const (
	SnowflakeType = internal.SnowflakeType
	UUIDType      = internal.UUIDType
	RedisType     = internal.RedisType
)

var (
	// 全局默认 ID 生成器实例
	defaultGenerator IDGenerator
	// 确保默认生成器只初始化一次
	defaultGeneratorOnce sync.Once
	// 模块日志器
	logger = clog.Module("idgen")
)

// getDefaultGenerator 获取全局默认 ID 生成器实例，使用懒加载和单例模式
func getDefaultGenerator() IDGenerator {
	defaultGeneratorOnce.Do(func() {
		cfg := DefaultConfig()
		var err error
		defaultGenerator, err = internal.NewIDGenerator(cfg)
		if err != nil {
			logger.Error("创建默认 ID 生成器失败", clog.Err(err))
			panic(err)
		}
		logger.Info("默认 ID 生成器初始化成功", clog.String("type", cfg.Type.String()))
	})
	return defaultGenerator
}

// New 根据提供的配置创建一个新的 IDGenerator 实例
//
// 示例：
//
//	cfg := idgen.Config{
//	  Type: idgen.SnowflakeType,
//	  Snowflake: &idgen.SnowflakeConfig{
//	    NodeID: 1,
//	    AutoNodeID: false,
//	  },
//	}
//	generator, err := idgen.New(cfg)
//	if err != nil {
//	  log.Fatal(err)
//	}
//	id, _ := generator.GenerateInt64(ctx)
func New(cfg Config) (IDGenerator, error) {
	return internal.NewIDGenerator(cfg)
}

// DefaultConfig 返回一个带有合理默认值的 Config
// 默认使用雪花算法，自动生成节点 ID
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// DefaultSnowflakeConfig 返回默认雪花算法配置
func DefaultSnowflakeConfig() Config {
	return internal.DefaultSnowflakeConfig()
}

// DefaultUUIDConfig 返回默认 UUID 配置
func DefaultUUIDConfig() Config {
	return internal.DefaultUUIDConfig()
}

// DefaultRedisConfig 返回默认 Redis 配置
func DefaultRedisConfig() Config {
	return internal.DefaultRedisConfig()
}

// ===== 全局 ID 生成方法 =====

// GenerateString 使用全局默认生成器生成字符串类型的 ID
func GenerateString(ctx context.Context) (string, error) {
	return getDefaultGenerator().GenerateString(ctx)
}

// GenerateInt64 使用全局默认生成器生成 int64 类型的 ID
func GenerateInt64(ctx context.Context) (int64, error) {
	return getDefaultGenerator().GenerateInt64(ctx)
}

// Type 返回全局默认生成器的类型
func Type() GeneratorType {
	return getDefaultGenerator().Type()
}

// ===== 便捷的生成器创建方法 =====

// NewSnowflakeGenerator 创建雪花算法 ID 生成器
func NewSnowflakeGenerator(config *SnowflakeConfig) (SnowflakeGenerator, error) {
	return internal.NewSnowflakeIDGenerator(config)
}

// NewUUIDGenerator 创建 UUID ID 生成器
func NewUUIDGenerator(config *UUIDConfig) (UUIDGenerator, error) {
	return internal.NewUUIDIDGenerator(config)
}

// NewRedisGenerator 创建 Redis 自增 ID 生成器
func NewRedisGenerator(config *RedisConfig) (RedisIDGenerator, error) {
	return internal.NewRedisIDIDGenerator(config)
}

// ===== 向后兼容的方法 =====

// GetSnowflakeID 生成一个全局唯一的64位整数ID（向后兼容）
// 使用全局默认的雪花算法生成器
func GetSnowflakeID() int64 {
	ctx := context.Background()

	// 如果默认生成器不是雪花算法类型，创建一个雪花算法生成器
	if getDefaultGenerator().Type() != SnowflakeType {
		generator, err := internal.CreateDefaultSnowflakeGenerator()
		if err != nil {
			logger.Error("创建雪花算法生成器失败", clog.Err(err))
			// 返回时间戳作为备用方案
			return time.Now().UnixNano()
		}
		defer generator.Close()

		id, err := generator.GenerateInt64(ctx)
		if err != nil {
			logger.Error("生成雪花 ID 失败", clog.Err(err))
			return time.Now().UnixNano()
		}
		return id
	}

	id, err := getDefaultGenerator().GenerateInt64(ctx)
	if err != nil {
		logger.Error("生成 ID 失败", clog.Err(err))
		// 返回时间戳作为备用方案
		return time.Now().UnixNano()
	}
	return id
}
