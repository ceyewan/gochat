# cache

一个现代化、高性能的 Go Redis 缓存库，基于 Redis Go 客户端 v9 构建。cache 提供简洁、可组合的接口，支持字符串、哈希、集合操作、分布式锁、布隆过滤器等高级特性。

## 功能特色

- 🚀 **基于 go-redis/v9**：充分利用最新的 Redis Go 客户端，性能与兼容性俱佳
- 🎯 **接口驱动**：抽象清晰，封装合理
- 🌟 **全局缓存方法**：支持 `cache.Get()` 等全局缓存方法，无需显式创建缓存实例
- 📦 **自定义缓存实例**：`cache.New(config)` 创建自定义配置的缓存实例
- 📝 **多数据结构支持**：支持字符串、哈希、集合等 Redis 数据结构
- 🔒 **分布式锁**：Redis 基础的分布式锁，支持过期时间和续期
- 🌸 **布隆过滤器**：Redis 基础的布隆过滤器，支持概率性成员测试
- 🔄 **连接池管理**：内置连接池和错误恢复机制
- 🏷️ **日志集成**：与 clog 日志库深度集成，提供详细的操作日志
- ⚡ **高性能**：优化的序列化和网络操作
- 🎨 **配置灵活**：丰富的配置选项和预设配置
- 🔧 **零额外依赖**：仅依赖 go-redis 和 clog

## 安装

```bash
go get github.com/ceyewan/gochat/im-infra/cache
```

## 快速开始

### 基本用法

#### 全局缓存方法（推荐）

```go
package main

import (
    "context"
    "time"
    "github.com/ceyewan/gochat/im-infra/cache"
)

func main() {
    ctx := context.Background()
    
    // 字符串操作
    err := cache.Set(ctx, "user:123", "John Doe", time.Hour)
    if err != nil {
        panic(err)
    }
    
    value, err := cache.Get(ctx, "user:123")
    if err != nil {
        panic(err)
    }
    fmt.Println("User:", value)
    
    // 哈希操作
    err = cache.HSet(ctx, "user:123:profile", "name", "John Doe")
    err = cache.HSet(ctx, "user:123:profile", "email", "john@example.com")
    
    profile, err := cache.HGetAll(ctx, "user:123:profile")
    fmt.Println("Profile:", profile)
    
    // 集合操作
    err = cache.SAdd(ctx, "user:123:tags", "developer", "golang", "redis")
    tags, err := cache.SMembers(ctx, "user:123:tags")
    fmt.Println("Tags:", tags)
}
```

#### 自定义缓存实例（推荐用于大型应用）

```go
package main

import (
    "context"
    "time"
    "github.com/ceyewan/gochat/im-infra/cache"
)

func main() {
    ctx := context.Background()

    // 创建自定义配置的缓存实例
    userCfg := cache.NewConfigBuilder().
        KeyPrefix("user").
        PoolSize(10).
        Build()
    userCache, _ := cache.New(userCfg)

    sessionCfg := cache.NewConfigBuilder().
        KeyPrefix("session").
        PoolSize(5).
        Build()
    sessionCache, _ := cache.New(sessionCfg)

    // 用户缓存操作
    err := userCache.Set(ctx, "123", userData, time.Hour)
    user, err := userCache.Get(ctx, "123")

    // 会话缓存操作
    err = sessionCache.Set(ctx, "abc", sessionData, time.Minute*30)
    session, err := sessionCache.Get(ctx, "abc")
}
```

### 自定义配置

```go
package main

import (
    "github.com/ceyewan/gochat/im-infra/cache"
)

func main() {
    // 使用预设配置
    cfg := cache.ProductionConfig()
    
    // 或者使用配置构建器
    cfg = cache.NewConfigBuilder().
        Addr("redis:6379").
        Password("secret").
        DB(0).
        PoolSize(20).
        KeyPrefix("myapp").
        EnableTracing().
        EnableMetrics().
        Build()
    
    cacheInstance, err := cache.New(cfg)
    if err != nil {
        panic(err)
    }
    
    // 使用自定义缓存实例
    err = cacheInstance.Set(ctx, "key", "value", time.Hour)
}
```

