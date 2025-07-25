# Coordination 模块

Coordination 模块是 gochat 项目的内部基础设施库，专注于为微服务架构提供三大核心功能：**分布式锁**、**服务注册发现**、**配置中心管理**。

## 设计理念

本模块采用实用主义原则，去除过度复杂的企业级功能，专注于满足 gochat 项目的实际需求：

- **简化架构**：去除过度设计，专注核心功能
- **实用性优先**：只实现必需的功能，避免过度工程化
- **易于使用**：提供简洁清晰的 API 接口
- **日志驱动**：使用 clog 日志系统替代复杂的监控系统

## 核心功能

### 🔒 分布式锁
- 互斥锁获取与释放
- 锁自动续期机制
- TTL 管理
- 非阻塞锁获取

### 🔍 服务注册发现
- 服务注册与注销
- 服务发现
- 服务变化监听
- 服务 TTL 自动续期

### ⚙️ 配置中心
- 任意类型配置值存储
- 配置变更监听
- 配置前缀管理
- JSON 对象支持

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "time"
    
    "github.com/ceyewan/gochat/im-infra/coordination"
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
    
    // 使用分布式锁
    lock, err := coord.Lock().Acquire(ctx, "my-lockimpl", 30*time.Second)
    if err != nil {
        panic(err)
    }
    defer lock.Unlock(ctx)
    
    // 使用配置中心
    err = coord.Config().Set(ctx, "app.name", "gochat")
    if err != nil {
        panic(err)
    }
    
    value, err := coord.Config().Get(ctx, "app.name")
    if err != nil {
        panic(err)
    }
    
    // 使用服务注册
    service := coordination.ServiceInfo{
        ID:      "service-001",
        Name:    "chat-service",
        Address: "127.0.0.1",
        Port:    8080,
        TTL:     30 * time.Second,
    }
    
    err = coord.Registry().Register(ctx, service)
    if err != nil {
        panic(err)
    }
}
```

### 全局方法使用

```go
package main

import (
    "context"
    "time"
    
    "github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
    ctx := context.Background()
    
    // 全局锁方法
    lock, err := coordination.AcquireLock(ctx, "global-lockimpl", 30*time.Second)
    if err != nil {
        panic(err)
    }
    defer lock.Unlock(ctx)
    
    // 全局配置方法
    err = coordination.SetConfig(ctx, "global.setting", "value")
    if err != nil {
        panic(err)
    }
    
    value, err := coordination.GetConfig(ctx, "global.setting")
    if err != nil {
        panic(err)
    }
    
    // 全局服务注册方法
    service := coordination.ServiceInfo{
        ID:      "global-service-001",
        Name:    "global-service",
        Address: "127.0.0.1",
        Port:    9090,
        TTL:     30 * time.Second,
    }
    
    err = coordination.RegisterService(ctx, service)
    if err != nil {
        panic(err)
    }
}
```



## API 参考

### 核心接口

#### Coordinator

主协调器接口，提供三大功能模块的统一访问入口：

```go
type Coordinator interface {
    Lock() DistributedLock      // 获取分布式锁服务
    Registry() ServiceRegistry  // 获取服务注册发现
    Config() ConfigCenter       // 获取配置中心
    Close() error              // 关闭协调器
}
```

#### DistributedLock

分布式锁接口：

```go
type DistributedLock interface {
    // 获取互斥锁（阻塞）
    Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
    
    // 尝试获取锁（非阻塞）
    TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}

type Lock interface {
    Unlock(ctx context.Context) error                    // 释放锁
    Renew(ctx context.Context, ttl time.Duration) error // 续期锁
    TTL(ctx context.Context) (time.Duration, error)     // 获取剩余时间
    Key() string                                         // 获取锁键
}
```

#### ServiceRegistry

服务注册发现接口：

```go
type ServiceRegistry interface {
    Register(ctx context.Context, service ServiceInfo) error           // 注册服务
    Unregister(ctx context.Context, serviceID string) error           // 注销服务
    Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error) // 发现服务
    Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error) // 监听变化
}
```

#### ConfigCenter

配置中心接口：

```go
type ConfigCenter interface {
    Get(ctx context.Context, key string) (interface{}, error)        // 获取配置
    Set(ctx context.Context, key string, value interface{}) error   // 设置配置
    Delete(ctx context.Context, key string) error                   // 删除配置
    Watch(ctx context.Context, key string) (<-chan ConfigEvent, error) // 监听变化
    List(ctx context.Context, prefix string) ([]string, error)      // 列出配置键
}
```

### 配置选项

```go
type CoordinatorOptions struct {
    Endpoints   []string       `json:"endpoints"`    // etcd 服务器地址列表
    Username    string         `json:"username"`     // etcd 用户名（可选）
    Password    string         `json:"password"`     // etcd 密码（可选）
    Timeout     time.Duration  `json:"timeout"`      // 连接超时时间
    RetryConfig *RetryConfig   `json:"retry_config"` // 重试配置
}

