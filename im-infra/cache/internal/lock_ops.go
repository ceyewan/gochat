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

// lockOperations 实现锁操作的结构体
type lockOperations struct {
	client    *redis.Client
	logger    clog.Logger
	keyPrefix string
}

// newLockOperations 创建锁操作实例
func newLockOperations(client *redis.Client, logger clog.Logger, keyPrefix string) *lockOperations {
	return &lockOperations{
		client:    client,
		logger:    logger,
		keyPrefix: keyPrefix,
	}
}

// formatKey 格式化键名，添加前缀
func (l *lockOperations) formatKey(key string) string {
	if l.keyPrefix == "" {
		return "lock:" + key
	}
	// 如果前缀已经以冒号结尾，直接拼接
	if len(l.keyPrefix) > 0 && l.keyPrefix[len(l.keyPrefix)-1] == ':' {
		return l.keyPrefix + "lock:" + key
	}
	return l.keyPrefix + ":lock:" + key
}

// Lock 定义了一个分布式锁实例的接口。
type Lock interface {
	// Unlock 释放锁。
	Unlock(ctx context.Context) error
	// Refresh 尝试为锁续期。
	Refresh(ctx context.Context, expiration time.Duration) error
	// Key 返回锁的键名。
	Key() string
	// IsLocked 检查当前实例是否仍然持有锁。
	IsLocked(ctx context.Context) (bool, error)
	// Value 返回锁的唯一值，可用于验证所有权。
	Value() string
}

// Acquire 获取分布式锁
func (l *lockOperations) Acquire(ctx context.Context, key string, expiration time.Duration) (Locker, error) {
	formattedKey := l.formatKey(key)
	value := generateUniqueValue()

	// 使用 SETNX 获取锁
	set, err := l.client.SetNX(ctx, formattedKey, value, expiration).Result()
	if err != nil {
		l.logger.Error("Failed to acquire lock", clog.String("key", formattedKey), clog.Err(err))
		return nil, err
	}
	if !set {
		return nil, fmt.Errorf("lock already acquired") // 锁已被占用
	}

	return &distributedLock{
		key:    formattedKey,
		value:  value,
		client: l.client,
		logger: l.logger,
	}, nil
}

// distributedLock 分布式锁的实现
type distributedLock struct {
	key    string
	value  string
	client *redis.Client
	logger clog.Logger
}

// Unlock 释放锁
func (d *distributedLock) Unlock(ctx context.Context) error {
	// 使用 Lua 脚本确保只删除自己的锁
	luaScript := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `
	result, err := d.client.Eval(ctx, luaScript, []string{d.key}, d.value).Result()
	if err != nil {
		d.logger.Error("Failed to unlock", clog.String("key", d.key), clog.Err(err))
		return err
	}
	if result.(int64) == 0 {
		d.logger.Warn("Lock not owned by this instance", clog.String("key", d.key))
	}
	return nil
}

// Refresh 续期锁的过期时间
func (d *distributedLock) Refresh(ctx context.Context, expiration time.Duration) error {
	// 使用 Lua 脚本检查值并续期
	luaScript := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("pexpire", KEYS[1], ARGV[2])
        else
            return 0
        end
    `
	result, err := d.client.Eval(ctx, luaScript, []string{d.key}, d.value, expiration.Milliseconds()).Result()
	if err != nil {
		d.logger.Error("Failed to refresh lock", clog.String("key", d.key), clog.Err(err))
		return err
	}
	if result.(int64) == 0 {
		d.logger.Warn("Lock not owned by this instance, cannot refresh", clog.String("key", d.key))
		return nil // 或者返回错误
	}
	return nil
}

// Key 返回锁的键名
func (d *distributedLock) Key() string {
	return d.key
}

// IsLocked 检查锁是否仍然有效
func (d *distributedLock) IsLocked(ctx context.Context) (bool, error) {
	result, err := d.client.Get(ctx, d.key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		d.logger.Error("Failed to check lock status", clog.String("key", d.key), clog.Err(err))
		return false, err
	}
	return result == d.value, nil
}

// Value 返回锁的值（用于验证锁的所有权）
func (d *distributedLock) Value() string {
	return d.value
}

// generateUniqueValue 生成唯一值
func generateUniqueValue() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
