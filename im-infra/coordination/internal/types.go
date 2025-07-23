package internal

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// Coordinator 定义分布式协调操作的核心接口。
// 提供服务注册发现、分布式锁、配置中心等功能的统一访问入口。
type Coordinator interface {
	// ServiceRegistry 返回服务注册与发现实例
	ServiceRegistry() ServiceRegistry

	// Lock 返回分布式锁实例
	Lock() DistributedLock

	// ConfigCenter 返回配置中心实例
	ConfigCenter() ConfigCenter

	// Ping 检查 etcd 连接是否正常
	Ping(ctx context.Context) error

	// Close 关闭协调器并释放资源
	Close() error
}

// ServiceRegistry 定义服务注册与发现的接口。
// 提供服务注册、发现、健康检查和负载均衡等功能。
type ServiceRegistry interface {
	// Register 注册服务实例
	Register(ctx context.Context, service ServiceInfo) error

	// Deregister 注销服务实例
	Deregister(ctx context.Context, serviceName, instanceID string) error

	// Discover 发现指定服务的所有健康实例
	Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error)

	// Watch 监听指定服务的实例变化
	Watch(ctx context.Context, serviceName string) (<-chan []ServiceInfo, error)

	// UpdateHealth 更新服务实例的健康状态
	UpdateHealth(ctx context.Context, serviceName, instanceID string, status HealthStatus) error

	// GetConnection 获取到指定服务的 gRPC 连接，支持负载均衡
	GetConnection(ctx context.Context, serviceName string, strategy LoadBalanceStrategy) (*grpc.ClientConn, error)
}

// DistributedLock 定义分布式锁的接口。
// 提供基础锁、可重入锁、读写锁等分布式锁定机制。
type DistributedLock interface {
	// Acquire 获取基础分布式锁
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)

	// AcquireReentrant 获取可重入分布式锁
	AcquireReentrant(ctx context.Context, key string, ttl time.Duration) (ReentrantLock, error)

	// AcquireReadLock 获取读锁
	AcquireReadLock(ctx context.Context, key string, ttl time.Duration) (Lock, error)

	// AcquireWriteLock 获取写锁
	AcquireWriteLock(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}

// ConfigCenter 定义配置中心的接口。
// 提供动态配置管理、版本控制、变更通知等功能。
type ConfigCenter interface {
	// Get 获取配置值
	Get(ctx context.Context, key string) (*ConfigValue, error)

	// Set 设置配置值，支持版本控制
	Set(ctx context.Context, key string, value interface{}, version int64) error

	// Delete 删除配置，支持版本控制
	Delete(ctx context.Context, key string, version int64) error

	// GetVersion 获取配置的当前版本号
	GetVersion(ctx context.Context, key string) (int64, error)

	// GetHistory 获取配置的历史版本
	GetHistory(ctx context.Context, key string, limit int) ([]ConfigVersion, error)

	// Watch 监听指定配置的变更
	Watch(ctx context.Context, key string) (<-chan *ConfigChange, error)

	// WatchPrefix 监听指定前缀下所有配置的变更
	WatchPrefix(ctx context.Context, prefix string) (<-chan *ConfigChange, error)
}

// Lock 定义锁实例的接口。
type Lock interface {
	// Release 释放锁
	Release(ctx context.Context) error

	// Renew 续期锁
	Renew(ctx context.Context, ttl time.Duration) error

	// IsHeld 检查锁是否仍被持有
	IsHeld(ctx context.Context) (bool, error)

	// Key 返回锁的键名
	Key() string

	// TTL 返回锁的剩余生存时间
	TTL(ctx context.Context) (time.Duration, error)
}

// ReentrantLock 定义可重入锁实例的接口。
type ReentrantLock interface {
	Lock

	// AcquireCount 返回当前锁的获取次数
	AcquireCount() int

	// Acquire 再次获取锁（可重入）
	Acquire(ctx context.Context) error

	// Release 释放一次锁，只有当获取次数为0时才真正释放
	Release(ctx context.Context) error
}

