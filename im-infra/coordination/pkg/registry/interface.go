package registry

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination"
)

// ServiceRegistry 服务注册发现接口
type ServiceRegistry interface {
	// Register 注册服务
	Register(ctx context.Context, service coordination.ServiceInfo) error

	// Unregister 注销服务
	Unregister(ctx context.Context, serviceID string) error

	// Discover 发现服务
	Discover(ctx context.Context, serviceName string) ([]coordination.ServiceInfo, error)

	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string) (<-chan coordination.ServiceEvent, error)
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
	Type coordination.EventType `json:"type"`

	// Service 服务信息
	Service coordination.ServiceInfo `json:"service"`

	// Timestamp 事件时间
	Timestamp time.Time `json:"timestamp"`
}
