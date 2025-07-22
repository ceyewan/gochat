# cache API 文档

## 概述

`cache` 是一个基于 Redis 的高性能缓存库，提供了简洁易用的 API 和丰富的功能。

## 核心特性

- 🌟 **全局缓存方法**：支持 `cache.Get()` 等全局方法，无需显式创建缓存实例
- 📦 **自定义缓存实例**：`cache.New(config)` 创建自定义配置的缓存实例
- 🚀 **基于 go-redis/v9**：充分利用最新的 Redis Go 客户端，性能与兼容性俱佳
- 📝 **多数据结构支持**：支持字符串、哈希、集合等 Redis 数据结构
- 🔒 **分布式锁**：Redis 基础的分布式锁，支持过期时间和续期
- 🌸 **布隆过滤器**：Redis 基础的布隆过滤器，支持概率性成员测试
- 🏷️ **日志集成**：与 clog 日志库深度集成，提供详细的操作日志

## 全局缓存方法

```go
// 基础全局缓存方法
func Get(ctx context.Context, key string) (string, error)
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
func Incr(ctx context.Context, key string) (int64, error)
func Decr(ctx context.Context, key string) (int64, error)
func Expire(ctx context.Context, key string, expiration time.Duration) error
func TTL(ctx context.Context, key string) (time.Duration, error)
func Del(ctx context.Context, keys ...string) error
func Exists(ctx context.Context, keys ...string) (int64, error)

// 哈希操作方法
func HGet(ctx context.Context, key, field string) (string, error)
func HSet(ctx context.Context, key, field string, value interface{}) error
func HGetAll(ctx context.Context, key string) (map[string]string, error)
func HDel(ctx context.Context, key string, fields ...string) error
func HExists(ctx context.Context, key, field string) (bool, error)
func HLen(ctx context.Context, key string) (int64, error)

// 集合操作方法
func SAdd(ctx context.Context, key string, members ...interface{}) error
func SRem(ctx context.Context, key string, members ...interface{}) error
func SMembers(ctx context.Context, key string) ([]string, error)
func SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
func SCard(ctx context.Context, key string) (int64, error)

// 分布式锁方法
func Lock(ctx context.Context, key string, expiration time.Duration) (Lock, error)

// 布隆过滤器方法
func BloomAdd(ctx context.Context, key string, item string) error
func BloomExists(ctx context.Context, key string, item string) (bool, error)
func BloomInit(ctx context.Context, key string, capacity uint64, errorRate float64) error

// 连接管理方法
func Ping(ctx context.Context) error
```

**使用示例：**
```go
// 字符串操作
cache.Set(ctx, "user:123", "John Doe", time.Hour)
user, _ := cache.Get(ctx, "user:123")

// 哈希操作
cache.HSet(ctx, "user:123:profile", "name", "John")
cache.HSet(ctx, "user:123:profile", "email", "john@example.com")
profile, _ := cache.HGetAll(ctx, "user:123:profile")

// 集合操作
cache.SAdd(ctx, "user:123:tags", "developer", "golang")
tags, _ := cache.SMembers(ctx, "user:123:tags")
```



## 工厂方法

### New 函数

```go
func New(cfg Config) (Cache, error)
```

根据配置创建新的缓存实例。

### Default 函数

```go
func Default() Cache
```

返回默认缓存实例，与全局缓存方法使用相同的缓存实例。

### 高级工厂方法

```go
func NewWithOptions(opts *redis.Options) Cache
func NewWithClient(client *redis.Client) Cache
```

使用自定义 Redis 选项或现有客户端创建缓存实例。

## 配置管理

### 配置结构

```go
type Config struct {
    Addr            string        // Redis 地址
    Password        string        // Redis 密码
    DB              int           // 数据库编号
    PoolSize        int           // 连接池大小
    MinIdleConns    int           // 最小空闲连接数
    MaxIdleConns    int           // 最大空闲连接数
    ConnMaxIdleTime time.Duration // 连接最大空闲时间
    ConnMaxLifetime time.Duration // 连接最大生存时间
    DialTimeout     time.Duration // 连接超时
    ReadTimeout     time.Duration // 读取超时
    WriteTimeout    time.Duration // 写入超时
    PoolTimeout     time.Duration // 连接池超时
    MaxRetries      int           // 最大重试次数
    MinRetryBackoff time.Duration // 最小重试间隔
    MaxRetryBackoff time.Duration // 最大重试间隔
    EnableTracing   bool          // 启用链路追踪
    EnableMetrics   bool          // 启用指标收集
    KeyPrefix       string        // 键名前缀
    Serializer      string        // 序列化器
    Compression     bool          // 启用压缩
}
```

