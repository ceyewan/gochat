package cache

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache/internal"
)

// ScriptingOperations 定义了与 Redis Lua 脚本相关的操作。
type ScriptingOperations interface {
	// EvalSha 执行已加载的 Lua 脚本。
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error)
	// ScriptLoad 将 Lua 脚本加载到 Redis 中并返回其 SHA1 哈希值。
	ScriptLoad(ctx context.Context, script string) (string, error)
}

// Cache 定义了缓存服务的核心接口，整合了所有数据结构的操作。
// 它的设计遵循面向接口的原则，便于测试和扩展。
type Cache interface {
	StringOperations
	HashOperations
	SetOperations
	LockOperations
	BloomFilterOperations
	ScriptingOperations

	// Ping 检查与 Redis 服务器的连接是否正常。
	Ping(ctx context.Context) error
	// Close 关闭与 Redis 服务器的连接。
	Close() error
}

// StringOperations 定义了所有与 Redis 字符串相关的操作。
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

// HashOperations 定义了所有与 Redis 哈希相关的操作。
type HashOperations interface {
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key, field string, value interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error
	HExists(ctx context.Context, key, field string) (bool, error)
	HLen(ctx context.Context, key string) (int64, error)
}

// SetOperations 定义了所有与 Redis 集合相关的操作。
type SetOperations interface {
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SRem(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SCard(ctx context.Context, key string) (int64, error)
}

// LockOperations 定义了分布式锁的操作。
type LockOperations interface {
	// Lock 尝试获取一个分布式锁。
	Lock(ctx context.Context, key string, expiration time.Duration) (Lock, error)
}

type Lock = internal.Lock

// BloomFilterOperations 定义了布隆过滤器的操作。
// 注意：需要 RedisBloom 模块的支持。
type BloomFilterOperations interface {
	// BFAdd 向布隆过滤器中添加一个元素。
	BFAdd(ctx context.Context, key string, item string) error
	// BFExists 检查一个元素是否存在于布隆过滤器中。
	BFExists(ctx context.Context, key string, item string) (bool, error)
	// BFInit 初始化一个布隆过滤器，如果它不存在。
	// errorRate 是期望的错误率，capacity 是期望的容量。
	BFInit(ctx context.Context, key string, errorRate float64, capacity int64) error
}
