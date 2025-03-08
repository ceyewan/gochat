package tools

import (
	"context"
	"fmt"
	"gochat/config"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	client *redis.Client
	once   sync.Once
)

// GetRedisClient 返回 Redis 客户端实例，如果未初始化则自动初始化
func GetRedisClient() (*redis.Client, error) {
	var initErr error
	once.Do(func() {
		client = redis.NewClient(&redis.Options{
			Addr:       config.Conf.Redis.Addr,
			Password:   config.Conf.Redis.Password,
			DB:         config.Conf.Redis.DB,
			MaxConnAge: 20 * time.Second,
		})
		// 测试连接是否成功
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			initErr = fmt.Errorf("redis connection failed: %v", err)
			client = nil
		}
	})
	if initErr != nil {
		return nil, initErr
	}
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	return client, nil
}
