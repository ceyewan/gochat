package coordination

import (
	"github.com/ceyewan/gochat/im-infra/coordination/pkg/config"
	"github.com/ceyewan/gochat/im-infra/coordination/pkg/lock"
	"github.com/ceyewan/gochat/im-infra/coordination/pkg/registry"
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
type DistributedLock = lock.DistributedLock

// Lock 锁对象接口
type Lock = lock.Lock

// ConfigCenter 配置中心接口
type ConfigCenter = config.ConfigCenter

// ConfigEvent 配置变化事件
type ConfigEvent = config.ConfigEvent

// EventType 事件类型
type EventType = config.EventType

// 事件类型常量
const (
	EventTypePut    = config.EventTypePut    // 设置事件
	EventTypeDelete = config.EventTypeDelete // 删除事件
)

// ServiceRegistry 服务注册发现接口
type ServiceRegistry = registry.ServiceRegistry

// ServiceInfo 服务信息
type ServiceInfo = registry.ServiceInfo

// ServiceEvent 服务变化事件
type ServiceEvent = registry.ServiceEvent
