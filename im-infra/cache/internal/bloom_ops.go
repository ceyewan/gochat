package internal

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// bloomFilterOperations 实现了 BloomFilterOperations 接口
type bloomFilterOperations struct {
	client    *redis.Client
	logger    clog.Logger
	keyPrefix string
}

// newBloomFilterOperations 创建一个新的 bloomFilterOperations 实例
func newBloomFilterOperations(client *redis.Client, logger clog.Logger, keyPrefix string) *bloomFilterOperations {
	return &bloomFilterOperations{
		client:    client,
		logger:    logger,
		keyPrefix: keyPrefix,
	}
}

// formatKey 格式化键名，添加前缀
func (b *bloomFilterOperations) formatKey(key string) string {
	if b.keyPrefix == "" {
		return key
	}
	return b.keyPrefix + ":bf:" + key
}

// BFAdd 向布隆过滤器中添加一个元素
func (b *bloomFilterOperations) BFAdd(ctx context.Context, key string, item string) error {
	formattedKey := b.formatKey(key)
	err := b.client.Do(ctx, "BF.ADD", formattedKey, item).Err()
	if err != nil {
		b.logger.Error("Failed to add to bloom filter", clog.String("key", formattedKey), clog.String("item", item), clog.Err(err))
		return fmt.Errorf("failed to add to bloom filter: %w", err)
	}
	return nil
}

// BFExists 检查一个元素是否存在于布隆过滤器中
func (b *bloomFilterOperations) BFExists(ctx context.Context, key string, item string) (bool, error) {
	formattedKey := b.formatKey(key)
	result, err := b.client.Do(ctx, "BF.EXISTS", formattedKey, item).Bool()
	if err != nil {
		b.logger.Error("Failed to check bloom filter existence", clog.String("key", formattedKey), clog.String("item", item), clog.Err(err))
		return false, fmt.Errorf("failed to check bloom filter existence: %w", err)
	}
	return result, nil
}

// BFInit 初始化一个布隆过滤器
func (b *bloomFilterOperations) BFInit(ctx context.Context, key string, errorRate float64, capacity int64) error {
	formattedKey := b.formatKey(key)
	err := b.client.Do(ctx, "BF.RESERVE", formattedKey, errorRate, capacity).Err()
	// 忽略 "item exists" 错误，因为我们希望它能被重复调用
	if err != nil && err.Error() != "ERR item exists" {
		b.logger.Error("Failed to init bloom filter", clog.String("key", formattedKey), clog.Err(err))
		return fmt.Errorf("failed to init bloom filter: %w", err)
	}
	return nil
}
