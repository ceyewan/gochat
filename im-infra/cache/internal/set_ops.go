package internal

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// SAdd 向集合添加一个或多个成员
func (c *cache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}
	if len(members) == 0 {
		return fmt.Errorf("members cannot be empty")
	}

	formattedKey := c.formatKey(key)

	// 序列化所有成员
	serializedMembers := make([]interface{}, len(members))
	for i, member := range members {
		serializedValue, err := c.serialize(member)
		if err != nil {
			c.logger.Error("序列化成员失败",
				clog.String("key", key),
				clog.Int("memberIndex", i),
				clog.Err(err),
			)
			return fmt.Errorf("failed to serialize member at index %d for key %s: %w", i, key, err)
		}
		serializedMembers[i] = serializedValue
	}

	return c.executeWithLogging(ctx, "SADD", key, func() error {
		addedCount, err := c.client.SAdd(ctx, formattedKey, serializedMembers...).Result()
		if err != nil {
			return c.handleRedisError("SADD", key, err)
		}

		c.logger.Debug("添加集合成员完成",
			clog.String("key", key),
			clog.Int("memberCount", len(members)),
			clog.Int64("addedCount", addedCount),
		)

		return nil
	})
}

// SRem 从集合移除一个或多个成员
func (c *cache) SRem(ctx context.Context, key string, members ...interface{}) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}
	if len(members) == 0 {
		return fmt.Errorf("members cannot be empty")
	}

	formattedKey := c.formatKey(key)

	// 序列化所有成员
	serializedMembers := make([]interface{}, len(members))
	for i, member := range members {
		serializedValue, err := c.serialize(member)
		if err != nil {
			c.logger.Error("序列化成员失败",
				clog.String("key", key),
				clog.Int("memberIndex", i),
				clog.Err(err),
			)
			return fmt.Errorf("failed to serialize member at index %d for key %s: %w", i, key, err)
		}
		serializedMembers[i] = serializedValue
	}

	return c.executeWithLogging(ctx, "SREM", key, func() error {
		removedCount, err := c.client.SRem(ctx, formattedKey, serializedMembers...).Result()
		if err != nil {
			return c.handleRedisError("SREM", key, err)
		}

		c.logger.Debug("移除集合成员完成",
			clog.String("key", key),
			clog.Int("memberCount", len(members)),
			clog.Int64("removedCount", removedCount),
		)

		return nil
	})
}

// SMembers 获取集合的所有成员
func (c *cache) SMembers(ctx context.Context, key string) ([]string, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKey(key); err != nil {
		return nil, err
	}

	formattedKey := c.formatKey(key)

	var result []string
	err := c.executeWithLogging(ctx, "SMEMBERS", key, func() error {
		members, err := c.client.SMembers(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("SMEMBERS", key, err)
		}
		result = members
		return nil
	})

	return result, err
}

// SIsMember 检查成员是否在集合中
func (c *cache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	if err := c.validateContext(ctx); err != nil {
		return false, err
	}
	if err := c.validateKey(key); err != nil {
		return false, err
	}

	formattedKey := c.formatKey(key)
	serializedMember, err := c.serialize(member)
	if err != nil {
		c.logger.Error("序列化成员失败",
			clog.String("key", key),
			clog.Err(err),
		)
		return false, fmt.Errorf("failed to serialize member for key %s: %w", key, err)
	}

	var result bool
	err = c.executeWithLogging(ctx, "SISMEMBER", key, func() error {
		isMember, err := c.client.SIsMember(ctx, formattedKey, serializedMember).Result()
		if err != nil {
			return c.handleRedisError("SISMEMBER", key, err)
		}
		result = isMember
		return nil
	})

	return result, err
}

// SCard 获取集合的成员数量
func (c *cache) SCard(ctx context.Context, key string) (int64, error) {
	if err := c.validateContext(ctx); err != nil {
		return 0, err
	}
	if err := c.validateKey(key); err != nil {
		return 0, err
	}

	formattedKey := c.formatKey(key)

	var result int64
	err := c.executeWithLogging(ctx, "SCARD", key, func() error {
		count, err := c.client.SCard(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("SCARD", key, err)
		}
		result = count
		return nil
	})

	return result, err
}

