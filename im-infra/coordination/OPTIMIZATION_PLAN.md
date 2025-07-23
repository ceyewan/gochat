# Coordination 模块简化优化计划

## 项目概述

Coordination 模块是 gochat 项目的内部基础设施库，专注于为微服务架构提供三大核心功能：分布式锁、服务注册发现、配置中心管理。本优化计划采用实用主义原则，去除过度复杂的企业级功能，专注于满足 gochat 项目的实际需求。

## 优化目标

- **简化架构**：去除过度设计，专注核心功能
- **实用性优先**：只实现必需的功能，避免过度工程化
- **易于使用**：提供简洁清晰的 API 接口
- **日志驱动**：使用日志系统替代复杂的监控系统

## 1. 项目架构设计

### 1.1 目录结构

```
im-infra/coordination/
├── pkg/
│   ├── lock/           # 分布式锁实现
│   │   ├── etcd_lock.go
│   │   └── interface.go
│   ├── registry/       # 服务注册发现
│   │   ├── etcd_registry.go
│   │   └── interface.go
│   ├── config/         # 配置中心
│   │   ├── etcd_config.go
│   │   └── interface.go
│   └── client/         # etcd 客户端封装
│       └── etcd_client.go
├── examples/           # 使用示例
│   ├── lock_example.go
│   ├── registry_example.go
│   └── config_example.go
├── coordinator.go      # 主入口
├── options.go          # 配置选项
└── README.md
```

### 1.2 核心接口设计

#### 主协调器接口

```go
// Coordinator 主协调器接口
type Coordinator interface {
    // 获取分布式锁服务
    Lock() DistributedLock

    // 获取服务注册发现
    Registry() ServiceRegistry

    // 获取配置中心
    Config() ConfigCenter

    // 关闭协调器
    Close() error
}
```

#### 构建器模式配置

```go
// CoordinatorOptions 协调器配置选项
type CoordinatorOptions struct {
    Endpoints []string
    Username  string
    Password  string
    Timeout   time.Duration
}

// NewCoordinator 创建协调器实例
func NewCoordinator(opts CoordinatorOptions) (Coordinator, error)
```

## 2. 核心功能实现

### 2.1 分布式锁

#### 接口设计

```go
// DistributedLock 分布式锁接口
type DistributedLock interface {
    // 获取互斥锁
    Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)

    // 尝试获取锁（非阻塞）
    TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}

// Lock 锁对象接口
type Lock interface {
    // 释放锁
    Unlock(ctx context.Context) error

    // 续期锁
    Renew(ctx context.Context, ttl time.Duration) error

    // 获取锁的剩余有效时间
    TTL(ctx context.Context) (time.Duration, error)

    // 获取锁的键
    Key() string
}
```

#### 实现特性

- 基于 etcd 的分布式锁
- 支持锁超时自动释放
- 支持锁续期功能
- 支持非阻塞获取锁

### 2.2 配置中心

#### 接口设计

```go
// ConfigCenter 配置中心接口
type ConfigCenter interface {
    // 获取配置值
    Get(ctx context.Context, key string) (interface{}, error)
    
    // 设置配置值（支持任意可序列化对象）
    Set(ctx context.Context, key string, value interface{}) error
    
    // 删除配置
    Delete(ctx context.Context, key string) error
    
    // 监听配置变化
    Watch(ctx context.Context, key string) (<-chan ConfigEvent, error)
    
    // 列出所有配置键
    List(ctx context.Context, prefix string) ([]string, error)
}

// ConfigEvent 配置变化事件
type ConfigEvent struct {
    Type      EventType   `json:"type"`      // 事件类型：PUT, DELETE
    Key       string      `json:"key"`       // 配置键
    Value     interface{} `json:"value"`     // 配置值（支持任意类型）
    Timestamp time.Time   `json:"timestamp"` // 事件时间
}
```
```go
// EventType 事件类型
type EventType string

const (
    EventTypePut    EventType = "PUT"
    EventTypeDelete EventType = "DELETE"
)
```

#### 实现特性

- 基于 etcd 的配置存储
- 支持任意类型配置值（字符串、数字、JSON 对象等）
- 支持配置的增删改查
- 支持配置变化监听
- 支持按前缀列出配置

### 2.3 服务注册发现

#### 接口设计

