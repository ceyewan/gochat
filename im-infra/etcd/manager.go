package etcd

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// etcdManager 实现 EtcdManager 接口
type etcdManager struct {
	options *ManagerOptions
	logger  Logger

	// 组件管理
	connMgr   ConnectionManager
	leaseMgr  LeaseManager
	registry  ServiceRegistry
	discovery ServiceDiscovery
	lock      DistributedLock

	// 状态管理
	ready  int32 // 原子操作标志
	closed int32 // 原子操作标志
	mu     sync.RWMutex

	// 健康检查
	healthTicker *time.Ticker
	stopHealth   chan struct{}
}

// NewEtcdManager 创建新的 etcd 管理器
func NewEtcdManager(options *ManagerOptions) (EtcdManager, error) {
	etcdLogger := clog.Module("etcd")
	etcdLogger.Info("开始创建 etcd 管理器")

	if options == nil {
		etcdLogger.Info("选项为空，使用默认选项")
		options = DefaultManagerOptions()
	}

	etcdLogger.Info("etcd 管理器配置",
		clog.Any("endpoints", options.Endpoints),
		clog.Duration("dial_timeout", options.DialTimeout),
		clog.String("service_prefix", options.ServicePrefix),
		clog.Int64("default_ttl", options.DefaultTTL))

	manager := &etcdManager{
		options:    options,
		logger:     options.Logger,
		stopHealth: make(chan struct{}),
	}

	// 初始化组件
	etcdLogger.Info("开始初始化组件")
	if err := manager.initializeComponents(); err != nil {
		etcdLogger.Error("初始化组件失败", clog.Err(err))
		return nil, WrapConfigurationError(err, "failed to initialize manager components")
	}
	etcdLogger.Info("组件初始化成功")

	// 启动健康检查
	etcdLogger.Info("启动健康检查")
	manager.startHealthCheck()

	atomic.StoreInt32(&manager.ready, 1)
	manager.logger.Info("etcd 管理器初始化成功")
	etcdLogger.Info("etcd 管理器创建完成")

	return manager, nil
}

// initializeComponents 初始化所有组件
func (m *etcdManager) initializeComponents() error {
	m.logger.Info("开始初始化 etcd 组件")

	// 创建连接管理器
	m.logger.Debug("创建连接管理器")
	m.connMgr = NewConnectionManager(m.options)

	// 建立连接
	m.logger.Info("建立 etcd 连接", clog.Any("endpoints", m.options.Endpoints))
	ctx, cancel := context.WithTimeout(context.Background(), m.options.DialTimeout)
	defer cancel()

	if err := m.connMgr.Connect(ctx); err != nil {
		m.logger.Error("连接 etcd 失败", clog.Err(err))
		return WrapConnectionError(err, "failed to connect to etcd")
	}
	m.logger.Info("etcd 连接建立成功")

	// 创建租约管理器
	m.logger.Debug("创建租约管理器")
	m.leaseMgr = NewLeaseManager(m.connMgr, m.logger)

	// 创建服务注册组件
	m.logger.Debug("创建服务注册组件")
	registry, err := NewServiceRegistryWithManager(m.connMgr, m.leaseMgr, m.logger, m.options)
	if err != nil {
		m.logger.Error("创建服务注册组件失败", clog.Err(err))
		return WrapRegistryError(err, "failed to create service registry")
	}
	m.registry = registry
	m.logger.Debug("服务注册组件创建成功")

	// 创建服务发现组件
	m.logger.Debug("创建服务发现组件")
	discovery, err := NewServiceDiscoveryWithManager(m.connMgr, m.logger, m.options)
	if err != nil {
		m.logger.Error("创建服务发现组件失败", clog.Err(err))
		return WrapDiscoveryError(err, "failed to create service discovery")
	}
	m.discovery = discovery
	m.logger.Debug("服务发现组件创建成功")

	// 创建分布式锁组件
	m.logger.Debug("创建分布式锁组件")
	m.lock = NewDistributedLock(m.connMgr, m.leaseMgr, m.logger, m.options.LockPrefix)
	m.logger.Debug("分布式锁组件创建成功")

	m.logger.Info("所有 etcd 组件初始化完成")
	return nil
}

// ServiceRegistry 返回服务注册组件
func (m *etcdManager) ServiceRegistry() ServiceRegistry {
	return m.registry
}

// ServiceDiscovery 返回服务发现组件
func (m *etcdManager) ServiceDiscovery() ServiceDiscovery {
	return m.discovery
}

// DistributedLock 返回分布式锁组件
func (m *etcdManager) DistributedLock() DistributedLock {
	return m.lock
}

// LeaseManager 返回租约管理组件
func (m *etcdManager) LeaseManager() LeaseManager {
	return m.leaseMgr
}

// ConnectionManager 返回连接管理组件
func (m *etcdManager) ConnectionManager() ConnectionManager {
	return m.connMgr
}

// HealthCheck 执行健康检查
func (m *etcdManager) HealthCheck(ctx context.Context) error {
	if !m.IsReady() {
		return ErrInvalidState
	}

	// 检查连接状态
	if err := m.connMgr.HealthCheck(ctx); err != nil {
		m.logger.Errorf("Connection health check failed: %v", err)
		return err
	}

	// 检查各组件状态
	// 这里可以添加更多的健康检查逻辑

	return nil
}

// IsReady 检查管理器是否就绪
func (m *etcdManager) IsReady() bool {
	return atomic.LoadInt32(&m.ready) == 1 && atomic.LoadInt32(&m.closed) == 0
}

