package logic

import (
	"context"
	"gochat/clog"
	"gochat/tools"
	"time"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func InitRedisClient() error {
	// 使用全局Redis客户端
	client, err := tools.GetRedisClient()
	if err != nil {
		clog.Error("Redis队列初始化失败: %s", err.Error())
		return err
	}
	RedisClient = client
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pong, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		clog.Error("Redis队列连接测试失败: %s", err.Error())
		return err
	}
	clog.Info("Redis队列连接成功: %s", pong)
	return nil
}

