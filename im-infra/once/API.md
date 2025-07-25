# idempotent API 使用文档

本文档提供 idempotent 库的完整 API 使用指南，包含所有接口的使用方法和示例代码。

## 概述

`idempotent` 是一个基于 Redis setnx 命令的高性能分布式幂等库，提供了简洁易用的 API 和丰富的功能。

## 核心特性

- 🌟 **全局方法**：支持 `idempotent.Set()` 等全局方法，无需显式创建客户端
- 📦 **自定义客户端**：`idempotent.New(config)` 创建自定义配置的客户端实例
- 🚀 **基于 Redis setnx**：利用 Redis 原子性操作保证幂等性
- 📝 **结果存储**：支持存储操作结果，避免重复计算
- 🔄 **TTL 支持**：支持设置幂等键的过期时间
- 🏷️ **日志集成**：与 clog 日志库深度集成

## 全局方法 API

### 基础幂等操作

#### Check

检查指定键是否已经存在（是否已执行过）。

```go
func Check(ctx context.Context, key string) (bool, error)
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名

**返回值：**
- `bool`：键是否存在
- `error`：错误信息

**示例：**
```go
exists, err := idempotent.Check(ctx, "user:create:123")
if err != nil {
    log.Fatal(err)
}
if exists {
    fmt.Println("操作已执行过")
}
```

#### Set

设置幂等标记，如果键已存在则返回 false。

```go
func Set(ctx context.Context, key string, ttl time.Duration) (bool, error)
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名
- `ttl`：过期时间，0 表示使用默认 TTL

**返回值：**
- `bool`：是否成功设置（首次设置）
- `error`：错误信息

**示例：**
```go
success, err := idempotent.Set(ctx, "user:create:123", time.Hour)
if err != nil {
    log.Fatal(err)
}
if success {
    fmt.Println("首次执行，进行实际操作")
} else {
    fmt.Println("操作已执行过")
}
```

#### CheckAndSet

原子性地检查并设置幂等标记。

```go
func CheckAndSet(ctx context.Context, key string, ttl time.Duration) (bool, error)
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名
- `ttl`：过期时间

**返回值：**
- `bool`：是否首次设置
- `error`：错误信息

**示例：**
```go
firstTime, err := idempotent.CheckAndSet(ctx, "user:create:123", time.Hour)
if err != nil {
    log.Fatal(err)
}
if firstTime {
    fmt.Println("首次执行")
} else {
    fmt.Println("重复执行")
}
```

### 结果存储操作

#### SetWithResult

设置幂等标记并存储操作结果。

```go
func SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error)
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名
- `result`：要存储的结果
- `ttl`：过期时间

**返回值：**
- `bool`：是否成功设置
- `error`：错误信息

**示例：**
```go
result := map[string]interface{}{
    "user_id": 123,
    "status":  "created",
}

success, err := idempotent.SetWithResult(ctx, "user:create:123", result, time.Hour)
if err != nil {
    log.Fatal(err)
}
```

#### GetResult

获取存储的操作结果。

```go
func GetResult(ctx context.Context, key string) (interface{}, error)
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名

**返回值：**
- `interface{}`：存储的结果
- `error`：错误信息

**示例：**
```go
result, err := idempotent.GetResult(ctx, "user:create:123")
if err != nil {
    log.Fatal(err)
}
if result != nil {
    fmt.Printf("缓存的结果: %+v\n", result)
}
```

### TTL 管理操作

#### TTL

获取键的剩余过期时间。

```go
func TTL(ctx context.Context, key string) (time.Duration, error)
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名

**返回值：**
- `time.Duration`：剩余过期时间
- `error`：错误信息

**示例：**
```go
ttl, err := idempotent.TTL(ctx, "user:create:123")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("剩余时间: %v\n", ttl)
```

#### Refresh

刷新键的过期时间。

```go
func Refresh(ctx context.Context, key string, ttl time.Duration) error
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名
- `ttl`：新的过期时间

**返回值：**
- `error`：错误信息

**示例：**
```go
err := idempotent.Refresh(ctx, "user:create:123", 2*time.Hour)
if err != nil {
    log.Fatal(err)
}
```

### 其他操作

#### Delete

删除幂等标记。

```go
func Delete(ctx context.Context, key string) error
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名

**返回值：**
- `error`：错误信息

**示例：**
```go
err := idempotent.Delete(ctx, "user:create:123")
if err != nil {
    log.Fatal(err)
}
```

#### Exists

检查键是否存在（别名方法，与 Check 功能相同）。

```go
func Exists(ctx context.Context, key string) (bool, error)
```

**示例：**
```go
exists, err := idempotent.Exists(ctx, "user:create:123")
if err != nil {
    log.Fatal(err)
}
```

## 便捷方法 API

### Execute

执行幂等操作，如果是首次执行则调用回调函数。

