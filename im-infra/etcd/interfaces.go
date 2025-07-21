package etcd

import (
	"context"
	"io"
	"time"

	"google.golang.org/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdManager 是 etcd-grpc 组件的主要管理接口
// 它提供了统一的入口来访问所有功能组件
type EtcdManager interface {
	io.Closer

	// ServiceRegistry 返回服务注册组件
	ServiceRegistry() ServiceRegistry

	// ServiceDiscovery 返回服务发现组件
	ServiceDiscovery() ServiceDiscovery

	// DistributedLock 返回分布式锁组件
	DistributedLock() DistributedLock

	// LeaseManager 返回租约管理组件
	LeaseManager() LeaseManager

	// ConnectionManager 返回连接管理组件
	ConnectionManager() ConnectionManager

	// HealthCheck 执行健康检查
	HealthCheck(ctx context.Context) error

	// IsReady 检查管理器是否就绪
	IsReady() bool
}

// ServiceRegistry 定义服务注册接口
type ServiceRegistry interface {
	// Register 注册服务实例
	// serviceName: 服务名称
	// instanceID: 实例ID
	// addr: 服务地址
	// options: 注册选项
	Register(ctx context.Context, serviceName, instanceID, addr string, options ...RegisterOption) error

	// Deregister 注销服务实例
	Deregister(ctx context.Context, serviceName, instanceID string) error

	// UpdateService 更新服务信息
	UpdateService(ctx context.Context, serviceName, instanceID, addr string) error

	// ListServices 列出所有已注册的服务
	ListServices(ctx context.Context) ([]ServiceInfo, error)

	// GetServiceInstances 获取指定服务的所有实例
	GetServiceInstances(ctx context.Context, serviceName string) ([]ServiceInstance, error)
}

// ServiceDiscovery 定义服务发现接口
type ServiceDiscovery interface {
	// GetConnection 获取服务的 gRPC 连接
	GetConnection(ctx context.Context, serviceName string, options ...DiscoveryOption) (*grpc.ClientConn, error)

	// GetServiceEndpoints 获取服务的所有端点
	GetServiceEndpoints(ctx context.Context, serviceName string) ([]string, error)

	// WatchService 监听服务变化
	WatchService(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)

	// ResolveService 解析服务地址
	ResolveService(ctx context.Context, serviceName string) ([]ServiceInstance, error)
}

// DistributedLock 定义分布式锁接口
type DistributedLock interface {
	// Lock 获取分布式锁
	// key: 锁的键名
	// ttl: 锁的生存时间
	Lock(ctx context.Context, key string, ttl time.Duration) error

	// TryLock 尝试获取分布式锁，不阻塞
	// 返回是否成功获取锁
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Unlock 释放分布式锁
	Unlock(ctx context.Context, key string) error

	// Refresh 刷新锁的TTL
	Refresh(ctx context.Context, key string, ttl time.Duration) error

	// IsLocked 检查锁是否被持有
	IsLocked(ctx context.Context, key string) (bool, error)

	// GetLockInfo 获取锁的详细信息
	GetLockInfo(ctx context.Context, key string) (*LockInfo, error)
}

// LeaseManager 定义租约管理接口
type LeaseManager interface {
	// CreateLease 创建租约
	CreateLease(ctx context.Context, ttl int64) (clientv3.LeaseID, error)

	// KeepAlive 保持租约活跃
	KeepAlive(ctx context.Context, leaseID clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error)

	// RevokeLease 撤销租约
	RevokeLease(ctx context.Context, leaseID clientv3.LeaseID) error

	// GetLeaseInfo 获取租约信息
	GetLeaseInfo(ctx context.Context, leaseID clientv3.LeaseID) (*clientv3.LeaseTimeToLiveResponse, error)

	// ListLeases 列出所有租约
	ListLeases(ctx context.Context) ([]clientv3.LeaseStatus, error)

	// RefreshLease 刷新租约TTL
	RefreshLease(ctx context.Context, leaseID clientv3.LeaseID, ttl int64) error
}

// ConnectionManager 定义连接管理接口
type ConnectionManager interface {
	io.Closer

	// Connect 建立连接
	Connect(ctx context.Context) error

	// Disconnect 断开连接
	Disconnect() error

	// IsConnected 检查连接状态
	IsConnected() bool

	// HealthCheck 执行健康检查
	HealthCheck(ctx context.Context) error

	// GetClient 获取底层 etcd 客户端（内部使用）
	GetClient() *clientv3.Client

	// Reconnect 重新连接
	Reconnect(ctx context.Context) error

	// GetConnectionStatus 获取连接状态信息
	GetConnectionStatus() ConnectionStatus
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name      string            `json:"name"`
	Instances []ServiceInstance `json:"instances"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ServiceInstance 服务实例信息
type ServiceInstance struct {
	ID       string            `json:"id"`
	Address  string            `json:"address"`
	Metadata map[string]string `json:"metadata,omitempty"`
	LeaseID  clientv3.LeaseID  `json:"lease_id,omitempty"`
	TTL      int64             `json:"ttl,omitempty"`
}

// ServiceEvent 服务变化事件
type ServiceEvent struct {
	Type     ServiceEventType `json:"type"`
	Service  string           `json:"service"`
	Instance ServiceInstance  `json:"instance"`
}

// ServiceEventType 服务事件类型
type ServiceEventType int

const (
	ServiceEventAdd ServiceEventType = iota
	ServiceEventUpdate
	ServiceEventDelete
)

// LockInfo 锁信息
type LockInfo struct {
	Key     string            `json:"key"`
	Owner   string            `json:"owner"`
	LeaseID clientv3.LeaseID  `json:"lease_id"`
	TTL     int64             `json:"ttl"`
	Created time.Time         `json:"created"`
}

// ConnectionStatus 连接状态
type ConnectionStatus struct {
	Connected bool      `json:"connected"`
	Endpoint  string    `json:"endpoint"`
	LastPing  time.Time `json:"last_ping"`
	Error     string    `json:"error,omitempty"`
}

// RegisterOption 注册选项
type RegisterOption func(*RegisterOptions)

// RegisterOptions 注册选项配置
type RegisterOptions struct {
	TTL      int64             `json:"ttl"`
	Metadata map[string]string `json:"metadata"`
	LeaseID  clientv3.LeaseID  `json:"lease_id,omitempty"`
}

// DiscoveryOption 发现选项
type DiscoveryOption func(*DiscoveryOptions)

// DiscoveryOptions 发现选项配置
type DiscoveryOptions struct {
	LoadBalancer string            `json:"load_balancer"`
	Timeout      time.Duration     `json:"timeout"`
	Metadata     map[string]string `json:"metadata"`
}
