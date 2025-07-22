package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// client 是 Idempotent 接口的内部实现。
// 它包装了一个 cache.Cache，并提供幂等操作方法。
type client struct {
	cache  cache.Cache
	config Config
	logger clog.Logger
}

// NewIdempotentClient 创建新的幂等客户端
func NewIdempotentClient(cfg Config) (Idempotent, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建缓存实例
	cacheInstance, err := cache.New(cfg.CacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache instance: %w", err)
	}

	return &client{
		cache:  cacheInstance,
		config: cfg,
		logger: clog.Module("idempotent"),
	}, nil
}

// Check 检查指定键是否已经存在（是否已执行过）
func (c *client) Check(ctx context.Context, key string) (bool, error) {
	if err := c.validateKey(key); err != nil {
		return false, err
	}

	formattedKey := c.formatKey(key)

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("Check 操作完成",
			clog.String("key", key),
			clog.Duration("duration", duration),
		)
	}()

	count, err := c.cache.Exists(ctx, formattedKey)
	if err != nil {
		c.logger.Error("检查键存在性失败",
			clog.String("key", key),
			clog.ErrorValue(err),
		)
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	exists := count > 0
	c.logger.Debug("键存在性检查完成",
		clog.String("key", key),
		clog.Bool("exists", exists),
	)

	return exists, nil
}

// Set 设置幂等标记，如果键已存在则返回 false
func (c *client) Set(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if err := c.validateKey(key); err != nil {
		return false, err
	}

	formattedKey := c.formatKey(key)
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("Set 操作完成",
			clog.String("key", key),
			clog.Duration("duration", duration),
		)
	}()

	// 使用 SetNX 实现原子性的设置操作
	success, err := c.cache.SetNX(ctx, formattedKey, "1", ttl)
	if err != nil {
		c.logger.Error("设置幂等标记失败",
			clog.String("key", key),
			clog.ErrorValue(err),
		)
		return false, fmt.Errorf("failed to set idempotent key: %w", err)
	}

	c.logger.Debug("幂等标记设置完成",
		clog.String("key", key),
		clog.Bool("success", success),
		clog.Duration("ttl", ttl),
	)

	return success, nil
}

// CheckAndSet 原子性地检查并设置幂等标记
func (c *client) CheckAndSet(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	// CheckAndSet 实际上就是 Set 操作，因为 SetNX 本身就是原子性的
	return c.Set(ctx, key, ttl)
}

// SetWithResult 设置幂等标记并存储操作结果
func (c *client) SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error) {
	if err := c.validateKey(key); err != nil {
		return false, err
	}

	formattedKey := c.formatKey(key)
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("SetWithResult 操作完成",
			clog.String("key", key),
			clog.Duration("duration", duration),
		)
	}()

	// 序列化结果
	serializedResult, err := c.serialize(result)
	if err != nil {
		c.logger.Error("序列化结果失败",
			clog.String("key", key),
			clog.ErrorValue(err),
		)
		return false, fmt.Errorf("failed to serialize result: %w", err)
	}

	// 使用 SetNX 设置结果
	success, err := c.cache.SetNX(ctx, formattedKey, serializedResult, ttl)
	if err != nil {
		c.logger.Error("设置幂等标记和结果失败",
			clog.String("key", key),
			clog.ErrorValue(err),
		)
		return false, fmt.Errorf("failed to set idempotent key with result: %w", err)
	}

	c.logger.Debug("幂等标记和结果设置完成",
		clog.String("key", key),
		clog.Bool("success", success),
		clog.Duration("ttl", ttl),
	)

	return success, nil
}

// GetResult 获取存储的操作结果
func (c *client) GetResult(ctx context.Context, key string) (interface{}, error) {
	if err := c.validateKey(key); err != nil {
		return nil, err
	}

	formattedKey := c.formatKey(key)

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("GetResult 操作完成",
			clog.String("key", key),
			clog.Duration("duration", duration),
		)
	}()

	value, err := c.cache.Get(ctx, formattedKey)
	if err != nil {
		c.logger.Error("获取结果失败",
			clog.String("key", key),
			clog.ErrorValue(err),
		)
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	// 如果值是简单的标记（"1"），返回 nil
	if value == "1" {
		return nil, nil
	}

	// 反序列化结果
	var result interface{}
	if err := c.deserialize([]byte(value), &result); err != nil {
		c.logger.Error("反序列化结果失败",
			clog.String("key", key),
			clog.ErrorValue(err),
		)
		return nil, fmt.Errorf("failed to deserialize result: %w", err)
	}

	c.logger.Debug("获取结果完成",
		clog.String("key", key),
	)

	return result, nil
}

