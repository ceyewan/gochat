package etcd

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EtcdManagerFactory etcd 管理器工厂
type EtcdManagerFactory struct {
	defaultOptions *ManagerOptions
	mu             sync.RWMutex
}

// NewEtcdManagerFactory 创建新的工厂实例
func NewEtcdManagerFactory() *EtcdManagerFactory {
	return &EtcdManagerFactory{
		defaultOptions: DefaultManagerOptions(),
	}
}

// CreateManager 使用默认配置创建管理器
func (f *EtcdManagerFactory) CreateManager() (EtcdManager, error) {
	f.mu.RLock()
	options := f.copyOptions(f.defaultOptions)
	f.mu.RUnlock()

	return NewEtcdManager(options)
}

// CreateManagerWithOptions 使用指定配置创建管理器
func (f *EtcdManagerFactory) CreateManagerWithOptions(options *ManagerOptions) (EtcdManager, error) {
	if options == nil {
		return f.CreateManager()
	}

	// 验证配置
	builder := NewManagerBuilder()
	builder.options = options
	if err := builder.validateOptions(); err != nil {
		return nil, WrapConfigurationError(err, "invalid manager options")
	}

	return NewEtcdManager(options)
}

// CreateManagerWithBuilder 使用建造者创建管理器
func (f *EtcdManagerFactory) CreateManagerWithBuilder(builderFunc func(*ManagerBuilder) *ManagerBuilder) (EtcdManager, error) {
	builder := NewManagerBuilder()
	builder = builderFunc(builder)
	return builder.Build()
}

// SetDefaultOptions 设置默认配置
func (f *EtcdManagerFactory) SetDefaultOptions(options *ManagerOptions) error {
	if options == nil {
		return WrapConfigurationError(ErrInvalidConfiguration, "options cannot be nil")
	}

	// 验证配置
	builder := NewManagerBuilder()
	builder.options = options
	if err := builder.validateOptions(); err != nil {
		return WrapConfigurationError(err, "invalid default options")
	}

	f.mu.Lock()
	f.defaultOptions = f.copyOptions(options)
	f.mu.Unlock()

	return nil
}

// GetDefaultOptions 获取默认配置
func (f *EtcdManagerFactory) GetDefaultOptions() *ManagerOptions {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.copyOptions(f.defaultOptions)
}

// copyOptions 深拷贝配置选项
func (f *EtcdManagerFactory) copyOptions(src *ManagerOptions) *ManagerOptions {
	if src == nil {
		return DefaultManagerOptions()
	}

	dst := &ManagerOptions{
		Endpoints:           make([]string, len(src.Endpoints)),
		DialTimeout:         src.DialTimeout,
		Username:            src.Username,
		Password:            src.Password,
		Logger:              src.Logger,
		DefaultTTL:          src.DefaultTTL,
		ServicePrefix:       src.ServicePrefix,
		LockPrefix:          src.LockPrefix,
		MaxIdleConns:        src.MaxIdleConns,
		MaxActiveConns:      src.MaxActiveConns,
		ConnMaxLifetime:     src.ConnMaxLifetime,
		HealthCheckInterval: src.HealthCheckInterval,
		HealthCheckTimeout:  src.HealthCheckTimeout,
	}

	copy(dst.Endpoints, src.Endpoints)

	if src.TLSConfig != nil {
		dst.TLSConfig = &TLSConfig{
			CertFile:   src.TLSConfig.CertFile,
			KeyFile:    src.TLSConfig.KeyFile,
			CAFile:     src.TLSConfig.CAFile,
			ServerName: src.TLSConfig.ServerName,
			Insecure:   src.TLSConfig.Insecure,
		}
	}

	if src.RetryConfig != nil {
		dst.RetryConfig = &RetryConfig{
			MaxRetries:      src.RetryConfig.MaxRetries,
			InitialInterval: src.RetryConfig.InitialInterval,
			MaxInterval:     src.RetryConfig.MaxInterval,
			Multiplier:      src.RetryConfig.Multiplier,
		}
	}

	if src.DefaultMetadata != nil {
		dst.DefaultMetadata = make(map[string]string)
		for k, v := range src.DefaultMetadata {
			dst.DefaultMetadata[k] = v
		}
	}

	return dst
}

// 全局工厂实例
var (
	globalFactory     *EtcdManagerFactory
	globalFactoryOnce sync.Once
)

// GetGlobalFactory 获取全局工厂实例
func GetGlobalFactory() *EtcdManagerFactory {
	globalFactoryOnce.Do(func() {
		globalFactory = NewEtcdManagerFactory()
	})
	return globalFactory
}

// 便捷函数

// NewManager 使用默认配置创建管理器
func NewManager() (EtcdManager, error) {
	return GetGlobalFactory().CreateManager()
}

// NewManagerWithEndpoints 使用指定端点创建管理器
func NewManagerWithEndpoints(endpoints []string) (EtcdManager, error) {
	return NewManagerBuilder().
		WithEndpoints(endpoints).
		Build()
}

