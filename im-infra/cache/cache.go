package cache

import (
	"context"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache/internal"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// Cache 定义缓存操作的核心接口。
// 提供 Redis 数据结构的抽象操作方法。
type Cache = internal.Cache

// Lock 定义分布式锁的接口。
// 提供锁的获取、释放、续期等操作。
type Lock = internal.Lock

// Config 是 cache 的主配置结构体。
// 用于声明式地定义缓存行为和 Redis 连接参数。
type Config = internal.Config

// LockConfig 分布式锁的配置
type LockConfig = internal.LockConfig

// BloomConfig 布隆过滤器的配置
type BloomConfig = internal.BloomConfig

var (
	// 全局默认缓存实例
	defaultCache Cache
	// 确保默认缓存只初始化一次
	defaultCacheOnce sync.Once
	// 模块日志器
	logger = clog.Module("cache")
)

// getDefaultCache 获取全局默认缓存实例
func getDefaultCache() Cache {
	defaultCacheOnce.Do(func() {
		defaultCache = internal.NewDefaultCache()
		if defaultCache == nil {
			logger.Error("创建默认缓存实例失败")
		} else {
			logger.Info("默认缓存实例创建成功")
		}
	})
	return defaultCache
}

// New 根据提供的配置创建一个新的 Cache 实例。
// 这是核心工厂函数，按配置组装所有组件。
//
// 示例：
//
//	cfg := cache.Config{
//	  Addr: "localhost:6379",
//	  DB: 0,
//	  PoolSize: 10,
//	}
//	cache, err := cache.New(cfg)
//	if err != nil {
//	  log.Fatal(err)
//	}
func New(cfg Config) (Cache, error) {
	return internal.NewCache(cfg)
}

// Default 返回一个带有合理默认配置的 Cache。
// 默认缓存连接到 localhost:6379，使用数据库 0。
//
// 等价于：
//
//	cfg := cache.Config{
//	  Addr: "localhost:6379",
//	  DB: 0,
//	  PoolSize: 10,
//	}
//	cache, _ := cache.New(cfg)
//
// 示例：
//
//	cache := cache.Default()
//	err := cache.Set(ctx, "key", "value", time.Hour)
func Default() Cache {
	return getDefaultCache()
}

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置适用于大多数开发和测试场景。
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// ===== 全局缓存方法 =====

// Get 使用全局默认缓存获取字符串值
func Get(ctx context.Context, key string) (string, error) {
	return getDefaultCache().Get(ctx, key)
}

// Set 使用全局默认缓存设置字符串值
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return getDefaultCache().Set(ctx, key, value, expiration)
}

// SetNX 使用全局默认缓存仅在键不存在时设置值
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return getDefaultCache().SetNX(ctx, key, value, expiration)
}

// Incr 使用全局默认缓存递增整数值
func Incr(ctx context.Context, key string) (int64, error) {
	return getDefaultCache().Incr(ctx, key)
}

// Decr 使用全局默认缓存递减整数值
func Decr(ctx context.Context, key string) (int64, error) {
	return getDefaultCache().Decr(ctx, key)
}

// Expire 使用全局默认缓存设置键的过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return getDefaultCache().Expire(ctx, key, expiration)
}

// TTL 使用全局默认缓存获取键的剩余生存时间
func TTL(ctx context.Context, key string) (time.Duration, error) {
	return getDefaultCache().TTL(ctx, key)
}

// Del 使用全局默认缓存删除一个或多个键
func Del(ctx context.Context, keys ...string) error {
	return getDefaultCache().Del(ctx, keys...)
}

// Exists 使用全局默认缓存检查一个或多个键是否存在
func Exists(ctx context.Context, keys ...string) (int64, error) {
	return getDefaultCache().Exists(ctx, keys...)
}

// HGet 使用全局默认缓存获取哈希字段的值
func HGet(ctx context.Context, key, field string) (string, error) {
	return getDefaultCache().HGet(ctx, key, field)
}

// HSet 使用全局默认缓存设置哈希字段的值
func HSet(ctx context.Context, key, field string, value interface{}) error {
	return getDefaultCache().HSet(ctx, key, field, value)
}

// HGetAll 使用全局默认缓存获取哈希的所有字段和值
func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return getDefaultCache().HGetAll(ctx, key)
}

// HDel 使用全局默认缓存删除哈希的一个或多个字段
func HDel(ctx context.Context, key string, fields ...string) error {
	return getDefaultCache().HDel(ctx, key, fields...)
}

// HExists 使用全局默认缓存检查哈希字段是否存在
func HExists(ctx context.Context, key, field string) (bool, error) {
	return getDefaultCache().HExists(ctx, key, field)
}

// HLen 使用全局默认缓存获取哈希字段的数量
func HLen(ctx context.Context, key string) (int64, error) {
	return getDefaultCache().HLen(ctx, key)
}

// SAdd 使用全局默认缓存向集合添加一个或多个成员
func SAdd(ctx context.Context, key string, members ...interface{}) error {
	return getDefaultCache().SAdd(ctx, key, members...)
}

// SRem 使用全局默认缓存从集合移除一个或多个成员
func SRem(ctx context.Context, key string, members ...interface{}) error {
	return getDefaultCache().SRem(ctx, key, members...)
}

// SMembers 使用全局默认缓存获取集合的所有成员
func SMembers(ctx context.Context, key string) ([]string, error) {
	return getDefaultCache().SMembers(ctx, key)
}

// SIsMember 使用全局默认缓存检查成员是否在集合中
func SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return getDefaultCache().SIsMember(ctx, key, member)
}

// SCard 使用全局默认缓存获取集合的成员数量
func SCard(ctx context.Context, key string) (int64, error) {
	return getDefaultCache().SCard(ctx, key)
}

// AcquireLock 使用全局默认缓存获取分布式锁
func AcquireLock(ctx context.Context, key string, expiration time.Duration) (Lock, error) {
	return getDefaultCache().Lock(ctx, key, expiration)
}

// BloomAdd 使用全局默认缓存向布隆过滤器添加元素
func BloomAdd(ctx context.Context, key string, item string) error {
	return getDefaultCache().BloomAdd(ctx, key, item)
}

// BloomExists 使用全局默认缓存检查元素是否可能存在于布隆过滤器中
func BloomExists(ctx context.Context, key string, item string) (bool, error) {
	return getDefaultCache().BloomExists(ctx, key, item)
}

// BloomInit 使用全局默认缓存初始化布隆过滤器
func BloomInit(ctx context.Context, key string, capacity uint64, errorRate float64) error {
	return getDefaultCache().BloomInit(ctx, key, capacity, errorRate)
}

// Ping 使用全局默认缓存检查 Redis 连接是否正常
func Ping(ctx context.Context) error {
	return getDefaultCache().Ping(ctx)
}

// ===== 高级工厂方法 =====

// NewWithOptions 使用自定义 Redis 选项创建缓存实例
func NewWithOptions(opts *redis.Options) Cache {
	return internal.NewCacheWithOptions(opts)
}

// NewWithClient 使用现有的 Redis 客户端创建缓存实例
func NewWithClient(client *redis.Client) Cache {
	return internal.NewCacheWithClient(client)
}
