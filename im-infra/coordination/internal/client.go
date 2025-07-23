package internal

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// coordinator 是 Coordinator 接口的内部实现。
// 它包装了一个 etcd 客户端，并提供分布式协调功能。
type coordinator struct {
	client          *clientv3.Client
	config          Config
	logger          clog.Logger
	serviceRegistry ServiceRegistry
	distributedLock DistributedLock
	configCenter    ConfigCenter
	closed          bool
	mu              sync.RWMutex
}

// NewCoordinator 创建新的协调器实例
func NewCoordinator(cfg Config) (Coordinator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建 etcd 客户端配置
	etcdConfig := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	}

	// 配置认证
	if cfg.Username != "" {
		etcdConfig.Username = cfg.Username
		etcdConfig.Password = cfg.Password
	}

	// 配置 TLS
	if cfg.TLS != nil {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}

		if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		etcdConfig.TLS = tlsConfig
	}

	// 创建 etcd 客户端
	client, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	logger := clog.Module("coordination")

	c := &coordinator{
		client: client,
		config: cfg,
		logger: logger,
	}

	// 初始化各个模块
	c.serviceRegistry = newServiceRegistry(client, cfg.ServiceRegistry, logger.With(clog.String("module", "registry")))
	c.distributedLock = newDistributedLock(client, cfg.DistributedLock, logger.With(clog.String("module", "lock")))
	c.configCenter = newConfigCenter(client, cfg.ConfigCenter, logger.With(clog.String("module", "config")))

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := c.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to etcd at %v: %w\n\n"+
			"请确保 etcd 服务正在运行。快速启动 etcd 的方法：\n"+
			"Docker: docker run -d --name etcd -p 2379:2379 -p 2380:2380 quay.io/coreos/etcd:v3.5.9 /usr/local/bin/etcd --name s1 --data-dir /etcd-data --listen-client-urls http://0.0.0.0:2379 --advertise-client-urls http://0.0.0.0:2379 --listen-peer-urls http://0.0.0.0:2380 --initial-advertise-peer-urls http://0.0.0.0:2380 --initial-cluster s1=http://0.0.0.0:2380 --initial-cluster-token tkn --initial-cluster-state new\n"+
			"Homebrew: brew install etcd && etcd", cfg.Endpoints, err)
	}

	logger.Info("协调器创建成功",
		clog.Strings("endpoints", cfg.Endpoints),
		clog.Duration("dial_timeout", cfg.DialTimeout),
	)

	return c, nil
}

// NewModuleCoordinator 创建模块特定的协调器实例
func NewModuleCoordinator(base Coordinator, moduleName string) Coordinator {
	if baseCoord, ok := base.(*coordinator); ok {
		return &moduleCoordinator{
			coordinator: baseCoord,
			moduleName:  moduleName,
			logger:      clog.Module("coordination").With(clog.String("service", moduleName)),
		}
	}
	return base
}

// moduleCoordinator 是模块特定的协调器包装器
type moduleCoordinator struct {
	*coordinator
	moduleName string
	logger     clog.Logger
}

// ServiceRegistry 返回服务注册与发现实例
func (c *coordinator) ServiceRegistry() ServiceRegistry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serviceRegistry
}

// Lock 返回分布式锁实例
func (c *coordinator) Lock() DistributedLock {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.distributedLock
}

// ConfigCenter 返回配置中心实例
func (c *coordinator) ConfigCenter() ConfigCenter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configCenter
}

// Ping 检查 etcd 连接是否正常
func (c *coordinator) Ping(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("coordinator is closed")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("Ping 操作完成",
			clog.Duration("duration", duration),
		)
	}()

	// 使用 etcd 的状态检查
	_, err := c.client.Status(ctx, c.config.Endpoints[0])
	if err != nil {
		c.logger.Error("etcd Ping 失败", clog.ErrorValue(err))
		return fmt.Errorf("etcd ping failed: %w", err)
	}

	c.logger.Debug("etcd Ping 成功")
	return nil
}

