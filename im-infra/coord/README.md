# Coord 模块

Coord 模块是 gochat 项目的分布式协调基础设施库，基于 etcd 提供三大核心功能：**分布式锁**、**服务注册发现**、**配置中心管理**。

## 设计理念

本模块采用实用主义原则，专注于满足 gochat 项目的实际需求：

- **简化架构**：基于 etcd，去除过度设计。
- **实用性优先**：只实现必需的功能，避免过度工程化。
- **易于使用**：提供简洁清晰的 API 接口。
- **高可靠性**：基于 etcd 的强一致性保证，并内置连接重试机制。
- **gRPC 集成**：原生支持 gRPC 服务发现和客户端负载均衡。

## 核心功能

### 🔒 分布式锁
- 基于 etcd 的互斥锁。
- 支持阻塞 (`Acquire`) 和非阻塞 (`TryAcquire`) 获取。
- 锁持有者通过租约（Lease）实现 TTL，并自动续约。
- 支持通过 `context` 取消阻塞的获取操作。
- 提供了 `Unlock`, `TTL`, `Key` 等完整的锁操作接口。

### 🔍 服务注册发现
- 服务注册与基于租约的自动注销。
- 实时服务发现和变化监听。
- **内置 gRPC 连接管理**，提供负载均衡的 `grpc.ClientConn`。
- 基于租约的健康检查机制。

### ⚙️ 配置中心
- 强类型配置的 Get/Set/Delete/List 操作。
- 支持对单个 Key 或指定前缀（Prefix）进行实时变更监听。
- 泛型支持，提供类型安全的事件通知。

## 快速开始

### 1. 安装

```bash
go get github.com/ceyewan/gochat/im-infra/coord
```

### 2. 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/ceyewan/gochat/im-infra/coord"
    "github.com/ceyewan/gochat/im-infra/coord/registry"
)

func main() {
    // 1. 创建协调器实例 (使用默认配置)
    coordinator, err := coord.New()
    if err != nil {
        log.Fatalf("Failed to create coordinator: %v", err)
    }
    defer coordinator.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
    defer cancel()

    // 2. 使用分布式锁
    fmt.Println("Acquiring lock...")
    lock, err := coordinator.Lock().Acquire(ctx, "my-lock-key", 15*time.Second)
    if err != nil {
        log.Fatalf("Failed to acquire lock: %v", err)
    }
    defer lock.Unlock(ctx)
    fmt.Printf("Lock '%s' acquired.\n", lock.Key())

    // 3. 使用服务注册
    fmt.Println("Registering service...")
    service := registry.ServiceInfo{
        ID:      "user-service-1",
        Name:    "user-service",
        Address: "127.0.0.1",
        Port:    8080,
    }
    if err := coordinator.Registry().Register(ctx, service, 30*time.Second); err != nil {
        log.Fatalf("Failed to register service: %v", err)
    }
    defer coordinator.Registry().Unregister(ctx, service.ID)
    fmt.Printf("Service '%s' registered.\n", service.Name)

    // 4. 使用配置中心
    fmt.Println("Setting config...")
    configKey := "app/settings/theme"
    if err := coordinator.Config().Set(ctx, configKey, "dark"); err != nil {
        log.Fatalf("Failed to set config: %v", err)
    }

    var theme string
    if err := coordinator.Config().Get(ctx, configKey, &theme); err != nil {
        log.Fatalf("Failed to get config: %v", err)
    }
    fmt.Printf("Config '%s' has value: '%s'\n", configKey, theme)
}
```

### 3. 配置选项

可以通过 `coord.New()` 传入自定义配置。

```go
import "time"
import "github.com/ceyewan/gochat/im-infra/coord"

// 自定义配置
config := coord.CoordinatorConfig{
    Endpoints: []string{"etcd-1:2379", "etcd-2:2379", "etcd-3:2379"},
    Username:  "user",
    Password:  "password",
    Timeout:   10 * time.Second,
    RetryConfig: &coord.RetryConfig{
        MaxAttempts:  5,
        InitialDelay: 200 * time.Millisecond,
        MaxDelay:     5 * time.Second,
        Multiplier:   2.0,
    },
}
coordinator, err := coord.New(config)
```

## API 参考

详细的 API 文档请参阅 [`API.md`](./API.md)。以下为核心接口概览。

### Provider

主协调器接口，提供三大功能模块的统一访问入口。

```go
type Provider interface {
    Lock() lock.DistributedLock
    Registry() registry.ServiceRegistry
    Config() config.ConfigCenter
    Close() error
}
```

### DistributedLock

分布式锁接口。

```go
type DistributedLock interface {
    Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
    TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}

type Lock interface {
    Unlock(ctx context.Context) error
    TTL(ctx context.Context) (time.Duration, error)
    Key() string
}
```

### ServiceRegistry

服务注册发现接口。

```go
type ServiceRegistry interface {
    Register(ctx context.Context, service ServiceInfo, ttl time.Duration) error
    Unregister(ctx context.Context, serviceID string) error
    Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error)
    Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)
    GetConnection(ctx context.Context, serviceName string) (*grpc.ClientConn, error)
}
```

### ConfigCenter

配置中心接口。

```go
type ConfigCenter interface {
    Get(ctx context.Context, key string, v interface{}) error
    Set(ctx context.Context, key string, value interface{}) error
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) ([]string, error)
    Watch(ctx context.Context, key string, v interface{}) (Watcher[any], error)
    WatchPrefix(ctx context.Context, prefix string, v interface{}) (Watcher[any], error)
}
```

## 目录结构

```
coord/
├── internal/          # 内部实现
│   ├── client/        # etcd 客户端封装
│   ├── configimpl/    # 配置中心实现
│   ├── lockimpl/      # 分布式锁实现
│   └── registryimpl/  # 服务注册发现实现
├── config/            # 配置中心接口定义
├── lock/              # 分布式锁接口定义
├── registry/          # 服务注册发现接口定义
├── examples/          # 使用示例
├── coord.go           # 主协调器实现
├── config.go          # 配置结构定义
├── API.md             # 详细 API 文档
└── README.md          # 本文档
```

## 测试

运行所有测试：
```bash
go test -v ./...
```

运行测试并生成覆盖率报告：
```bash
go test -v -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out
```

## 依赖

- Go 1.21+
- etcd v3.5+
- gRPC v1.50+
- `github.com/ceyewan/gochat/im-infra/clog`