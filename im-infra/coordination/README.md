# coordination

一个现代化、高性能的 Go 分布式协调库，基于 etcd 构建。coordination 提供服务注册发现、分布式锁、配置中心等企业级分布式协调功能，遵循 im-infra 模块设计模式。

## 功能特色

- 🚀 **基于 etcd**：充分利用 etcd 的强一致性和高可用性
- 🎯 **接口驱动**：抽象清晰，封装合理，易于测试和扩展
- 🌟 **全局方法**：支持 `coordination.RegisterService()` 等全局方法，无需显式创建协调器
- 📦 **模块协调器**：`coordination.Module("service-name")` 创建服务特定协调器，单例模式，配置继承
- 🔧 **服务注册发现**：支持健康检查、负载均衡、服务监听
- 🔒 **分布式锁**：支持基础锁、可重入锁、读写锁，自动续期
- ⚙️ **配置中心**：支持版本控制、变更通知、历史追踪
- 🔄 **多环境配置**：提供开发、测试、生产环境的预设配置
- 📁 **重试机制**：内置指数退避重试策略
- 🏷️ **日志集成**：与 clog 日志库深度集成
- ⚡ **高性能**：优化的连接管理和会话复用
- 🎨 **企业级**：支持 TLS、认证、指标收集、链路追踪

## 安装

```bash
go get github.com/ceyewan/gochat/im-infra/coordination
```

## 快速开始

### 基本用法

#### 全局方法（推荐）

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
    ctx := context.Background()
    
    // 1. 服务注册
    service := coordination.ServiceInfo{
        Name:       "user-service",
        InstanceID: "instance-1",
        Address:    "localhost:8080",
        Metadata: map[string]string{
            "version": "1.0.0",
            "region":  "us-west-1",
        },
    }
    
    err := coordination.RegisterService(ctx, service)
    if err != nil {
        log.Fatal(err)
    }
    defer coordination.DeregisterService(ctx, service.Name, service.InstanceID)
    
    // 2. 服务发现
    services, err := coordination.DiscoverServices(ctx, "user-service")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("发现 %d 个服务实例", len(services))
    
    // 3. 分布式锁
    lock, err := coordination.AcquireLock(ctx, "critical-section", 30*time.Second)
    if err != nil {
        log.Fatal(err)
    }
    defer lock.Release(ctx)
    
    // 执行临界区代码
    log.Println("执行临界区操作")
    
    // 4. 配置管理
    err = coordination.SetConfig(ctx, "app.database.url", "postgresql://localhost:5432/myapp", 0)
    if err != nil {
        log.Fatal(err)
    }
    
    config, err := coordination.GetConfig(ctx, "app.database.url")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("数据库配置: %s", config.Value)
}
```

#### 模块协调器

```go
package main

import (
    "context"
    "log"
    
    "github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
    // 为特定服务创建协调器
    userServiceCoordinator := coordination.Module("user-service")
    
    // 使用服务特定的协调器
    registry := userServiceCoordinator.ServiceRegistry()
    lockManager := userServiceCoordinator.Lock()
    configCenter := userServiceCoordinator.ConfigCenter()
    
    ctx := context.Background()
    
    // 服务注册会自动添加服务上下文
    service := coordination.ServiceInfo{
        Name:       "user-service",
        InstanceID: "instance-1",
        Address:    "localhost:8080",
    }
    
    err := registry.Register(ctx, service)
    if err != nil {
        log.Fatal(err)
    }
}
```

#### 自定义配置

```go
package main

import (
    "time"
    
    "github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
    // 创建自定义配置
    cfg := coordination.Config{
        Endpoints:   []string{"etcd-1:2379", "etcd-2:2379", "etcd-3:2379"},
        DialTimeout: 10 * time.Second,
        ServiceRegistry: coordination.ServiceRegistryConfig{
            KeyPrefix:           "/my-services",
            TTL:                 60 * time.Second,
            HealthCheckInterval: 20 * time.Second,
            EnableHealthCheck:   true,
        },
        DistributedLock: coordination.DistributedLockConfig{
            KeyPrefix:       "/my-locks",
            DefaultTTL:      45 * time.Second,
            RenewInterval:   15 * time.Second,
            EnableReentrant: true,
        },
        ConfigCenter: coordination.ConfigCenterConfig{
            KeyPrefix:         "/my-config",
            EnableVersioning:  true,
            MaxVersionHistory: 200,
            EnableValidation:  true,
        },
    }
    
    coordinator, err := coordination.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer coordinator.Close()
    
    // 使用自定义协调器
    // ...
}
```

### 预设配置

```go
// 开发环境配置
cfg := coordination.DevelopmentConfig()

// 生产环境配置
cfg := coordination.ProductionConfig()

// 测试环境配置
cfg := coordination.TestConfig()