```go
// ServiceRegistry 服务注册发现接口
type ServiceRegistry interface {
    // 注册服务
    Register(ctx context.Context, service ServiceInfo) error

    // 注销服务
    Unregister(ctx context.Context, serviceID string) error

    // 发现服务
    Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error)

    // 监听服务变化
    Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)
}

// ServiceInfo 服务信息
type ServiceInfo struct {
    ID       string            `json:"id"`       // 服务实例ID
    Name     string            `json:"name"`     // 服务名称
    Address  string            `json:"address"`  // 服务地址
    Port     int               `json:"port"`     // 服务端口
    Metadata map[string]string `json:"metadata"` // 服务元数据
    TTL      time.Duration     `json:"ttl"`      // 服务TTL
}

// ServiceEvent 服务变化事件
type ServiceEvent struct {
    Type      EventType   `json:"type"`      // 事件类型
    Service   ServiceInfo `json:"service"`   // 服务信息
    Timestamp time.Time   `json:"timestamp"` // 事件时间
}
```

#### 实现特性

- 基于 etcd 的服务注册
- 支持服务自动续期
- 支持服务发现和监听
- 支持服务元数据存储

## 3. 日志系统集成

### 3.1 使用 clog 日志库

```go
// 日志配置
type LogConfig struct {
    Level  string `json:"level"`  // 日志级别
    Format string `json:"format"` // 日志格式
}

// 日志使用示例
func (c *coordinator) logOperation(operation, key string, err error) {
    if err != nil {
        clog.Error("operation failed",
            clog.String("operation", operation),
            clog.String("key", key),
            clog.Error("error", err),
        )
    } else {
        clog.Info("operation success",
            clog.String("operation", operation),
            clog.String("key", key),
        )
    }
}
```

### 3.2 日志规范

- **INFO 级别**：记录正常的操作日志
- **WARN 级别**：记录潜在问题
- **ERROR 级别**：记录错误和异常
- **日志字段**：统一使用结构化日志，包含操作类型、键名、错误信息等

## 4. 错误处理

### 4.1 标准化错误类型

```go
// CoordinationError 协调器错误类型
type CoordinationError struct {
    Code    ErrorCode `json:"code"`    // 错误码
    Message string    `json:"message"` // 错误消息
    Cause   error     `json:"cause"`   // 原始错误
}

// ErrorCode 错误码定义
type ErrorCode string

const (
    ErrCodeConnection  ErrorCode = "CONNECTION_ERROR"
    ErrCodeTimeout     ErrorCode = "TIMEOUT_ERROR"
    ErrCodeNotFound    ErrorCode = "NOT_FOUND"
    ErrCodeConflict    ErrorCode = "CONFLICT"
    ErrCodeValidation  ErrorCode = "VALIDATION_ERROR"
    ErrCodeUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

func (e *CoordinationError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
```

### 4.2 重试机制

```go
// RetryConfig 重试配置
type RetryConfig struct {
    MaxAttempts int           `json:"max_attempts"`     // 最大重试次数
    InitialDelay time.Duration `json:"initial_delay"`   // 初始延迟
    MaxDelay     time.Duration `json:"max_delay"`       // 最大延迟
    Multiplier   float64       `json:"multiplier"`      // 退避倍数
}

// 重试实现
func (c *coordinator) retryOperation(ctx context.Context, operation func() error) error {
    var lastErr error
    delay := c.retryConfig.InitialDelay

    for attempt := 0; attempt < c.retryConfig.MaxAttempts; attempt++ {
        if err := operation(); err == nil {
            return nil
        } else {
            lastErr = err
            clog.Warn("operation retry",
                clog.Int("attempt", attempt+1),
                clog.Duration("delay", delay),
                clog.Error("error", err),
            )
        }

        if attempt < c.retryConfig.MaxAttempts-1 {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
            }

            delay = time.Duration(float64(delay) * c.retryConfig.Multiplier)
            if delay > c.retryConfig.MaxDelay {
                delay = c.retryConfig.MaxDelay
            }
        }
    }

    return lastErr
}
```

## 5. 示例代码

### 5.1 基础使用示例

