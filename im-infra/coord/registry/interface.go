package registry

import (
	"context"
	"time"
)

// EventType 事件类型
type EventType string

const (
	EventTypePut    EventType = "PUT"
	EventTypeDelete EventType = "DELETE"
)

// ServiceInfo 服务信息
type ServiceInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ServiceEvent 服务变化事件
type ServiceEvent struct {
	Type    EventType
	Service ServiceInfo
}

// ServiceRegistry 服务注册发现接口
type ServiceRegistry interface {
	// Register 注册服务，ttl 是租约的有效期
	Register(ctx context.Context, service ServiceInfo, ttl time.Duration) error
	// Unregister 注销服务
	Unregister(ctx context.Context, serviceID string) error
	// Discover 发现服务
	Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error)
	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)
}
