# coordination API 文档

## 概述

`coordination` 是一个基于 etcd 的高性能分布式协调库，提供了服务注册发现、分布式锁、配置中心等企业级功能。

## 核心特性

- 🌟 **全局方法**：支持 `coordination.RegisterService()` 等全局方法，无需显式创建协调器
- 🚀 **基于 etcd**：充分利用 etcd 的强一致性和高可用性
- 🔧 **服务注册发现**：支持健康检查、负载均衡、服务监听
- 🔒 **分布式锁**：支持基础锁、可重入锁、读写锁，自动续期
- ⚙️ **配置中心**：支持版本控制、变更通知、历史追踪

## 全局方法

### 协调器管理

```go
// 创建协调器实例
func New(cfg Config) (Coordinator, error)

// 检查连接状态
func Ping(ctx context.Context) error

// 关闭全局协调器
func Close() error
```

### 服务注册发现

```go
// 注册服务
func RegisterService(ctx context.Context, service ServiceInfo) error

// 注销服务
func DeregisterService(ctx context.Context, serviceName, instanceID string) error

// 发现服务
func DiscoverServices(ctx context.Context, serviceName string) ([]ServiceInfo, error)

// 获取服务连接
func GetServiceConnection(ctx context.Context, serviceName string, strategy LoadBalanceStrategy) (*grpc.ClientConn, error)
```

### 分布式锁

```go
// 获取基础锁
func AcquireLock(ctx context.Context, key string, ttl time.Duration) (Lock, error)

// 获取可重入锁
func AcquireReentrantLock(ctx context.Context, key string, ttl time.Duration) (ReentrantLock, error)

// 获取读锁
func AcquireReadLock(ctx context.Context, key string, ttl time.Duration) (Lock, error)

// 获取写锁
func AcquireWriteLock(ctx context.Context, key string, ttl time.Duration) (Lock, error)
```

### 配置中心

```go
// 获取配置
func GetConfig(ctx context.Context, key string) (*ConfigValue, error)

// 设置配置
func SetConfig(ctx context.Context, key string, value interface{}, version int64) error

// 监听配置变更
func WatchConfig(ctx context.Context, key string) (<-chan *ConfigChange, error)
```

## 核心接口

### Coordinator

```go
type Coordinator interface {
    // 获取服务注册与发现实例
    ServiceRegistry() ServiceRegistry
    
    // 获取分布式锁实例
    Lock() DistributedLock
    
    // 获取配置中心实例
    ConfigCenter() ConfigCenter
    
    // 检查 etcd 连接是否正常
    Ping(ctx context.Context) error
    
    // 关闭协调器并释放资源
    Close() error
}
```

### ServiceRegistry

```go
type ServiceRegistry interface {
    // 注册服务实例
    Register(ctx context.Context, service ServiceInfo) error
    
    // 注销服务实例
    Deregister(ctx context.Context, serviceName, instanceID string) error
    
    // 发现指定服务的所有健康实例
    Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error)
    
    // 监听指定服务的实例变化
    Watch(ctx context.Context, serviceName string) (<-chan []ServiceInfo, error)
    
    // 更新服务实例的健康状态
    UpdateHealth(ctx context.Context, serviceName, instanceID string, status HealthStatus) error
    
    // 获取到指定服务的 gRPC 连接，支持负载均衡
    GetConnection(ctx context.Context, serviceName string, strategy LoadBalanceStrategy) (*grpc.ClientConn, error)
}
```

### DistributedLock

```go
type DistributedLock interface {
    // 获取基础分布式锁
    Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
    
    // 获取可重入分布式锁
    AcquireReentrant(ctx context.Context, key string, ttl time.Duration) (ReentrantLock, error)
    
    // 获取读锁
    AcquireReadLock(ctx context.Context, key string, ttl time.Duration) (Lock, error)
    
    // 获取写锁
    AcquireWriteLock(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}
```

### ConfigCenter

```go
type ConfigCenter interface {
    // 获取配置值
    Get(ctx context.Context, key string) (*ConfigValue, error)
    
    // 设置配置值，支持版本控制
    Set(ctx context.Context, key string, value interface{}, version int64) error
    
    // 删除配置，支持版本控制
    Delete(ctx context.Context, key string, version int64) error
    
    // 获取配置的当前版本号
    GetVersion(ctx context.Context, key string) (int64, error)
    
    // 获取配置的历史版本
    GetHistory(ctx context.Context, key string, limit int) ([]ConfigVersion, error)
    
    // 监听指定配置的变更
    Watch(ctx context.Context, key string) (<-chan *ConfigChange, error)
    
    // 监听指定前缀下所有配置的变更
    WatchPrefix(ctx context.Context, prefix string) (<-chan *ConfigChange, error)
}
```

### Lock

```go
type Lock interface {
    // 释放锁
    Release(ctx context.Context) error
    
    // 续期锁
    Renew(ctx context.Context, ttl time.Duration) error
    
    // 检查锁是否仍被持有
    IsHeld(ctx context.Context) (bool, error)
    
    // 返回锁的键名
    Key() string
    
    // 返回锁的剩余生存时间
    TTL(ctx context.Context) (time.Duration, error)
}
```

