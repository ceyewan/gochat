package coordination

import (
	"context"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coordination/internal"
	"google.golang.org/grpc"
)

// Coordinator 定义分布式协调操作的核心接口。
// 提供服务注册发现、分布式锁、配置中心等功能的统一访问入口。
type Coordinator = internal.Coordinator

// ServiceRegistry 定义服务注册与发现的接口。
// 提供服务注册、发现、健康检查和负载均衡等功能。
type ServiceRegistry = internal.ServiceRegistry

// DistributedLock 定义分布式锁的接口。
// 提供基础锁、可重入锁、读写锁等分布式锁定机制。
type DistributedLock = internal.DistributedLock

// ConfigCenter 定义配置中心的接口。
// 提供动态配置管理、版本控制、变更通知等功能。
type ConfigCenter = internal.ConfigCenter

// Lock 定义锁实例的接口。
type Lock = internal.Lock

// ReentrantLock 定义可重入锁实例的接口。
type ReentrantLock = internal.ReentrantLock

// ServiceInfo 定义服务信息结构。
type ServiceInfo = internal.ServiceInfo

// HealthStatus 定义服务健康状态。
type HealthStatus = internal.HealthStatus

// LoadBalanceStrategy 定义负载均衡策略。
type LoadBalanceStrategy = internal.LoadBalanceStrategy

// ConfigValue 定义配置值结构。
type ConfigValue = internal.ConfigValue

// ConfigChange 定义配置变更事件。
type ConfigChange = internal.ConfigChange

// ConfigVersion 定义配置版本信息。
type ConfigVersion = internal.ConfigVersion

// 健康状态常量
const (
	HealthUnknown     = internal.HealthUnknown
	HealthHealthy     = internal.HealthHealthy
	HealthUnhealthy   = internal.HealthUnhealthy
	HealthMaintenance = internal.HealthMaintenance
)

// 负载均衡策略常量
const (
	LoadBalanceRoundRobin = internal.LoadBalanceRoundRobin
	LoadBalanceRandom     = internal.LoadBalanceRandom
	LoadBalanceWeighted   = internal.LoadBalanceWeighted
	LoadBalanceLeastConn  = internal.LoadBalanceLeastConn
)

// 配置变更类型常量
const (
	ConfigChangeCreate = internal.ConfigChangeCreate
	ConfigChangeUpdate = internal.ConfigChangeUpdate
	ConfigChangeDelete = internal.ConfigChangeDelete
)

var (
	// 全局默认协调器实例
	defaultCoordinator Coordinator
	// 确保默认协调器只初始化一次
	defaultCoordinatorOnce sync.Once
	// 模块协调器缓存
	moduleCoordinators = make(map[string]Coordinator)
	// 保护模块协调器缓存的读写锁
	moduleCoordinatorsMutex sync.RWMutex
	// 模块日志器
	logger = clog.Module("coordination")
)

// getDefaultCoordinator 获取全局默认协调器实例，使用懒加载和单例模式
func getDefaultCoordinator() Coordinator {
	defaultCoordinatorOnce.Do(func() {
		cfg := DefaultConfig()
		var err error
		defaultCoordinator, err = internal.NewCoordinator(cfg)
		if err != nil {
			logger.Error("创建默认协调器实例失败", clog.ErrorValue(err))
			panic(err)
		}
		logger.Info("默认协调器实例创建成功")
	})
	return defaultCoordinator
}

// New 根据提供的配置创建一个新的 Coordinator 实例。
// 用于自定义协调器实例的主要入口。
//
// 示例：
//
//	cfg := coordination.Config{
//		Endpoints:   []string{"localhost:2379"},
//		DialTimeout: 5 * time.Second,
//		ServiceRegistry: coordination.ServiceRegistryConfig{
//			KeyPrefix: "/services",
//			TTL:       30 * time.Second,
//		},
//	}
//
//	coordinator, err := coordination.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer coordinator.Close()
//
//	// 使用服务注册
//	registry := coordinator.ServiceRegistry()
//	err = registry.Register(ctx, coordination.ServiceInfo{
//		Name:       "user-service",
//		InstanceID: "instance-1",
//		Address:    "localhost:50051",
//		Metadata:   map[string]string{"version": "1.0.0"},
//	})
func New(cfg Config) (Coordinator, error) {
	return internal.NewCoordinator(cfg)
}

