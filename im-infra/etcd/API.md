# etcd-grpc API 文档

本文档提供了 etcd-grpc 组件的完整 API 参考，包括所有公共接口、方法、参数和使用示例。

## 目录

- [核心接口](#核心接口)
- [管理器接口](#管理器接口)
- [服务注册接口](#服务注册接口)
- [服务发现接口](#服务发现接口)
- [分布式锁接口](#分布式锁接口)
- [租约管理接口](#租约管理接口)
- [连接管理接口](#连接管理接口)
- [配置选项](#配置选项)
- [错误处理](#错误处理)
- [工厂方法](#工厂方法)

## 核心接口

### EtcdManager

顶层管理接口，统一管理所有功能组件。

```go
type EtcdManager interface {
    io.Closer
    
    // 获取各种功能组件
    ServiceRegistry() ServiceRegistry
    ServiceDiscovery() ServiceDiscovery
    DistributedLock() DistributedLock
    LeaseManager() LeaseManager
    ConnectionManager() ConnectionManager
    
    // 健康检查和状态管理
    HealthCheck(ctx context.Context) error
    IsReady() bool
}
```

#### 方法说明

**ServiceRegistry() ServiceRegistry**
- 返回服务注册组件实例
- 用于注册和管理服务实例

**ServiceDiscovery() ServiceDiscovery**
- 返回服务发现组件实例
- 用于发现和连接服务

**DistributedLock() DistributedLock**
- 返回分布式锁组件实例
- 用于分布式锁操作

**LeaseManager() LeaseManager**
- 返回租约管理组件实例
- 用于管理 etcd 租约

**ConnectionManager() ConnectionManager**
- 返回连接管理组件实例
- 用于管理 etcd 连接

**HealthCheck(ctx context.Context) error**
- 执行健康检查
- 参数：
  - `ctx`: 上下文，用于超时控制
- 返回：错误信息，nil 表示健康

**IsReady() bool**
- 检查管理器是否就绪
- 返回：true 表示就绪，false 表示未就绪

**Close() error**
- 关闭管理器，释放所有资源
- 返回：错误信息，nil 表示成功

#### 使用示例

```go
// 创建管理器
manager, err := etcd.QuickStart("localhost:2379")
if err != nil {
    log.Fatalf("Failed to create manager: %v", err)
}
defer manager.Close()

// 检查状态
if !manager.IsReady() {
    log.Fatal("Manager not ready")
}

// 执行健康检查
ctx := context.Background()
if err := manager.HealthCheck(ctx); err != nil {
    log.Printf("Health check failed: %v", err)
}

// 获取各种组件
registry := manager.ServiceRegistry()
discovery := manager.ServiceDiscovery()
lock := manager.DistributedLock()
```

## 管理器接口

### ManagerBuilder

建造者模式配置管理器。

```go
type ManagerBuilder struct {
    // 私有字段
}
```

#### 方法说明

**NewManagerBuilder() *ManagerBuilder**
- 创建新的管理器建造者
- 返回：建造者实例

**WithEndpoints(endpoints []string) *ManagerBuilder**
- 设置 etcd 服务器端点
- 参数：
  - `endpoints`: etcd 服务器地址列表
- 返回：建造者实例（支持链式调用）

**WithDialTimeout(timeout time.Duration) *ManagerBuilder**
- 设置连接超时时间
- 参数：
  - `timeout`: 超时时间
- 返回：建造者实例

**WithAuth(username, password string) *ManagerBuilder**
- 设置认证信息
- 参数：
  - `username`: 用户名
  - `password`: 密码
- 返回：建造者实例

**WithTLS(config *TLSConfig) *ManagerBuilder**
- 设置 TLS 配置
- 参数：
  - `config`: TLS 配置
- 返回：建造者实例

**WithDefaultTTL(ttl int64) *ManagerBuilder**
- 设置默认 TTL（秒）
- 参数：
  - `ttl`: 生存时间
- 返回：建造者实例

**WithServicePrefix(prefix string) *ManagerBuilder**
- 设置服务注册前缀
- 参数：
  - `prefix`: 前缀路径
- 返回：建造者实例

**WithLockPrefix(prefix string) *ManagerBuilder**
- 设置分布式锁前缀
- 参数：
  - `prefix`: 前缀路径
- 返回：建造者实例

**WithHealthCheck(interval, timeout time.Duration) *ManagerBuilder**
- 设置健康检查配置
- 参数：
  - `interval`: 检查间隔
  - `timeout`: 检查超时
- 返回：建造者实例

**WithRetryConfig(config *RetryConfig) *ManagerBuilder**
- 设置重试配置
- 参数：
  - `config`: 重试配置
- 返回：建造者实例

**Build() (EtcdManager, error)**
- 构建管理器实例
- 返回：管理器实例和错误信息

#### 使用示例

```go
manager, err := etcd.NewManagerBuilder().
    WithEndpoints([]string{"etcd1:2379", "etcd2:2379", "etcd3:2379"}).
    WithDialTimeout(10 * time.Second).
    WithDefaultTTL(60).
    WithServicePrefix("/prod/services").
    WithLockPrefix("/prod/locks").
    WithHealthCheck(30*time.Second, 5*time.Second).
    WithAuth("username", "password").
    WithRetryConfig(&etcd.RetryConfig{
        MaxRetries:      5,
        InitialInterval: 200 * time.Millisecond,
        MaxInterval:     5 * time.Second,
        Multiplier:      2.0,
    }).
    Build()
```

## 服务注册接口

### ServiceRegistry

服务注册和管理接口。

```go
type ServiceRegistry interface {
    Register(ctx context.Context, serviceName, instanceID, addr string, options ...RegisterOption) error
    Deregister(ctx context.Context, serviceName, instanceID string) error
    UpdateService(ctx context.Context, serviceName, instanceID, addr string) error
    ListServices(ctx context.Context) ([]ServiceInfo, error)
    GetServiceInstances(ctx context.Context, serviceName string) ([]ServiceInstance, error)
}
```

#### 方法说明

**Register(ctx context.Context, serviceName, instanceID, addr string, options ...RegisterOption) error**
- 注册服务实例
- 参数：
  - `ctx`: 上下文
  - `serviceName`: 服务名称
  - `instanceID`: 实例ID
  - `addr`: 服务地址
  - `options`: 注册选项
- 返回：错误信息

**Deregister(ctx context.Context, serviceName, instanceID string) error**
- 注销服务实例
- 参数：
  - `ctx`: 上下文
  - `serviceName`: 服务名称
  - `instanceID`: 实例ID
- 返回：错误信息

**UpdateService(ctx context.Context, serviceName, instanceID, addr string) error**
- 更新服务信息
- 参数：
  - `ctx`: 上下文
  - `serviceName`: 服务名称
  - `instanceID`: 实例ID
  - `addr`: 新的服务地址
- 返回：错误信息

**ListServices(ctx context.Context) ([]ServiceInfo, error)**
- 列出所有已注册的服务
- 参数：
  - `ctx`: 上下文
- 返回：服务信息列表和错误信息

**GetServiceInstances(ctx context.Context, serviceName string) ([]ServiceInstance, error)**
- 获取指定服务的所有实例
- 参数：
  - `ctx`: 上下文
  - `serviceName`: 服务名称
- 返回：服务实例列表和错误信息

#### 注册选项

**WithTTL(ttl int64) RegisterOption**
- 设置服务 TTL
- 参数：
  - `ttl`: 生存时间（秒）

**WithMetadata(metadata map[string]string) RegisterOption**
- 设置服务元数据
- 参数：
  - `metadata`: 元数据键值对

**WithLeaseID(leaseID clientv3.LeaseID) RegisterOption**
- 使用指定的租约ID
- 参数：
  - `leaseID`: 租约ID

#### 使用示例

```go
registry := manager.ServiceRegistry()

// 基本注册
err := registry.Register(ctx, "user-service", "instance-1", "localhost:50051")

// 带选项的注册
err = registry.Register(ctx, "user-service", "instance-2", "localhost:50052",
    etcd.WithTTL(60),
    etcd.WithMetadata(map[string]string{
        "version": "1.2.0",
        "region":  "us-west",
        "env":     "production",
    }),
)

// 列出所有服务
services, err := registry.ListServices(ctx)
for _, service := range services {
    fmt.Printf("Service: %s, Instances: %d\n", service.Name, len(service.Instances))
}

// 获取特定服务的实例
instances, err := registry.GetServiceInstances(ctx, "user-service")
for _, instance := range instances {
    fmt.Printf("Instance: %s at %s\n", instance.ID, instance.Address)
}

// 注销服务
defer registry.Deregister(ctx, "user-service", "instance-1")
```

## 服务发现接口

### ServiceDiscovery

服务发现和连接管理接口。

```go
type ServiceDiscovery interface {
    GetConnection(ctx context.Context, serviceName string, options ...DiscoveryOption) (*grpc.ClientConn, error)
    GetServiceEndpoints(ctx context.Context, serviceName string) ([]string, error)
    WatchService(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)
    ResolveService(ctx context.Context, serviceName string) ([]ServiceInstance, error)
}
```

#### 方法说明

**GetConnection(ctx context.Context, serviceName string, options ...DiscoveryOption) (*grpc.ClientConn, error)**
- 获取服务的 gRPC 连接
- 参数：
  - `ctx`: 上下文
  - `serviceName`: 服务名称
  - `options`: 发现选项
- 返回：gRPC 连接和错误信息

**GetServiceEndpoints(ctx context.Context, serviceName string) ([]string, error)**
- 获取服务的所有端点地址
- 参数：
  - `ctx`: 上下文
  - `serviceName`: 服务名称
- 返回：端点地址列表和错误信息

**WatchService(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)**
- 监听服务变化事件
- 参数：
  - `ctx`: 上下文
  - `serviceName`: 服务名称
- 返回：事件通道和错误信息

**ResolveService(ctx context.Context, serviceName string) ([]ServiceInstance, error)**
- 解析服务的所有实例
- 参数：
  - `ctx`: 上下文
  - `serviceName`: 服务名称
- 返回：服务实例列表和错误信息

#### 发现选项

**WithLoadBalancer(lb string) DiscoveryOption**
- 设置负载均衡策略
- 参数：
  - `lb`: 负载均衡策略（"round_robin", "random", "pick_first"）

**WithDiscoveryTimeout(timeout time.Duration) DiscoveryOption**
- 设置发现超时时间
- 参数：
  - `timeout`: 超时时间

**WithDiscoveryMetadata(metadata map[string]string) DiscoveryOption**
- 设置元数据过滤条件
- 参数：
  - `metadata`: 过滤条件

#### 使用示例

```go
discovery := manager.ServiceDiscovery()

// 获取 gRPC 连接
conn, err := discovery.GetConnection(ctx, "user-service",
    etcd.WithLoadBalancer("round_robin"),
    etcd.WithDiscoveryTimeout(5*time.Second),
)
if err != nil {
    log.Fatalf("Failed to get connection: %v", err)
}
defer conn.Close()

// 创建客户端并调用服务
client := pb.NewUserServiceClient(conn)
response, err := client.GetUser(ctx, &pb.GetUserRequest{Id: "123"})

// 获取服务端点
endpoints, err := discovery.GetServiceEndpoints(ctx, "user-service")
for _, endpoint := range endpoints {
    fmt.Printf("Available endpoint: %s\n", endpoint)
}

// 监听服务变化
eventCh, err := discovery.WatchService(ctx, "user-service")
if err != nil {
    log.Fatalf("Failed to watch service: %v", err)
}

go func() {
    for event := range eventCh {
        switch event.Type {
        case etcd.ServiceEventAdd:
            fmt.Printf("Service instance added: %s\n", event.Instance.Address)
        case etcd.ServiceEventDelete:
            fmt.Printf("Service instance removed: %s\n", event.Instance.Address)
        case etcd.ServiceEventUpdate:
            fmt.Printf("Service instance updated: %s\n", event.Instance.Address)
        }
    }
}()

// 解析服务实例
instances, err := discovery.ResolveService(ctx, "user-service")
for _, instance := range instances {
    fmt.Printf("Instance: %s at %s, metadata: %v\n",
        instance.ID, instance.Address, instance.Metadata)
}
```

## 分布式锁接口

### DistributedLock

分布式锁管理接口。

```go
type DistributedLock interface {
    Lock(ctx context.Context, key string, ttl time.Duration) error
    TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
    Unlock(ctx context.Context, key string) error
    Refresh(ctx context.Context, key string, ttl time.Duration) error
    IsLocked(ctx context.Context, key string) (bool, error)
    GetLockInfo(ctx context.Context, key string) (*LockInfo, error)
}
```

#### 方法说明

**Lock(ctx context.Context, key string, ttl time.Duration) error**
- 获取分布式锁（阻塞）
- 参数：
  - `ctx`: 上下文
  - `key`: 锁的键名
  - `ttl`: 锁的生存时间
- 返回：错误信息

**TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)**
- 尝试获取分布式锁（非阻塞）
- 参数：
  - `ctx`: 上下文
  - `key`: 锁的键名
  - `ttl`: 锁的生存时间
- 返回：是否成功获取锁和错误信息

**Unlock(ctx context.Context, key string) error**
- 释放分布式锁
- 参数：
  - `ctx`: 上下文
  - `key`: 锁的键名
- 返回：错误信息

**Refresh(ctx context.Context, key string, ttl time.Duration) error**
- 刷新锁的 TTL
- 参数：
  - `ctx`: 上下文
  - `key`: 锁的键名
  - `ttl`: 新的生存时间
- 返回：错误信息

**IsLocked(ctx context.Context, key string) (bool, error)**
- 检查锁是否被持有
- 参数：
  - `ctx`: 上下文
  - `key`: 锁的键名
- 返回：是否被锁定和错误信息

**GetLockInfo(ctx context.Context, key string) (*LockInfo, error)**
- 获取锁的详细信息
- 参数：
  - `ctx`: 上下文
  - `key`: 锁的键名
- 返回：锁信息和错误信息

#### 使用示例

```go
lock := manager.DistributedLock()

// 获取锁（阻塞）
err := lock.Lock(ctx, "my-critical-section", 30*time.Second)
if err != nil {
    log.Fatalf("Failed to acquire lock: %v", err)
}

// 执行临界区代码
performCriticalOperation()

// 释放锁
defer lock.Unlock(ctx, "my-critical-section")

// 尝试获取锁（非阻塞）
acquired, err := lock.TryLock(ctx, "another-lock", 30*time.Second)
if err != nil {
    log.Fatalf("Error trying to acquire lock: %v", err)
}
if acquired {
    defer lock.Unlock(ctx, "another-lock")
    // 执行需要锁保护的操作
} else {
    fmt.Println("Lock is held by another process")
}

// 检查锁状态
locked, err := lock.IsLocked(ctx, "my-lock")
if err != nil {
    log.Printf("Failed to check lock status: %v", err)
} else if locked {
    fmt.Println("Lock is currently held")
}

// 获取锁信息
lockInfo, err := lock.GetLockInfo(ctx, "my-lock")
if err != nil {
    log.Printf("Failed to get lock info: %v", err)
} else {
    fmt.Printf("Lock owner: %s, TTL: %d\n", lockInfo.Owner, lockInfo.TTL)
}

// 刷新锁的 TTL
err = lock.Refresh(ctx, "my-lock", 60*time.Second)
if err != nil {
    log.Printf("Failed to refresh lock: %v", err)
}
```

## 租约管理接口

### LeaseManager

租约管理接口。

```go
type LeaseManager interface {
    CreateLease(ctx context.Context, ttl int64) (clientv3.LeaseID, error)
    KeepAlive(ctx context.Context, leaseID clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error)
    RevokeLease(ctx context.Context, leaseID clientv3.LeaseID) error
    GetLeaseInfo(ctx context.Context, leaseID clientv3.LeaseID) (*clientv3.LeaseTimeToLiveResponse, error)
    ListLeases(ctx context.Context) ([]clientv3.LeaseStatus, error)
    RefreshLease(ctx context.Context, leaseID clientv3.LeaseID, ttl int64) error
}
```

#### 方法说明

**CreateLease(ctx context.Context, ttl int64) (clientv3.LeaseID, error)**
- 创建新的租约
- 参数：
  - `ctx`: 上下文
  - `ttl`: 租约生存时间（秒）
- 返回：租约ID和错误信息

**KeepAlive(ctx context.Context, leaseID clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error)**
- 启动租约保活
- 参数：
  - `ctx`: 上下文
  - `leaseID`: 租约ID
- 返回：保活响应通道和错误信息

**RevokeLease(ctx context.Context, leaseID clientv3.LeaseID) error**
- 撤销租约
- 参数：
  - `ctx`: 上下文
  - `leaseID`: 租约ID
- 返回：错误信息

**GetLeaseInfo(ctx context.Context, leaseID clientv3.LeaseID) (*clientv3.LeaseTimeToLiveResponse, error)**
- 获取租约信息
- 参数：
  - `ctx`: 上下文
  - `leaseID`: 租约ID
- 返回：租约信息和错误信息

**ListLeases(ctx context.Context) ([]clientv3.LeaseStatus, error)**
- 列出所有租约
- 参数：
  - `ctx`: 上下文
- 返回：租约状态列表和错误信息

**RefreshLease(ctx context.Context, leaseID clientv3.LeaseID, ttl int64) error**
- 刷新租约 TTL
- 参数：
  - `ctx`: 上下文
  - `leaseID`: 租约ID
  - `ttl`: 新的生存时间
- 返回：错误信息

#### 使用示例

```go
leaseMgr := manager.LeaseManager()

// 创建租约
leaseID, err := leaseMgr.CreateLease(ctx, 60) // 60秒TTL
if err != nil {
    log.Fatalf("Failed to create lease: %v", err)
}
fmt.Printf("Created lease: %d\n", leaseID)

// 启动租约保活
keepAliveCh, err := leaseMgr.KeepAlive(ctx, leaseID)
if err != nil {
    log.Fatalf("Failed to start keepalive: %v", err)
}

// 监听保活响应
go func() {
    for resp := range keepAliveCh {
        if resp != nil {
            fmt.Printf("Lease %d renewed, TTL: %d\n", resp.ID, resp.TTL)
        }
    }
}()

// 获取租约信息
info, err := leaseMgr.GetLeaseInfo(ctx, leaseID)
if err != nil {
    log.Printf("Failed to get lease info: %v", err)
} else {
    fmt.Printf("Lease TTL: %d seconds\n", info.TTL)
}

// 列出所有租约
leases, err := leaseMgr.ListLeases(ctx)
if err != nil {
    log.Printf("Failed to list leases: %v", err)
} else {
    fmt.Printf("Total leases: %d\n", len(leases))
}

// 刷新租约 TTL
err = leaseMgr.RefreshLease(ctx, leaseID, 120) // 延长到120秒
if err != nil {
    log.Printf("Failed to refresh lease: %v", err)
}

// 撤销租约
defer leaseMgr.RevokeLease(ctx, leaseID)
```

## 连接管理接口

### ConnectionManager

连接管理接口。

```go
type ConnectionManager interface {
    io.Closer

    Connect(ctx context.Context) error
    Disconnect() error
    IsConnected() bool
    HealthCheck(ctx context.Context) error
    GetClient() *clientv3.Client
    Reconnect(ctx context.Context) error
    GetConnectionStatus() ConnectionStatus
}
```

#### 方法说明

**Connect(ctx context.Context) error**
- 建立到 etcd 的连接
- 参数：
  - `ctx`: 上下文
- 返回：错误信息

**Disconnect() error**
- 断开 etcd 连接
- 返回：错误信息

**IsConnected() bool**
- 检查连接状态
- 返回：是否已连接

**HealthCheck(ctx context.Context) error**
- 执行连接健康检查
- 参数：
  - `ctx`: 上下文
- 返回：错误信息

**GetClient() *clientv3.Client**
- 获取底层 etcd 客户端（内部使用）
- 返回：etcd 客户端实例

**Reconnect(ctx context.Context) error**
- 重新连接到 etcd
- 参数：
  - `ctx`: 上下文
- 返回：错误信息

**GetConnectionStatus() ConnectionStatus**
- 获取连接状态信息
- 返回：连接状态

**Close() error**
- 关闭连接管理器
- 返回：错误信息

#### 使用示例

```go
connMgr := manager.ConnectionManager()

// 检查连接状态
if !connMgr.IsConnected() {
    // 建立连接
    err := connMgr.Connect(ctx)
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
}

// 执行健康检查
err := connMgr.HealthCheck(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)

    // 尝试重连
    err = connMgr.Reconnect(ctx)
    if err != nil {
        log.Printf("Reconnect failed: %v", err)
    }
}

// 获取连接状态
status := connMgr.GetConnectionStatus()
fmt.Printf("Connected: %v, Endpoint: %s, Last Ping: %v\n",
    status.Connected, status.Endpoint, status.LastPing)

// 关闭连接
defer connMgr.Close()
```

## 配置选项

### ManagerOptions

管理器配置选项结构。

```go
type ManagerOptions struct {
    // etcd 连接配置
    Endpoints   []string      `json:"endpoints"`
    DialTimeout time.Duration `json:"dial_timeout"`
    Username    string        `json:"username,omitempty"`
    Password    string        `json:"password,omitempty"`

    // TLS 配置
    TLSConfig *TLSConfig `json:"tls_config,omitempty"`

    // 日志配置
    Logger Logger `json:"-"`

    // 重试配置
    RetryConfig *RetryConfig `json:"retry_config,omitempty"`

    // 服务注册默认配置
    DefaultTTL      int64             `json:"default_ttl"`
    ServicePrefix   string            `json:"service_prefix"`
    LockPrefix      string            `json:"lock_prefix"`
    DefaultMetadata map[string]string `json:"default_metadata,omitempty"`

    // 连接池配置
    MaxIdleConns    int           `json:"max_idle_conns"`
    MaxActiveConns  int           `json:"max_active_conns"`
    ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`

    // 健康检查配置
    HealthCheckInterval time.Duration `json:"health_check_interval"`
    HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
}
```

### TLSConfig

TLS 安全配置。

```go
type TLSConfig struct {
    CertFile   string `json:"cert_file,omitempty"`
    KeyFile    string `json:"key_file,omitempty"`
    CAFile     string `json:"ca_file,omitempty"`
    ServerName string `json:"server_name,omitempty"`
    Insecure   bool   `json:"insecure"`
}
```

### RetryConfig

重试策略配置。

```go
type RetryConfig struct {
    MaxRetries      int           `json:"max_retries"`
    InitialInterval time.Duration `json:"initial_interval"`
    MaxInterval     time.Duration `json:"max_interval"`
    Multiplier      float64       `json:"multiplier"`
}
```

### 数据结构

#### ServiceInfo

服务信息结构。

```go
type ServiceInfo struct {
    Name      string            `json:"name"`
    Instances []ServiceInstance `json:"instances"`
    Metadata  map[string]string `json:"metadata,omitempty"`
}
```

#### ServiceInstance

服务实例信息结构。

```go
type ServiceInstance struct {
    ID       string            `json:"id"`
    Address  string            `json:"address"`
    Metadata map[string]string `json:"metadata,omitempty"`
    LeaseID  clientv3.LeaseID  `json:"lease_id,omitempty"`
    TTL      int64             `json:"ttl,omitempty"`
}
```

#### ServiceEvent

服务变化事件结构。

```go
type ServiceEvent struct {
    Type     ServiceEventType `json:"type"`
    Service  string           `json:"service"`
    Instance ServiceInstance  `json:"instance"`
}

type ServiceEventType int

const (
    ServiceEventAdd ServiceEventType = iota
    ServiceEventUpdate
    ServiceEventDelete
)
```

#### LockInfo

锁信息结构。

```go
type LockInfo struct {
    Key     string            `json:"key"`
    Owner   string            `json:"owner"`
    LeaseID clientv3.LeaseID  `json:"lease_id"`
    TTL     int64             `json:"ttl"`
    Created time.Time         `json:"created"`
}
```

#### ConnectionStatus

连接状态结构。

```go
type ConnectionStatus struct {
    Connected bool      `json:"connected"`
    Endpoint  string    `json:"endpoint"`
    LastPing  time.Time `json:"last_ping"`
    Error     string    `json:"error,omitempty"`
}
```

## 错误处理

### 错误类型

组件定义了以下错误类型常量：

```go
const (
    ErrTypeConnection    = "CONNECTION_ERROR"
    ErrTypeRegistry      = "REGISTRY_ERROR"
    ErrTypeDiscovery     = "DISCOVERY_ERROR"
    ErrTypeLock          = "LOCK_ERROR"
    ErrTypeLease         = "LEASE_ERROR"
    ErrTypeConfiguration = "CONFIGURATION_ERROR"
    ErrTypeTimeout       = "TIMEOUT_ERROR"
    ErrTypeNotFound      = "NOT_FOUND_ERROR"
    ErrTypeAlreadyExists = "ALREADY_EXISTS_ERROR"
    ErrTypeInvalidState  = "INVALID_STATE_ERROR"
)
```

### 预定义错误

```go
var (
    // 连接相关错误
    ErrConnectionFailed    = NewEtcdError(ErrTypeConnection, 1001, "failed to connect to etcd", nil)
    ErrConnectionLost      = NewEtcdError(ErrTypeConnection, 1002, "connection to etcd lost", nil)
    ErrConnectionTimeout   = NewEtcdError(ErrTypeConnection, 1003, "connection timeout", nil)
    ErrNotConnected        = NewEtcdError(ErrTypeConnection, 1004, "not connected to etcd", nil)

    // 注册相关错误
    ErrServiceAlreadyRegistered = NewEtcdError(ErrTypeRegistry, 2001, "service already registered", nil)
    ErrServiceNotRegistered     = NewEtcdError(ErrTypeRegistry, 2002, "service not registered", nil)

    // 发现相关错误
    ErrServiceNotFound      = NewEtcdError(ErrTypeDiscovery, 3001, "service not found", nil)
    ErrNoAvailableInstances = NewEtcdError(ErrTypeDiscovery, 3002, "no available service instances", nil)

    // 锁相关错误
    ErrLockAcquisitionFailed = NewEtcdError(ErrTypeLock, 4001, "failed to acquire lock", nil)
    ErrLockNotHeld           = NewEtcdError(ErrTypeLock, 4002, "lock not held", nil)
    ErrLockAlreadyHeld       = NewEtcdError(ErrTypeLock, 4003, "lock already held", nil)

    // 租约相关错误
    ErrLeaseCreationFailed = NewEtcdError(ErrTypeLease, 5001, "failed to create lease", nil)
    ErrLeaseNotFound       = NewEtcdError(ErrTypeLease, 5002, "lease not found", nil)
    ErrLeaseExpired        = NewEtcdError(ErrTypeLease, 5003, "lease expired", nil)

    // 配置相关错误
    ErrInvalidConfiguration = NewEtcdError(ErrTypeConfiguration, 6001, "invalid configuration", nil)
    ErrMissingEndpoints     = NewEtcdError(ErrTypeConfiguration, 6002, "missing etcd endpoints", nil)
)
```

### 错误检查函数

```go
// 错误类型检查
func IsConnectionError(err error) bool
func IsRegistryError(err error) bool
func IsDiscoveryError(err error) bool
func IsLockError(err error) bool
func IsLeaseError(err error) bool
func IsTimeoutError(err error) bool
func IsNotFoundError(err error) bool

// 重试相关
func IsRetryableError(err error) bool
func GetRetryDelay(attempt int) int
```

### 错误处理示例

```go
// 基本错误处理
if err != nil {
    if etcd.IsConnectionError(err) {
        log.Printf("Connection error: %v", err)
        // 实现重连逻辑
    } else if etcd.IsRetryableError(err) {
        // 实现重试逻辑
        delay := etcd.GetRetryDelay(attempt)
        time.Sleep(time.Duration(delay) * time.Millisecond)
        // 重试操作
    } else {
        log.Printf("Non-retryable error: %v", err)
        return err
    }
}

// 错误包装
if err != nil {
    return etcd.WrapRegistryError(err, "failed to register service")
}
```

## 工厂方法

### 便捷创建函数

```go
// 快速启动 - 支持配置优先级
func QuickStart(endpoints ...string) (EtcdManager, error)

// 使用默认配置创建管理器
func NewManager() (EtcdManager, error)

// 使用指定端点创建管理器
func NewManagerWithEndpoints(endpoints []string) (EtcdManager, error)

// 预设环境配置
func NewDevelopmentManager() (EtcdManager, error)
func NewProductionManager(endpoints []string) (EtcdManager, error)
func NewTestManager() (EtcdManager, error)
```

### EtcdManagerFactory

工厂类用于创建和管理 EtcdManager 实例。

```go
type EtcdManagerFactory struct {
    // 私有字段
}

func NewEtcdManagerFactory() *EtcdManagerFactory
func (f *EtcdManagerFactory) CreateManager() (EtcdManager, error)
func (f *EtcdManagerFactory) CreateManagerWithOptions(options *ManagerOptions) (EtcdManager, error)
func (f *EtcdManagerFactory) SetDefaultOptions(options *ManagerOptions) error
func (f *EtcdManagerFactory) GetDefaultOptions() *ManagerOptions
```

## 配置优先级

### 配置加载优先级

etcd-grpc 支持多种配置方式，按以下优先级顺序：

1. **用户输入参数** (最高优先级)
2. **配置文件**
3. **默认值** (最低优先级)

### 配置文件支持

支持 JSON 和 YAML 两种格式的配置文件：

**etcd-config.json**
```json
{
  "endpoints": ["localhost:23791", "localhost:23792", "localhost:23793"],
  "dial_timeout": "5s",
  "log_level": "info"
}
```

**etcd-config.yaml**
```yaml
endpoints:
  - localhost:23791
  - localhost:23792
  - localhost:23793
dial_timeout: 5s
log_level: info
```

### 配置文件查找顺序

当未指定配置文件路径时，系统会按以下顺序查找：

1. `etcd-config.json`
2. `etcd-config.yaml`
3. `etcd-config.yml`
4. `config/etcd.json`
5. `config/etcd.yaml`
6. `config/etcd.yml`

### QuickStart 行为变化

```go
// 方式1: 无参数 - 优先使用配置文件，回退到默认值
manager, err := etcd.QuickStart()

// 方式2: 有参数 - 用户输入覆盖配置文件和默认值
manager, err := etcd.QuickStart("custom:2379")
```

### 使用示例

```go
// 快速启动 - 智能配置加载
manager, err := etcd.QuickStart()
if err != nil {
    log.Fatalf("Failed to start: %v", err)
}
defer manager.Close()

// 用户指定端点（覆盖配置文件）
manager, err := etcd.QuickStart("localhost:2379")
if err != nil {
    log.Fatalf("Failed to start: %v", err)
}
defer manager.Close()

// 开发环境
devManager, err := etcd.NewDevelopmentManager()
if err != nil {
    log.Fatalf("Failed to create dev manager: %v", err)
}
defer devManager.Close()

// 生产环境
prodManager, err := etcd.NewProductionManager([]string{
    "etcd1.prod.com:2379",
    "etcd2.prod.com:2379",
    "etcd3.prod.com:2379",
})
if err != nil {
    log.Fatalf("Failed to create prod manager: %v", err)
}
defer prodManager.Close()

// 使用工厂
factory := etcd.NewEtcdManagerFactory()
manager, err := factory.CreateManagerWithOptions(&etcd.ManagerOptions{
    Endpoints:   []string{"localhost:2379"},
    DialTimeout: 10 * time.Second,
    DefaultTTL:  60,
})
if err != nil {
    log.Fatalf("Failed to create manager: %v", err)
}
defer manager.Close()
```

## 最佳实践

### 1. 资源管理

```go
// 总是使用 defer 确保资源释放
manager, err := etcd.QuickStart("localhost:2379")
if err != nil {
    return err
}
defer manager.Close() // 确保资源释放

// 检查管理器状态
if !manager.IsReady() {
    return fmt.Errorf("manager not ready")
}
```

### 2. 错误处理

```go
// 实现重试逻辑
func registerWithRetry(registry etcd.ServiceRegistry, ctx context.Context, serviceName, instanceID, addr string) error {
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        err := registry.Register(ctx, serviceName, instanceID, addr)
        if err == nil {
            return nil
        }

        lastErr = err
        if !etcd.IsRetryableError(err) {
            break
        }

        delay := etcd.GetRetryDelay(attempt)
        time.Sleep(time.Duration(delay) * time.Millisecond)
    }
    return lastErr
}
```

### 3. 上下文管理

```go
// 使用带超时的上下文
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// 传递上下文到所有操作
err := registry.Register(ctx, "service", "instance", "addr")
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Operation timed out")
    }
    return err
}
```

### 4. 服务注册

```go
// 使用合适的 TTL 和元数据
err := registry.Register(ctx, "user-service", "instance-1", "localhost:50051",
    etcd.WithTTL(60), // 60秒 TTL
    etcd.WithMetadata(map[string]string{
        "version":    "1.2.0",
        "region":     "us-west",
        "datacenter": "dc1",
        "health":     "healthy",
    }),
)
```

### 5. 分布式锁

```go
// 使用适当的锁超时
lockKey := fmt.Sprintf("process-%s", processID)
err := lock.Lock(ctx, lockKey, 30*time.Second)
if err != nil {
    return err
}
defer func() {
    // 确保锁被释放
    if unlockErr := lock.Unlock(context.Background(), lockKey); unlockErr != nil {
        log.Printf("Failed to unlock: %v", unlockErr)
    }
}()

// 执行临界区操作
return performCriticalOperation()
```

### 6. 服务发现

```go
// 缓存连接以提高性能
var (
    connCache = make(map[string]*grpc.ClientConn)
    connMutex sync.RWMutex
)

func getServiceConnection(discovery etcd.ServiceDiscovery, serviceName string) (*grpc.ClientConn, error) {
    connMutex.RLock()
    if conn, exists := connCache[serviceName]; exists {
        connMutex.RUnlock()
        return conn, nil
    }
    connMutex.RUnlock()

    connMutex.Lock()
    defer connMutex.Unlock()

    // 双重检查
    if conn, exists := connCache[serviceName]; exists {
        return conn, nil
    }

    conn, err := discovery.GetConnection(context.Background(), serviceName)
    if err != nil {
        return nil, err
    }

    connCache[serviceName] = conn
    return conn, nil
}
```

### 7. 配置管理

```go
// 使用环境变量配置
func createManagerFromEnv() (etcd.EtcdManager, error) {
    endpoints := strings.Split(os.Getenv("ETCD_ENDPOINTS"), ",")
    if len(endpoints) == 0 {
        endpoints = []string{"localhost:2379"}
    }

    dialTimeout, _ := time.ParseDuration(os.Getenv("ETCD_DIAL_TIMEOUT"))
    if dialTimeout == 0 {
        dialTimeout = 5 * time.Second
    }

    return etcd.NewManagerBuilder().
        WithEndpoints(endpoints).
        WithDialTimeout(dialTimeout).
        WithServicePrefix(os.Getenv("SERVICE_PREFIX")).
        Build()
}
```

## 注意事项

1. **线程安全**：所有公共接口都是线程安全的，可以在多个 goroutine 中并发使用。

2. **资源清理**：始终调用 `Close()` 方法释放资源，建议使用 `defer` 语句。

3. **错误处理**：检查所有返回的错误，使用提供的错误检查函数进行分类处理。

4. **上下文使用**：为所有操作提供适当的上下文，特别是设置合理的超时时间。

5. **配置验证**：在生产环境中使用前验证所有配置参数。

6. **监控和日志**：实现适当的监控和日志记录，使用自定义日志器接口。

7. **版本兼容性**：确保 etcd 服务器版本与客户端库兼容。
