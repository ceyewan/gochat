package es

import (
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
)

// Option 是一个配置 es provider 的函数
type Option func(*providerOptions)

type providerOptions struct {
	logger clog.Logger
	coord  coord.Provider
}

// WithLogger 为 es provider 设置日志记录器
func WithLogger(logger clog.Logger) Option {
	return func(o *providerOptions) {
		o.logger = logger
	}
}

// WithCoordinator 为 es provider 设置协调器
func WithCoordinator(coord coord.Provider) Option {
	return func(o *providerOptions) {
		o.coord = coord
	}
}
