# 基础设施: Coord 分布式协调

## 1. 设计理念

`coord` 是 `gochat` 项目的分布式协调核心，基于 `etcd` 构建。它为整个微服务集群提供了一个统一的、可靠的协调层，封装了服务发现、分布式锁和动态配置管理等复杂性。

`coord` 的设计严格遵循 `im-infra` 的核心规范，旨在成为一个稳定、可预测且易于依赖的基础服务。

## 2. 核心 API 契约

`coord` 通过一个统一的 `Provider` 接口暴露其所有能力。

### 2.1 构造函数

```go
// Config 是 coord 组件的配置结构体。
type Config struct {
    // Endpoints 是 etcd 集群的地址列表。
    Endpoints []string `json:"endpoints"`
    // DialTimeout 是连接 etcd 的超时时间。
    DialTimeout time.Duration `json:"dialTimeout"`
    // ... 其他 etcd 相关配置
}

// New 创建一个新的 coord Provider 实例。
// 这是与 coord 组件交互的唯一入口。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Provider 接口

`Provider` 接口是所有协调服务的总入口，它通过功能将不同的职责分离到独立的子接口中。

```go
// Provider 定义了 coord 组件提供的所有能力。
type Provider interface {
	// Registry 返回服务注册与发现的客户端。
	Registry() ServiceRegistry
	// Config 返回配置中心的客户端。
	Config() ConfigCenter
	// Lock 返回分布式锁的客户端。
	Lock() DistributedLock
    // InstanceIDAllocator 获取一个服务实例ID分配器。
    InstanceIDAllocator(serviceName string, maxID int) (InstanceIDAllocator, error)

	// Close 关闭与 etcd 的连接并释放所有资源。
	Close() error
}
```

### 2.3 `ConfigCenter` 接口 (重点)

`ConfigCenter` 提供了对配置的读、写和监听能力。

```go
// ConfigCenter 定义了配置中心的核心操作。
type ConfigCenter interface {
	// Get 获取指定 key 的配置，并将其反序列化到 v 中。
	Get(ctx context.Context, key string, v interface{}) error

	// List 返回指定前缀下的所有 key。
	List(ctx context.Context, prefix string) ([]string, error)

	// Set 将 v 序列化为 JSON 并写入指定的 key。
	Set(ctx context.Context, key string, v interface{}) error

	// Watch 监听一个键或一个前缀的变更。
	//
	// 行为约定:
	// - 如果 key 不以 "/" 结尾，则只监听该单个键的变更。
	// - 如果 key 以 "/" 结尾，则监听该前缀下的所有键的变更 (WatchPrefix 行为)。
	//
	// 返回的 Watcher 会在后台自动处理重连和错误。
	Watch(ctx context.Context, key string) (Watcher, error)
}

// Watcher 定义了配置监听器。
type Watcher interface {
    // Chan 返回一个只读通道，用于接收配置变更事件。
    Chan() <-chan Event
    // Close 关闭监听器。
    Close()
}

// Event 代表一次配置变更。
type Event struct {
    Type EventType // PUT 或 DELETE
    Key  string
    // Value 是变更后的值，已反序列化。
    // 对于 DELETE 事件，Value 为 nil。
    Value []byte
}
```

### 2.4 其他核心接口

```go
// ServiceRegistry 定义了服务注册与发现的操作。
type ServiceRegistry interface {
	Register(ctx context.Context, service ServiceInfo, ttl time.Duration) error
	Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error)
	// ...
}

// DistributedLock 定义了分布式锁的操作。
type DistributedLock interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (Locker, error)
	// ...
}

// InstanceIDAllocator 为一类服务的实例分配唯一的、可自动回收的ID。
type InstanceIDAllocator interface {
    AcquireID(ctx context.Context) (int, error)
    ReleaseID(ctx context.Context) error
    // ...
}
```

## 3. 标准用法

### 场景：实现 `ratelimit` 的动态配置

`ratelimit` 组件在其 `New` 函数中，将使用 `coord.Config()` 来实现其“组件自治”的动态配置。

```go
// 在 ratelimit 的 New 函数内部
func New(ctx context.Context, config *Config, opts ...Option) (RateLimiter, error) {
    // ... 初始化 limiter 实例，并从 opts 中获取 coord.Provider ...

    // 1. 初始加载全量规则
    keys, err := coordProvider.Config().List(ctx, config.RulesPath)
    if err != nil {
        // ... handle error
    }
    for _, key := range keys {
        // ... coordProvider.Config().Get(ctx, key, &rule) ...
    }

    // 2. 启动后台 goroutine 监听配置变更
    go func() {
        // config.RulesPath 必须以 "/" 结尾，例如 "/config/dev/myservice/ratelimit/"
        watcher, err := coordProvider.Config().Watch(ctx, config.RulesPath)
        if err != nil {
            limiter.logger.Error("无法监听配置变更", clog.Err(err))
            return
        }
        defer watcher.Close()

        for {
            select {
            case <-ctx.Done():
                return
            case event := <-watcher.Chan():
                // 检测到变更，重新全量加载所有规则
                limiter.logger.Info("检测到规则变更，重新加载...", clog.String("key", event.Key))
                if err := limiter.reloadAllRules(); err != nil {
                    limiter.logger.Error("重新加载规则失败", clog.Err(err))
                }
            }
        }
    }()

    return limiter, nil
}