package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// zsetOperations 是 ZSetOperations 接口的实现
type zsetOperations struct {
	client   *redis.Client
	logger   clog.Logger
	keyPrefix string
}

// newZSetOperations 创建一个新的 ZSetOperations 实例
func newZSetOperations(client *redis.Client, logger clog.Logger, keyPrefix string) *zsetOperations {
	return &zsetOperations{
		client:    client,
		logger:    logger,
		keyPrefix: keyPrefix,
	}
}

// ZAdd 添加一个或多个成员到有序集合
func (z *zsetOperations) ZAdd(ctx context.Context, key string, members ...*ZMember) error {
	formattedKey := z.formatKey(key)

	// 转换为 redis.Z 结构
	zMembers := make([]redis.Z, len(members))
	for i, member := range members {
		zMembers[i] = redis.Z{
			Score:  member.Score,
			Member: member.Member,
		}
	}

	err := z.client.ZAdd(ctx, formattedKey, zMembers...).Err()
	if err != nil {
		z.logger.Error("Failed to ZAdd", clog.String("key", formattedKey), clog.Err(err))
		return fmt.Errorf("zadd failed: %w", err)
	}

	z.logger.Debug("ZAdd successful", clog.String("key", formattedKey), clog.Int("count", len(members)))
	return nil
}

// ZRange 获取有序集合中指定范围内的成员，按分数从低到高排序
func (z *zsetOperations) ZRange(ctx context.Context, key string, start, stop int64) ([]*ZMember, error) {
	formattedKey := z.formatKey(key)

	result, err := z.client.ZRange(ctx, formattedKey, start, stop).Result()
	if err != nil {
		z.logger.Error("Failed to ZRange", clog.String("key", formattedKey), clog.Err(err))
		return nil, fmt.Errorf("zrange failed: %w", err)
	}

	// 获取每个成员的分数
	members := make([]*ZMember, len(result))
	for i, member := range result {
		score, err := z.client.ZScore(ctx, formattedKey, member).Result()
		if err != nil && err != redis.Nil {
			z.logger.Error("Failed to get score", clog.String("key", formattedKey), clog.String("member", fmt.Sprintf("%v", member)), clog.Err(err))
			return nil, fmt.Errorf("failed to get score: %w", err)
		}
		members[i] = &ZMember{
			Member: member,
			Score:  score,
		}
	}

	z.logger.Debug("ZRange successful", clog.String("key", formattedKey), clog.Int64("start", start), clog.Int64("stop", stop), clog.Int("count", len(members)))
	return members, nil
}

// ZRevRange 获取有序集合中指定范围内的成员，按分数从高到低排序
func (z *zsetOperations) ZRevRange(ctx context.Context, key string, start, stop int64) ([]*ZMember, error) {
	formattedKey := z.formatKey(key)

	result, err := z.client.ZRevRange(ctx, formattedKey, start, stop).Result()
	if err != nil {
		z.logger.Error("Failed to ZRevRange", clog.String("key", formattedKey), clog.Err(err))
		return nil, fmt.Errorf("zrevrange failed: %w", err)
	}

	// 获取每个成员的分数
	members := make([]*ZMember, len(result))
	for i, member := range result {
		score, err := z.client.ZScore(ctx, formattedKey, member).Result()
		if err != nil && err != redis.Nil {
			z.logger.Error("Failed to get score", clog.String("key", formattedKey), clog.String("member", fmt.Sprintf("%v", member)), clog.Err(err))
			return nil, fmt.Errorf("failed to get score: %w", err)
		}
		members[i] = &ZMember{
			Member: member,
			Score:  score,
		}
	}

	z.logger.Debug("ZRevRange successful", clog.String("key", formattedKey), clog.Int64("start", start), clog.Int64("stop", stop), clog.Int("count", len(members)))
	return members, nil
}

// ZRangeByScore 获取指定分数范围内的成员
func (z *zsetOperations) ZRangeByScore(ctx context.Context, key string, min, max float64) ([]*ZMember, error) {
	formattedKey := z.formatKey(key)

	opt := &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", min),
		Max: fmt.Sprintf("%f", max),
	}

	result, err := z.client.ZRangeByScore(ctx, formattedKey, opt).Result()
	if err != nil {
		z.logger.Error("Failed to ZRangeByScore", clog.String("key", formattedKey), clog.Err(err))
		return nil, fmt.Errorf("zrangebyscore failed: %w", err)
	}

	// 获取每个成员的分数
	members := make([]*ZMember, len(result))
	for i, member := range result {
		score, err := z.client.ZScore(ctx, formattedKey, member).Result()
		if err != nil && err != redis.Nil {
			z.logger.Error("Failed to get score", clog.String("key", formattedKey), clog.String("member", fmt.Sprintf("%v", member)), clog.Err(err))
			return nil, fmt.Errorf("failed to get score: %w", err)
		}
		members[i] = &ZMember{
			Member: member,
			Score:  score,
		}
	}

	z.logger.Debug("ZRangeByScore successful", clog.String("key", formattedKey), clog.Float64("min", min), clog.Float64("max", max), clog.Int("count", len(members)))
	return members, nil
}

