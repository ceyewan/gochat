package internal

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// hashOperations 实现哈希操作的结构体
type hashOperations struct {
	client    *redis.Client
	logger    clog.Logger
	keyPrefix string
}

// newHashOperations 创建哈希操作实例
func newHashOperations(client *redis.Client, logger clog.Logger, keyPrefix string) *hashOperations {
	return &hashOperations{
		client:    client,
		logger:    logger,
		keyPrefix: keyPrefix,
	}
}

// formatKey 格式化键名，添加前缀
func (h *hashOperations) formatKey(key string) string {
	if h.keyPrefix == "" {
		return key
	}
	return h.keyPrefix + ":" + key
}

// HGet 获取哈希字段值
func (h *hashOperations) HGet(ctx context.Context, key, field string) (string, error) {
	formattedKey := h.formatKey(key)
	result, err := h.client.HGet(ctx, formattedKey, field).Result()
	if err != nil {
		h.logger.Error("Failed to HGet", clog.String("key", formattedKey), clog.String("field", field), clog.Err(err))
		return "", err
	}
	return result, nil
}

// HSet 设置哈希字段值
func (h *hashOperations) HSet(ctx context.Context, key, field string, value interface{}) error {
	formattedKey := h.formatKey(key)
	err := h.client.HSet(ctx, formattedKey, field, value).Err()
	if err != nil {
		h.logger.Error("Failed to HSet", clog.String("key", formattedKey), clog.String("field", field), clog.Err(err))
		return err
	}
	return nil
}

// HGetAll 获取哈希的所有字段和值
func (h *hashOperations) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	formattedKey := h.formatKey(key)
	result, err := h.client.HGetAll(ctx, formattedKey).Result()
	if err != nil {
		h.logger.Error("Failed to HGetAll", clog.String("key", formattedKey), clog.Err(err))
		return nil, err
	}
	return result, nil
}

// HDel 删除哈希字段
func (h *hashOperations) HDel(ctx context.Context, key string, fields ...string) error {
	formattedKey := h.formatKey(key)
	err := h.client.HDel(ctx, formattedKey, fields...).Err()
	if err != nil {
		h.logger.Error("Failed to HDel", clog.String("key", formattedKey), clog.Any("fields", fields), clog.Err(err))
		return err
	}
	return nil
}

// HExists 检查哈希字段是否存在
func (h *hashOperations) HExists(ctx context.Context, key, field string) (bool, error) {
	formattedKey := h.formatKey(key)
	result, err := h.client.HExists(ctx, formattedKey, field).Result()
	if err != nil {
		h.logger.Error("Failed to HExists", clog.String("key", formattedKey), clog.String("field", field), clog.Err(err))
		return false, err
	}
	return result, nil
}

// HLen 获取哈希字段数量
func (h *hashOperations) HLen(ctx context.Context, key string) (int64, error) {
	formattedKey := h.formatKey(key)
	result, err := h.client.HLen(ctx, formattedKey).Result()
	if err != nil {
		h.logger.Error("Failed to HLen", clog.String("key", formattedKey), clog.Err(err))
		return 0, err
	}
	return result, err
}