// Close 关闭协调器并释放资源
func (c *coordinator) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.logger.Info("正在关闭协调器")

	var errs []error

	// 关闭各个模块
	if closer, ok := c.serviceRegistry.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close service registry: %w", err))
		}
	}

	if closer, ok := c.distributedLock.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close distributed lock: %w", err))
		}
	}

	if closer, ok := c.configCenter.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close config center: %w", err))
		}
	}

	// 关闭 etcd 客户端
	if err := c.client.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close etcd client: %w", err))
	}

	c.closed = true

	if len(errs) > 0 {
		c.logger.Error("关闭协调器时发生错误", clog.Any("errors", errs))
		return fmt.Errorf("multiple errors occurred while closing coordinator: %v", errs)
	}

	c.logger.Info("协调器已成功关闭")
	return nil
}

// 模块协调器的方法重写，添加模块上下文

// ServiceRegistry 返回服务注册与发现实例（模块版本）
func (mc *moduleCoordinator) ServiceRegistry() ServiceRegistry {
	return &moduleServiceRegistry{
		ServiceRegistry: mc.coordinator.ServiceRegistry(),
		moduleName:      mc.moduleName,
		logger:          mc.logger,
	}
}

// Lock 返回分布式锁实例（模块版本）
func (mc *moduleCoordinator) Lock() DistributedLock {
	return &moduleDistributedLock{
		DistributedLock: mc.coordinator.Lock(),
		moduleName:      mc.moduleName,
		logger:          mc.logger,
	}
}

// ConfigCenter 返回配置中心实例（模块版本）
func (mc *moduleCoordinator) ConfigCenter() ConfigCenter {
	return &moduleConfigCenter{
		ConfigCenter: mc.coordinator.ConfigCenter(),
		moduleName:   mc.moduleName,
		logger:       mc.logger,
	}
}

// moduleServiceRegistry 是模块特定的服务注册包装器
type moduleServiceRegistry struct {
	ServiceRegistry
	moduleName string
	logger     clog.Logger
}

// moduleDistributedLock 是模块特定的分布式锁包装器
type moduleDistributedLock struct {
	DistributedLock
	moduleName string
	logger     clog.Logger
}

// moduleConfigCenter 是模块特定的配置中心包装器
type moduleConfigCenter struct {
	ConfigCenter
	moduleName string
	logger     clog.Logger
}

// 辅助函数：创建带重试的 etcd 操作
func withRetry(ctx context.Context, retryConfig *RetryConfig, operation func() error, logger clog.Logger) error {
	if retryConfig == nil || retryConfig.MaxRetries == 0 {
		return operation()
	}

	var lastErr error
	interval := retryConfig.InitialInterval

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interval):
			}

			// 计算下一次重试间隔
			interval = time.Duration(float64(interval) * retryConfig.Multiplier)
			if interval > retryConfig.MaxInterval {
				interval = retryConfig.MaxInterval
			}

			// 添加随机化
			if retryConfig.RandomizationFactor > 0 {
				jitter := time.Duration(float64(interval) * retryConfig.RandomizationFactor)
				randomFactor := float64(2*time.Now().UnixNano()%2 - 1)
				interval += time.Duration(float64(jitter) * randomFactor)
			}
		}

		lastErr = operation()
		if lastErr == nil {
			if attempt > 0 {
				logger.Debug("重试操作成功", clog.Int("attempt", attempt+1))
			}
			return nil
		}

		if attempt < retryConfig.MaxRetries {
			logger.Warn("操作失败，准备重试",
				clog.ErrorValue(lastErr),
				clog.Int("attempt", attempt+1),
				clog.Int("max_retries", retryConfig.MaxRetries),
				clog.Duration("next_interval", interval),
			)
		}
	}

	logger.Error("重试操作最终失败",
		clog.ErrorValue(lastErr),
		clog.Int("max_retries", retryConfig.MaxRetries),
	)
	return fmt.Errorf("operation failed after %d retries: %w", retryConfig.MaxRetries, lastErr)
}
