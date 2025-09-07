# 基础设施: Cache 分布式缓存

## 1. 概述

`cache` 是 `gochat` 项目的统一分布式缓存服务，基于 `go-redis` 构建。它提供了一个高性能、功能完备、类型安全且易于使用的 Redis 操作层。

其核心设计原则是**封装与易用**：将 `go-redis` 的复杂性隐藏在一个简洁的 API 之后，同时提供丰富的功能，如分布式锁和布隆过滤器。

## 2. 核心用法

### 2.1 初始化

所有缓存操作都通过 `cache.Cache` 接口进行。

```go
import "github.com/ceyewan/gochat/im-infra/cache"

// 1. 获取默认配置，并指定 Redis 地址
cfg := cache.DefaultConfig()
cfg.Addr = "localhost:6379"
cfg.KeyPrefix = "gochat:" // 推荐为所有键设置统一前缀

// 2. 创建 Cache 实例，并注入 logger
logger := clog.Module("cache-example")
cacheClient, err := cache.New(ctx, cfg, cache.WithLogger(logger))
if err != nil {
    log.Fatalf("无法创建缓存客户端: %v", err)
}
defer cacheClient.Close()
```

### 2.2 字符串操作 (String)

最常用的键值对操作。

```go
// 设置一个键，过期时间为 5 分钟
err := cacheClient.Set(ctx, "user:1001:profile", profileJSON, 5*time.Minute)

// 获取一个键
value, err := cacheClient.Get(ctx, "user:1001:profile")
if err == redis.Nil {
    // 缓存未命中
}

// 原子增/减
onlineCount, err := cacheClient.Incr(ctx, "stats:online_users")
```

### 2.3 哈希操作 (Hash)

适用于存储对象。

```go
// 设置哈希中的一个字段
err := cacheClient.HSet(ctx, "user:1001", "name", "Alice")

// 获取整个哈希对象
profile, err := cacheClient.HGetAll(ctx, "user:1001")
// profile 是一个 map[string]string
```

### 2.4 分布式锁

用于确保在分布式环境中只有一个实例可以执行某个任务。

```go
// 尝试获取一个租期为 30 秒的锁
lock, err := cacheClient.Lock(ctx, "lock:process_daily_report", 30*time.Second)
if err != nil {
    // 获取锁失败，可能已被其他实例持有
    logger.Info("获取报表处理锁失败，退出任务")
    return
}
// 确保任务完成后释放锁
defer lock.Unlock(ctx)

// --- 执行需要同步的任务 ---
```

### 2.5 布隆过滤器

用于以极高的空间效率判断一个元素“一定不存在”或“可能存在”，非常适合用于防止缓存穿透。

```go
// 检查一个可能不存在的用户ID
exists, err := cacheClient.BFExists(ctx, "bloom:user_ids", "user_id_abcdef")
if !exists {
    // 元素一定不存在，可以直接拒绝请求，无需查询数据库
}
```

## 3. API 参考

`cache` 模块通过组合多个小接口来定义其完整功能。

```go
// Cache 是所有缓存操作的主入口接口。
type Cache interface {
	StringOperations
	HashOperations
	SetOperations
	LockOperations
	BloomFilterOperations
	ScriptingOperations

	Ping(ctx context.Context) error
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
	Lock(ctx context.Context, key string, expiration time.Duration) (Lock, error)
}

// Lock 定义了锁对象的接口。
type Lock interface {
    Unlock(ctx context.Context) error
    Refresh(ctx context.Context, expiration time.Duration) error
}

// BloomFilterOperations 定义了布隆过滤器的操作 (需要 RedisBloom 模块)。
type BloomFilterOperations interface {
	BFAdd(ctx context.Context, key string, item string) error
	BFExists(ctx context.Context, key string, item string) (bool, error)
	BFInit(ctx context.Context, key string, errorRate float64, capacity int64) error
}

// ScriptingOperations 定义了与 Redis Lua 脚本相关的操作。
type ScriptingOperations interface {
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error)
	ScriptLoad(ctx context.Context, script string) (string, error)
}
```

## 4. 配置

`cache` 组件通过 `Config` 结构体进行配置，通常由 `coord` 配置中心管理。

```go
type Config struct {
	Addr            string        `json:"addr"`
	Password        string        `json:"password"`
	DB              int           `json:"db"`
	PoolSize        int           `json:"poolSize"`
	DialTimeout     time.Duration `json:"dialTimeout"`
	ReadTimeout     time.Duration `json:"readTimeout"`
	WriteTimeout    time.Duration `json:"writeTimeout"`
	KeyPrefix       string        `json:"keyPrefix"` // 推荐设置，用于命名空间隔离
	// ... 更多连接池和重试选项
}