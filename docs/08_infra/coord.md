# 基础设施: Coord 分布式协调

## 1. 概述

`coord` 是 `gochat` 项目的分布式协调核心，基于 `etcd` 构建。它为整个系统提供三大基础能力：

1.  **分布式锁 (Distributed Lock)**: 保证在分布式环境下对共享资源的互斥访问。
2.  **服务注册与发现 (Service Registry & Discovery)**: 让各个微服务能够动态地找到彼此。
3.  **配置中心 (Configuration Center)**: 提供一个统一的地方来管理和分发服务配置。

`coord` 组件设计精良，实现健壮，是支撑 `gochat` 微服务架构的基石。

## 2. 核心用法

### 2.1 初始化

所有 `coord` 的功能都通过一个统一的 `Provider` 接口暴露。

```go
import "github.com/ceyewan/gochat/im-infra/coord"

// 创建协调器，连接到默认的 etcd 地址
// 在真实场景中，应通过 Option 注入日志器
coordinator, err := coord.New(context.Background(), coord.DefaultConfig(), coord.WithLogger(logger))
if err != nil {
    log.Fatal(err)
}
defer coordinator.Close()
```

### 2.2 分布式锁

用于需要保证操作唯一性的场景，例如定时任务、资源初始化等。

```go
// 获取分布式锁服务
lockService := coordinator.Lock()

// 1. 阻塞式获取锁，最长等待 ctx 的超时时间
// "my-unique-task" 是锁的名称，30s 是锁的租期 (TTL)
lock, err := lockService.Acquire(ctx, "my-unique-task", 30*time.Second)
if err != nil {
    // 获取锁失败
    return
}
// 确保在操作完成后释放锁
defer lock.Unlock(ctx)

// --- 在这里执行你的临界区代码 ---

// 2. 非阻塞式获取锁
// 如果锁已被占用，会立即返回错误
lock, err = lockService.TryAcquire(ctx, "another-task", 30*time.Second)
if err != nil {
    logger.Info("获取锁失败，可能已有其他实例在执行任务")
    return
}
defer lock.Unlock(ctx)
```

### 2.3 服务注册与发现

每个微服务在启动时，都需要向 `coord` 注册自己，并在关闭时注销。

```go
// 获取服务注册发现服务
registryService := coordinator.Registry()

// 1. 注册服务
serviceInfo := registry.ServiceInfo{
    ID:       "im-logic-instance-01", // 服务实例的唯一ID
    Name:     "im-logic",             // 服务名
    Address:  "10.0.1.10",            // 实例地址
    Port:     8080,
    Metadata: map[string]string{"region": "us-east-1"},
}
// 注册并设置 30s 的租期，服务需要在此期间保持心跳（由coord自动处理）
err := registryService.Register(ctx, serviceInfo, 30*time.Second)

// 2. 发现服务
// 客户端可以通过服务名发现所有可用的实例
instances, err := registryService.Discover(ctx, "im-logic")
for _, inst := range instances {
    fmt.Printf("发现服务实例: %s -> %s:%d\n", inst.ID, inst.Address, inst.Port)
}

// 3. gRPC 客户端直连 (推荐)
// coord 提供了 gRPC resolver，可以透明地处理服务发现和负载均衡
conn, err := registryService.GetConnection(ctx, "im-logic")
if err != nil {
    // ...
}
logicClient := im_logic_v1.NewMessageServiceClient(conn)
```

### 2.4 配置中心

用于动态获取服务的配置。

```go
// 获取配置中心服务
configService := coordinator.Config()

// 1. 获取配置
var mqConfig mq.Config
// "common/mq" 是配置在 etcd 中的 key
err := configService.Get(ctx, "common/mq", &mqConfig)
if err != nil {
    // ...
}

// 2. 监听配置变更
watcher, err := configService.Watch(ctx, "common/mq", &mqConfig)
go func() {
    defer watcher.Close()
    for event := range watcher.Chan() {
        // 配置已自动更新到 mqConfig 变量中
        logger.Info("MQ 配置已更新", clog.Any("new_config", event.Value))
    }
}()
```

## 3. API 参考

`coord` 提供的核心接口如下：

```go
// Provider 是 coord 的主入口
type Provider interface {
	Lock() lock.DistributedLock
	Registry() registry.ServiceRegistry
	Config() config.ConfigCenter
    // InstanceIDAllocator 获取一个服务实例ID分配器
    InstanceIDAllocator(serviceName string, maxID int) (InstanceIDAllocator, error)
	Close() error
}

// InstanceIDAllocator 为一类服务的实例分配唯一的、可自动回收的ID。
// 例如 im-gateway, im-logic 等都需要唯一的 workerID/instanceID。
type InstanceIDAllocator interface {
    // AcquireID 获取一个唯一的实例ID。此方法会阻塞直到成功获取或上下文超时。
    // ID的范围是 [1, maxID)。
    AcquireID(ctx context.Context) (int, error)
    // GetID 返回当前实例已获取的ID。如果还未获取，返回0。
    GetID() int
    // ReleaseID 释放当前实例持有的ID。通常在服务正常关闭时调用。
    ReleaseID(ctx context.Context) error
}

// DistributedLock 分布式锁接口
type DistributedLock interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}

// ServiceRegistry 服务注册发现接口
type ServiceRegistry interface {
	Register(ctx context.Context, service ServiceInfo, ttl time.Duration) error
	Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error)
	Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)
	GetConnection(ctx context.Context, serviceName string) (*grpc.ClientConn, error)
}

// ConfigCenter 配置中心接口
type ConfigCenter interface {
	Get(ctx context.Context, key string, v interface{}) error
	Set(ctx context.Context, key string, value interface{}) error
	Watch(ctx context.Context, key string, v interface{}) (Watcher[any], error)
}
```

### 3.1 `InstanceIDAllocator` 使用示例

任何需要唯一 WorkerID 的服务（如 `im-gateway` 或 `im-logic`）都应在启动时使用此功能。

```go
// 1. 获取特定服务的ID分配器
// "im-gateway" 是服务名，1023 是ID上限
idAllocator, err := coordinator.InstanceIDAllocator("im-gateway", 1023)
if err != nil {
    log.Fatalf("无法创建ID分配器: %v", err)
}

// 2. 获取唯一ID
// 此过程是阻塞和自动重试的，直到成功获取ID
instanceID, err := idAllocator.AcquireID(context.Background())
if err != nil {
    log.Fatalf("无法获取实例ID: %v", err)
}
logger.Info("成功获取实例ID", clog.Int("instanceID", instanceID))

// 3. 在服务关闭时，确保释放ID
defer func() {
    if err := idAllocator.ReleaseID(context.Background()); err != nil {
        logger.Error("释放实例ID失败", clog.Err(err))
    }
}()

// 4. 使用获取到的 instanceID
// 例如，构建 Kafka Topic
kafkaTopic := fmt.Sprintf("gochat.messages.downstream.%d", instanceID)
// 或者，初始化雪花算法生成器
snowflakeNode, err := snowflake.NewNode(int64(instanceID))
```