```go
func Execute(ctx context.Context, key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error)
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名
- `ttl`：过期时间
- `callback`：回调函数

**返回值：**
- `interface{}`：操作结果
- `error`：错误信息

**示例：**
```go
result, err := idempotent.Execute(ctx, "user:create:123", time.Hour, func() (interface{}, error) {
    // 执行实际的业务逻辑
    user := createUser(123)
    return user, nil
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("用户创建结果: %+v\n", result)
```

### ExecuteSimple

执行简单的幂等操作，只设置标记不存储结果。

```go
func ExecuteSimple(ctx context.Context, key string, ttl time.Duration, callback func() error) error
```

**参数：**
- `ctx`：上下文
- `key`：幂等键名
- `ttl`：过期时间
- `callback`：回调函数

**返回值：**
- `error`：错误信息

**示例：**
```go
err := idempotent.ExecuteSimple(ctx, "notification:send:123", time.Hour, func() error {
    return sendNotification(123)
})
if err != nil {
    log.Fatal(err)
}
```

## 工厂方法 API

### New

根据配置创建幂等客户端。

```go
func New(cfg Config) (Idempotent, error)
```

**参数：**
- `cfg`：客户端配置

**返回值：**
- `Idempotent`：客户端实例
- `error`：错误信息

**示例：**
```go
config := idempotent.Config{
    KeyPrefix:   "myapp",
    DefaultTTL:  time.Hour,
    CacheConfig: cache.Config{
        Addr: "localhost:6379",
    },
}

client, err := idempotent.New(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### Default

返回全局默认幂等客户端实例。

```go
func Default() Idempotent
```

**返回值：**
- `Idempotent`：默认客户端实例

**示例：**
```go
client := idempotent.Default()
success, err := client.Set(ctx, "operation:123", time.Hour)
```

## 配置 API

### 预设配置

#### DefaultConfig

返回默认配置。

```go
func DefaultConfig() Config
```

#### DevelopmentConfig

返回开发环境配置。

```go
func DevelopmentConfig() Config
```

#### ProductionConfig

返回生产环境配置。

```go
func ProductionConfig() Config
```

#### TestConfig

返回测试环境配置。

```go
func TestConfig() Config
```

### 配置构建器

#### NewConfigBuilder

创建新的配置构建器。

```go
func NewConfigBuilder() *ConfigBuilder
```

**示例：**
```go
cfg := idempotent.NewConfigBuilder().
    KeyPrefix("myapp").
    DefaultTTL(time.Hour).
    Serializer("json").
    EnableCompression().
    Build()
```

## 接口方法 API

### Idempotent 接口

所有客户端都实现的基础接口。

```go
type Idempotent interface {
    Check(ctx context.Context, key string) (bool, error)
    Set(ctx context.Context, key string, ttl time.Duration) (bool, error)
    CheckAndSet(ctx context.Context, key string, ttl time.Duration) (bool, error)
    SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error)
    GetResult(ctx context.Context, key string) (interface{}, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    TTL(ctx context.Context, key string) (time.Duration, error)
    Refresh(ctx context.Context, key string, ttl time.Duration) error
    Close() error
}
```

## 配置结构

### Config

主配置结构体。

```go
type Config struct {
    CacheConfig       cache.Config  // Redis 连接配置
    KeyPrefix         string        // 键前缀
    DefaultTTL        time.Duration // 默认过期时间
    Serializer        string        // 序列化方式
    EnableCompression bool          // 是否启用压缩
    MaxKeyLength      int           // 最大键长度
    KeyValidator      string        // 键名验证器
    EnableMetrics     bool          // 是否启用指标收集
    EnableTracing     bool          // 是否启用链路追踪
    RetryConfig       *RetryConfig  // 重试配置
}
```

### RetryConfig

重试配置结构体。

```go
type RetryConfig struct {
    MaxRetries          int           // 最大重试次数
    InitialInterval     time.Duration // 初始重试间隔
    MaxInterval         time.Duration // 最大重试间隔
    Multiplier          float64       // 重试间隔倍数
    RandomizationFactor float64       // 随机化因子
}
```

## 错误处理

### 常见错误类型

```go
// 配置错误
err := config.Validate()
if err != nil {
    // 处理配置验证错误
}

// 连接错误
client, err := idempotent.New(config)
if err != nil {
    // 处理客户端创建错误
}

// 操作错误
success, err := client.Set(ctx, "key", time.Hour)
if err != nil {
    // 处理幂等操作错误
}
```

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/ceyewan/gochat/im-infra/idempotent"
    "github.com/ceyewan/gochat/im-infra/cache"
)

func main() {
    ctx := context.Background()

    // 1. 使用全局方法
    fmt.Println("=== 全局方法示例 ===")
    
    // 简单的幂等检查和设置
    success, err := idempotent.Set(ctx, "user:create:123", time.Hour)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("设置成功: %t\n", success)

    // 检查是否存在
    exists, err := idempotent.Check(ctx, "user:create:123")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("键存在: %t\n", exists)

    // 2. 使用便捷方法
    fmt.Println("\n=== 便捷方法示例 ===")
    
    result, err := idempotent.Execute(ctx, "user:create:456", time.Hour, func() (interface{}, error) {
        return map[string]interface{}{
            "id":   456,
            "name": "用户456",
        }, nil
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("执行结果: %+v\n", result)

    // 3. 自定义客户端
    fmt.Println("\n=== 自定义客户端示例 ===")
    
    cfg := idempotent.NewConfigBuilder().
        KeyPrefix("custom").
        DefaultTTL(30 * time.Minute).
        CacheConfig(cache.DevelopmentConfig()).
        Build()

    client, err := idempotent.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    success, err = client.SetWithResult(ctx, "operation:789", "操作结果", time.Hour)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("自定义客户端设置成功: %t\n", success)
}
```
