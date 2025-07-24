package coordination

import (
	"context"
	"time"
)

// Coordinator 主协调器接口
// 提供分布式锁、服务注册发现、配置中心三大核心功能的统一访问入口
type Coordinator interface {
	// Lock 获取分布式锁服务
	Lock() DistributedLock

	// Registry 获取服务注册发现
	Registry() ServiceRegistry

	// Config 获取配置中心
	Config() ConfigCenter

	// Close 关闭协调器
	Close() error
}

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// Acquire 获取互斥锁
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)

	// TryAcquire 尝试获取锁（非阻塞）
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}

// Lock 锁对象接口
type Lock interface {
	// Unlock 释放锁
	Unlock(ctx context.Context) error

	// Renew 续期锁
	Renew(ctx context.Context, ttl time.Duration) error

	// TTL 获取锁的剩余有效时间
	TTL(ctx context.Context) (time.Duration, error)

	// Key 获取锁的键
	Key() string
}

// ConfigCenter 配置中心接口
type ConfigCenter interface {
	// Get 获取配置值
	Get(ctx context.Context, key string) (interface{}, error)

	// Set 设置配置值（支持任意可序列化对象）
	Set(ctx context.Context, key string, value interface{}) error

	// Delete 删除配置
	Delete(ctx context.Context, key string) error

	// Watch 监听配置变化
	Watch(ctx context.Context, key string) (<-chan ConfigEvent, error)

	// List 列出所有配置键
	List(ctx context.Context, prefix string) ([]string, error)
}

// ConfigEvent 配置变化事件
type ConfigEvent struct {
	// Type 事件类型：PUT, DELETE
	Type EventType `json:"type"`

	// Key 配置键
	Key string `json:"key"`

	// Value 配置值（支持任意类型）
	Value interface{} `json:"value"`

	// Timestamp 事件时间
	Timestamp time.Time `json:"timestamp"`
}

// EventType 事件类型
type EventType string

const (
	// EventTypePut 设置事件
	EventTypePut EventType = "PUT"

	// EventTypeDelete 删除事件
	EventTypeDelete EventType = "DELETE"
)

// ServiceRegistry 服务注册发现接口
type ServiceRegistry interface {
	// Register 注册服务
	Register(ctx context.Context, service ServiceInfo) error

	// Unregister 注销服务
	Unregister(ctx context.Context, serviceID string) error

	// Discover 发现服务
	Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error)

	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	// ID 服务实例ID
	ID string `json:"id"`

	// Name 服务名称
	Name string `json:"name"`

	// Address 服务地址
	Address string `json:"address"`

	// Port 服务端口
	Port int `json:"port"`

	// Metadata 服务元数据
	Metadata map[string]string `json:"metadata"`

	// TTL 服务TTL
	TTL time.Duration `json:"ttl"`
}

// ServiceEvent 服务变化事件
type ServiceEvent struct {
	// Type 事件类型
	Type EventType `json:"type"`

	// Service 服务信息
	Service ServiceInfo `json:"service"`

	// Timestamp 事件时间
	Timestamp time.Time `json:"timestamp"`
}
