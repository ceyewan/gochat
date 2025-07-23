package internal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// lock 是 Lock 接口的内部实现
type lock struct {
	client     *redis.Client
	key        string
	value      string
	expiration time.Duration
	logger     clog.Logger
}

// Lock 获取分布式锁
func (c *cache) Lock(ctx context.Context, key string, expiration time.Duration) (Lock, error) {
	if err := c.validateContext(ctx); err != nil {
		return nil, err
	}
	if err := c.validateKey(key); err != nil {
		return nil, err
	}
	if expiration <= 0 {
		expiration = c.lockConfig.DefaultExpiration
	}

	formattedKey := c.formatKey("lock:" + key)
	lockValue := generateLockValue()

	lockLogger := clog.Module("cache.lock")

	// 尝试获取锁，使用 SET NX EX 命令
	var acquired bool
	err := c.executeWithLogging(ctx, "LOCK", key, func() error {
		result, err := c.client.SetNX(ctx, formattedKey, lockValue, expiration).Result()
		if err != nil {
			return c.handleRedisError("LOCK", key, err)
		}
		acquired = result
		return nil
	})

	if err != nil {
		return nil, err
	}

	if !acquired {
		lockLogger.Debug("获取锁失败，锁已被占用",
			clog.String("key", key),
			clog.Duration("expiration", expiration),
		)
		return nil, fmt.Errorf("failed to acquire lock for key %s: lock already held", key)
	}

	lockLogger.Info("成功获取锁",
		clog.String("key", key),
		clog.String("value", lockValue),
		clog.Duration("expiration", expiration),
	)

	return &lock{
		client:     c.client,
		key:        formattedKey,
		value:      lockValue,
		expiration: expiration,
		logger:     lockLogger,
	}, nil
}

// Unlock 释放锁
func (l *lock) Unlock(ctx context.Context) error {
	if err := validateContext(ctx); err != nil {
		return err
	}

	// 使用 Lua 脚本确保只有锁的持有者才能释放锁
	luaScript := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	start := time.Now()
	result, err := l.client.Eval(ctx, luaScript, []string{l.key}, l.value).Result()
	duration := time.Since(start)

	if err != nil {
		l.logger.Error("释放锁失败",
			clog.String("key", l.key),
			clog.String("value", l.value),
			clog.Duration("duration", duration),
			clog.Err(err),
		)
		return fmt.Errorf("failed to unlock key %s: %w", l.key, err)
	}

	deleted := result.(int64)
	if deleted == 0 {
		l.logger.Warn("释放锁失败，锁不存在或已被其他进程持有",
			clog.String("key", l.key),
			clog.String("value", l.value),
			clog.Duration("duration", duration),
		)
		return fmt.Errorf("failed to unlock key %s: lock not held or expired", l.key)
	}

	l.logger.Info("成功释放锁",
		clog.String("key", l.key),
		clog.String("value", l.value),
		clog.Duration("duration", duration),
	)

	return nil
}

// Refresh 续期锁的过期时间
func (l *lock) Refresh(ctx context.Context, expiration time.Duration) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if expiration <= 0 {
		return fmt.Errorf("expiration must be positive")
	}

	// 使用 Lua 脚本确保只有锁的持有者才能续期
	luaScript := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("EXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	start := time.Now()
	result, err := l.client.Eval(ctx, luaScript, []string{l.key}, l.value, int(expiration.Seconds())).Result()
	duration := time.Since(start)

	if err != nil {
		l.logger.Error("续期锁失败",
			clog.String("key", l.key),
			clog.String("value", l.value),
			clog.Duration("newExpiration", expiration),
			clog.Duration("duration", duration),
			clog.Err(err),
		)
		return fmt.Errorf("failed to refresh lock for key %s: %w", l.key, err)
	}

	refreshed := result.(int64)
	if refreshed == 0 {
		l.logger.Warn("续期锁失败，锁不存在或已被其他进程持有",
			clog.String("key", l.key),
			clog.String("value", l.value),
			clog.Duration("newExpiration", expiration),
			clog.Duration("duration", duration),
		)
		return fmt.Errorf("failed to refresh lock for key %s: lock not held or expired", l.key)
	}

	l.expiration = expiration
	l.logger.Debug("成功续期锁",
		clog.String("key", l.key),
		clog.String("value", l.value),
		clog.Duration("newExpiration", expiration),
		clog.Duration("duration", duration),
	)

	return nil
}

// Key 返回锁的键名
func (l *lock) Key() string {
	return l.key
}

// IsLocked 检查锁是否仍然有效
func (l *lock) IsLocked(ctx context.Context) (bool, error) {
	if err := validateContext(ctx); err != nil {
		return false, err
	}

	start := time.Now()
	value, err := l.client.Get(ctx, l.key).Result()
	duration := time.Since(start)

	if err != nil {
		if err == redis.Nil {
			l.logger.Debug("锁已过期或不存在",
				clog.String("key", l.key),
				clog.Duration("duration", duration),
			)
			return false, nil
		}

		l.logger.Error("检查锁状态失败",
			clog.String("key", l.key),
			clog.Duration("duration", duration),
			clog.Err(err),
		)
		return false, fmt.Errorf("failed to check lock status for key %s: %w", l.key, err)
	}

	isLocked := value == l.value
	l.logger.Debug("检查锁状态完成",
		clog.String("key", l.key),
		clog.Bool("isLocked", isLocked),
		clog.Duration("duration", duration),
	)

	return isLocked, nil
}

// Value 返回锁的值
func (l *lock) Value() string {
	return l.value
}

// generateLockValue 生成唯一的锁值
func generateLockValue() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// 如果随机数生成失败，使用时间戳作为备选方案
		return fmt.Sprintf("lock_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// validateContext 验证上下文（独立函数，避免循环依赖）
func validateContext(ctx context.Context) error {
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

// TryLock 尝试获取锁，如果失败则重试
func (c *cache) TryLock(ctx context.Context, key string, expiration time.Duration, maxRetries int) (Lock, error) {
	if maxRetries <= 0 {
		maxRetries = c.lockConfig.MaxRetries
	}

	var lastErr error
	lockLogger := clog.Module("cache.lock")

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			// 等待一段时间再重试
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.lockConfig.RetryDelay):
			}

			lockLogger.Debug("重试获取锁",
				clog.String("key", key),
				clog.Int("attempt", i+1),
				clog.Int("maxRetries", maxRetries),
			)
		}

		lock, err := c.Lock(ctx, key, expiration)
		if err == nil {
			if i > 0 {
				lockLogger.Info("重试获取锁成功",
					clog.String("key", key),
					clog.Int("attempts", i+1),
				)
			}
			return lock, nil
		}

		lastErr = err

		// 如果是上下文取消或超时，不再重试
		if err == context.Canceled || err == context.DeadlineExceeded {
			break
		}
	}

	lockLogger.Error("获取锁重试失败",
		clog.String("key", key),
		clog.Int("maxRetries", maxRetries),
		clog.Err(lastErr),
	)

	return nil, lastErr
}
