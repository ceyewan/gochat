# Coord API 文档

## 概述

`coord` 是一个基于 etcd 的分布式协调库，为 gochat 项目提供分布式锁、服务注册发现、配置中心等核心功能。

## 核心特性

- 🚀 **基于 etcd**：充分利用 etcd 的强一致性和高可用性。
- 🔧 **gRPC 动态服务发现**：实现标准 gRPC resolver 插件，支持实时服务发现和动态负载均衡。
- ⚡ **高性能连接管理**：连接复用 + gRPC 原生负载均衡，毫秒级故障转移。
- 🔒 **分布式锁**：基于 etcd 的互斥锁，支持 TTL、自动续约和上下文取消。
- ⚙️ **配置中心**：强类型配置管理，支持实时监听和 Key-Value 操作。
- 🎯 **高可靠性**：内置可配置的连接重试和指数退避策略。

## 创建协调器

### 基本用法

```go
import "github.com/ceyewan/gochat/im-infra/coord"

// 使用默认配置（连接 "localhost:2379"）
coordinator, err := coord.New()
if err != nil {
    log.Fatal(err)
}
defer coordinator.Close()
```

### 自定义配置

```go
import (
    "time"
    "github.com/ceyewan/gochat/im-infra/coord"
)

config := coord.CoordinatorConfig{
    Endpoints: []string{"etcd-1:2379", "etcd-2:2379"},
    Username:  "your-username",
    Password:  "your-password",
    Timeout:   10 * time.Second,
    RetryConfig: &coord.RetryConfig{
        MaxAttempts:  5,
        InitialDelay: 200 * time.Millisecond,
        MaxDelay:     5 * time.Second,
        Multiplier:   2.0,
    },
}

coordinator, err := coord.New(config)
if err != nil {
    log.Fatal(err)
}
defer coordinator.Close()
```

## 核心接口

### Provider

主协调器接口，提供三大功能模块的统一访问入口：

```go
type Provider interface {
    // Lock 获取分布式锁服务
    Lock() lock.DistributedLock

    // Registry 获取服务注册发现服务
    Registry() registry.ServiceRegistry

    // Config 获取配置中心服务
    Config() config.ConfigCenter

    // Close 关闭协调器并释放资源
    Close() error
}
```

---

### DistributedLock

分布式锁接口：

```go
type DistributedLock interface {
    // Acquire 获取互斥锁（阻塞直到获取成功或 context 取消）
    Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)

    // TryAcquire 尝试获取锁（非阻塞），如果锁已被占用，会立即返回错误
    TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}
```

#### Lock

已获取的锁实例接口：

```go
type Lock interface {
    // Unlock 释放锁
    Unlock(ctx context.Context) error

    // TTL 获取锁的剩余有效时间
    TTL(ctx context.Context) (time.Duration, error)

    // Key 获取锁的键
    Key() string
}
```

---

### ServiceRegistry

服务注册发现接口，支持 gRPC 动态服务发现：

```go
type ServiceRegistry interface {
    // Register 注册服务，ttl 是租约的有效期
    Register(ctx context.Context, service ServiceInfo, ttl time.Duration) error

    // Unregister 注销服务
    Unregister(ctx context.Context, serviceID string) error

    // Discover 发现服务
    Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error)

    // Watch 监听服务变化
    Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)

    // GetConnection 获取到指定服务的 gRPC 连接，使用动态服务发现和负载均衡
    // 🚀 新特性：基于 gRPC resolver 插件，支持实时服务发现和故障转移
    GetConnection(ctx context.Context, serviceName string) (*grpc.ClientConn, error)
}
```

#### ServiceInfo & ServiceEvent

服务信息与事件结构：

```go
// ServiceInfo 服务信息
type ServiceInfo struct {
    ID       string            `json:"id"`
    Name     string            `json:"name"`
    Address  string            `json:"address"`
    Port     int               `json:"port"`
    Metadata map[string]string `json:"metadata,omitempty"`
}

// ServiceEvent 服务变化事件
type ServiceEvent struct {
    Type    EventType
    Service ServiceInfo
}

// EventType 事件类型
type EventType string
const (
    EventTypePut    EventType = "PUT"
    EventTypeDelete EventType = "DELETE"
)
```

---

### ConfigCenter

配置中心接口：

```go
type ConfigCenter interface {
    // Get 检索一个配置值并将其反序列化到提供的类型中
    Get(ctx context.Context, key string, v interface{}) error

    // Set 序列化并存储一个配置值
    Set(ctx context.Context, key string, value interface{}) error

    // Delete 删除一个配置键
    Delete(ctx context.Context, key string) error

    // Watch 监听单个键的变化
    Watch(ctx context.Context, key string, v interface{}) (Watcher[any], error)

    // WatchPrefix 监听给定前缀下的所有键的变化
    WatchPrefix(ctx context.Context, prefix string, v interface{}) (Watcher[any], error)
    
    // List 列出给定前缀下的所有键
    List(ctx context.Context, prefix string) ([]string, error)
}
```

#### Watcher & ConfigEvent

配置监听器与事件结构：

```go
// Watcher 配置监听器接口
type Watcher[T any] interface {
    Chan() <-chan ConfigEvent[T]
    Close()
}

// ConfigEvent 配置变化事件
type ConfigEvent[T any] struct {
    Type  EventType
    Key   string
    Value T
}
```