// SPop 随机移除并返回集合中的一个成员
func (c *cache) SPop(ctx context.Context, key string) (string, error) {
	if err := c.validateContext(ctx); err != nil {
		return "", err
	}
	if err := c.validateKey(key); err != nil {
		return "", err
	}

	formattedKey := c.formatKey(key)

	var result string
	err := c.executeWithLogging(ctx, "SPOP", key, func() error {
		member, err := c.client.SPop(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("SPOP", key, err)
		}
		result = member
		return nil
	})

	return result, err
}

// SPopN 随机移除并返回集合中的多个成员
func (c *cache) SPopN(ctx context.Context, key string, count int64) ([]string, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKey(key); err != nil {
		return nil, err
	}
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	formattedKey := c.formatKey(key)

	var result []string
	err := c.executeWithLogging(ctx, "SPOPN", key, func() error {
		members, err := c.client.SPopN(ctx, formattedKey, count).Result()
		if err != nil {
			return c.handleRedisError("SPOPN", key, err)
		}
		result = members
		return nil
	})

	return result, err
}

// SRandMember 随机返回集合中的一个成员（不移除）
func (c *cache) SRandMember(ctx context.Context, key string) (string, error) {
	if err := c.validateContext(ctx); err != nil {
		return "", err
	}
	if err := c.validateKey(key); err != nil {
		return "", err
	}

	formattedKey := c.formatKey(key)

	var result string
	err := c.executeWithLogging(ctx, "SRANDMEMBER", key, func() error {
		member, err := c.client.SRandMember(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("SRANDMEMBER", key, err)
		}
		result = member
		return nil
	})

	return result, err
}

// SRandMemberN 随机返回集合中的多个成员（不移除）
func (c *cache) SRandMemberN(ctx context.Context, key string, count int64) ([]string, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKey(key); err != nil {
		return nil, err
	}

	formattedKey := c.formatKey(key)

	var result []string
	err := c.executeWithLogging(ctx, "SRANDMEMBERN", key, func() error {
		members, err := c.client.SRandMemberN(ctx, formattedKey, count).Result()
		if err != nil {
			return c.handleRedisError("SRANDMEMBERN", key, err)
		}
		result = members
		return nil
	})

	return result, err
}

// SUnion 返回多个集合的并集
func (c *cache) SUnion(ctx context.Context, keys ...string) ([]string, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKeys(keys); err != nil {
		return nil, err
	}

	// 格式化所有键名
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = c.formatKey(key)
	}

	var result []string
	err := c.executeWithLogging(ctx, "SUNION", fmt.Sprintf("%v", keys), func() error {
		members, err := c.client.SUnion(ctx, formattedKeys...).Result()
		if err != nil {
			return c.handleRedisError("SUNION", fmt.Sprintf("%v", keys), err)
		}
		result = members
		return nil
	})

	return result, err
}

// SInter 返回多个集合的交集
func (c *cache) SInter(ctx context.Context, keys ...string) ([]string, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKeys(keys); err != nil {
		return nil, err
	}

	// 格式化所有键名
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = c.formatKey(key)
	}

	var result []string
	err := c.executeWithLogging(ctx, "SINTER", fmt.Sprintf("%v", keys), func() error {
		members, err := c.client.SInter(ctx, formattedKeys...).Result()
		if err != nil {
			return c.handleRedisError("SINTER", fmt.Sprintf("%v", keys), err)
		}
		result = members
		return nil
	})

	return result, err
}

// SDiff 返回多个集合的差集
func (c *cache) SDiff(ctx context.Context, keys ...string) ([]string, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKeys(keys); err != nil {
		return nil, err
	}

	// 格式化所有键名
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = c.formatKey(key)
	}

	var result []string
	err := c.executeWithLogging(ctx, "SDIFF", fmt.Sprintf("%v", keys), func() error {
		members, err := c.client.SDiff(ctx, formattedKeys...).Result()
		if err != nil {
			return c.handleRedisError("SDIFF", fmt.Sprintf("%v", keys), err)
		}
		result = members
		return nil
	})

	return result, err
}
