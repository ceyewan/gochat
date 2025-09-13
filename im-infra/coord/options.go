package coord

import "github.com/ceyewan/gochat/im-infra/clog"

// Options holds configuration for the coordinator.
type Options struct {
	Logger    clog.Logger
	Namespace string
}

// Option configures a coordinator.
type Option func(*Options)

// WithLogger provides a logger for the coordinator.
func WithLogger(logger clog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithNamespace sets the namespace for the coordinator.
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.Namespace = namespace
	}
}

// DefaultOptions returns default options for coordinator.
func DefaultOptions() *Options {
	return &Options{
		Logger:    clog.Namespace("coord"),
		Namespace: "coord",
	}
}
