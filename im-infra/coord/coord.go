package coord

import (
	"context"
	"github.com/ceyewan/gochat/im-infra/coord/lock"
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	"github.com/ceyewan/gochat/im-infra/coord/internal/configimpl"
	"github.com/ceyewan/gochat/im-infra/coord/internal/lockimpl"
	"github.com/ceyewan/gochat/im-infra/coord/internal/registryimpl"
	"github.com/ceyewan/gochat/im-infra/coord/internal/types"
)

// coordinator 主协调器实现
type coordinator struct {
	client   *client.EtcdClient
	lock     lockimpl.DistributedLock
	registry registryimpl.ServiceRegistry
	config   configimpl.EtcdConfigCenter
	logger   clog.Logger
	closed   bool
	mu       sync.RWMutex
}

// New 创建并返回一个新的协调器实例
// 这是符合 infra-style-guide 的标准构造函数
func New(ctx context.Context, opts ...types.Option) (types.Provider, error) {
	// 1. 初始化选项
	finalOpts := DefaultCoordinatorConfig()

	logger := clog.Module("coord")
	logger.Info("creating new coordinator",
		clog.Strings("endpoints", finalOpts.Endpoints))

	// 3. 创建内部 etcd 客户端
	// 手动转换配置结构体
	clientCfg := client.CoordinatorOptions{
		Endpoints:   finalOpts.Endpoints,
		Username:    finalOpts.Username,
		Password:    finalOpts.Password,
		Timeout:     finalOpts.Timeout,
		RetryConfig: (*client.RetryConfig)(finalOpts.RetryConfig),
	}
	etcdClient, err := client.NewEtcdClient(clientCfg)
	if err != nil {
		logger.Error("failed to create etcd client", clog.Err(err))
		return nil, err
	}

	// 4. 创建内部服务
	lockService := lockimpl.NewEtcdDistributedLock(etcdClient, "/locks")
	registryService := registryimpl.NewEtcdServiceRegistry(etcdClient, "/services")
	configService := configimpl.NewEtcdConfigCenter(etcdClient, "/configimpl")

	// 5. 组装 coordinator
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

// ===== coordinator 方法实现 =====

// Lock 获取分布式锁服务
func (c *coordinator) Lock() lock.DistributedLock {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		c.logger.Warn("尝试从已关闭的协调器获取锁服务")
		return nil
	}

	return c.lock
}

// Registry 获取服务注册发现
func (c *coordinator) Registry() types.ServiceRegistry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		c.logger.Warn("尝试从已关闭的协调器获取注册发现服务")
		return nil
	}

	return c.registry
}

// Config 获取配置中心
func (c *coordinator) Config() types.ConfigCenter {
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
