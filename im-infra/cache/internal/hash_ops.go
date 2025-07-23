package internal

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// HGet 获取哈希字段的值
func (c *cache) HGet(ctx context.Context, key, field string) (string, error) {
	if err := c.validateContext(ctx); err != nil {
		return "", err
	}
	if err := c.validateKey(key); err != nil {
		return "", err
	}
	if field == "" {
		return "", fmt.Errorf("field cannot be empty")
	}

	formattedKey := c.formatKey(key)

	var result string
	err := c.executeWithLogging(ctx, "HGET", fmt.Sprintf("%s.%s", key, field), func() error {
		val, err := c.client.HGet(ctx, formattedKey, field).Result()
		if err != nil {
			return c.handleRedisError("HGET", key, err)
		}
		result = val
		return nil
	})

	return result, err
}

// HSet 设置哈希字段的值
func (c *cache) HSet(ctx context.Context, key, field string, value interface{}) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}
	if field == "" {
		return fmt.Errorf("field cannot be empty")
	}

	formattedKey := c.formatKey(key)
	serializedValue, err := c.serialize(value)
	if err != nil {
		c.logger.Error("序列化值失败",
			clog.String("key", key),
			clog.String("field", field),
			clog.Err(err),
		)
		return fmt.Errorf("failed to serialize value for key %s field %s: %w", key, field, err)
	}

	return c.executeWithLogging(ctx, "HSET", fmt.Sprintf("%s.%s", key, field), func() error {
		err := c.client.HSet(ctx, formattedKey, field, serializedValue).Err()
		return c.handleRedisError("HSET", key, err)
	})
}

// HGetAll 获取哈希的所有字段和值
func (c *cache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKey(key); err != nil {
		return nil, err
	}

	formattedKey := c.formatKey(key)

	var result map[string]string
	err := c.executeWithLogging(ctx, "HGETALL", key, func() error {
		val, err := c.client.HGetAll(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("HGETALL", key, err)
		}
		result = val
		return nil
	})

	return result, err
}

// HDel 删除哈希的一个或多个字段
func (c *cache) HDel(ctx context.Context, key string, fields ...string) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}
	if len(fields) == 0 {
		return fmt.Errorf("fields cannot be empty")
	}

	for i, field := range fields {
		if field == "" {
			return fmt.Errorf("field at index %d cannot be empty", i)
		}
	}

	formattedKey := c.formatKey(key)

	return c.executeWithLogging(ctx, "HDEL", fmt.Sprintf("%s.%v", key, fields), func() error {
		deletedCount, err := c.client.HDel(ctx, formattedKey, fields...).Result()
		if err != nil {
			return c.handleRedisError("HDEL", key, err)
		}

		c.logger.Debug("删除哈希字段完成",
			clog.String("key", key),
			clog.Strings("fields", fields),
			clog.Int64("deletedCount", deletedCount),
		)

		return nil
	})
}

// HExists 检查哈希字段是否存在
func (c *cache) HExists(ctx context.Context, key, field string) (bool, error) {
	if err := c.validateContext(ctx); err != nil {
		return false, err
	}
	if err := c.validateKey(key); err != nil {
		return false, err
	}
	if field == "" {
		return false, fmt.Errorf("field cannot be empty")
	}

	formattedKey := c.formatKey(key)

	var result bool
	err := c.executeWithLogging(ctx, "HEXISTS", fmt.Sprintf("%s.%s", key, field), func() error {
		exists, err := c.client.HExists(ctx, formattedKey, field).Result()
		if err != nil {
			return c.handleRedisError("HEXISTS", key, err)
		}
		result = exists
		return nil
	})

	return result, err
}

// HLen 获取哈希字段的数量
func (c *cache) HLen(ctx context.Context, key string) (int64, error) {
	if err := c.validateContext(ctx); err != nil {
		return 0, err
	}
	if err := c.validateKey(key); err != nil {
		return 0, err
	}

	formattedKey := c.formatKey(key)

	var result int64
	err := c.executeWithLogging(ctx, "HLEN", key, func() error {
		length, err := c.client.HLen(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("HLEN", key, err)
		}
		result = length
		return nil
	})

	return result, err
}

// HKeys 获取哈希的所有字段名
func (c *cache) HKeys(ctx context.Context, key string) ([]string, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKey(key); err != nil {
		return nil, err
	}

	formattedKey := c.formatKey(key)

	var result []string
	err := c.executeWithLogging(ctx, "HKEYS", key, func() error {
		keys, err := c.client.HKeys(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("HKEYS", key, err)
		}
		result = keys
		return nil
	})

	return result, err
}

// HVals 获取哈希的所有值
func (c *cache) HVals(ctx context.Context, key string) ([]string, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKey(key); err != nil {
		return nil, err
	}

	formattedKey := c.formatKey(key)

	var result []string
	err := c.executeWithLogging(ctx, "HVALS", key, func() error {
		vals, err := c.client.HVals(ctx, formattedKey).Result()
		if err != nil {
			return c.handleRedisError("HVALS", key, err)
		}
		result = vals
		return nil
	})

	return result, err
}

// HMGet 获取多个哈希字段的值
func (c *cache) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKey(key); err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields cannot be empty")
	}

	for i, field := range fields {
		if field == "" {
			return nil, fmt.Errorf("field at index %d cannot be empty", i)
		}
	}

	formattedKey := c.formatKey(key)

	var result []interface{}
	err := c.executeWithLogging(ctx, "HMGET", fmt.Sprintf("%s.%v", key, fields), func() error {
		vals, err := c.client.HMGet(ctx, formattedKey, fields...).Result()
		if err != nil {
			return c.handleRedisError("HMGET", key, err)
		}
		result = vals
		return nil
	})

	return result, err
}

// HMSet 设置多个哈希字段的值
func (c *cache) HMSet(ctx context.Context, key string, values map[string]interface{}) error {
	if err := c.validateContext(ctx); err != nil {
		return err
	}
	if err := c.validateKey(key); err != nil {
		return err
	}
	if len(values) == 0 {
		return fmt.Errorf("values cannot be empty")
	}

	formattedKey := c.formatKey(key)

	// 序列化所有值
	serializedValues := make(map[string]interface{})
	for field, value := range values {
		if field == "" {
			return fmt.Errorf("field cannot be empty")
		}

		serializedValue, err := c.serialize(value)
		if err != nil {
			c.logger.Error("序列化值失败",
				clog.String("key", key),
				clog.String("field", field),
				clog.Err(err),
			)
			return fmt.Errorf("failed to serialize value for key %s field %s: %w", key, field, err)
		}
		serializedValues[field] = serializedValue
	}

	return c.executeWithLogging(ctx, "HMSET", key, func() error {
		err := c.client.HMSet(ctx, formattedKey, serializedValues).Err()
		return c.handleRedisError("HMSET", key, err)
	})
}

// HIncrBy 将哈希字段的整数值增加指定数量
func (c *cache) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	if err := c.validateContext(ctx); err != nil {
		return 0, err
	}
	if err := c.validateKey(key); err != nil {
		return 0, err
	}
	if field == "" {
		return 0, fmt.Errorf("field cannot be empty")
	}

	formattedKey := c.formatKey(key)

	var result int64
	err := c.executeWithLogging(ctx, "HINCRBY", fmt.Sprintf("%s.%s", key, field), func() error {
		val, err := c.client.HIncrBy(ctx, formattedKey, field, incr).Result()
		if err != nil {
			return c.handleRedisError("HINCRBY", key, err)
		}
		result = val
		return nil
	})

	return result, err
}
