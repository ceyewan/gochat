package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// client 是 Cache 接口的内部实现。
// 它包装了一个 *redis.Client，并提供接口方法。
type client struct {
	redisClient *redis.Client
	logger      clog.Logger
	config      Config

	// 嵌入各种操作
	*stringOperations
	*hashOperations
	*setOperations
	*lockOperations
	*bloomFilterOperations
	*scriptingOperations
}

// Config 配置结构体（内部使用）
type Config struct {
	Addr            string
	Password        string
	DB              int
	PoolSize        int
	MinIdleConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	PoolTimeout     time.Duration
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
	KeyPrefix       string
}

// NewCache 根据提供的配置创建一个新的 Cache 实例。
// 这是核心工厂函数，按配置组装所有组件。
func NewCache(ctx context.Context, cfg Config, logger clog.Logger) (*client, error) {
	// 创建 Redis 客户端选项
	redisOpts := &redis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolTimeout:     cfg.PoolTimeout,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	}

	// 创建 Redis 客户端
	redisCache := redis.NewClient(redisOpts)

	// 测试连接
	if err := redisCache.Ping(ctx).Err(); err != nil {
		logger.Error("Redis 连接测试失败", clog.Err(err))
		redisCache.Close()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	// 创建客户端实例
	c := &client{
		redisClient:           redisCache,
		logger:                logger,
		config:                cfg,
		stringOperations:      newStringOperations(redisCache, logger, cfg.KeyPrefix),
		hashOperations:        newHashOperations(redisCache, logger, cfg.KeyPrefix),
		setOperations:         newSetOperations(redisCache, logger, cfg.KeyPrefix),
		lockOperations:        newLockOperations(redisCache, logger, cfg.KeyPrefix),
		bloomFilterOperations: newBloomFilterOperations(redisCache, logger, cfg.KeyPrefix),
		scriptingOperations:   newScriptingOperations(redisCache, logger),
	}

	logger.Info("Cache 实例创建成功")
	return c, nil
}

// Ping 检查 Redis 连接是否正常
func (c *client) Ping(ctx context.Context) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("ping operation completed",
			clog.Duration("duration", duration),
		)
	}()

	err := c.redisClient.Ping(ctx).Err()
	if err != nil {
		c.logger.Error("redis ping failed", clog.Err(err))
		return fmt.Errorf("redis ping failed: %w", err)
	}

	c.logger.Debug("redis ping successful")
	return nil
}

// Close 关闭 Redis 连接
func (c *client) Close() error {
	c.logger.Info("closing redis connection")
	err := c.redisClient.Close()
	if err != nil {
		c.logger.Error("failed to close redis connection", clog.Err(err))
		return fmt.Errorf("failed to close redis client: %w", err)
	}
	c.logger.Info("redis connection closed")
	return nil
}