### ReentrantLock

```go
type ReentrantLock interface {
    Lock
    
    // 返回当前锁的获取次数
    AcquireCount() int
    
    // 释放一次锁，只有当获取次数为0时才真正释放
    Release(ctx context.Context) error
}
```

## 数据结构

### Config

```go
type Config struct {
    // etcd 服务器地址列表
    Endpoints []string
    
    // 连接超时时间
    DialTimeout time.Duration
    
    // etcd 用户名（可选）
    Username string
    
    // etcd 密码（可选）
    Password string
    
    // TLS 配置（可选）
    TLS *TLSConfig
    
    // 服务注册与发现配置
    ServiceRegistry ServiceRegistryConfig
    
    // 分布式锁配置
    DistributedLock DistributedLockConfig
    
    // 配置中心配置
    ConfigCenter ConfigCenterConfig
    
    // 重试策略配置
    Retry *RetryConfig
    
    // 日志级别
    LogLevel string
    
    // 是否启用指标收集
    EnableMetrics bool
    
    // 是否启用链路追踪
    EnableTracing bool
}
```

### ServiceInfo

```go
type ServiceInfo struct {
    // 服务名称
    Name string
    
    // 服务实例ID
    InstanceID string
    
    // 服务地址，格式为 "host:port"
    Address string
    
    // 服务元数据
    Metadata map[string]string
    
    // 服务健康状态
    Health HealthStatus
    
    // 注册时间
    RegisterTime time.Time
    
    // 最后心跳时间
    LastHeartbeat time.Time
}
```

### ConfigValue

```go
type ConfigValue struct {
    // 配置键
    Key string
    
    // 配置值
    Value string
    
    // 配置版本号
    Version int64
    
    // 创建时间
    CreateTime time.Time
    
    // 更新时间
    UpdateTime time.Time
    
    // 配置元数据
    Metadata map[string]string
}
```

### ConfigChange

```go
type ConfigChange struct {
    // 变更类型
    Type ConfigChangeType
    
    // 配置键
    Key string
    
    // 旧值
    OldValue *ConfigValue
    
    // 新值
    NewValue *ConfigValue
    
    // 变更时间戳
    Timestamp time.Time
}
```

## 枚举类型

### HealthStatus

```go
const (
    HealthUnknown     HealthStatus = iota // 未知状态
    HealthHealthy                         // 健康状态
    HealthUnhealthy                       // 不健康状态
    HealthMaintenance                     // 维护状态
)
```

### LoadBalanceStrategy

```go
const (
    LoadBalanceRoundRobin LoadBalanceStrategy = iota // 轮询策略
    LoadBalanceRandom                                // 随机策略
    LoadBalanceWeighted                              // 加权策略
    LoadBalanceLeastConn                             // 最少连接策略
)
```

### ConfigChangeType

```go
const (
    ConfigChangeCreate ConfigChangeType = iota // 创建配置
    ConfigChangeUpdate                         // 更新配置
    ConfigChangeDelete                         // 删除配置
)
```

## 配置工厂函数

### 预设配置

```go
// 返回默认配置
func DefaultConfig() Config

// 返回适用于开发环境的配置
func DevelopmentConfig() Config

// 返回适用于生产环境的配置
func ProductionConfig() Config

// 返回适用于测试环境的配置
func TestConfig() Config
```

### 自定义配置

```go
// 创建服务注册配置
func NewServiceRegistryConfig(keyPrefix string, ttl, healthCheckInterval time.Duration, enableHealthCheck bool) ServiceRegistryConfig

// 创建分布式锁配置
func NewDistributedLockConfig(keyPrefix string, defaultTTL, renewInterval time.Duration, enableReentrant bool) DistributedLockConfig

// 创建配置中心配置
func NewConfigCenterConfig(keyPrefix string, enableVersioning bool, maxVersionHistory int, enableValidation bool) ConfigCenterConfig

// 创建重试配置
func NewRetryConfig(maxRetries int, initialInterval, maxInterval time.Duration, multiplier float64) RetryConfig
```

## 使用示例

### 基础用法

```go
// 使用默认配置
coordinator, err := coordination.New(coordination.DefaultConfig())
if err != nil {
    log.Fatal(err)
}
defer coordinator.Close()

// 或使用全局方法
err = coordination.RegisterService(ctx, serviceInfo)
```



### 自定义配置

```go
cfg := coordination.Config{
    Endpoints:   []string{"etcd-1:2379", "etcd-2:2379"},
    DialTimeout: 10 * time.Second,
    ServiceRegistry: coordination.NewServiceRegistryConfig(
        "/my-services", 60*time.Second, 20*time.Second, true,
    ),
}

coordinator, err := coordination.New(cfg)
```

## 错误处理

所有方法都返回标准的 Go error 类型。常见错误包括：

- 连接错误：etcd 不可用或网络问题
- 验证错误：配置参数无效
- 冲突错误：版本冲突或锁竞争
- 超时错误：操作超时

建议使用适当的重试策略和错误监控。

## 性能考虑

- 复用协调器实例以减少连接开销
- 合理设置超时和重试参数
- 监控 etcd 集群性能和健康状态
