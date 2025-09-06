package coord

import "github.com/ceyewan/gochat/im-infra/clog"

// Options holds configuration for the coordinator.
type Options struct {
	Logger clog.Logger
}

// Option configures a coordinator.
type Option func(*Options)

// WithLogger provides a logger for the coordinator.
func WithLogger(logger clog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}
