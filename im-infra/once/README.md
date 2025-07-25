# idempotent - 分布式幂等组件

一个现代化、高性能的 Go 分布式幂等库，基于 Redis setnx 命令实现。本项目是 gochat 即时通讯系统基础设施库的重要组成部分，提供了企业级分布式系统中幂等操作的最佳实践。

## 功能特色

- 🚀 **基于 Redis setnx**：利用 Redis 原子性操作保证幂等性
- 🎯 **接口驱动**：抽象清晰，封装合理
- 🌟 **全局方法支持**：支持 `idempotent.Set()` 等全局方法，无需显式创建客户端实例
- 📦 **自定义客户端实例**：`idempotent.New(config)` 创建自定义配置的客户端实例
- 📝 **结果存储**：支持存储操作结果，避免重复计算
- 🔄 **TTL 支持**：支持设置幂等键的过期时间
- 🏷️ **日志集成**：与 clog 日志库深度集成，提供详细的操作日志
- ⚡ **高性能**：优化的序列化和网络操作
- 🎨 **配置灵活**：丰富的配置选项和预设配置
- 🔧 **零额外依赖**：仅依赖 cache 和 clog 组件

## 安装

```bash
go get github.com/ceyewan/gochat/im-infra/idempotent
```

## 快速开始

### 基本用法

#### 全局方法（推荐）

```go
package main

import (
    "context"
    "time"
    "github.com/ceyewan/gochat/im-infra/idempotent"
)

func main() {
    ctx := context.Background()
    
    // 检查操作是否已执行
    exists, err := idempotent.Check(ctx, "user:create:123")
    if err != nil {
        panic(err)
    }
    
    if exists {
        fmt.Println("操作已执行过")
        return
    }
    
    // 设置幂等标记
    success, err := idempotent.Set(ctx, "user:create:123", time.Hour)
    if err != nil {
        panic(err)
    }
    
    if success {
        fmt.Println("首次执行，进行实际操作")
        // 执行实际的业务逻辑
    } else {
        fmt.Println("并发情况下，其他协程已执行")
    }
}
```

#### 便捷的执行方法

```go
package main

import (
    "context"
    "time"
    "github.com/ceyewan/gochat/im-infra/idempotent"
)

func main() {
    ctx := context.Background()
    
    // 执行幂等操作，自动处理首次执行和重复执行
    result, err := idempotent.Execute(ctx, "user:create:123", time.Hour, func() (interface{}, error) {
        // 执行实际的业务逻辑
        user := createUser(123)
        return user, nil
    })
    
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("用户创建结果: %+v\n", result)
}

func createUser(id int) map[string]interface{} {
    return map[string]interface{}{
        "id":   id,
        "name": "用户" + fmt.Sprintf("%d", id),
    }
}
```

#### 简单的幂等操作

```go
package main

import (
    "context"
    "time"
    "github.com/ceyewan/gochat/im-infra/idempotent"
)

func main() {
    ctx := context.Background()
    
    // 执行简单的幂等操作，只设置标记不存储结果
    err := idempotent.ExecuteSimple(ctx, "notification:send:123", time.Hour, func() error {
        return sendNotification(123)
    })
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println("通知发送完成")
}

func sendNotification(userID int) error {
    // 发送通知的逻辑
    fmt.Printf("发送通知给用户 %d\n", userID)
    return nil
}
```

### 自定义客户端实例

```go
package main

import (
    "context"
    "time"
    "github.com/ceyewan/gochat/im-infra/idempotent"
    "github.com/ceyewan/gochat/im-infra/cache"
)

func main() {
    ctx := context.Background()
    
    // 创建自定义配置的幂等客户端
    userCfg := idempotent.NewConfigBuilder().
        KeyPrefix("user").
        DefaultTTL(time.Hour).
        CacheConfig(cache.NewConfigBuilder().
            Addr("localhost:6379").
            PoolSize(10).
            Build()).
        Build()
    
    userClient, err := idempotent.New(userCfg)
    if err != nil {
        panic(err)
    }
    defer userClient.Close()
    
    // 使用自定义客户端
    success, err := userClient.Set(ctx, "create:123", time.Hour)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("操作成功: %t\n", success)
}
```

### 自定义配置

```go
package main

import (
    "github.com/ceyewan/gochat/im-infra/idempotent"
    "github.com/ceyewan/gochat/im-infra/cache"
)

func main() {
    // 使用预设配置
    cfg := idempotent.ProductionConfig()
    
    // 或者使用配置构建器
    cfg = idempotent.NewConfigBuilder().
        KeyPrefix("myapp").
        DefaultTTL(2 * time.Hour).
        CacheConfig(cache.NewConfigBuilder().
            Addr("redis:6379").
            Password("secret").
            DB(0).
            PoolSize(20).
            Build()).
        Serializer("json").
        EnableCompression().
        Build()
    
    client, err := idempotent.New(cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // 使用自定义客户端
    // ...
}
```