// NewManagerWithConfig 使用指定配置创建管理器
func NewManagerWithConfig(config *Config) (EtcdManager, error) {
	if config == nil {
		return NewManager()
	}

	builder := NewManagerBuilder().
		WithEndpoints(config.Endpoints).
		WithDialTimeout(config.DialTimeout)

	return builder.Build()
}

// QuickStart 快速启动，支持配置优先级：用户输入 > 配置文件 > 默认值
func QuickStart(endpoints ...string) (EtcdManager, error) {
	// 使用智能配置加载
	config, err := LoadConfig(endpoints, "")
	if err != nil {
		return nil, WrapConfigurationError(err, "加载配置失败")
	}

	// 转换为 ManagerOptions
	options := config.ToManagerOptions()

	// 创建管理器
	manager, err := NewEtcdManager(options)
	if err != nil {
		return nil, err
	}

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), config.DialTimeout)
	defer cancel()

	if err := manager.HealthCheck(ctx); err != nil {
		manager.Close()
		return nil, WrapConnectionError(err, "快速启动时连接失败")
	}

	return manager, nil
}

// 预设配置工厂函数

// NewDevelopmentManager 创建开发环境管理器
func NewDevelopmentManager() (EtcdManager, error) {
	return NewManagerBuilder().
		WithEndpoints([]string{"localhost:23791", "localhost:23792", "localhost:23793"}).
		WithDialTimeout(5*time.Second).
		WithDefaultTTL(30).
		WithHealthCheck(10*time.Second, 3*time.Second).
		WithServicePrefix("/dev/services").
		WithLockPrefix("/dev/locks").
		Build()
}

// NewProductionManager 创建生产环境管理器
func NewProductionManager(endpoints []string) (EtcdManager, error) {
	if len(endpoints) == 0 {
		return nil, WrapConfigurationError(ErrMissingEndpoints, "production manager requires endpoints")
	}

	return NewManagerBuilder().
		WithEndpoints(endpoints).
		WithDialTimeout(10*time.Second).
		WithDefaultTTL(60).
		WithHealthCheck(30*time.Second, 5*time.Second).
		WithRetryConfig(&RetryConfig{
			MaxRetries:      5,
			InitialInterval: 200 * time.Millisecond,
			MaxInterval:     5 * time.Second,
			Multiplier:      2.0,
		}).
		WithConnectionPool(20, 200, 60*time.Minute).
		Build()
}

// NewTestManager 创建测试环境管理器
func NewTestManager() (EtcdManager, error) {
	return NewManagerBuilder().
		WithEndpoints([]string{"localhost:23791", "localhost:23792", "localhost:23793"}).
		WithDialTimeout(2*time.Second).
		WithDefaultTTL(10).
		WithServicePrefix("/test/services").
		WithLockPrefix("/test/locks").
		WithHealthCheck(0, 0). // 禁用健康检查
		Build()
}

// ComponentFactory 组件工厂
type ComponentFactory struct {
	manager EtcdManager
}

// NewComponentFactory 创建组件工厂
func NewComponentFactory(manager EtcdManager) *ComponentFactory {
	return &ComponentFactory{manager: manager}
}

// CreateServiceRegistry 创建服务注册组件
func (cf *ComponentFactory) CreateServiceRegistry() ServiceRegistry {
	return cf.manager.ServiceRegistry()
}

// CreateServiceDiscovery 创建服务发现组件
func (cf *ComponentFactory) CreateServiceDiscovery() ServiceDiscovery {
	return cf.manager.ServiceDiscovery()
}

// CreateDistributedLock 创建分布式锁组件
func (cf *ComponentFactory) CreateDistributedLock() DistributedLock {
	return cf.manager.DistributedLock()
}

// CreateLeaseManager 创建租约管理组件
func (cf *ComponentFactory) CreateLeaseManager() LeaseManager {
	return cf.manager.LeaseManager()
}

// 单例管理器

var (
	singletonManager     EtcdManager
	singletonManagerOnce sync.Once
	singletonManagerMu   sync.RWMutex
)

// GetSingletonManager 获取单例管理器
func GetSingletonManager() EtcdManager {
	singletonManagerMu.RLock()
	if singletonManager != nil {
		defer singletonManagerMu.RUnlock()
		return singletonManager
	}
	singletonManagerMu.RUnlock()

	singletonManagerOnce.Do(func() {
		singletonManagerMu.Lock()
		defer singletonManagerMu.Unlock()

		manager, err := NewManager()
		if err != nil {
			panic(fmt.Sprintf("Failed to create singleton manager: %v", err))
		}
		singletonManager = manager
	})

	return singletonManager
}

// SetSingletonManager 设置单例管理器
func SetSingletonManager(manager EtcdManager) {
	singletonManagerMu.Lock()
	defer singletonManagerMu.Unlock()

	if singletonManager != nil {
		singletonManager.Close()
	}
	singletonManager = manager
}

// CloseSingletonManager 关闭单例管理器
func CloseSingletonManager() error {
	singletonManagerMu.Lock()
	defer singletonManagerMu.Unlock()

	if singletonManager != nil {
		err := singletonManager.Close()
		singletonManager = nil
		return err
	}
	return nil
}
