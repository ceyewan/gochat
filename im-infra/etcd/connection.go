package etcd

import (
	"context"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// etcdConnectionManager 实现 ConnectionManager 接口
type etcdConnectionManager struct {
	options   *ManagerOptions
	client    *clientv3.Client
	logger    Logger
	connected bool
	mu        sync.RWMutex
}

// NewConnectionManager 创建新的连接管理器
func NewConnectionManager(options *ManagerOptions) ConnectionManager {
	return &etcdConnectionManager{
		options:   options,
		logger:    options.Logger,
		connected: false,
	}
}

// Connect 建立到 etcd 的连接
func (cm *etcdConnectionManager) Connect(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.connected && cm.client != nil {
		return nil // 已经连接
	}

	config := clientv3.Config{
		Endpoints:   cm.options.Endpoints,
		DialTimeout: cm.options.DialTimeout,
	}

	// 添加认证配置
	if cm.options.Username != "" {
		config.Username = cm.options.Username
		config.Password = cm.options.Password
	}

	client, err := clientv3.New(config)
	if err != nil {
		return WrapConnectionError(err, "failed to create etcd client")
	}

	// 测试连接
	ctxWithTimeout, cancel := context.WithTimeout(ctx, cm.options.DialTimeout)
	defer cancel()

	_, err = client.Status(ctxWithTimeout, cm.options.Endpoints[0])
	if err != nil {
		client.Close()
		return WrapConnectionError(err, "failed to connect to etcd")
	}

	cm.client = client
	cm.connected = true
	cm.logger.Info("Successfully connected to etcd")

	return nil
}

// Disconnect 断开 etcd 连接
func (cm *etcdConnectionManager) Disconnect() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.client != nil {
		err := cm.client.Close()
		cm.client = nil
		cm.connected = false
		cm.logger.Info("Disconnected from etcd")
		return err
	}

	return nil
}

// IsConnected 检查连接状态
func (cm *etcdConnectionManager) IsConnected() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connected && cm.client != nil
}

// HealthCheck 执行连接健康检查
func (cm *etcdConnectionManager) HealthCheck(ctx context.Context) error {
	cm.mu.RLock()
	client := cm.client
	connected := cm.connected
	cm.mu.RUnlock()

	if !connected || client == nil {
		return ErrNotConnected
	}

	// 检查连接状态
	for _, endpoint := range cm.options.Endpoints {
		_, err := client.Status(ctx, endpoint)
		if err != nil {
			return WrapConnectionError(err, "health check failed for endpoint: "+endpoint)
		}
	}

	return nil
}

// GetClient 获取底层 etcd 客户端
func (cm *etcdConnectionManager) GetClient() *clientv3.Client {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.client
}

// Reconnect 重新连接到 etcd
func (cm *etcdConnectionManager) Reconnect(ctx context.Context) error {
	cm.logger.Info("Attempting to reconnect to etcd...")

	// 先断开现有连接
	if err := cm.Disconnect(); err != nil {
		cm.logger.Errorf("Error during disconnect: %v", err)
	}

	// 重新连接
	return cm.Connect(ctx)
}

// GetConnectionStatus 获取连接状态信息
func (cm *etcdConnectionManager) GetConnectionStatus() ConnectionStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	status := ConnectionStatus{
		Connected: cm.connected,
		LastPing:  time.Now(),
	}

	if len(cm.options.Endpoints) > 0 {
		status.Endpoint = cm.options.Endpoints[0]
	}

	return status
}

// Close 关闭连接管理器
func (cm *etcdConnectionManager) Close() error {
	return cm.Disconnect()
}