// ServiceInfo 定义服务信息结构。
type ServiceInfo struct {
	// Name 服务名称
	Name string `json:"name"`

	// InstanceID 服务实例ID，同一服务的不同实例必须有不同的ID
	InstanceID string `json:"instance_id"`

	// Address 服务地址，格式为 "host:port"
	Address string `json:"address"`

	// Metadata 服务元数据，可包含版本、标签等信息
	Metadata map[string]string `json:"metadata,omitempty"`

	// Health 服务健康状态
	Health HealthStatus `json:"health"`

	// RegisterTime 注册时间
	RegisterTime time.Time `json:"register_time"`

	// LastHeartbeat 最后心跳时间
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// HealthStatus 定义服务健康状态。
type HealthStatus int

const (
	// HealthUnknown 未知状态
	HealthUnknown HealthStatus = iota
	// HealthHealthy 健康状态
	HealthHealthy
	// HealthUnhealthy 不健康状态
	HealthUnhealthy
	// HealthMaintenance 维护状态
	HealthMaintenance
)

// String 返回健康状态的字符串表示
func (h HealthStatus) String() string {
	switch h {
	case HealthHealthy:
		return "healthy"
	case HealthUnhealthy:
		return "unhealthy"
	case HealthMaintenance:
		return "maintenance"
	default:
		return "unknown"
	}
}

// LoadBalanceStrategy 定义负载均衡策略。
type LoadBalanceStrategy int

const (
	// LoadBalanceRoundRobin 轮询策略
	LoadBalanceRoundRobin LoadBalanceStrategy = iota
	// LoadBalanceRandom 随机策略
	LoadBalanceRandom
	// LoadBalanceWeighted 加权策略
	LoadBalanceWeighted
	// LoadBalanceLeastConn 最少连接策略
	LoadBalanceLeastConn
)

// String 返回负载均衡策略的字符串表示
func (s LoadBalanceStrategy) String() string {
	switch s {
	case LoadBalanceRoundRobin:
		return "round_robin"
	case LoadBalanceRandom:
		return "random"
	case LoadBalanceWeighted:
		return "weighted"
	case LoadBalanceLeastConn:
		return "least_conn"
	default:
		return "round_robin"
	}
}

// ConfigValue 定义配置值结构。
type ConfigValue struct {
	// Key 配置键
	Key string `json:"key"`

	// Value 配置值
	Value string `json:"value"`

	// Version 配置版本号
	Version int64 `json:"version"`

	// CreateTime 创建时间
	CreateTime time.Time `json:"create_time"`

	// UpdateTime 更新时间
	UpdateTime time.Time `json:"update_time"`

	// Metadata 配置元数据
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ConfigChange 定义配置变更事件。
type ConfigChange struct {
	// Type 变更类型
	Type ConfigChangeType `json:"type"`

	// Key 配置键
	Key string `json:"key"`

	// OldValue 旧值
	OldValue *ConfigValue `json:"old_value,omitempty"`

	// NewValue 新值
	NewValue *ConfigValue `json:"new_value,omitempty"`

	// Timestamp 变更时间戳
	Timestamp time.Time `json:"timestamp"`
}

// ConfigChangeType 定义配置变更类型。
type ConfigChangeType int

const (
	// ConfigChangeCreate 创建配置
	ConfigChangeCreate ConfigChangeType = iota
	// ConfigChangeUpdate 更新配置
	ConfigChangeUpdate
	// ConfigChangeDelete 删除配置
	ConfigChangeDelete
)

// String 返回配置变更类型的字符串表示
func (t ConfigChangeType) String() string {
	switch t {
	case ConfigChangeCreate:
		return "create"
	case ConfigChangeUpdate:
		return "update"
	case ConfigChangeDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// ConfigVersion 定义配置版本信息。
type ConfigVersion struct {
	// Version 版本号
	Version int64 `json:"version"`

	// Value 配置值
	Value string `json:"value"`

	// CreateTime 创建时间
	CreateTime time.Time `json:"create_time"`

	// Author 修改者
	Author string `json:"author,omitempty"`

	// Comment 版本注释
	Comment string `json:"comment,omitempty"`
}
