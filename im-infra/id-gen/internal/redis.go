package internal

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// redisIDGenerator Redis 自增 ID 生成器的实现
type redisIDGenerator struct {
	config RedisConfig
	cache  cache.Cache
	logger clog.Logger
}

// NewRedisIDGenerator 创建新的 Redis 自增 ID 生成器
func NewRedisIDGenerator(config RedisConfig) (RedisIDGenerator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid redis config: %w", err)
	}

	logger := clog.Module("idgen")
	logger.Info("创建 Redis ID 生成器",
		clog.String("key_prefix", config.KeyPrefix),
		clog.String("default_key", config.DefaultKey),
		clog.Int64("step", config.Step),
		clog.Int64("initial_value", config.InitialValue),
		clog.Duration("ttl", config.TTL),
	)

	// 创建缓存实例
	cacheInstance, err := cache.New(config.CacheConfig)
	if err != nil {
		logger.Error("创建缓存实例失败", clog.Err(err))
		return nil, fmt.Errorf("failed to create cache instance: %w", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := cacheInstance.Ping(ctx); err != nil {
		logger.Error("Redis 连接测试失败", clog.Err(err))
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	logger.Info("Redis ID 生成器创建成功")
	return &redisIDGenerator{
		config: config,
		cache:  cacheInstance,
		logger: logger,
	}, nil
}

// GenerateString 生成字符串类型的 ID
func (g *redisIDGenerator) GenerateString(ctx context.Context) (string, error) {
	id, err := g.GenerateInt64(ctx)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

// GenerateInt64 生成 int64 类型的 ID
func (g *redisIDGenerator) GenerateInt64(ctx context.Context) (int64, error) {
	return g.GenerateWithKey(ctx, g.config.DefaultKey)
}

// GenerateWithKey 使用指定键生成 ID
func (g *redisIDGenerator) GenerateWithKey(ctx context.Context, key string) (int64, error) {
	return g.GenerateWithStep(ctx, g.config.Step)
}

// GenerateWithStep 使用指定步长生成 ID
func (g *redisIDGenerator) GenerateWithStep(ctx context.Context, step int64) (int64, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		g.logger.Debug("生成 Redis ID",
			clog.Duration("duration", duration),
			clog.Int64("step", step),
		)
	}()

	fullKey := g.formatKey(g.config.DefaultKey)

	// 检查键是否存在，如果不存在则初始化
	exists, err := g.cache.Exists(ctx, fullKey)
	if err != nil {
		g.logger.Error("检查键是否存在失败", clog.String("key", fullKey), clog.Err(err))
		return 0, fmt.Errorf("failed to check key existence: %w", err)
	}

	if exists == 0 {
		// 键不存在，初始化为初始值
		err = g.cache.Set(ctx, fullKey, g.config.InitialValue, g.config.TTL)
		if err != nil {
			g.logger.Error("初始化键失败", clog.String("key", fullKey), clog.Err(err))
			return 0, fmt.Errorf("failed to initialize key: %w", err)
		}
		g.logger.Debug("初始化 Redis 键", clog.String("key", fullKey), clog.Int64("initial_value", g.config.InitialValue))
	}

	// 使用 INCRBY 原子性地增加值
	var id int64
	for i := int64(0); i < step; i++ {
		id, err = g.cache.Incr(ctx, fullKey)
		if err != nil {
			g.logger.Error("自增操作失败", clog.String("key", fullKey), clog.Err(err))
			return 0, fmt.Errorf("failed to increment key: %w", err)
		}
	}

	// 如果设置了 TTL，更新过期时间
	if g.config.TTL > 0 {
		err = g.cache.Expire(ctx, fullKey, g.config.TTL)
		if err != nil {
			g.logger.Warn("设置过期时间失败", clog.String("key", fullKey), clog.Err(err))
			// 不返回错误，因为 ID 已经生成成功
		}
	}

	g.logger.Debug("生成 Redis ID 成功", clog.String("key", fullKey), clog.Int64("id", id))
	return id, nil
}

// Reset 重置指定键的计数器
func (g *redisIDGenerator) Reset(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		g.logger.Debug("重置计数器",
			clog.String("key", key),
			clog.Duration("duration", duration),
		)
	}()

	fullKey := g.formatKey(key)
	
	err := g.cache.Set(ctx, fullKey, g.config.InitialValue, g.config.TTL)
	if err != nil {
		g.logger.Error("重置计数器失败", clog.String("key", fullKey), clog.Err(err))
		return fmt.Errorf("failed to reset counter: %w", err)
	}

	g.logger.Info("重置计数器成功", clog.String("key", fullKey), clog.Int64("initial_value", g.config.InitialValue))
	return nil
}

// GetCurrent 获取当前计数值
func (g *redisIDGenerator) GetCurrent(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		g.logger.Debug("获取当前计数值",
			clog.String("key", key),
			clog.Duration("duration", duration),
		)
	}()

	fullKey := g.formatKey(key)
	
	value, err := g.cache.Get(ctx, fullKey)
	if err != nil {
		g.logger.Error("获取当前计数值失败", clog.String("key", fullKey), clog.Err(err))
		return 0, fmt.Errorf("failed to get current value: %w", err)
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		g.logger.Error("解析计数值失败", clog.String("key", fullKey), clog.String("value", value), clog.Err(err))
		return 0, fmt.Errorf("failed to parse current value: %w", err)
	}

	g.logger.Debug("获取当前计数值成功", clog.String("key", fullKey), clog.Int64("value", id))
	return id, nil
}

// formatKey 格式化键名，添加前缀
func (g *redisIDGenerator) formatKey(key string) string {
	if g.config.KeyPrefix == "" {
		return key
	}
	return g.config.KeyPrefix + ":" + key
}

// Type 返回生成器类型
func (g *redisIDGenerator) Type() GeneratorType {
	return RedisType
}

// Close 关闭生成器
func (g *redisIDGenerator) Close() error {
	g.logger.Info("关闭 Redis ID 生成器")
	if g.cache != nil {
		return g.cache.Close()
	}
	return nil
}
