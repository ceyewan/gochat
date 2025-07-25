package registryimpl

import (
	"context"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
)

// ServiceRegistry 服务注册发现接口
type ServiceRegistry interface {
	// Register 注册服务
	Register(ctx context.Context, service registry.ServiceInfo) error

	// Unregister 注销服务
	Unregister(ctx context.Context, serviceID string) error

	// Discover 发现服务
	Discover(ctx context.Context, serviceName string) ([]registry.ServiceInfo, error)

	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string) (<-chan registry.ServiceEvent, error)
}