// Module 返回一个带有指定模块名的协调器实例。
// 对于相同的模块名，返回相同的协调器实例（单例模式）。
// 模块协调器继承默认协调器的配置，并添加模块特定的日志上下文。
//
// 示例：
//
//	coordinator := coordination.Module("user-service")
//	registry := coordinator.ServiceRegistry()
//	err := registry.Register(ctx, serviceInfo)
func Module(name string) Coordinator {
	// 先尝试读锁获取已存在的模块协调器
	moduleCoordinatorsMutex.RLock()
	if coordinator, exists := moduleCoordinators[name]; exists {
		moduleCoordinatorsMutex.RUnlock()
		return coordinator
	}
	moduleCoordinatorsMutex.RUnlock()

	// 如果不存在，使用写锁创建新的模块协调器
	moduleCoordinatorsMutex.Lock()
	defer moduleCoordinatorsMutex.Unlock()

	// 双重检查，防止在获取写锁期间其他 goroutine 已经创建了
	if coordinator, exists := moduleCoordinators[name]; exists {
		return coordinator
	}

	// 基于默认协调器创建模块协调器
	moduleCoordinator := internal.NewModuleCoordinator(getDefaultCoordinator(), name)
	moduleCoordinators[name] = moduleCoordinator
	return moduleCoordinator
}

// ===== 全局服务注册方法 =====

// RegisterService 使用全局默认协调器注册服务
func RegisterService(ctx context.Context, service ServiceInfo) error {
	return getDefaultCoordinator().ServiceRegistry().Register(ctx, service)
}

// DeregisterService 使用全局默认协调器注销服务
func DeregisterService(ctx context.Context, serviceName, instanceID string) error {
	return getDefaultCoordinator().ServiceRegistry().Deregister(ctx, serviceName, instanceID)
}

// DiscoverServices 使用全局默认协调器发现服务
func DiscoverServices(ctx context.Context, serviceName string) ([]ServiceInfo, error) {
	return getDefaultCoordinator().ServiceRegistry().Discover(ctx, serviceName)
}

// GetServiceConnection 使用全局默认协调器获取服务连接
func GetServiceConnection(ctx context.Context, serviceName string, strategy LoadBalanceStrategy) (*grpc.ClientConn, error) {
	return getDefaultCoordinator().ServiceRegistry().GetConnection(ctx, serviceName, strategy)
}

// ===== 全局分布式锁方法 =====

// AcquireLock 使用全局默认协调器获取分布式锁
func AcquireLock(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	return getDefaultCoordinator().Lock().Acquire(ctx, key, ttl)
}

// AcquireReentrantLock 使用全局默认协调器获取可重入锁
func AcquireReentrantLock(ctx context.Context, key string, ttl time.Duration) (ReentrantLock, error) {
	return getDefaultCoordinator().Lock().AcquireReentrant(ctx, key, ttl)
}

// AcquireReadLock 使用全局默认协调器获取读锁
func AcquireReadLock(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	return getDefaultCoordinator().Lock().AcquireReadLock(ctx, key, ttl)
}

// AcquireWriteLock 使用全局默认协调器获取写锁
func AcquireWriteLock(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	return getDefaultCoordinator().Lock().AcquireWriteLock(ctx, key, ttl)
}

// ===== 全局配置中心方法 =====

// GetConfig 使用全局默认协调器获取配置
func GetConfig(ctx context.Context, key string) (*ConfigValue, error) {
	return getDefaultCoordinator().ConfigCenter().Get(ctx, key)
}

// SetConfig 使用全局默认协调器设置配置
func SetConfig(ctx context.Context, key string, value interface{}, version int64) error {
	return getDefaultCoordinator().ConfigCenter().Set(ctx, key, value, version)
}

// WatchConfig 使用全局默认协调器监听配置变更
func WatchConfig(ctx context.Context, key string) (<-chan *ConfigChange, error) {
	return getDefaultCoordinator().ConfigCenter().Watch(ctx, key)
}

// ===== 全局管理方法 =====

// Ping 使用全局默认协调器检查连接状态
func Ping(ctx context.Context) error {
	return getDefaultCoordinator().Ping(ctx)
}

// Close 关闭全局默认协调器
func Close() error {
	if defaultCoordinator != nil {
		return defaultCoordinator.Close()
	}
	return nil
}
