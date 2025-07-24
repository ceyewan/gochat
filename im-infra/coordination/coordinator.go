package coordination

import (
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coordination/pkg/client"
	"github.com/ceyewan/gochat/im-infra/coordination/pkg/config"
	"github.com/ceyewan/gochat/im-infra/coordination/pkg/lock"
	"github.com/ceyewan/gochat/im-infra/coordination/pkg/registry"
)

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

// 全局变量管理
var (
	// 全局默认协调器实例
	defaultCoordinator Coordinator
	// 确保默认协调器只初始化一次
	defaultCoordinatorOnce sync.Once
	// 模块协调器缓存
	moduleCoordinators = make(map[string]Coordinator)
	// 保护模块协调器缓存的读写锁
	moduleCoordinatorsMutex sync.RWMutex
)

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

// Module 返回一个带有指定模块名的协调器实例
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
	moduleCoordinator := &moduleCoordinatorWrapper{
		base:       getDefaultCoordinator(),
		moduleName: name,
		logger:     clog.Module("coordination." + name),
	}
	moduleCoordinators[name] = moduleCoordinator
	return moduleCoordinator
}

// moduleCoordinatorWrapper 模块协调器包装器
type moduleCoordinatorWrapper struct {
	base       Coordinator
	moduleName string
	logger     clog.Logger
}

func (m *moduleCoordinatorWrapper) Lock() DistributedLock {
	return m.base.Lock()
}

func (m *moduleCoordinatorWrapper) Registry() ServiceRegistry {
	return m.base.Registry()
}

func (m *moduleCoordinatorWrapper) Config() ConfigCenter {
	return m.base.Config()
}

func (m *moduleCoordinatorWrapper) Close() error {
	// 模块协调器不直接关闭底层资源
	return nil
}
