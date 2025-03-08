package logic

import (
	"gochat/clog"
	"gochat/config"
	"time"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func InitRedisClient() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:       config.Conf.Redis.Host,
		Password:   config.Conf.Redis.Password,
		DB:         config.Conf.Redis.DB,
		MaxConnAge: 20 * time.Second,
	})
	// 测试连接
	_, err := RedisClient.Ping(RedisClient.Context()).Result()
	if err != nil {
		clog.Error("Redis连接失败: %v", err)
		return
	}
	clog.Info("Redis连接成功")
}