// ZRem 从有序集合中移除一个或多个成员
func (z *zsetOperations) ZRem(ctx context.Context, key string, members ...interface{}) error {
	formattedKey := z.formatKey(key)

	removed, err := z.client.ZRem(ctx, formattedKey, members...).Result()
	if err != nil {
		z.logger.Error("Failed to ZRem", clog.String("key", formattedKey), clog.Err(err))
		return fmt.Errorf("zrem failed: %w", err)
	}

	z.logger.Debug("ZRem successful", clog.String("key", formattedKey), clog.Int64("removed", removed))
	return nil
}

// ZRemRangeByRank 移除有序集合中指定排名区间内的成员
func (z *zsetOperations) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error {
	formattedKey := z.formatKey(key)

	removed, err := z.client.ZRemRangeByRank(ctx, formattedKey, start, stop).Result()
	if err != nil {
		z.logger.Error("Failed to ZRemRangeByRank", clog.String("key", formattedKey), clog.Err(err))
		return fmt.Errorf("zremrangebyrank failed: %w", err)
	}

	z.logger.Debug("ZRemRangeByRank successful", clog.String("key", formattedKey), clog.Int64("start", start), clog.Int64("stop", stop), clog.Int64("removed", removed))
	return nil
}

// ZCard 获取有序集合的成员数量
func (z *zsetOperations) ZCard(ctx context.Context, key string) (int64, error) {
	formattedKey := z.formatKey(key)

	count, err := z.client.ZCard(ctx, formattedKey).Result()
	if err != nil {
		z.logger.Error("Failed to ZCard", clog.String("key", formattedKey), clog.Err(err))
		return 0, fmt.Errorf("zcard failed: %w", err)
	}

	z.logger.Debug("ZCard successful", clog.String("key", formattedKey), clog.Int64("count", count))
	return count, nil
}

// ZCount 获取指定分数范围内的成员数量
func (z *zsetOperations) ZCount(ctx context.Context, key string, min, max float64) (int64, error) {
	formattedKey := z.formatKey(key)

	minStr := fmt.Sprintf("%f", min)
	maxStr := fmt.Sprintf("%f", max)

	count, err := z.client.ZCount(ctx, formattedKey, minStr, maxStr).Result()
	if err != nil {
		z.logger.Error("Failed to ZCount", clog.String("key", formattedKey), clog.Err(err))
		return 0, fmt.Errorf("zcount failed: %w", err)
	}

	z.logger.Debug("ZCount successful", clog.String("key", formattedKey), clog.Float64("min", min), clog.Float64("max", max), clog.Int64("count", count))
	return count, nil
}

// ZScore 获取成员的分数
func (z *zsetOperations) ZScore(ctx context.Context, key string, member string) (float64, error) {
	formattedKey := z.formatKey(key)

	score, err := z.client.ZScore(ctx, formattedKey, member).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, ErrCacheMiss
		}
		z.logger.Error("Failed to ZScore", clog.String("key", formattedKey), clog.String("member", fmt.Sprintf("%v", member)), clog.Err(err))
		return 0, fmt.Errorf("zscore failed: %w", err)
	}

	z.logger.Debug("ZScore successful", clog.String("key", formattedKey), clog.String("member", fmt.Sprintf("%v", member)), clog.Float64("score", score))
	return score, nil
}

// ZSetExpire 为有序集合设置过期时间
func (z *zsetOperations) ZSetExpire(ctx context.Context, key string, expiration time.Duration) error {
	formattedKey := z.formatKey(key)

	err := z.client.Expire(ctx, formattedKey, expiration).Err()
	if err != nil {
		z.logger.Error("Failed to set expire for zset", clog.String("key", formattedKey), clog.Err(err))
		return fmt.Errorf("zset expire failed: %w", err)
	}

	z.logger.Debug("ZSetExpire successful", clog.String("key", formattedKey), clog.Duration("expiration", expiration))
	return nil
}

// formatKey 格式化键名，添加前缀
func (z *zsetOperations) formatKey(key string) string {
	if z.keyPrefix == "" {
		return key
	}
	return z.keyPrefix + key
}