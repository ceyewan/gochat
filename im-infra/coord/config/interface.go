package config

import "context"

// EventType 事件类型
type EventType string

const (
	EventTypePut    EventType = "PUT"
	EventTypeDelete EventType = "DELETE"
)

// ConfigEvent 配置变化事件
type ConfigEvent struct {
	Type  EventType
	Key   string
	Value interface{}
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
