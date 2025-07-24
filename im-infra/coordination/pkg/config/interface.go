package config

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination"
)

// ConfigCenter 配置中心接口
type ConfigCenter interface {
	// Get 获取配置值
	Get(ctx context.Context, key string) (interface{}, error)

	// Set 设置配置值（支持任意可序列化对象）
	Set(ctx context.Context, key string, value interface{}) error

	// Delete 删除配置
	Delete(ctx context.Context, key string) error

	// Watch 监听配置变化
	Watch(ctx context.Context, key string) (<-chan coordination.ConfigEvent, error)

	// List 列出所有配置键
	List(ctx context.Context, prefix string) ([]string, error)
}

// ConfigEvent 配置变化事件
type ConfigEvent struct {
	// Type 事件类型：PUT, DELETE
	Type coordination.EventType `json:"type"`

	// Key 配置键
	Key string `json:"key"`

	// Value 配置值（支持任意类型）
	Value interface{} `json:"value"`

	// Timestamp 事件时间
	Timestamp time.Time `json:"timestamp"`
}
