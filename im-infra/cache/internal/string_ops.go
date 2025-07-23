package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// Get 获取字符串值
func (c *cache) Get(ctx context.Context, key string) (string, error) {
	if err := c.validateContext(ctx); err != nil {
		return "", err
	}
	if err := c.validateKey(key); err != nil {
		return "", err
	}

	formattedKey := c.formatKey(key)

	var result string
	err := c.executeWithLogging(ctx, "GET", key, func() error {
		val, err := c.client.Get(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("GET", key, err)
		}
		result = val
		return nil
	})

	return result, err
}

// Set 设置字符串值
func (c *cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}

	formattedKey := c.formatKey(key)
	serializedValue, err := c.serialize(value)
	if err != nil {
		c.logger.Error("序列化值失败",
			clog.String("key", key),
			clog.Err(err),
		)
		return fmt.Errorf("failed to serialize value for key %s: %w", key, err)
	}

	return c.executeWithLogging(ctx, "SET", key, func() error {
		err := c.client.Set(ctx, formattedKey, serializedValue, expiration).Err()
		return c.handleRedisError("SET", key, err)
	})
}

// Incr 递增整数值
func (c *cache) Incr(ctx context.Context, key string) (int64, error) {
	if err := c.validateContext(ctx); err != nil {
		return 0, err
	}
	if err := c.validateKey(key); err != nil {
		return 0, err
	}

	formattedKey := c.formatKey(key)

	var result int64
	err := c.executeWithLogging(ctx, "INCR", key, func() error {
		val, err := c.client.Incr(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("INCR", key, err)
		}
		result = val
		return nil
	})

	return result, err
}

// Decr 递减整数值
func (c *cache) Decr(ctx context.Context, key string) (int64, error) {
	if err := c.validateContext(ctx); err != nil {
		return 0, err
	}
	if err := c.validateKey(key); err != nil {
		return 0, err
	}

	formattedKey := c.formatKey(key)

	var result int64
	err := c.executeWithLogging(ctx, "DECR", key, func() error {
		val, err := c.client.Decr(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("DECR", key, err)
		}
		result = val
		return nil
	})

	return result, err
}

// Expire 设置键的过期时间
func (c *cache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}
	if expiration <= 0 {
		return fmt.Errorf("expiration must be positive")
	}

	formattedKey := c.formatKey(key)

	return c.executeWithLogging(ctx, "EXPIRE", key, func() error {
		success, err := c.client.Expire(ctx, formattedKey, expiration).Result()
		if err != nil {
			return c.handleRedisError("EXPIRE", key, err)
		}
		if !success {
			return fmt.Errorf("failed to set expiration for key %s: key does not exist", key)
		}
		return nil
	})
}

// TTL 获取键的剩余生存时间
func (c *cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	if err := c.validateContext(ctx); err != nil {
		return 0, err
	}
	if err := c.validateKey(key); err != nil {
		return 0, err
	}

	formattedKey := c.formatKey(key)

	var result time.Duration
	err := c.executeWithLogging(ctx, "TTL", key, func() error {
		ttl, err := c.client.TTL(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("TTL", key, err)
		}
		result = ttl
		return nil
	})

	return result, err
}

// Del 删除一个或多个键
func (c *cache) Del(ctx context.Context, keys ...string) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKeys(keys); err != nil {
		return err
	}

	// 格式化所有键名
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = c.formatKey(key)
	}

	return c.executeWithLogging(ctx, "DEL", fmt.Sprintf("%v", keys), func() error {
		deletedCount, err := c.client.Del(ctx, formattedKeys...).Result()
		if err != nil {
			return c.handleRedisError("DEL", fmt.Sprintf("%v", keys), err)
		}

		c.logger.Debug("删除键完成",
			clog.Strings("keys", keys),
			clog.Int64("deletedCount", deletedCount),
		)

		return nil
	})
}

// Exists 检查一个或多个键是否存在
func (c *cache) Exists(ctx context.Context, keys ...string) (int64, error) {
	if err := c.validateContext(ctx); err != nil {
		return 0, err
	}
	if err := c.validateKeys(keys); err != nil {
		return 0, err
	}

	// 格式化所有键名
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = c.formatKey(key)
	}

	var result int64
	err := c.executeWithLogging(ctx, "EXISTS", fmt.Sprintf("%v", keys), func() error {
		count, err := c.client.Exists(ctx, formattedKeys...).Result()
		if err != nil {
			return c.handleRedisError("EXISTS", fmt.Sprintf("%v", keys), err)
		}
		result = count
		return nil
	})

	return result, err
}

// SetNX 仅在键不存在时设置值
func (c *cache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	if err := c.validateContext(ctx); err != nil {
		return false, err
	}
	if err := c.validateKey(key); err != nil {
		return false, err
	}

	formattedKey := c.formatKey(key)
	serializedValue, err := c.serialize(value)
	if err != nil {
		c.logger.Error("序列化值失败",
			clog.String("key", key),
			clog.Err(err),
		)
		return false, fmt.Errorf("failed to serialize value for key %s: %w", key, err)
	}

	var result bool
	err = c.executeWithLogging(ctx, "SETNX", key, func() error {
		success, err := c.client.SetNX(ctx, formattedKey, serializedValue, expiration).Result()
		if err != nil {
			return c.handleRedisError("SETNX", key, err)
		}
		result = success
		return nil
	})

	return result, err
}

// GetSet 设置新值并返回旧值
func (c *cache) GetSet(ctx context.Context, key string, value interface{}) (string, error) {
	if err := c.validateContext(ctx); err != nil {
		return "", err
	}
	if err := c.validateKey(key); err != nil {
		return "", err
	}

	formattedKey := c.formatKey(key)
	serializedValue, err := c.serialize(value)
	if err != nil {
		c.logger.Error("序列化值失败",
			clog.String("key", key),
			clog.Err(err),
		)
		return "", fmt.Errorf("failed to serialize value for key %s: %w", key, err)
	}

	var result string
	err = c.executeWithLogging(ctx, "GETSET", key, func() error {
		oldValue, err := c.client.GetSet(ctx, formattedKey, serializedValue).Result()
		if err != nil && err != redis.Nil {
			return c.handleRedisError("GETSET", key, err)
		}
		result = oldValue
		return nil
	})

	return result, err
}
