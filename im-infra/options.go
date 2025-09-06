package infra

import (
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/metrics"
)

// Options 结构体持有组件的可配置依赖项。
type Options struct {
	Logger          clog.Logger
	MetricsProvider metrics.Provider
	Coordinator     coord.Provider
	ComponentName   string
}

// Option 定义了一个用于配置 Options 结构体的函数类型。
type Option func(*Options)

// WithLogger 创建一个用于设置日志记录器的 Option。
func WithLogger(logger clog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithMetricsProvider 创建一个用于设置指标提供者的 Option。
func WithMetricsProvider(provider metrics.Provider) Option {
	return func(o *Options) {
		o.MetricsProvider = provider
	}
}

// WithCoordinator 创建一个用于设置分布式协调器的 Option。
func WithCoordinator(c coord.Provider) Option {
	return func(o *Options) {
		o.Coordinator = c
	}
}

// WithComponentName 创建一个用于设置组件名称的 Option，以增强可观测性。
func WithComponentName(name string) Option {
	return func(o *Options) {
		o.ComponentName = name
	}
}
