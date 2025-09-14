package cache

import "github.com/ceyewan/gochat/im-infra/clog"

// Option 定义了用于定制 cache Provider 的函数。
type Option func(*options)

// WithLogger 将一个 clog.Logger 实例注入 cache，用于记录内部日志。
func WithLogger(logger clog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithCoordProvider 注入 coord.Provider，用于从配置中心获取动态配置。
func WithCoordProvider(provider any) Option {
	return func(o *options) {
		o.coordProvider = provider
	}
}

// options 是 cache 组件的内部选项结构体
type options struct {
	logger        clog.Logger
	coordProvider any
}
