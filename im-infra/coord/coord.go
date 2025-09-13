package coord

import (
	"context"
	"fmt"
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/allocator"
	"github.com/ceyewan/gochat/im-infra/coord/config"
	"github.com/ceyewan/gochat/im-infra/coord/internal/allocatorimpl"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	"github.com/ceyewan/gochat/im-infra/coord/internal/configimpl"
	"github.com/ceyewan/gochat/im-infra/coord/internal/lockimpl"
	"github.com/ceyewan/gochat/im-infra/coord/internal/registryimpl"
	"github.com/ceyewan/gochat/im-infra/coord/lock"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
)

// Provider 定义协调器的核心接口
type Provider interface {
	// Lock 获取分布式锁服务
	Lock() lock.DistributedLock
	// Registry 获取服务注册发现服务
	Registry() registry.ServiceRegistry
	// Config 获取配置中心服务
	Config() config.ConfigCenter
	// InstanceIDAllocator 获取一个服务实例ID分配器
	// 此方法是可重入的：为同一个 serviceName 多次调用，将返回同一个共享的分配器实例
	InstanceIDAllocator(serviceName string, maxID int) (allocator.InstanceIDAllocator, error)
	// Close 关闭协调器并释放资源
	Close() error
}

// coordinator 主协调器实现
type coordinator struct {
	client          *client.EtcdClient
	lock            lock.DistributedLock
	registry        registry.ServiceRegistry
	config          config.ConfigCenter
	logger          clog.Logger
	closed          bool
	mu              sync.RWMutex
	allocators      map[string]allocator.InstanceIDAllocator // 缓存分配器实例
	allocatorsMu    sync.RWMutex
}

// New 创建一个新的 coord Provider 实例
// 这是与 coord 组件交互的唯一入口
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error) {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	var logger clog.Logger
	if options.Logger != nil {
		logger = options.Logger.With(clog.String("component", "coord"))
	} else {
		logger = clog.Namespace("coord")
	}

	logger.Info("creating new coordinator",
		clog.Strings("endpoints", config.Endpoints))

	// 2. 创建内部 etcd 客户端
	clientCfg := client.Config{
		Endpoints: config.Endpoints,
		Username:  config.Username,
		Password:  config.Password,
		Timeout:   config.DialTimeout,
		Logger:    logger.With(clog.String("component", "etcd-client")),
	}
	etcdClient, err := client.New(clientCfg)
	if err != nil {
		logger.Error("failed to create etcd client", clog.Err(err))
		return nil, err
	}

	// 3. 创建内部服务
	lockService := lockimpl.NewEtcdLockFactory(etcdClient, "/locks", logger.With(clog.String("component", "lock")))
	registryService := registryimpl.NewEtcdServiceRegistry(etcdClient, "/services", logger.With(clog.String("component", "registry")))
	configService := configimpl.NewEtcdConfigCenter(etcdClient, "/config", logger.With(clog.String("component", "config")))

	// 4. 组装 coordinator
	coord := &coordinator{
		client:       etcdClient,
		lock:         lockService,
		registry:     registryService,
		config:       configService,
		logger:       logger,
		closed:       false,
		allocators:   make(map[string]allocator.InstanceIDAllocator),
	}

	logger.Info("coordinator created successfully")
	return coord, nil
}

// Lock 实现 Provider 接口 - 获取分布式锁服务
func (c *coordinator) Lock() lock.DistributedLock {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lock
}

// Registry 实现 Provider 接口 - 获取服务注册发现服务
func (c *coordinator) Registry() registry.ServiceRegistry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registry
}

// Config 实现 Provider 接口 - 获取配置中心服务
func (c *coordinator) Config() config.ConfigCenter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// InstanceIDAllocator 实现 Provider 接口 - 获取服务实例ID分配器
// 此方法是可重入的：为同一个 serviceName 多次调用，将返回同一个共享的分配器实例
func (c *coordinator) InstanceIDAllocator(serviceName string, maxID int) (allocator.InstanceIDAllocator, error) {
	c.allocatorsMu.RLock()
	
	// 生成缓存键
	cacheKey := fmt.Sprintf("%s:%d", serviceName, maxID)
	
	// 检查是否已存在
	if allocator, exists := c.allocators[cacheKey]; exists {
		c.allocatorsMu.RUnlock()
		return allocator, nil
	}
	c.allocatorsMu.RUnlock()

	// 创建新的分配器
	c.allocatorsMu.Lock()
	defer c.allocatorsMu.Unlock()

	// 再次检查，防止并发创建
	if allocator, exists := c.allocators[cacheKey]; exists {
		return allocator, nil
	}

	// 获取 etcd 原始客户端
	etcdClient := c.client.Client()

	// 创建分配器
	allocator, err := allocatorimpl.NewEtcdInstanceIDAllocator(
		etcdClient,
		serviceName,
		maxID,
		c.logger.With(clog.String("service", serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance ID allocator: %w", err)
	}

	// 缓存分配器
	c.allocators[cacheKey] = allocator

	c.logger.Info("instance ID allocator created",
		clog.String("service", serviceName),
		clog.Int("max_id", maxID))

	return allocator, nil
}

// Close 实现 Provider 接口 - 关闭协调器并释放资源
func (c *coordinator) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.logger.Info("closing coordinator")

	// 关闭所有分配器
	c.allocatorsMu.Lock()
	for key, allocator := range c.allocators {
		if closer, ok := allocator.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				c.logger.Error("failed to close allocator", clog.String("key", key), clog.Err(err))
			}
		}
		delete(c.allocators, key)
	}
	c.allocatorsMu.Unlock()

	// 关闭 etcd 客户端
	if c.client != nil {
		if err := c.client.Close(); err != nil {
			c.logger.Error("failed to close etcd client", clog.Err(err))
			return err
		}
	}

	c.closed = true
	c.logger.Info("coordinator closed successfully")
	return nil
}