## 高级用法

### 带结果存储的幂等操作

```go
// 设置幂等标记并存储结果
result := map[string]interface{}{
    "user_id": 123,
    "status":  "created",
}

success, err := idempotent.SetWithResult(ctx, "user:create:123", result, time.Hour)
if err != nil {
    panic(err)
}

if !success {
    // 获取已存储的结果
    cachedResult, err := idempotent.GetResult(ctx, "user:create:123")
    if err != nil {
        panic(err)
    }
    fmt.Printf("缓存的结果: %+v\n", cachedResult)
}
```

### TTL 管理

```go
// 获取剩余过期时间
ttl, err := idempotent.TTL(ctx, "user:create:123")
if err != nil {
    panic(err)
}
fmt.Printf("剩余时间: %v\n", ttl)

// 刷新过期时间
err = idempotent.Refresh(ctx, "user:create:123", 2*time.Hour)
if err != nil {
    panic(err)
}
```

### 删除幂等标记

```go
// 删除幂等标记，允许重新执行
err := idempotent.Delete(ctx, "user:create:123")
if err != nil {
    panic(err)
}
```

## 配置选项

### 预设配置

```go
// 开发环境
cfg := idempotent.DevelopmentConfig()

// 生产环境
cfg := idempotent.ProductionConfig()

// 测试环境
cfg := idempotent.TestConfig()
```

### 配置构建器

```go
cfg := idempotent.NewConfigBuilder().
    KeyPrefix("myapp").                    // 键前缀
    DefaultTTL(time.Hour).                 // 默认过期时间
    Serializer("json").                    // 序列化方式
    EnableCompression().                   // 启用压缩
    MaxKeyLength(200).                     // 最大键长度
    KeyValidator("strict").                // 键名验证器
    EnableMetrics().                       // 启用指标收集
    EnableTracing().                       // 启用链路追踪
    Build()
```

## 最佳实践

### 1. 键名设计

```go
// ✅ 使用有意义的键名
idempotent.Set(ctx, "user:create:123", time.Hour)
idempotent.Set(ctx, "order:payment:456", time.Hour)
idempotent.Set(ctx, "notification:send:789", time.Hour)

// ❌ 避免使用无意义的键名
idempotent.Set(ctx, "abc123", time.Hour)
```

### 2. TTL 设置

```go
// ✅ 根据业务场景设置合适的 TTL
idempotent.Set(ctx, "user:create:123", time.Hour)        // 用户创建，1小时
idempotent.Set(ctx, "payment:process:456", 10*time.Minute) // 支付处理，10分钟
idempotent.Set(ctx, "email:send:789", 24*time.Hour)     // 邮件发送，24小时
```

### 3. 错误处理

```go
// ✅ 适当的错误处理
success, err := idempotent.Set(ctx, "operation:123", time.Hour)
if err != nil {
    log.Printf("设置幂等标记失败: %v", err)
    return err
}

if !success {
    log.Printf("操作已执行过，跳过")
    return nil
}
```

### 4. 使用便捷方法

```go
// ✅ 使用 Execute 方法简化代码
result, err := idempotent.Execute(ctx, "user:create:123", time.Hour, func() (interface{}, error) {
    return createUser(123)
})

// ✅ 使用 ExecuteSimple 方法处理无返回值的操作
err := idempotent.ExecuteSimple(ctx, "notification:send:123", time.Hour, func() error {
    return sendNotification(123)
})
```

## 监控和日志

idempotent 与 clog 深度集成，提供详细的操作日志：

- 操作成功/失败日志
- 幂等检查结果
- TTL 管理操作
- 性能指标统计

## 示例

详见 [API.md](./API.md) 文档，包含完整的 API 使用方法和示例代码。

## 性能

idempotent 基于高性能的 cache 组件和 Redis，具备优秀性能：

- 基于 Redis setnx 的原子性操作
- 高效的序列化和网络操作
- 连接池管理和错误恢复
- 最小化的内存分配

## 架构设计

### 组件依赖

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

1. **原子性保证**：使用 Redis setnx 命令确保幂等检查和设置的原子性
2. **TTL 管理**：支持设置键的过期时间，自动清理过期的幂等标记
3. **结果存储**：可选择存储操作结果，避免重复计算
4. **错误处理**：完善的错误处理和重试机制
5. **日志记录**：详细的操作日志，便于调试和监控
