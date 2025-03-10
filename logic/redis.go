package logic

import (
	"context"
	"time"

	"gochat/clog"
	"gochat/tools"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// InitRedisClient 初始化Redis客户端
func InitRedisClient() error {
	// 获取全局Redis客户端
	client, err := tools.GetRedisClient()
	if err != nil {
		clog.Error("[Redis] Initialization failed: %v", err)
		return err
	}
	RedisClient = client

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := testRedisConnection(ctx); err != nil {
		return err
	}

	clog.Info("[Redis] Connection successful")
	return nil
}

// testRedisConnection 测试Redis连接
func testRedisConnection(ctx context.Context) error {
	pong, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		clog.Error("[Redis] Connection test failed: %v", err)
		return err
	}
	clog.Debug("[Redis] Ping response: %s", pong)
	return nil
}