// Delete 删除幂等标记
func (c *client) Delete(ctx context.Context, key string) error {
	if err := c.validateKey(key); err != nil {
		return err
	}

	formattedKey := c.formatKey(key)

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("Delete 操作完成",
			clog.String("key", key),
			clog.Duration("duration", duration),
		)
	}()

	err := c.cache.Del(ctx, formattedKey)
	if err != nil {
		c.logger.Error("删除幂等标记失败",
			clog.String("key", key),
			clog.ErrorValue(err),
		)
		return fmt.Errorf("failed to delete idempotent key: %w", err)
	}

	c.logger.Debug("幂等标记删除完成",
		clog.String("key", key),
	)

	return nil
}

// Exists 检查键是否存在（别名方法，与 Check 功能相同）
func (c *client) Exists(ctx context.Context, key string) (bool, error) {
	return c.Check(ctx, key)
}

// TTL 获取键的剩余过期时间
func (c *client) TTL(ctx context.Context, key string) (time.Duration, error) {
	if err := c.validateKey(key); err != nil {
		return 0, err
	}

	formattedKey := c.formatKey(key)

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("TTL 操作完成",
			clog.String("key", key),
			clog.Duration("duration", duration),
		)
	}()

	ttl, err := c.cache.TTL(ctx, formattedKey)
	if err != nil {
		c.logger.Error("获取 TTL 失败",
			clog.String("key", key),
			clog.ErrorValue(err),
		)
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	c.logger.Debug("TTL 获取完成",
		clog.String("key", key),
		clog.Duration("ttl", ttl),
	)

	return ttl, nil
}

// Refresh 刷新键的过期时间
func (c *client) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	if err := c.validateKey(key); err != nil {
		return err
	}

	formattedKey := c.formatKey(key)
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("Refresh 操作完成",
			clog.String("key", key),
			clog.Duration("duration", duration),
		)
	}()

	err := c.cache.Expire(ctx, formattedKey, ttl)
	if err != nil {
		c.logger.Error("刷新过期时间失败",
			clog.String("key", key),
			clog.ErrorValue(err),
		)
		return fmt.Errorf("failed to refresh TTL: %w", err)
	}

	c.logger.Debug("过期时间刷新完成",
		clog.String("key", key),
		clog.Duration("ttl", ttl),
	)

	return nil
}

// Close 关闭幂等客户端，释放资源
func (c *client) Close() error {
	c.logger.Info("关闭幂等客户端")
	// cache 组件可能没有 Close 方法，这里暂时不做处理
	return nil
}

// ===== 辅助方法 =====

// formatKey 格式化键名，添加前缀
func (c *client) formatKey(key string) string {
	if c.config.KeyPrefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", c.config.KeyPrefix, key)
}

// validateKey 验证键名是否有效
func (c *client) validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if len(key) > c.config.MaxKeyLength {
		return fmt.Errorf("key length exceeds maximum: %d > %d", len(key), c.config.MaxKeyLength)
	}

	// 根据配置的验证器进行验证
	switch c.config.KeyValidator {
	case "strict":
		return c.validateKeyStrict(key)
	case "loose":
		return c.validateKeyLoose(key)
	default:
		return c.validateKeyDefault(key)
	}
}

// validateKeyDefault 默认键名验证
func (c *client) validateKeyDefault(key string) error {
	// 不允许包含空格和特殊字符
	if strings.ContainsAny(key, " \t\n\r") {
		return fmt.Errorf("key cannot contain whitespace characters")
	}
	return nil
}

// validateKeyStrict 严格键名验证
func (c *client) validateKeyStrict(key string) error {
	// 只允许字母、数字、下划线、连字符和冒号
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_:-]+$`, key)
	if !matched {
		return fmt.Errorf("key contains invalid characters (strict mode)")
	}
	return nil
}

// validateKeyLoose 宽松键名验证
func (c *client) validateKeyLoose(key string) error {
	// 只检查是否包含换行符
	if strings.ContainsAny(key, "\n\r") {
		return fmt.Errorf("key cannot contain newline characters")
	}
	return nil
}

// serialize 序列化值
func (c *client) serialize(value interface{}) (string, error) {
	switch c.config.Serializer {
	case "json":
		data, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		// 默认使用 JSON
		data, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

// deserialize 反序列化值
func (c *client) deserialize(data []byte, value interface{}) error {
	switch c.config.Serializer {
	case "json":
		return json.Unmarshal(data, value)
	default:
		// 默认使用 JSON
		return json.Unmarshal(data, value)
	}
}