type RetryConfig struct {
    MaxAttempts  int           `json:"max_attempts"`  // 最大重试次数
    InitialDelay time.Duration `json:"initial_delay"` // 初始延迟
    MaxDelay     time.Duration `json:"max_delay"`     // 最大延迟
    Multiplier   float64       `json:"multiplier"`    // 退避倍数
}
```

### 数据类型

```go
type ServiceInfo struct {
    ID       string            `json:"id"`       // 服务实例ID
    Name     string            `json:"name"`     // 服务名称
    Address  string            `json:"address"`  // 服务地址
    Port     int               `json:"port"`     // 服务端口
    Metadata map[string]string `json:"metadata"` // 服务元数据
    TTL      time.Duration     `json:"ttl"`      // 服务TTL
}

type ServiceEvent struct {
    Type      EventType   `json:"type"`      // 事件类型：PUT, DELETE
    Service   ServiceInfo `json:"service"`   // 服务信息
    Timestamp time.Time   `json:"timestamp"` // 事件时间
}

type ConfigEvent struct {
    Type      EventType   `json:"type"`      // 事件类型：PUT, DELETE
    Key       string      `json:"key"`       // 配置键
    Value     interface{} `json:"value"`     // 配置值
    Timestamp time.Time   `json:"timestamp"` // 事件时间
}
```

## 错误处理

本模块提供标准化的错误处理机制：

```go
type CoordinationError struct {
    Code    ErrorCode `json:"code"`    // 错误码
    Message string    `json:"message"` // 错误消息
    Cause   error     `json:"cause"`   // 原始错误
}

// 错误码定义
const (
    ErrCodeConnection  ErrorCode = "CONNECTION_ERROR"
    ErrCodeTimeout     ErrorCode = "TIMEOUT_ERROR"
    ErrCodeNotFound    ErrorCode = "NOT_FOUND"
    ErrCodeConflict    ErrorCode = "CONFLICT"
    ErrCodeValidation  ErrorCode = "VALIDATION_ERROR"
    ErrCodeUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// 错误检查和处理
if err != nil {
    if coordination.IsCoordinationError(err) {
        code := coordination.GetErrorCode(err)
        switch code {
        case coordination.ErrCodeNotFound:
            // 处理资源未找到
        case coordination.ErrCodeTimeout:
            // 处理超时
        }
    }
}
```

## 配置示例

### 默认配置

```go
opts := coordination.DefaultCoordinatorOptions()
coord, err := coordination.NewCoordinator(opts)
```

### 自定义配置

```go
opts := coordination.CoordinatorOptions{
    Endpoints: []string{"etcd-1:2379", "etcd-2:2379", "etcd-3:2379"},
    Username:  "your-username",
    Password:  "your-password",
    Timeout:   10 * time.Second,
    RetryConfig: &coordination.RetryConfig{
        MaxAttempts:  5,
        InitialDelay: 200 * time.Millisecond,
        MaxDelay:     5 * time.Second,
        Multiplier:   2.0,
    },
}
```

## 监听示例

### 配置变化监听

```go
watchCh, err := coord.Config().Watch(ctx, "app.configimpl")
if err != nil {
    panic(err)
}

go func() {
    for event := range watchCh {
        fmt.Printf("配置变化: %s = %v (类型: %s)\n", 
            event.Key, event.Value, event.Type)
    }
}()
```

### 服务变化监听

```go
watchCh, err := coord.Registry().Watch(ctx, "chat-service")
if err != nil {
    panic(err)
}

go func() {
    for event := range watchCh {
        fmt.Printf("服务变化: %s %s (类型: %s)\n",
            event.Service.Name, event.Service.ID, event.Type)
    }
}()
```

## 最佳实践

1. **资源管理**：总是调用 `Close()` 方法释放资源
   ```go
   coord, err := coordination.NewCoordinator(opts)
   if err != nil {
       return err
   }
   defer coord.Close() // 重要：释放资源
   ```

2. **错误处理**：使用标准化的错误检查
   ```go
   if err != nil {
       if coordination.IsCoordinationError(err) {
           code := coordination.GetErrorCode(err)
           // 根据错误码进行相应处理
       }
       return err
   }
   ```

3. **超时控制**：为所有操作设置合适的超时时间
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
   defer cancel()
   ```

4. **日志观察**：关注结构化日志输出，有助于问题诊断

5. **测试环境**：确保 etcd 服务可用，参考测试用例进行集成测试

## 测试

运行单元测试：
```bash
go test -v ./...
```

运行集成测试（需要 etcd）：
```bash
go test -v -tags integration ./...
```

运行基准测试：
```bash
go test -bench=. -v ./...
```

## 依赖

- etcd v3.5+
- Go 1.18+
- clog 日志库

## 目录结构

```
coordination/
├── pkg/
│   ├── client/          # etcd 客户端封装
│   ├── lock/            # 分布式锁实现
│   ├── registry/        # 服务注册发现实现
│   └── config/          # 配置中心实现
├── examples/            # 使用示例
├── coordinator.go       # 主协调器实现
├── coordination.go      # 全局方法
├── interfaces.go        # 核心接口定义
├── options.go           # 配置选项和错误处理
└── README.md           # 本文档
```

## 许可证

内部项目，仅供 gochat 团队使用。