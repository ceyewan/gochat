# idempotent - 分布式幂等组件

一个轻量级、高性能的 Go 分布式幂等库，基于 Redis setnx 命令实现。专为 GoChat 系统设计，提供简洁易用的幂等操作能力。

## 功能特色

- 🚀 **基于 Redis setnx**：利用 Redis 原子性操作保证幂等性
- 🎯 **接口简洁**：提供核心幂等操作，API 简单易用
- 🌟 **全局方法支持**：支持 `idempotent.Do()` 等全局方法，无需显式创建客户端
- 📦 **自定义客户端**：`idempotent.New(config)` 创建自定义配置的客户端实例
- 📝 **结果存储**：支持存储操作结果，避免重复计算
- 🔄 **TTL 支持**：支持设置幂等键的过期时间
- ⚡ **高性能**：优化的 Redis 操作，最小化网络开销
- 🔧 **零额外依赖**：仅依赖 cache 和 clog 组件

## 安装

```bash
go get github.com/ceyewan/gochat/im-infra/idempotent
```

## 快速开始

### 基本用法

#### 核心 Do 操作（推荐）

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/ceyewan/gochat/im-infra/idempotent"
)

func main() {
    ctx := context.Background()
    
    // 执行幂等操作 - 如果已执行过则跳过，否则执行函数
    err := idempotent.Do(ctx, "user:create:123", func() error {
        // 执行实际的业务逻辑
        fmt.Println("创建用户 123")
        return createUser(123)
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // 第二次执行会被跳过
    err = idempotent.Do(ctx, "user:create:123", func() error {
        fmt.Println("这不会被执行")
        return nil
    })
    
    if err != nil {
        log.Fatal(err)
    }
}

func createUser(id int) error {
    // 实际的用户创建逻辑
    return nil
}
```

#### 基础幂等操作

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/ceyewan/gochat/im-infra/idempotent"
)

func main() {
    ctx := context.Background()
    
    // 检查操作是否已执行
    exists, err := idempotent.Check(ctx, "user:create:123")
    if err != nil {
        log.Fatal(err)
    }
    
    if exists {
        fmt.Println("操作已执行过")
        return
    }
    
    // 设置幂等标记
    success, err := idempotent.Set(ctx, "user:create:123", time.Hour)
    if err != nil {
        log.Fatal(err)
    }
    
    if success {
        fmt.Println("首次执行，进行实际操作")
        // 执行实际的业务逻辑
    } else {
        fmt.Println("操作已执行过")
    }
}
```

#### 带结果存储的幂等操作

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/ceyewan/gochat/im-infra/idempotent"
)

func main() {
    ctx := context.Background()
    
    // 设置幂等标记并存储结果
    result := map[string]interface{}{
        "user_id": 123,
        "status":  "created",
    }
    
    success, err := idempotent.SetWithResult(ctx, "user:create:123", result, time.Hour)
    if err != nil {
        log.Fatal(err)
    }
    
    if success {
        fmt.Println("首次执行并存储结果")
    }
    
    // 获取存储的结果
    cachedResult, err := idempotent.GetResult(ctx, "user:create:123")
    if err != nil {
        log.Fatal(err)
    }
    
    if cachedResult != nil {
        fmt.Printf("缓存的结果: %+v\n", cachedResult)
    }
}
```

### 自定义客户端实例

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/ceyewan/gochat/im-infra/cache"
    "github.com/ceyewan/gochat/im-infra/idempotent"
)

func main() {
    ctx := context.Background()
    
    // 创建自定义配置
    cfg := idempotent.NewConfigBuilder().
        KeyPrefix("myapp").
        DefaultTTL(time.Hour).
        CacheConfig(cache.NewConfigBuilder().
            Addr("localhost:6379").
            PoolSize(10).
            Build()).
        Build()
    
    client, err := idempotent.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 使用自定义客户端
    success, err := client.Set(ctx, "operation:789", time.Hour)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("操作成功: %t\n", success)
}
```

## 核心 API

### 主要接口

```go
type Idempotent interface {
    // Check 检查指定键是否已经存在
    Check(ctx context.Context, key string) (bool, error)
    
    // Set 设置幂等标记，如果键已存在则返回 false
    Set(ctx context.Context, key string, ttl time.Duration) (bool, error)
    
    // Do 执行幂等操作，如果已执行过则跳过，否则执行函数
    Do(ctx context.Context, key string, f func() error) error
    
    // SetWithResult 设置幂等标记并存储操作结果
    SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error)
    
    // GetResult 获取存储的操作结果
    GetResult(ctx context.Context, key string) (interface{}, error)
    
    // Delete 删除幂等标记
    Delete(ctx context.Context, key string) error
    
    // 其他方法...
}
```

### 全局方法

- `idempotent.Do(ctx, key, f)` - 核心幂等操作
- `idempotent.Check(ctx, key)` - 检查是否已执行
- `idempotent.Set(ctx, key, ttl)` - 设置幂等标记
- `idempotent.SetWithResult(ctx, key, result, ttl)` - 设置标记并存储结果
- `idempotent.GetResult(ctx, key)` - 获取存储的结果
- `idempotent.Delete(ctx, key)` - 删除幂等标记

### 配置选项

```go
cfg := idempotent.Config{
    KeyPrefix:   "myapp",           // 键前缀，用于业务隔离
    DefaultTTL:  time.Hour,         // 默认过期时间
    CacheConfig: cache.Config{      // Redis 配置
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
        PoolSize: 10,
    },
}
```

## 使用场景

### 1. 防止重复提交

```go
// 防止用户重复提交表单
err := idempotent.Do(ctx, fmt.Sprintf("form:submit:%d", userID), func() error {
    return processFormSubmission(data)
})
```

### 2. 防止重复支付

```go
// 防止重复支付同一订单
err := idempotent.Do(ctx, fmt.Sprintf("payment:%s", orderID), func() error {
    return processPayment(orderID, amount)
})
```

### 3. 防止重复发送通知

```go
// 防止重复发送通知
err := idempotent.Do(ctx, fmt.Sprintf("notification:%d:%s", userID, notificationType), func() error {
    return sendNotification(userID, message)
})
```

### 4. 缓存复杂计算结果

```go
// 缓存复杂计算的结果
result, err := idempotent.SetWithResult(ctx, "calculation:complex:123", 
    calculateComplexData(input), time.Hour)
if err != nil {
    return err
}

// 后续获取缓存结果
cachedResult, err := idempotent.GetResult(ctx, "calculation:complex:123")
```

## 最佳实践

### 1. 键名设计

```go
// ✅ 使用有意义的键名，包含业务信息和唯一标识
idempotent.Do(ctx, "user:create:123", func() error { ... })
idempotent.Do(ctx, "order:payment:456", func() error { ... })
idempotent.Do(ctx, "notification:send:789", func() error { ... })

// ❌ 避免使用无意义的键名
idempotent.Do(ctx, "abc123", func() error { ... })
```

### 2. TTL 设置

```go
// ✅ 根据业务场景设置合适的 TTL
idempotent.Do(ctx, "user:create:123", func() error { ... }) // 用户创建，默认TTL
idempotent.Set(ctx, "payment:process:456", 10*time.Minute) // 支付处理，10分钟
idempotent.Set(ctx, "email:send:789", 24*time.Hour) // 邮件发送，24小时
```

### 3. 错误处理

```go
// ✅ 适当的错误处理
err := idempotent.Do(ctx, "operation:123", func() error {
    return doSomething()
})
if err != nil {
    log.Printf("幂等操作失败: %v", err)
    return err
}
```

## 架构设计

```
idempotent
├── cache (Redis 操作)
├── clog (日志记录)
└── internal (内部实现)
    ├── interfaces.go (接口定义)
    ├── config.go (配置管理)
    └── client.go (核心实现)
```

### 核心原理

1. **原子性保证**：使用 Redis `SETNX` 命令确保幂等检查和设置的原子性
2. **TTL 管理**：支持设置键的过期时间，自动清理过期的幂等标记
3. **结果存储**：可选择存储操作结果，避免重复计算
4. **错误处理**：执行失败时自动清理标记，允许重试
5. **日志记录**：详细的操作日志，便于调试和监控

## 性能特点

- **基于 Redis**：高性能的内存操作
- **最小化网络**：优化的 Redis 命令使用
- **连接池**：复用 cache 组件的连接池管理
- **零内存泄漏**：完善的资源管理

## 测试

运行测试套件：

```bash
go test ./im-infra/once/...
```

## 许可证

此组件是 GoChat 项目的一部分，遵循相同的许可条款。