package db

import (
	"github.com/ceyewan/gochat/im-infra/clog"
)

// Options holds configuration for the database.
type Options struct {
	Logger       clog.Logger
	Namespace    string
	ComponentName string // For backward compatibility
}

// Option 定义配置选项的函数类型
type Option func(*Options)

// WithLogger 设置自定义日志记录器
//
// 示例：
//
// logger := clog.Namespace("my-app")
// database, err := db.New(ctx, cfg, db.WithLogger(logger))
func WithLogger(logger clog.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

// WithNamespace sets the namespace for the database.
func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Namespace = namespace
	}
}

// WithComponentName sets the component name for backward compatibility.
//
// Example:
//
// database, err := db.New(ctx, cfg, db.WithComponentName("user-db"))
func WithComponentName(name string) Option {
	return func(opts *Options) {
		opts.ComponentName = name
		opts.Namespace = name
	}
}

// DefaultOptions returns default options for database.
func DefaultOptions() *Options {
	return &Options{
		Logger:    clog.Namespace("db"),
		Namespace: "db",
	}
}
