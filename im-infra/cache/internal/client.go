package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// cache 是 Cache 接口的内部实现。
// 它包装了一个 *redis.Client，并提供接口方法。
type cache struct {
	client      *redis.Client
	config      Config
	lockConfig  LockConfig
	bloomConfig BloomConfig
	logger      clog.Logger
}

// Ping 检查 Redis 连接是否正常
func (c *cache) Ping(ctx context.Context) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("Ping 操作完成",
			clog.Duration("duration", duration),
		)
	}()

	err := c.client.Ping(ctx).Err()
	if err != nil {
		c.logger.Error("Redis Ping 失败", clog.Err(err))
		return fmt.Errorf("redis ping failed: %w", err)
	}

	c.logger.Debug("Redis Ping 成功")
	return nil
}

// Close 关闭 Redis 连接
func (c *cache) Close() error {
	c.logger.Info("关闭 Redis 连接")
	err := c.client.Close()
	if err != nil {
		c.logger.Error("关闭 Redis 连接失败", clog.Err(err))
		return fmt.Errorf("failed to close redis client: %w", err)
	}
	c.logger.Info("Redis 连接已关闭")
	return nil
}

// formatKey 格式化键名，添加前缀
func (c *cache) formatKey(key string) string {
	if c.config.KeyPrefix == "" {
		return key
	}
	return c.config.KeyPrefix + ":" + key
}

// serialize 序列化值
func (c *cache) serialize(value interface{}) (string, error) {
	if value == nil {
		return "", nil
	}

	// 如果已经是字符串，直接返回
	if str, ok := value.(string); ok {
		return str, nil
	}

	switch c.config.Serializer {
	case "json":
		data, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("json serialization failed: %w", err)
		}
		return string(data), nil
	default:
		// 默认使用 JSON
		data, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("json serialization failed: %w", err)
		}
		return string(data), nil
	}
}

// deserialize 反序列化值
func (c *cache) deserialize(data string, target interface{}) error {
	if data == "" {
		return nil
	}

	switch c.config.Serializer {
	case "json":
		return json.Unmarshal([]byte(data), target)
	default:
		// 默认使用 JSON
		return json.Unmarshal([]byte(data), target)
	}
}

// logOperation 记录操作日志
func (c *cache) logOperation(operation string, key string, duration time.Duration, err error) {
	fields := []clog.Field{
		clog.String("operation", operation),
		clog.String("key", key),
		clog.Duration("duration", duration),
	}

	if err != nil {
		fields = append(fields, clog.Err(err))
		c.logger.Error("缓存操作失败", fields...)
	} else {
		c.logger.Debug("缓存操作成功", fields...)
	}
}

// logSlowOperation 记录慢操作
func (c *cache) logSlowOperation(operation string, key string, duration time.Duration) {
	slowThreshold := 100 * time.Millisecond // 慢操作阈值
	if duration > slowThreshold {
		c.logger.Warn("检测到慢缓存操作",
			clog.String("operation", operation),
			clog.String("key", key),
			clog.Duration("duration", duration),
			clog.Duration("threshold", slowThreshold),
		)
	}
}

// executeWithLogging 执行操作并记录日志
func (c *cache) executeWithLogging(ctx context.Context, operation string, key string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	c.logOperation(operation, key, duration, err)
	c.logSlowOperation(operation, key, duration)

	return err
}

// handleRedisError 处理 Redis 错误
func (c *cache) handleRedisError(operation string, key string, err error) error {
	if err == nil {
		return nil
	}

	// 特殊处理一些常见错误
	switch err {
	case redis.Nil:
		return fmt.Errorf("key not found: %s", key)
	default:
		c.logger.Error("Redis 操作错误",
			clog.String("operation", operation),
			clog.String("key", key),
			clog.Err(err),
		)
		return fmt.Errorf("redis %s operation failed for key %s: %w", operation, key, err)
	}
}

// isKeyNotFoundError 检查是否是键不存在错误
func (c *cache) isKeyNotFoundError(err error) bool {
	return err == redis.Nil
}

// retryOperation 重试操作
func (c *cache) retryOperation(ctx context.Context, operation string, maxRetries int, fn func() error) error {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			// 等待一段时间再重试
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(i) * 100 * time.Millisecond):
			}

			c.logger.Debug("重试操作",
				clog.String("operation", operation),
				clog.Int("attempt", i+1),
				clog.Int("maxRetries", maxRetries),
			)
		}

		lastErr = fn()
		if lastErr == nil {
			if i > 0 {
				c.logger.Info("操作重试成功",
					clog.String("operation", operation),
					clog.Int("attempts", i+1),
				)
			}
			return nil
		}

		// 如果是上下文取消或超时，不再重试
		if lastErr == context.Canceled || lastErr == context.DeadlineExceeded {
			break
		}
	}

	c.logger.Error("操作重试失败",
		clog.String("operation", operation),
		clog.Int("maxRetries", maxRetries),
		clog.Err(lastErr),
	)

	return lastErr
}

// validateContext 验证上下文
func (c *cache) validateContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// validateKey 验证键名
func (c *cache) validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	return nil
}

// validateKeys 验证多个键名
func (c *cache) validateKeys(keys []string) error {
	if len(keys) == 0 {
		return fmt.Errorf("keys cannot be empty")
	}

	for i, key := range keys {
		if key == "" {
			return fmt.Errorf("key at index %d cannot be empty", i)
		}
	}

	return nil
}

// ===== Scripting Operations =====

// ScriptLoad loads a Lua script into the Redis script cache.
func (c *cache) ScriptLoad(ctx context.Context, script string) (string, error) {
	return c.client.ScriptLoad(ctx, script).Result()
}

// EvalSha executes a pre-loaded Lua script.
func (c *cache) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	return c.client.EvalSha(ctx, sha1, keys, args...).Result()
}

// ===== Connection Management =====

// Client returns the underlying go-redis client.
// Note: This is for advanced use cases. Prefer using the abstracted methods.
func (c *cache) Client() *redis.Client {
	return c.client
}