```go
// examples/basic_example.go
package main

import (
    "context"
    "fmt"
    "time"

    "gochat/im-infra/coordination"
)

func main() {
    // 创建协调器实例
    opts := coordination.CoordinatorOptions{
        Endpoints: []string{"localhost:2379"},
        Timeout:   5 * time.Second,
    }
    coord, err := coordination.NewCoordinator(opts)
    if err != nil {
        panic(err)
    }
    defer coord.Close()

    ctx := context.Background()

    // 分布式锁示例
    lockExample(ctx, coord)

    // 配置中心示例
    configExample(ctx, coord)

    // 服务注册发现示例
    registryExample(ctx, coord)
}

func lockExample(ctx context.Context, coord coordination.Coordinator) {
    fmt.Println("=== 分布式锁示例 ===")

    // 获取锁
    lock, err := coord.Lock().Acquire(ctx, "my-lock", 30*time.Second)
    if err != nil {
        fmt.Printf("acquire lock failed: %v\n", err)
        return
    }
    fmt.Println("lock acquired successfully")

    // 使用锁保护的业务逻辑
    fmt.Println("executing critical section...")
    time.Sleep(2 * time.Second)

    // 释放锁
    if err := lock.Unlock(ctx); err != nil {
        fmt.Printf("unlock failed: %v\n", err)
        return
    }
    fmt.Println("lock released successfully")
}

func configExample(ctx context.Context, coord coordination.Coordinator) {
    fmt.Println("=== 配置中心示例 ===")

    config := coord.Config()

    // 设置简单字符串配置
    if err := config.Set(ctx, "app.name", "gochat"); err != nil {
        fmt.Printf("set config failed: %v\n", err)
        return
    }
    fmt.Println("simple config set successfully")

    // 设置 JSON 对象配置
    dbConfig := map[string]interface{}{
        "host":     "localhost",
        "port":     3306,
        "database": "gochat",
        "username": "root",
        "password": "123456",
        "charset":  "utf8mb4",
        "timeout":  "30s",
    }
    if err := config.Set(ctx, "database.mysql", dbConfig); err != nil {
        fmt.Printf("set database config failed: %v\n", err)
        return
    }
    fmt.Println("database config set successfully")

    // 获取配置并反序列化
    var retrievedDBConfig map[string]interface{}
    if err := config.GetObject(ctx, "database.mysql", &retrievedDBConfig); err != nil {
        fmt.Printf("get database config failed: %v\n", err)
        return
    }

    go func() {
        for event := range watchCh {
            fmt.Printf("config changed: %+v\n", event)
        }
    }()

    // 更新配置触发监听
    time.Sleep(1 * time.Second)
    config.Set(ctx, "app.name", "gochat-v2")
    time.Sleep(1 * time.Second)
}

func registryExample(ctx context.Context, coord coordination.Coordinator) {
    fmt.Println("=== 服务注册发现示例 ===")

    registry := coord.Registry()

    // 注册服务
    service := coordination.ServiceInfo{
        ID:      "chat-service-001",
        Name:    "chat-service",
        Address: "127.0.0.1",
        Port:    8080,
        Metadata: map[string]string{
            "version": "1.0.0",
            "region":  "us-west",
        },
        TTL: 30 * time.Second,
    }

    if err := registry.Register(ctx, service); err != nil {
        fmt.Printf("register service failed: %v\n", err)
        return
    }
    fmt.Println("service registered successfully")

    // 发现服务
    services, err := registry.Discover(ctx, "chat-service")
    if err != nil {
        fmt.Printf("discover service failed: %v\n", err)
        return
    }
    fmt.Printf("discovered %d services\n", len(services))
    for _, svc := range services {
        fmt.Printf("  service: %s at %s:%d\n", svc.ID, svc.Address, svc.Port)
    }

    // 监听服务变化
    watchCh, err := registry.Watch(ctx, "chat-service")
    if err != nil {
        fmt.Printf("watch service failed: %v\n", err)
        return
    }

    go func() {
        for event := range watchCh {
            fmt.Printf("service event: %s - %s\n", event.Type, event.Service.ID)
        }
    }()

    time.Sleep(2 * time.Second)

    // 注销服务
    if err := registry.Unregister(ctx, service.ID); err != nil {
        fmt.Printf("unregister service failed: %v\n", err)
        return
    }
    fmt.Println("service unregistered successfully")
}
```

### 5.2 gRPC 集成示例