### 预设配置

```go
func DefaultConfig() Config
func DevelopmentConfig() Config
func ProductionConfig() Config
func TestConfig() Config
func HighPerformanceConfig() Config
```

### 配置构建器

```go
cfg := cache.NewConfigBuilder().
    Addr("localhost:6379").
    Password("secret").
    DB(0).
    PoolSize(20).
    KeyPrefix("myapp").
    EnableTracing().
    Build()
```

## 接口定义

### Cache 接口

```go
type Cache interface {
    // 字符串操作
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
    Incr(ctx context.Context, key string) (int64, error)
    Decr(ctx context.Context, key string) (int64, error)
    Expire(ctx context.Context, key string, expiration time.Duration) error
    TTL(ctx context.Context, key string) (time.Duration, error)
    Del(ctx context.Context, keys ...string) error
    Exists(ctx context.Context, keys ...string) (int64, error)

    // 哈希操作
    HGet(ctx context.Context, key, field string) (string, error)
    HSet(ctx context.Context, key, field string, value interface{}) error
    HGetAll(ctx context.Context, key string) (map[string]string, error)
    HDel(ctx context.Context, key string, fields ...string) error
    HExists(ctx context.Context, key, field string) (bool, error)
    HLen(ctx context.Context, key string) (int64, error)

    // 集合操作
    SAdd(ctx context.Context, key string, members ...interface{}) error
    SRem(ctx context.Context, key string, members ...interface{}) error
    SMembers(ctx context.Context, key string) ([]string, error)
    SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
    SCard(ctx context.Context, key string) (int64, error)

    // 分布式锁
    Lock(ctx context.Context, key string, expiration time.Duration) (Lock, error)

    // 布隆过滤器
    BloomAdd(ctx context.Context, key string, item string) error
    BloomExists(ctx context.Context, key string, item string) (bool, error)
    BloomInit(ctx context.Context, key string, capacity uint64, errorRate float64) error

    // 连接管理
    Ping(ctx context.Context) error
    Close() error
}
```

### Lock 接口

```go
type Lock interface {
    Unlock(ctx context.Context) error
    Refresh(ctx context.Context, expiration time.Duration) error
    Key() string
    IsLocked(ctx context.Context) (bool, error)
    Value() string
}
```

## 错误处理

cache 提供了详细的错误信息和类型：

```go
// 检查键不存在错误
value, err := cache.Get(ctx, "nonexistent")
if cache.IsKeyNotFoundError(err) {
    // 处理键不存在的情况
}

// 检查连接错误
err := cache.Set(ctx, "key", "value", time.Hour)
if cache.IsConnectionError(err) {
    // 处理连接错误
}
```

## 最佳实践

### 1. 上下文使用

```go
// ✅ 使用带超时的上下文
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

value, err := cache.Get(ctx, "key")
```

### 2. 错误处理

```go
// ✅ 完整的错误处理
value, err := cache.Get(ctx, "key")
if err != nil {
    if cache.IsKeyNotFoundError(err) {
        return defaultValue, nil
    }
    return "", fmt.Errorf("cache get failed: %w", err)
}
```

### 3. 资源清理

```go
// ✅ 正确释放锁
lock, err := cache.Lock(ctx, "resource", time.Minute)
if err != nil {
    return err
}
defer lock.Unlock(ctx)

// 执行临界区代码
```

### 4. 性能优化

```go
// ✅ 缓存自定义缓存实例
var userCache Cache
func init() {
    cfg := cache.NewConfigBuilder().KeyPrefix("user").Build()
    userCache, _ = cache.New(cfg)
}

// ✅ 使用批量操作
cache.Del(ctx, "key1", "key2", "key3")
count, _ := cache.Exists(ctx, "key1", "key2", "key3")
```

## 监控和日志

cache 与 clog 深度集成，提供详细的操作日志：

- 操作成功/失败日志
- 慢操作警告
- 连接状态变化
- 锁获取/释放事件
- 性能指标统计

## 迁移指南

### 从其他 Redis 库迁移

```go
// 其他库
client := redis.NewClient(&redis.Options{...})
client.Set(ctx, "key", "value", time.Hour)

// cache 库
cache := cache.New(config)
cache.Set(ctx, "key", "value", time.Hour)

// 或使用全局方法
cache.Set(ctx, "key", "value", time.Hour)
```