## 核心功能

### 字符串操作

```go
ctx := context.Background()

// 基本操作
cache.Set(ctx, "key", "value", time.Hour)
value, _ := cache.Get(ctx, "key")

// 数值操作
cache.Set(ctx, "counter", 0, time.Hour)
newValue, _ := cache.Incr(ctx, "counter")  // 1
newValue, _ := cache.Decr(ctx, "counter")  // 0

// 过期时间
cache.Expire(ctx, "key", time.Minute*30)
ttl, _ := cache.TTL(ctx, "key")

// 删除和检查
cache.Del(ctx, "key1", "key2")
count, _ := cache.Exists(ctx, "key1", "key2")
```

### 哈希操作

```go
ctx := context.Background()

// 设置和获取字段
cache.HSet(ctx, "user:123", "name", "John")
cache.HSet(ctx, "user:123", "email", "john@example.com")
name, _ := cache.HGet(ctx, "user:123", "name")

// 获取所有字段
fields, _ := cache.HGetAll(ctx, "user:123")

// 删除字段
cache.HDel(ctx, "user:123", "email")

// 检查字段存在
exists, _ := cache.HExists(ctx, "user:123", "name")

// 获取字段数量
count, _ := cache.HLen(ctx, "user:123")
```

### 集合操作

```go
ctx := context.Background()

// 添加成员
cache.SAdd(ctx, "tags", "golang", "redis", "cache")

// 检查成员
isMember, _ := cache.SIsMember(ctx, "tags", "golang")

// 获取所有成员
members, _ := cache.SMembers(ctx, "tags")

// 移除成员
cache.SRem(ctx, "tags", "cache")

// 获取成员数量
count, _ := cache.SCard(ctx, "tags")
```

### 分布式锁

```go
ctx := context.Background()

// 获取锁
lock, err := cache.Lock(ctx, "resource:123", time.Minute*5)
if err != nil {
    // 锁已被占用或其他错误
    return
}

// 执行临界区代码
defer lock.Unlock(ctx)

// 续期锁
err = lock.Refresh(ctx, time.Minute*10)

// 检查锁状态
isLocked, _ := lock.IsLocked(ctx)
```

### 布隆过滤器

```go
ctx := context.Background()

// 初始化布隆过滤器
err := cache.BloomInit(ctx, "users", 1000000, 0.01)

// 添加元素
cache.BloomAdd(ctx, "users", "user123")
cache.BloomAdd(ctx, "users", "user456")

// 检查元素是否存在
exists, _ := cache.BloomExists(ctx, "users", "user123")  // true
exists, _ = cache.BloomExists(ctx, "users", "user999")   // false (可能)
```

## 配置选项

### 预设配置

```go
// 开发环境
cfg := cache.DevelopmentConfig()

// 生产环境
cfg := cache.ProductionConfig()

// 测试环境
cfg := cache.TestConfig()

// 高性能场景
cfg := cache.HighPerformanceConfig()
```

### 配置构建器

```go
cfg := cache.NewConfigBuilder().
    Addr("localhost:6379").
    Password("secret").
    DB(0).
    PoolSize(20).
    IdleConns(5, 15).
    Timeouts(5*time.Second, 3*time.Second, 3*time.Second, 4*time.Second).
    Retries(3, 8*time.Millisecond, 512*time.Millisecond).
    KeyPrefix("myapp").
    Serializer("json").
    EnableTracing().
    EnableMetrics().
    EnableCompression().
    Build()
```

## 最佳实践

### 1. 选择合适的缓存方法

