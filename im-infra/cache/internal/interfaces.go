package internal

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache 定义缓存操作的核心接口。
// 提供 Redis 数据结构的抽象操作方法。
type Cache interface {
	StringOperations
	HashOperations
	SetOperations
	LockOperations
	// BloomOperations  // TODO: 暂时禁用布隆过滤器
	// ScriptOperations // TODO: 暂时禁用脚本操作

	// Connection management - 连接管理
	Ping(ctx context.Context) error
	Close() error
	Client() *redis.Client
}

// Lock 定义分布式锁的接口。
// 提供锁的获取、释放、续期等操作。
type Lock interface {
	// Unlock 释放锁
	Unlock(ctx context.Context) error

	// Refresh 续期锁的过期时间
	Refresh(ctx context.Context, expiration time.Duration) error

	// Key 返回锁的键名
	Key() string

	// IsLocked 检查锁是否仍然有效
	IsLocked(ctx context.Context) (bool, error)

	// Value 返回锁的值（用于验证锁的所有权）
	Value() string
}

// StringOperations 定义字符串操作的接口
type StringOperations interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
}

// HashOperations 定义哈希操作的接口
type HashOperations interface {
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key, field string, value interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error
	HExists(ctx context.Context, key, field string) (bool, error)
	HLen(ctx context.Context, key string) (int64, error)
}

// SetOperations 定义集合操作的接口
type SetOperations interface {
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SRem(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SCard(ctx context.Context, key string) (int64, error)
}

// LockOperations 定义分布式锁操作的接口
type LockOperations interface {
	Lock(ctx context.Context, key string, expiration time.Duration) (Lock, error)
}

// BloomOperations 定义布隆过滤器操作的接口
type BloomOperations interface {
	BloomAdd(ctx context.Context, key string, item string) error
	BloomExists(ctx context.Context, key string, item string) (bool, error)
	BloomInit(ctx context.Context, key string, capacity uint64, errorRate float64) error
}

// ScriptOperations 定义 Lua 脚本操作的接口
type ScriptOperations interface {
	ScriptLoad(ctx context.Context, script string) (string, error)
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error)
}
