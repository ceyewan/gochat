package internal

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// stringOperations 实现字符串操作的结构体
type stringOperations struct {
	client    *redis.Client
	logger    clog.Logger
	keyPrefix string
}

// newStringOperations 创建字符串操作实例
func newStringOperations(client *redis.Client, logger clog.Logger, keyPrefix string) *stringOperations {
	return &stringOperations{
		client:    client,
		logger:    logger,
		keyPrefix: keyPrefix,
	}
}

// formatKey 格式化键名，添加前缀
func (s *stringOperations) formatKey(key string) string {
	if s.keyPrefix == "" {
		return key
	}
	return s.keyPrefix + ":" + key
}

// Get 获取字符串值
func (s *stringOperations) Get(ctx context.Context, key string) (string, error) {
	formattedKey := s.formatKey(key)
	result, err := s.client.Get(ctx, formattedKey).Result()
	if err != nil {
		s.logger.Error("Failed to Get", clog.String("key", formattedKey), clog.Err(err))
		return "", err
	}
	return result, nil
}

// Set 设置字符串值
func (s *stringOperations) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	formattedKey := s.formatKey(key)
	err := s.client.Set(ctx, formattedKey, value, expiration).Err()
	if err != nil {
		s.logger.Error("Failed to Set", clog.String("key", formattedKey), clog.Any("value", value), clog.Duration("expiration", expiration), clog.Err(err))
		return err
	}
	return nil
}

// SetNX 当键不存在时设置字符串值
func (s *stringOperations) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	formattedKey := s.formatKey(key)
	result, err := s.client.SetNX(ctx, formattedKey, value, expiration).Result()
	if err != nil {
		s.logger.Error("Failed to SetNX", clog.String("key", formattedKey), clog.Any("value", value), clog.Duration("expiration", expiration), clog.Err(err))
		return false, err
	}
	return result, nil
}

// Incr 递增操作
func (s *stringOperations) Incr(ctx context.Context, key string) (int64, error) {
	formattedKey := s.formatKey(key)
	result, err := s.client.Incr(ctx, formattedKey).Result()
	if err != nil {
		s.logger.Error("Failed to Incr", clog.String("key", formattedKey), clog.Err(err))
		return 0, err
	}
	return result, nil
}

// Decr 递减操作
func (s *stringOperations) Decr(ctx context.Context, key string) (int64, error) {
	formattedKey := s.formatKey(key)
	result, err := s.client.Decr(ctx, formattedKey).Result()
	if err != nil {
		s.logger.Error("Failed to Decr", clog.String("key", formattedKey), clog.Err(err))
		return 0, err
	}
	return result, nil
}

// Expire 设置键的过期时间
func (s *stringOperations) Expire(ctx context.Context, key string, expiration time.Duration) error {
	formattedKey := s.formatKey(key)
	err := s.client.Expire(ctx, formattedKey, expiration).Err()
	if err != nil {
		s.logger.Error("Failed to Expire", clog.String("key", formattedKey), clog.Duration("expiration", expiration), clog.Err(err))
		return err
	}
	return nil
}

// TTL 获取键的剩余生存时间
func (s *stringOperations) TTL(ctx context.Context, key string) (time.Duration, error) {
	formattedKey := s.formatKey(key)
	result, err := s.client.TTL(ctx, formattedKey).Result()
	if err != nil {
		s.logger.Error("Failed to TTL", clog.String("key", formattedKey), clog.Err(err))
		return 0, err
	}
	return result, nil
}

// Del 删除键
func (s *stringOperations) Del(ctx context.Context, keys ...string) error {
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = s.formatKey(key)
	}
	err := s.client.Del(ctx, formattedKeys...).Err()
	if err != nil {
		s.logger.Error("Failed to Del", clog.Any("keys", formattedKeys), clog.Err(err))
		return err
	}
	return nil
}

// Exists 检查键是否存在
func (s *stringOperations) Exists(ctx context.Context, keys ...string) (int64, error) {
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = s.formatKey(key)
	}
	result, err := s.client.Exists(ctx, formattedKeys...).Result()
	if err != nil {
		s.logger.Error("Failed to Exists", clog.Any("keys", formattedKeys), clog.Err(err))
		return 0, err
	}
	return result, nil
}