```go
// examples/grpc_example.go
package main

import (
    "context"
    "fmt"
    "net"
    "time"

    "google.golang.org/grpc"
    "gochat/im-infra/coordination"
)

// GRPCServiceManager gRPC 服务管理器
type GRPCServiceManager struct {
    coord       coordination.Coordinator
    serviceName string
    address     string
    port        int
    server      *grpc.Server
}

func NewGRPCServiceManager(coord coordination.Coordinator, serviceName, address string, port int) *GRPCServiceManager {
    return &GRPCServiceManager{
        coord:       coord,
        serviceName: serviceName,
        address:     address,
        port:        port,
        server:      grpc.NewServer(),
    }
}

// Start 启动服务并注册到服务发现
func (m *GRPCServiceManager) Start(ctx context.Context) error {
    // 启动 gRPC 服务器
    listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", m.address, m.port))
    if err != nil {
        return fmt.Errorf("failed to listen: %v", err)
    }

    go func() {
        if err := m.server.Serve(listener); err != nil {
            fmt.Printf("grpc server error: %v\n", err)
        }
    }()

    // 注册服务到服务发现
    service := coordination.ServiceInfo{
        ID:      fmt.Sprintf("%s-%d", m.serviceName, time.Now().Unix()),
        Name:    m.serviceName,
        Address: m.address,
        Port:    m.port,
        Metadata: map[string]string{
            "protocol": "grpc",
            "version":  "1.0.0",
        },
        TTL: 30 * time.Second,
    }

    if err := m.coord.Registry().Register(ctx, service); err != nil {
        return fmt.Errorf("failed to register service: %v", err)
    }

    fmt.Printf("gRPC service started and registered: %s at %s:%d\n",
        service.ID, m.address, m.port)

    return nil
}

// Stop 停止服务并注销
func (m *GRPCServiceManager) Stop(ctx context.Context, serviceID string) error {
    // 注销服务
    if err := m.coord.Registry().Unregister(ctx, serviceID); err != nil {
        fmt.Printf("failed to unregister service: %v\n", err)
    }

    // 停止 gRPC 服务器
    m.server.GracefulStop()
    fmt.Println("gRPC service stopped and unregistered")

    return nil
}
```

## 6. 实施计划

### 6.1 开发阶段

#### 第一阶段：基础架构（1-2天）
- [ ] 设计核心接口
- [ ] 实现 etcd 客户端封装
- [ ] 创建项目目录结构

#### 第二阶段：核心功能（3-5天）
- [ ] 实现分布式锁功能
- [ ] 实现配置中心功能
- [ ] 实现服务注册发现功能

#### 第三阶段：日志和错误处理（1-2天）
- [ ] 集成 clog 日志系统
- [ ] 实现标准化错误处理
- [ ] 添加重试机制

#### 第四阶段：示例和测试（2-3天）
- [ ] 编写基础使用示例
- [ ] 编写 gRPC 集成示例
- [ ] 编写单元测试

### 6.2 质量保证

- **代码审查**：所有代码必须经过审查
- **单元测试**：核心功能覆盖率 > 80%
- **集成测试**：与 etcd 的集成测试
- **示例验证**：确保所有示例代码可以正常运行

## 7. 成功标准

### 7.1 功能标准
- [ ] 分布式锁功能完整可用
- [ ] 服务注册发现功能完整可用
- [ ] 配置中心功能完整可用
- [ ] 日志记录完整清晰

### 7.2 质量标准
- [ ] 代码结构清晰，易于理解
- [ ] 错误处理完善
- [ ] 单元测试覆盖率 > 80%
- [ ] 无明显性能问题

### 7.3 可用性标准
- [ ] API 接口简洁易用
- [ ] 示例代码完整可运行
- [ ] 文档清晰完整
- [ ] 与 gochat 项目集成顺畅

## 8. 总结

本优化计划遵循"简单实用"的原则，专注于为 gochat 项目提供三大核心功能：

1. **分布式锁**：用于实现分布式事务和并发控制
2. **服务注册发现**：用于 gRPC 服务的动态发现和负载均衡
3. **配置中心**：用于微服务配置的统一管理

通过去除过度复杂的企业级功能，我们能够快速交付一个稳定可用的基础设施库，满足 gochat 项目的实际需求，同时保持代码的简洁性和可维护性。
