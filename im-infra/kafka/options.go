package kafka

import (
	"github.com/ceyewan/gochat/im-infra/clog"
)

// options 定义了用于定制 kafka Producer/Consumer 的选项
type options struct {
	logger clog.Logger
}

// Option 定义了用于定制 kafka Producer/Consumer 的函数。
type Option func(*options)

// WithLogger 将一个 clog.Logger 实例注入 kafka，用于记录内部日志。
func WithLogger(logger clog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithNamespace 使用指定的命名空间创建 logger
func WithNamespace(namespace string) Option {
	return func(o *options) {
		o.logger = clog.Namespace(namespace)
	}
}