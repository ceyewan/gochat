package db

import (
	"github.com/ceyewan/gochat/im-infra/clog"
)

// Options 包含创建 db 实例时的可选配置
type Options struct {
	// Logger 自定义日志记录器
	Logger clog.Logger

	// ComponentName 组件名称，用于日志标识
	ComponentName string
}

// Option 定义配置选项的函数类型
type Option func(*Options)

// WithLogger 设置自定义日志记录器
//
// 示例：
//
// logger := clog.Module("my-app")
// database, err := db.New(ctx, cfg, db.WithLogger(logger))
func WithLogger(logger clog.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

// WithComponentName 设置组件名称
//
// 示例：
//
// database, err := db.New(ctx, cfg, db.WithComponentName("user-db"))
func WithComponentName(name string) Option {
	return func(opts *Options) {
		opts.ComponentName = name
	}
}
