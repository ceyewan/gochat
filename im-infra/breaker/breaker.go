package breaker

import (
	"context"
	"errors"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

var ErrBreakerOpen = errors.New("circuit breaker is open")


// Policy 定义了熔断器的行为策略
type Policy struct {
	FailureThreshold int           `json:"failureThreshold"`
	SuccessThreshold int           `json:"successThreshold"`
	OpenStateTimeout time.Duration `json:"openStateTimeout"`
}

// Config 是 breaker 组件的配置结构体
type Config struct {
	ServiceName string `json:"serviceName"`
	PoliciesPath string `json:"policiesPath"`
}

// Breaker 是熔断器的主接口
type Breaker interface {
	Do(ctx context.Context, op func() error) error
}

// Provider 是熔断器组件的提供者，负责创建和管理多个熔断器实例
type Provider interface {
	GetBreaker(name string) Breaker
	Close() error
}

// Option 是用于配置 breaker Provider 的函数式选项
type Option func(*providerOptions)

// providerOptions 是 Provider 的内部选项结构
type providerOptions struct {
	logger        Logger
	coordProvider CoordProvider
}

// Logger 直接使用 clog.Logger，保持完全兼容
type Logger = clog.Logger

// Field 直接使用 clog.Field，保持完全兼容
type Field = clog.Field


// CoordProvider 定义了配置中心的接口
type CoordProvider interface {
	Get(ctx context.Context, key string, v interface{}) error
	Set(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
	WatchPrefix(ctx context.Context, prefix string, v interface{}) (Watcher[any], error)
	List(ctx context.Context, prefix string) ([]string, error)
}

// Watcher 是用于监听配置变更的接口
type Watcher[T any] interface {
	Chan() <-chan ConfigEvent[T]
	Close()
}

// ConfigEvent 表示配置变更事件
type ConfigEvent[T any] struct {
	Type  EventType // 事件类型
	Key   string    // 配置键
	Value T         // 配置值
}

// EventType 表示事件类型
type EventType string

const (
	EventTypePut    EventType = "PUT"
	EventTypeDelete EventType = "DELETE"
)