coordinator, err := coordination.New(cfg)
```

## 核心功能

### 服务注册与发现

```go
// 注册服务
service := coordination.ServiceInfo{
    Name:       "api-gateway",
    InstanceID: "gateway-1",
    Address:    "192.168.1.100:8080",
    Metadata: map[string]string{
        "version":     "2.1.0",
        "datacenter":  "us-east-1",
        "environment": "production",
    },
    Health: coordination.HealthHealthy,
}

err := coordination.RegisterService(ctx, service)

// 服务发现
services, err := coordination.DiscoverServices(ctx, "api-gateway")

// 获取负载均衡连接
conn, err := coordination.GetServiceConnection(ctx, "api-gateway", coordination.LoadBalanceRoundRobin)

// 监听服务变化
registry := coordination.Module("monitor").ServiceRegistry()
ch, err := registry.Watch(ctx, "api-gateway")
for services := range ch {
    log.Printf("服务列表更新: %d 个实例", len(services))
}
```

### 分布式锁

```go
// 基础锁
lock, err := coordination.AcquireLock(ctx, "resource-lock", 30*time.Second)
defer lock.Release(ctx)

// 可重入锁
reentrantLock, err := coordination.AcquireReentrantLock(ctx, "reentrant-lock", 30*time.Second)
reentrantLock.Acquire(ctx) // 可以多次获取
defer reentrantLock.Release(ctx)

// 读写锁
readLock, err := coordination.AcquireReadLock(ctx, "data-lock", 30*time.Second)
writeLock, err := coordination.AcquireWriteLock(ctx, "data-lock", 30*time.Second)

// 检查锁状态
held, err := lock.IsHeld(ctx)
ttl, err := lock.TTL(ctx)
```

### 配置中心

```go
// 设置配置
err := coordination.SetConfig(ctx, "app.redis.host", "redis.example.com", 0)

// 获取配置
config, err := coordination.GetConfig(ctx, "app.redis.host")
log.Printf("Redis 主机: %s (版本: %d)", config.Value, config.Version)

// 监听配置变更
ch, err := coordination.WatchConfig(ctx, "app.redis.host")
for change := range ch {
    log.Printf("配置变更: %s -> %s", change.OldValue.Value, change.NewValue.Value)
}

// 获取配置历史
history, err := coordination.Module("admin").ConfigCenter().GetHistory(ctx, "app.redis.host", 10)
for _, version := range history {
    log.Printf("版本 %d: %s (%s)", version.Version, version.Value, version.CreateTime)
}
```

## 高级特性

### TLS 和认证

```go
cfg := coordination.DefaultConfig()
cfg.Username = "etcd-user"
cfg.Password = "etcd-password"
cfg.TLS = &coordination.TLSConfig{
    CertFile: "/path/to/client.crt",
    KeyFile:  "/path/to/client.key",
    CAFile:   "/path/to/ca.crt",
}

coordinator, err := coordination.New(cfg)
```

### 重试策略

```go
cfg := coordination.DefaultConfig()
cfg.Retry = &coordination.RetryConfig{
    MaxRetries:          5,
    InitialInterval:     200 * time.Millisecond,
    MaxInterval:         10 * time.Second,
    Multiplier:          2.0,
    RandomizationFactor: 0.1,
}
```

### 指标和追踪

```go
cfg := coordination.ProductionConfig()
cfg.EnableMetrics = true
cfg.EnableTracing = true
```

## 最佳实践

### 1. 服务注册
- 使用有意义的服务名和实例 ID
- 在服务元数据中包含版本和环境信息
- 启用健康检查以确保服务可用性
- 优雅关闭时注销服务

### 2. 分布式锁
- 设置合适的锁超时时间
- 使用 defer 确保锁被释放
- 对于长时间运行的操作，考虑使用自动续期
- 避免嵌套锁以防止死锁

### 3. 配置管理
- 使用层次化的配置键名
- 启用版本控制以支持回滚
- 监听配置变更以实现动态配置
- 验证配置值的有效性

### 4. 错误处理
- 实现适当的重试策略
- 监控和记录错误
- 使用断路器模式处理 etcd 不可用的情况

### 5. 性能优化
- 复用协调器实例
- 使用模块协调器进行服务隔离
- 合理设置连接池大小
- 监控 etcd 集群性能

## 故障排除

### 常见问题

1. **连接失败**
   - 检查 etcd 服务是否运行
   - 验证网络连接和防火墙设置
   - 确认 TLS 配置正确

2. **锁获取超时**
   - 检查是否有死锁
   - 调整锁超时时间
   - 监控锁的使用情况

3. **服务发现失败**
   - 确认服务已正确注册
   - 检查服务健康状态
   - 验证键前缀配置

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个库。

## 许可证

本项目采用 MIT 许可证。