## 配置选项

### CoordinatorConfig

```go
type CoordinatorConfig struct {
    // Endpoints etcd 服务器地址列表
    Endpoints []string `json:"endpoints"`

    // Username etcd 用户名（可选）
    Username string `json:"username,omitempty"`

    // Password etcd 密码（可选）
    Password string `json:"password,omitempty"`

    // Timeout 连接超时时间
    Timeout time.Duration `json:"timeout"`

    // RetryConfig 重试配置
    RetryConfig *RetryConfig `json:"retry_config,omitempty"`
}
```

### RetryConfig

```go
type RetryConfig struct {
    // MaxAttempts 最大重试次数
    MaxAttempts int `json:"max_attempts"`

    // InitialDelay 初始延迟
    InitialDelay time.Duration `json:"initial_delay"`

    // MaxDelay 最大延迟
    MaxDelay time.Duration `json:"max_delay"`

    // Multiplier 退避倍数
    Multiplier float64 `json:"multiplier"`
}
```

### 默认配置

```go
func DefaultConfig() CoordinatorConfig {
    return CoordinatorConfig{
        Endpoints: []string{"localhost:2379"},
        Timeout:   5 * time.Second,
        RetryConfig: &RetryConfig{
            MaxAttempts:  3,
            InitialDelay: 100 * time.Millisecond,
            MaxDelay:     2 * time.Second,
            Multiplier:   2.0,
        },
    }
}
```

## 使用示例

### 分布式锁示例

```go
// 获取分布式锁
lock, err := coordinator.Lock().Acquire(ctx, "my-resource", 30*time.Second)
if err != nil {
    panic(err)
}
defer lock.Unlock(ctx) // 使用 Unlock 释放锁

// 检查锁的剩余 TTL
ttl, err := lock.TTL(ctx)
if err == nil {
    fmt.Printf("Lock '%s' will expire in %v\n", lock.Key(), ttl)
}

// 执行需要互斥的操作...
```

### 服务注册发现示例

```go
// 注册服务
service := registry.ServiceInfo{
    ID:      "chat-service-001",
    Name:    "chat-service",
    Address: "127.0.0.1",
    Port:    8080,
}
err = coordinator.Registry().Register(ctx, service, 30*time.Second)
if err != nil {
    panic(err)
}
defer coordinator.Registry().Unregister(ctx, service.ID)

// 发现服务
services, err := coordinator.Registry().Discover(ctx, "chat-service")
if err != nil {
    panic(err)
}
fmt.Printf("Found services: %+v\n", services)

// 🚀 获取 gRPC 连接（使用动态服务发现）
conn, err := coordinator.Registry().GetConnection(ctx, "chat-service")
if err != nil {
    panic(err)
}
defer conn.Close()
// 使用 conn 创建 gRPC 客户端...
```

### 配置中心示例

```go
type AppConfig struct {
    Name    string `json:"name"`
    Port    int    `json:"port"`
    Enabled bool   `json:"enabled"`
}

// 设置配置
appConfig := AppConfig{Name: "gochat", Port: 8080, Enabled: true}
err = coordinator.Config().Set(ctx, "app/config", appConfig)
if err != nil {
    panic(err)
}

// 获取配置
var retrievedConfig AppConfig
err = coordinator.Config().Get(ctx, "app/config", &retrievedConfig)
if err != nil {
    panic(err)
}
fmt.Printf("Retrieved config: %+v\n", retrievedConfig)

// 列出前缀下的所有键
keys, err := coordinator.Config().List(ctx, "app/")
if err != nil {
    panic(err)
}
fmt.Printf("Keys under 'app/': %v\n", keys)
```

## gRPC 动态服务发现

coord 模块实现了标准的 gRPC resolver 插件机制，提供：

- **实时服务发现**：自动感知服务变化
- **智能负载均衡**：支持 round_robin、pick_first 等策略
- **自动故障转移**：毫秒级切换到可用实例
- **高性能连接**：连接复用，减少开销

### 基本用法

```go
// 创建协调器（自动注册 gRPC resolver）
coordinator, err := coord.New()

// 注册服务
service := registry.ServiceInfo{
    ID: "user-service-1", Name: "user-service",
    Address: "127.0.0.1", Port: 8080,
}
coordinator.Registry().Register(ctx, service, 30*time.Second)

// 创建 gRPC 连接（使用动态服务发现）
conn, err := coordinator.Registry().GetConnection(ctx, "user-service")
client := yourpb.NewYourServiceClient(conn)
```

### 高级配置

```go
// 自定义负载均衡策略
conn, err := grpc.DialContext(ctx, "etcd:///my-service",
    grpc.WithDefaultServiceConfig(`{
        "loadBalancingPolicy": "round_robin"
    }`),
)
```

## 最佳实践

1. **资源管理**: 总是使用 `defer coordinator.Close()` 确保连接关闭
2. **上下文管理**: 为阻塞操作提供带超时的 `context`
3. **gRPC 连接复用**: 创建一次连接后尽量复用
4. **监控服务健康**: 使用 `Watch` 方法监听服务变化
5. **错误处理**: 检查非阻塞操作的返回错误
