package coordination

import (
	"context"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/pkg/client"
	"github.com/ceyewan/gochat/im-infra/coord/pkg/config"
	"github.com/ceyewan/gochat/im-infra/coord/pkg/lock"
	"github.com/ceyewan/gochat/im-infra/coord/pkg/registry"
)

// 模块日志器
var logger = clog.Module("coordination")

// coordinator 主协调器实现
type coordinator struct {
	client   *client.EtcdClient
	lock     DistributedLock
	registry ServiceRegistry
	config   ConfigCenter
	logger   clog.Logger
	closed   bool
	mu       sync.RWMutex
}

// 全局变量管理
var (
	// 全局默认协调器实例
	defaultCoordinator Coordinator
	// 确保默认协调器只初始化一次
	defaultCoordinatorOnce sync.Once
)

// ===== 全局服务注册方法 =====

// RegisterService 使用全局默认协调器注册服务
func RegisterService(ctx context.Context, service ServiceInfo) error {
	logger.Info("注册服务 (全局方法)",
		clog.String("service_name", service.Name),
		clog.String("service_id", service.ID))

	return getDefaultCoordinator().Registry().Register(ctx, service)
}

// UnregisterService 使用全局默认协调器注销服务
func UnregisterService(ctx context.Context, serviceID string) error {
	logger.Info("注销服务 (全局方法)",
		clog.String("service_id", serviceID))

	return getDefaultCoordinator().Registry().Unregister(ctx, serviceID)
}

// DiscoverServices 使用全局默认协调器发现服务
func DiscoverServices(ctx context.Context, serviceName string) ([]ServiceInfo, error) {
	logger.Info("发现服务 (全局方法)",
		clog.String("service_name", serviceName))

	return getDefaultCoordinator().Registry().Discover(ctx, serviceName)
}

// WatchServices 使用全局默认协调器监听服务变化
func WatchServices(ctx context.Context, serviceName string) (<-chan ServiceEvent, error) {
	logger.Info("监听服务变化 (全局方法)",
		clog.String("service_name", serviceName))

	return getDefaultCoordinator().Registry().Watch(ctx, serviceName)
}

// ===== 全局分布式锁方法 =====

// AcquireLock 使用全局默认协调器获取分布式锁
func AcquireLock(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	logger.Info("获取分布式锁 (全局方法)",
		clog.String("key", key),
		clog.Duration("ttl", ttl))

	return getDefaultCoordinator().Lock().Acquire(ctx, key, ttl)
}

// TryAcquireLock 使用全局默认协调器尝试获取锁（非阻塞）
func TryAcquireLock(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	logger.Info("尝试获取分布式锁 (全局方法)",
		clog.String("key", key),
		clog.Duration("ttl", ttl))

	return getDefaultCoordinator().Lock().TryAcquire(ctx, key, ttl)
}

// ===== 全局配置中心方法 =====

// GetConfig 使用全局默认协调器获取配置
func GetConfig(ctx context.Context, key string) (interface{}, error) {
	logger.Info("获取配置 (全局方法)",
		clog.String("key", key))

	return getDefaultCoordinator().Config().Get(ctx, key)
}

// SetConfig 使用全局默认协调器设置配置
func SetConfig(ctx context.Context, key string, value interface{}) error {
	logger.Info("设置配置 (全局方法)",
		clog.String("key", key))

	return getDefaultCoordinator().Config().Set(ctx, key, value)
}

// DeleteConfig 使用全局默认协调器删除配置
func DeleteConfig(ctx context.Context, key string) error {
	logger.Info("删除配置 (全局方法)",
		clog.String("key", key))

	return getDefaultCoordinator().Config().Delete(ctx, key)
}

// WatchConfig 使用全局默认协调器监听配置变更
func WatchConfig(ctx context.Context, key string) (<-chan ConfigEvent, error) {
	logger.Info("监听配置变化 (全局方法)",
		clog.String("key", key))

	return getDefaultCoordinator().Config().Watch(ctx, key)
}

// ListConfigs 使用全局默认协调器列出配置键
func ListConfigs(ctx context.Context, prefix string) ([]string, error) {
	logger.Info("列出配置键 (全局方法)",
		clog.String("prefix", prefix))

	return getDefaultCoordinator().Config().List(ctx, prefix)
}

// ===== 全局管理方法 =====

// Close 关闭全局默认协调器
func Close() error {
	logger.Info("关闭全局协调器")

	if defaultCoordinator != nil {
		return defaultCoordinator.Close()
	}
	return nil
}

// ===== 协调器创建和管理 =====

// NewCoordinator 创建协调器实例
func NewCoordinator(opts CoordinatorOptions) (Coordinator, error) {
	// 验证选项
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	logger := clog.Module("coordination")
	logger.Info("创建协调器实例",
		clog.Strings("endpoints", opts.Endpoints),
		clog.Duration("timeout", opts.Timeout))

	// 创建 etcd 客户端
	etcdClient, err := client.NewEtcdClient(opts)
	if err != nil {
		logger.Error("创建 etcd 客户端失败", clog.Err(err))
		return nil, err
	}

	// 创建各个子模块
	lockService := lock.NewEtcdDistributedLock(etcdClient, "/locks")
	registryService := registry.NewEtcdServiceRegistry(etcdClient, "/services")
	configService := config.NewEtcdConfigCenter(etcdClient, "/config")

	coord := &coordinator{
		client:   etcdClient,
		lock:     lockService,
		registry: registryService,
		config:   configService,
		logger:   logger,
		closed:   false,
	}

	logger.Info("协调器实例创建成功")
	return coord, nil
}

// New 创建新的协调器实例（别名方法，保持向后兼容）
func New(opts CoordinatorOptions) (Coordinator, error) {
	return NewCoordinator(opts)
}

// Default 获取默认协调器实例
func Default() Coordinator {
	return getDefaultCoordinator()
}

// getDefaultCoordinator 获取全局默认协调器实例
func getDefaultCoordinator() Coordinator {
	defaultCoordinatorOnce.Do(func() {
		opts := DefaultCoordinatorOptions()
		var err error
		defaultCoordinator, err = NewCoordinator(opts)
		if err != nil {
			logger := clog.Module("coordination")
			logger.Error("创建默认协调器实例失败", clog.Err(err))
			panic(err)
		}
	})
	return defaultCoordinator
}

// ===== coordinator 方法实现 =====

// Lock 获取分布式锁服务
func (c *coordinator) Lock() DistributedLock {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		c.logger.Warn("尝试从已关闭的协调器获取锁服务")
		return nil
	}

	return c.lock
}

// Registry 获取服务注册发现
func (c *coordinator) Registry() ServiceRegistry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		c.logger.Warn("尝试从已关闭的协调器获取注册发现服务")
		return nil
	}

	return c.registry
}

// Config 获取配置中心
func (c *coordinator) Config() ConfigCenter {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		c.logger.Warn("尝试从已关闭的协调器获取配置中心服务")
		return nil
	}

	return c.config
}

// Close 关闭协调器
func (c *coordinator) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		c.logger.Debug("协调器已经关闭")
		return nil
	}

	c.logger.Info("关闭协调器")

	var err error
	if c.client != nil {
		if closeErr := c.client.Close(); closeErr != nil {
			c.logger.Error("关闭 etcd 客户端失败", clog.Err(closeErr))
			err = closeErr
		}
	}

	c.closed = true
	c.logger.Info("协调器关闭完成")

	return err
}