```go
// ✅ 简单场景：使用全局方法
cache.Set(ctx, "key", "value", time.Hour)

// ✅ 复杂配置：使用自定义缓存实例
cacheInstance, _ := cache.New(customConfig)
cacheInstance.Set(ctx, "key", "value", time.Hour)

// ✅ 模块化场景：创建专用缓存实例
userCfg := cache.NewConfigBuilder().KeyPrefix("user").Build()
userCache, _ := cache.New(userCfg)
userCache.Set(ctx, "123", userData, time.Hour)
```

### 2. 性能优化

```go
// ✅ 缓存自定义缓存实例
var (
    userCache    Cache
    sessionCache Cache
)

func init() {
    userCfg := cache.NewConfigBuilder().KeyPrefix("user").Build()
    userCache, _ = cache.New(userCfg)

    sessionCfg := cache.NewConfigBuilder().KeyPrefix("session").Build()
    sessionCache, _ = cache.New(sessionCfg)
}

func handleRequest() {
    userCache.Get(ctx, "123")    // 使用预创建的实例
    sessionCache.Get(ctx, "abc") // 使用预创建的实例
}

// ❌ 避免重复创建实例
func handleRequest() {
    userCfg := cache.NewConfigBuilder().KeyPrefix("user").Build()
    userCache, _ := cache.New(userCfg) // 有额外开销
    userCache.Get(ctx, "123")
}
```

### 3. 错误处理

```go
value, err := cache.Get(ctx, "key")
if err != nil {
    if cache.IsKeyNotFoundError(err) {
        // 键不存在，执行相应逻辑
        return defaultValue, nil
    }
    // 其他错误，记录日志并返回
    return "", fmt.Errorf("cache get failed: %w", err)
}
```

### 4. 上下文使用

```go
// ✅ 使用带超时的上下文
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

value, err := cache.Get(ctx, "key")
```

## 日志集成

cache 与 clog 日志库深度集成，提供详细的操作日志：

```go
// 缓存操作会自动记录日志
cache.Set(ctx, "key", "value", time.Hour)
// 日志输出: level=DEBUG msg="缓存操作成功" operation=SET key=key duration=2ms

// 慢操作会记录警告日志
// 日志输出: level=WARN msg="检测到慢缓存操作" operation=GET key=key duration=150ms threshold=100ms

// 错误会记录错误日志
// 日志输出: level=ERROR msg="缓存操作失败" operation=GET key=key duration=5ms error="connection refused"
```

## 监控和指标

启用指标收集后，cache 会收集以下指标：

- 操作延迟
- 操作成功/失败率
- 连接池状态
- 慢操作统计

## 常见问题

### Q: 全局方法和自定义缓存实例的区别？
A: 全局方法适用于简单场景，自定义缓存实例适用于需要不同配置或命名空间隔离的场景。自定义实例可以有独立的配置和键前缀。

### Q: 如何处理连接失败？
A: cache 内置了重试机制和连接池管理，会自动处理临时连接失败。持续失败会记录错误日志。

### Q: 分布式锁是否支持续期？
A: 是的，可以使用 `lock.Refresh()` 方法续期锁的过期时间。

### Q: 布隆过滤器的误判率如何控制？
A: 通过调整容量和错误率参数来控制。容量越大、错误率越小，所需的内存和哈希函数就越多。

### Q: 如何选择序列化器？
A: 默认使用 JSON 序列化器，适用于大多数场景。未来会支持 msgpack 和 gob 等更高效的序列化器。

## 性能基准

cache 在各种场景下都有优异的性能表现：

```
BenchmarkGet-8          1000000    1200 ns/op    128 B/op    3 allocs/op
BenchmarkSet-8           800000    1500 ns/op    256 B/op    5 allocs/op
BenchmarkHGet-8          900000    1300 ns/op    160 B/op    4 allocs/op
BenchmarkSAdd-8          700000    1800 ns/op    320 B/op    6 allocs/op
BenchmarkLock-8          500000    2500 ns/op    512 B/op    8 allocs/op
```

## 许可证

MIT License
