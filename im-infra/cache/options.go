package cache

import "github.com/ceyewan/gochat/im-infra/clog"

// Options holds configuration for the cache.
type Options struct {
	Logger    clog.Logger
	Namespace string
}

// Option configures a cache instance.
type Option func(*Options)

// WithLogger provides a logger for the cache.
func WithLogger(logger clog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithNamespace sets the namespace for the cache.
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.Namespace = namespace
	}
}

// DefaultOptions returns default options for cache.
func DefaultOptions() *Options {
	return &Options{
		Logger:    clog.Namespace("cache"),
		Namespace: "cache",
	}
}