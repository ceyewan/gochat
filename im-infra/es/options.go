package es

import (
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
)

// Option is a function that configures the es provider.
type Option func(*providerOptions)

type providerOptions struct {
	logger clog.Logger
	coord  coord.Provider
}

// WithLogger sets the logger for the es provider.
func WithLogger(logger clog.Logger) Option {
	return func(o *providerOptions) {
		o.logger = logger
	}
}

// WithCoordinator sets the coordinator for the es provider.
func WithCoordinator(coord coord.Provider) Option {
	return func(o *providerOptions) {
		o.coord = coord
	}
}
