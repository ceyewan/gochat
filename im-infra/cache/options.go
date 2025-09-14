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

type options struct {
	logger clog.Logger
}