// Close 关闭管理器
func (m *etcdManager) Close() error {
	if !atomic.CompareAndSwapInt32(&m.closed, 0, 1) {
		return nil // 已经关闭
	}

	m.logger.Info("开始关闭 etcd 管理器")

	// 停止健康检查
	m.logger.Debug("停止健康检查")
	m.stopHealthCheck()

	// 关闭各组件
	var lastErr error

	// 关闭分布式锁
	if m.lock != nil {
		m.logger.Debug("关闭分布式锁组件")
		if lockCloser, ok := m.lock.(*etcdDistributedLock); ok {
			if err := lockCloser.Close(); err != nil {
				m.logger.Errorf("关闭分布式锁失败: %v", err)
				lastErr = err
			} else {
				m.logger.Debug("分布式锁组件关闭成功")
			}
		}
	}

	// 关闭租约管理器
	if m.leaseMgr != nil {
		m.logger.Debug("关闭租约管理器")
		if leaseCloser, ok := m.leaseMgr.(*etcdLeaseManager); ok {
			if err := leaseCloser.Close(); err != nil {
				m.logger.Errorf("关闭租约管理器失败: %v", err)
				lastErr = err
			} else {
				m.logger.Debug("租约管理器关闭成功")
			}
		}
	}

	// 关闭连接管理器
	if m.connMgr != nil {
		m.logger.Debug("关闭连接管理器")
		if err := m.connMgr.Close(); err != nil {
			m.logger.Errorf("关闭连接管理器失败: %v", err)
			lastErr = err
		} else {
			m.logger.Debug("连接管理器关闭成功")
		}
	}

	atomic.StoreInt32(&m.ready, 0)
	if lastErr != nil {
		m.logger.Warnf("etcd 管理器关闭完成，但有错误: %v", lastErr)
	} else {
		m.logger.Info("etcd 管理器关闭成功")
	}

	return lastErr
}

// startHealthCheck 启动健康检查
func (m *etcdManager) startHealthCheck() {
	if m.options.HealthCheckInterval <= 0 {
		return // 禁用健康检查
	}

	m.healthTicker = time.NewTicker(m.options.HealthCheckInterval)

	go func() {
		defer m.healthTicker.Stop()

		for {
			select {
			case <-m.healthTicker.C:
				m.performHealthCheck()
			case <-m.stopHealth:
				return
			}
		}
	}()
}

// stopHealthCheck 停止健康检查
func (m *etcdManager) stopHealthCheck() {
	if m.healthTicker != nil {
		close(m.stopHealth)
		m.healthTicker.Stop()
	}
}

// performHealthCheck 执行健康检查
func (m *etcdManager) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), m.options.HealthCheckTimeout)
	defer cancel()

	if err := m.HealthCheck(ctx); err != nil {
		m.logger.Warnf("Health check failed: %v", err)

		// 尝试重连
		if IsConnectionError(err) {
			m.logger.Info("Attempting to reconnect...")
			if reconnectErr := m.connMgr.Reconnect(ctx); reconnectErr != nil {
				m.logger.Errorf("Reconnection failed: %v", reconnectErr)
			} else {
				m.logger.Info("Reconnection successful")
			}
		}
	}
}

// GetManagerStatus 获取管理器状态
func (m *etcdManager) GetManagerStatus() ManagerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := ManagerStatus{
		Ready:     m.IsReady(),
		Connected: m.connMgr.IsConnected(),
		StartTime: time.Now(), // 这里应该记录实际的启动时间
	}

	if m.connMgr != nil {
		status.Connection = m.connMgr.GetConnectionStatus()
	}

	return status
}

// ManagerStatus 管理器状态
type ManagerStatus struct {
	Ready      bool             `json:"ready"`
	Connected  bool             `json:"connected"`
	StartTime  time.Time        `json:"start_time"`
	Connection ConnectionStatus `json:"connection"`
}

// Restart 重启管理器
func (m *etcdManager) Restart(ctx context.Context) error {
	m.logger.Info("Restarting EtcdManager...")

	// 重新连接
	if err := m.connMgr.Reconnect(ctx); err != nil {
		return WrapConnectionError(err, "failed to reconnect during restart")
	}

	// 重新初始化组件
	if err := m.initializeComponents(); err != nil {
		return WrapConfigurationError(err, "failed to reinitialize components during restart")
	}

	m.logger.Info("EtcdManager restarted successfully")
	return nil
}

// UpdateConfiguration 更新配置
func (m *etcdManager) UpdateConfiguration(newOptions *ManagerOptions) error {
	if !m.IsReady() {
		return ErrInvalidState
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证新配置
	builder := NewManagerBuilder()
	builder.options = newOptions
	if err := builder.validateOptions(); err != nil {
		return WrapConfigurationError(err, "invalid new configuration")
	}

	// 更新配置
	oldOptions := m.options
	m.options = newOptions

	// 如果连接配置发生变化，需要重新连接
	if !m.isConnectionConfigSame(oldOptions, newOptions) {
		ctx, cancel := context.WithTimeout(context.Background(), newOptions.DialTimeout)
		defer cancel()

		if err := m.Restart(ctx); err != nil {
			// 恢复旧配置
			m.options = oldOptions
			return WrapConfigurationError(err, "failed to apply new configuration")
		}
	}

	m.logger.Info("Configuration updated successfully")
	return nil
}

// isConnectionConfigSame 检查连接配置是否相同
func (m *etcdManager) isConnectionConfigSame(old, new *ManagerOptions) bool {
	if len(old.Endpoints) != len(new.Endpoints) {
		return false
	}

	for i, endpoint := range old.Endpoints {
		if endpoint != new.Endpoints[i] {
			return false
		}
	}

	return old.DialTimeout == new.DialTimeout &&
		old.Username == new.Username &&
		old.Password == new.Password
}
