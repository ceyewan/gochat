package coord

import (
	"context"
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/config"
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
	// Close 关闭协调器并释放资源
	Close() error
}

// coordinator 主协调器实现
type coordinator struct {
	client   *client.EtcdClient
	lock     lock.DistributedLock
	registry registry.ServiceRegistry
	config   config.ConfigCenter
	logger   clog.Logger
	closed   bool
	mu       sync.RWMutex
}

// New 创建并返回一个新的协调器实例
func New(ctx context.Context, cfg CoordinatorConfig, opts ...Option) (Provider, error) {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	var logger clog.Logger
	if options.Logger != nil {
		logger = options.Logger.With(clog.String("component", "coord"))
	} else {
		logger = clog.Module("coord")
	}

	logger.Info("creating new coordinator",
		clog.Strings("endpoints", cfg.Endpoints))

	// 2. 创建内部 etcd 客户端
	clientCfg := client.Config{
		Endpoints:   cfg.Endpoints,
		Username:    cfg.Username,
		Password:    cfg.Password,
		Timeout:     cfg.Timeout,
		RetryConfig: (*client.RetryConfig)(cfg.RetryConfig),
		Logger:      logger.With(clog.String("component", "etcd-client")),
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
		client:   etcdClient,
		lock:     lockService,
		registry: registryService,
		config:   configService,
		logger:   logger,
		closed:   false,
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

// Close 实现 Provider 接口 - 关闭协调器并释放资源
func (c *coordinator) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.logger.Info("closing coordinator")

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
