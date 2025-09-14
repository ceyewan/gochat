package internal

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// setOperations 实现集合操作的结构体
type setOperations struct {
	client    *redis.Client
	logger    clog.Logger
	keyPrefix string
}

// newSetOperations 创建集合操作实例
func newSetOperations(client *redis.Client, logger clog.Logger, keyPrefix string) *setOperations {
	return &setOperations{
		client:    client,
		logger:    logger,
		keyPrefix: keyPrefix,
	}
}

// formatKey 格式化键名，添加前缀
func (s *setOperations) formatKey(key string) string {
	if s.keyPrefix == "" {
		return key
	}
	// 如果前缀已经以冒号结尾，直接拼接
	if len(s.keyPrefix) > 0 && s.keyPrefix[len(s.keyPrefix)-1] == ':' {
		return s.keyPrefix + key
	}
	return s.keyPrefix + ":" + key
}

// SAdd 向集合添加成员
func (s *setOperations) SAdd(ctx context.Context, key string, members ...interface{}) error {
	formattedKey := s.formatKey(key)
	err := s.client.SAdd(ctx, formattedKey, members...).Err()
	if err != nil {
		s.logger.Error("Failed to SAdd", clog.String("key", formattedKey), clog.Any("members", members), clog.Err(err))
		return err
	}
	return nil
}

// SRem 从集合移除成员
func (s *setOperations) SRem(ctx context.Context, key string, members ...interface{}) error {
	formattedKey := s.formatKey(key)
	err := s.client.SRem(ctx, formattedKey, members...).Err()
	if err != nil {
		s.logger.Error("Failed to SRem", clog.String("key", formattedKey), clog.Any("members", members), clog.Err(err))
		return err
	}
	return nil
}

// SMembers 获取集合所有成员
func (s *setOperations) SMembers(ctx context.Context, key string) ([]string, error) {
	formattedKey := s.formatKey(key)
	result, err := s.client.SMembers(ctx, formattedKey).Result()
	if err != nil {
		s.logger.Error("Failed to SMembers", clog.String("key", formattedKey), clog.Err(err))
		return nil, err
	}
	return result, nil
}

// SIsMember 检查成员是否在集合中
func (s *setOperations) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	formattedKey := s.formatKey(key)
	result, err := s.client.SIsMember(ctx, formattedKey, member).Result()
	if err != nil {
		s.logger.Error("Failed to SIsMember", clog.String("key", formattedKey), clog.Any("member", member), clog.Err(err))
		return false, err
	}
	return result, nil
}

// SCard 获取集合成员数量
func (s *setOperations) SCard(ctx context.Context, key string) (int64, error) {
	formattedKey := s.formatKey(key)
	result, err := s.client.SCard(ctx, formattedKey).Result()
	if err != nil {
		s.logger.Error("Failed to SCard", clog.String("key", formattedKey), clog.Err(err))
		return 0, err
	}
	return result, nil
}
