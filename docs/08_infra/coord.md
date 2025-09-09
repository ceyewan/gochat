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
    // Endpoints 是 etcd 集群的地址列表
    Endpoints []string `json:"endpoints"`
    // DialTimeout 是连接 etcd 的超时时间
    DialTimeout time.Duration `json:"dialTimeout"`
    // KeepAliveTime 是 keepalive 心跳间隔
    KeepAliveTime time.Duration `json:"keepAliveTime"`
    // KeepAliveTimeout 是 keepalive 超时时间
    KeepAliveTimeout time.Duration `json:"keepAliveTimeout"`
    // Username 是认证用户名，可选
    Username string `json:"username,omitempty"`
    // Password 是认证密码，可选
    Password string `json:"password,omitempty"`
    // TLS 相关配置，可选
    TLS *TLSConfig `json:"tls,omitempty"`
}

// TLSConfig 定义了 TLS 连接配置
type TLSConfig struct {
    CertFile string `json:"certFile,omitempty"`
    KeyFile  string `json:"keyFile,omitempty"`
    CAFile   string `json:"caFile,omitempty"`
}

// GetDefaultConfig 返回默认的 coord 配置。
func GetDefaultConfig(env string) *Config

// Option 定义了用于定制 coord Provider 的函数。
type Option func(*options)

// WithLogger 将一个 clog.Logger 实例注入 coord，用于记录内部日志。
func WithLogger(logger clog.Logger) Option

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
    // 此方法是可重入的：为同一个 serviceName 多次调用，将返回同一个共享的分配器实例。
    InstanceIDAllocator(serviceName string, maxID int) (InstanceIDAllocator, error)

	// Close 关闭与 etcd 的连接并释放所有资源，包括所有持有的锁和实例ID。
	Close() error
}
```

### 2.3 `ConfigCenter` 接口 (重点)

`ConfigCenter` 提供了对配置的读、写和监听能力。

```go
// ConfigCenter 定义了配置中心的核心操作。
type ConfigCenter interface {
	// Get 获取指定 key 的配置，并将其反序列化到 v (指针类型) 中。
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
    // 设计注记:
    // - 返回的 Event.Value 是原始的 []byte，需要使用者自行反序列化。
    //   这是为了在反序列化失败时不丢失事件通知，并将错误处理的灵活性交给调用方。
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
    // Value 是变更后的值的原始字节。对于 DELETE 事件，Value 为 nil。
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
// 详细工作原理请参见文末的“设计注记”。
type InstanceIDAllocator interface {
    // AcquireID 尝试获取一个未被使用的 ID。
    // ctx 用于控制本次获取操作的超时。
    // 返回的 AllocatedID 对象代表一个被成功占用的、会自动续租的 ID。
    AcquireID(ctx context.Context) (AllocatedID, error)
}

// AllocatedID 代表一个被当前服务实例持有的、会自动续租的 ID。
type AllocatedID interface {
    // ID 返回被分配的整数 ID。
    ID() int
    // Close 主动释放当前持有的 ID。这是一个幂等操作。
    // 如果不调用此方法，ID 将在服务实例关闭时通过 etcd 的租约机制自动释放。
    // ctx 用于控制本次释放操作的超时。
    Close(ctx context.Context) error
}
```

## 3. 标准用法

### ... (场景 1-4 保持不变) ...

### 场景 5: 实例 ID 分配器

```go
func (s *UserService) AssignInstanceID(ctx context.Context) (int, error) {
    allocator, err := s.coordProvider.InstanceIDAllocator("user-service", 1024)
    if err != nil {
        return -1, fmt.Errorf("获取实例ID分配器失败: %w", err)
    }
    
    // 获取一个 ID
    allocatedID, err := allocator.AcquireID(ctx)
    if err != nil {
        return -1, fmt.Errorf("分配实例ID失败: %w", err)
    }
    
    // 使用 defer 确保在函数退出时主动释放 ID，这是一个好习惯。
    // 注意：在 defer 中应使用独立的 context，以确保即使原始 ctx 已被取消，释放操作也能被尝试执行。
    defer func() {
        if err := allocatedID.Close(context.Background()); err != nil {
            s.logger.Error("释放实例ID失败", clog.Err(err))
        }
    }()
    
    instanceID := allocatedID.ID()
    s.logger.Info("成功分配实例ID", clog.Int("id", instanceID))
    
    // ... 使用该 ID 执行业务逻辑 ...
    
    return instanceID, nil
}
```

## 4. 设计注记

### 4.1 GetDefaultConfig 默认值说明
... (保持不变) ...

### 4.2 Watch 接口的前缀监听约定
... (保持不变) ...

### 4.3 组件自治的动态配置模式
... (保持不变) ...

### 4.4 InstanceIDAllocator 工作原理

`InstanceIDAllocator` 是一个基于 `etcd` 的租约（Lease）和临时节点（Ephemeral Node）实现的、高可用的唯一 ID 分配器。

-   **核心机制**:
    1.  **会话 (Session)**: 当首次为某个 `serviceName` 创建分配器时，它会与 `etcd` 建立一个会话。此会话包含一个租约，并由 `coord` 组件在后台自动续租（KeepAlive）。
    2.  **ID 获取**: 调用 `AcquireID` 时，分配器会在 `etcd` 的一个公共目录下（如 `/im-infra/allocators/user-service/ids/`）尝试以事务的方式创建一个以 ID 为名的临时节点（如 `.../5`）。此节点与上述会话的租约绑定。由于事务的原子性，只有一个实例能成功创建节点，从而成功获取该 ID。
    3.  **自动回收**:
        -   **正常关闭**: 当服务实例正常关闭并调用 `coordProvider.Close()` 时，etcd 会话被关闭，租约失效，所有与该租约绑定的临时节点（即所有持有的 ID）都会被 `etcd` 自动删除。
        -   **异常崩溃**: 当服务实例崩溃或与 `etcd` 网络中断时，`coord` 组件无法再为租约续期。当租约达到 TTL 后，`etcd` 会自动使其失效并删除所有关联的临时节点。
-   **API 行为**:
    -   `AllocatedID.Close()`: 此方法会主动删除 `etcd` 中对应的临时节点，用于提前释放不再需要的 ID。
    -   **可重入性**: `Provider.InstanceIDAllocator()` 是可重入的。在同一个 `coordProvider` 实例中，为相同的 `serviceName` 多次调用此方法，将返回同一个共享的、底层的分配器实例，不会产生额外的资源开销。