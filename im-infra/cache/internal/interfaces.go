package internal

import (
	"context"
	"time"
)

// StringOperations 定义了所有与 Redis 字符串相关的操作。
type StringOperations interface {
	// Get 获取一个 key。如果 key 不存在，将返回 redis.Nil 错误。
	Get(ctx context.Context, key string) (string, error)
	// Set 存入一个 key-value 对。
	// 注意：value (interface{}) 参数需要调用者自行序列化为字符串或字节数组。
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	Exists(ctx context.Context, keys ...string) (int64, error)
	// SetNX (Set if Not Exists) 存入一个 key-value 对，仅当 key 不存在时。
	// 注意：value (interface{}) 参数需要调用者自行序列化。
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	// GetSet 设置新值并返回旧值。如果 key 不存在，返回 redis.Nil。
	// 注意：value (interface{}) 参数需要调用者自行序列化。
	GetSet(ctx context.Context, key string, value interface{}) (string, error)
}

// HashOperations 定义了所有与 Redis 哈希相关的操作。
type HashOperations interface {
	// HGet 获取哈希表 key 中一个 field 的值。如果 key 或 field 不存在，返回 redis.Nil。
	HGet(ctx context.Context, key, field string) (string, error)
	// HSet 设置哈希表 key 中一个 field 的值。
	// 注意：value (interface{}) 参数需要调用者自行序列化。
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

// ZSetOperations 定义了所有与 Redis 有序集合相关的操作。
type ZSetOperations interface {
	// ZAdd 添加一个或多个成员到有序集合
	ZAdd(ctx context.Context, key string, members ...*ZMember) error
	// ZRange 获取有序集合中指定范围内的成员，按分数从低到高排序
	ZRange(ctx context.Context, key string, start, stop int64) ([]*ZMember, error)
	// ZRevRange 获取有序集合中指定范围内的成员，按分数从高到低排序
	ZRevRange(ctx context.Context, key string, start, stop int64) ([]*ZMember, error)
	// ZRangeByScore 获取指定分数范围内的成员
	ZRangeByScore(ctx context.Context, key string, min, max float64) ([]*ZMember, error)
	// ZRem 从有序集合中移除一个或多个成员
	ZRem(ctx context.Context, key string, members ...interface{}) error
	// ZRemRangeByRank 移除有序集合中指定排名区间内的成员
	ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error
	// ZCard 获取有序集合的成员数量
	ZCard(ctx context.Context, key string) (int64, error)
	// ZCount 获取指定分数范围内的成员数量
	ZCount(ctx context.Context, key string, min, max float64) (int64, error)
	// ZScore 获取成员的分数
	ZScore(ctx context.Context, key string, member string) (float64, error)
	// ZSetExpire 为有序集合设置过期时间
	ZSetExpire(ctx context.Context, key string, expiration time.Duration) error
}

// ZMember 表示有序集合中的成员
type ZMember struct {
	Member interface{} // 成员值
	Score  float64     // 分数
}

// LockOperations 定义了分布式锁的操作。
type LockOperations interface {
	// Acquire 尝试获取一个锁。如果成功，返回一个 Locker 对象；否则返回错误。
	Acquire(ctx context.Context, key string, expiration time.Duration) (Locker, error)
}

// Locker 定义了锁对象的接口。
type Locker interface {
	// Unlock 释放锁
	Unlock(ctx context.Context) error
	// Refresh 刷新锁的过期时间
	Refresh(ctx context.Context, expiration time.Duration) error
}

// BloomFilterOperations 定义了布隆过滤器的操作 (需要 RedisBloom 模块)。
type BloomFilterOperations interface {
	BFAdd(ctx context.Context, key string, item string) error
	BFExists(ctx context.Context, key string, item string) (bool, error)
	BFReserve(ctx context.Context, key string, errorRate float64, capacity uint64) error
}

// ScriptingOperations 定义了与 Redis Lua 脚本相关的操作。
type ScriptingOperations interface {
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error)
	ScriptLoad(ctx context.Context, script string) (string, error)
	ScriptExists(ctx context.Context, sha1 ...string) ([]bool, error)
}

// Provider 定义了 cache 组件提供的所有能力。
type Provider interface {
	String() StringOperations
	Hash() HashOperations
	Set() SetOperations
	ZSet() ZSetOperations
	Lock() LockOperations
	Bloom() BloomFilterOperations
	Script() ScriptingOperations

	// Ping 检查与 Redis 服务器的连接。
	Ping(ctx context.Context) error
	// Close 关闭所有与 Redis 的连接。
	Close() error
}