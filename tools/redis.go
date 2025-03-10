package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gochat/clog"
	"gochat/config"

	"github.com/redis/go-redis/v9"
)

const (
	// Redis 连接超时时间
	redisConnTimeout = 5 * time.Second
)

var (
	// Redis 客户端实例
	redisClient *redis.Client
	redisOnce   sync.Once
	redisErr    error
)

// GetRedisClient 返回 Redis 客户端实例，确保只初始化一次
func GetRedisClient() (*redis.Client, error) {
	redisOnce.Do(func() {
		// 获取 Redis 配置
		conf := config.Conf.Redis
		if conf.Addr == "" {
			redisErr = fmt.Errorf("redis address not configured")
			clog.Error("Redis initialization failed: %v", redisErr)
			return
		}

		clog.Debug("Initializing Redis client connection to %s", conf.Addr)

		// 创建 Redis 客户端实例
		redisClient = redis.NewClient(&redis.Options{
			Addr:         conf.Addr,
			Password:     conf.Password,
			DB:           conf.DB,
			PoolTimeout:  4 * time.Second,
			PoolSize:     10, // 连接池大小
			MinIdleConns: 2,  // 最小空闲连接
		})

		// 测试连接是否成功
		ctx, cancel := context.WithTimeout(context.Background(), redisConnTimeout)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			redisErr = fmt.Errorf("redis connection failed: %v", err)
			redisClient = nil
			clog.Error("Failed to connect to Redis at %s: %v", conf.Addr, err)
			return
		}

		clog.Info("Redis client connected successfully to %s (DB: %d)", conf.Addr, conf.DB)
	})

	// 返回初始化结果
	if redisErr != nil {
		return nil, redisErr
	}

	return redisClient, nil
}

// CloseRedisClient 关闭 Redis 连接
func CloseRedisClient() error {
	if redisClient != nil {
		clog.Debug("Closing Redis client connection")
		if err := redisClient.Close(); err != nil {
			clog.Error("Failed to close Redis connection: %v", err)
			return err
		}
		clog.Info("Redis connection closed successfully")
	}
	return nil
}
