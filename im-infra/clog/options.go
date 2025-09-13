package clog

// Options holds configuration options for the clog logger instance.
type Options struct {
	// Namespace is the root namespace for the logger, typically the service name.
	// This namespace will appear in all logs produced by this logger instance.
	Namespace string
}

// Option defines a function type for configuring clog options.
type Option func(*Options)

// WithNamespace sets the namespace for the logger.
//
// Example:
//
//	logger, err := clog.New(ctx, config, clog.WithNamespace("im-gateway"))
func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Namespace = namespace
	}
}


// DefaultOptions returns default options for clog.
func DefaultOptions() *Options {
	return &Options{
		Namespace: "",
	}
}

// ParseOptions applies the provided options and returns a configured Options struct.
func ParseOptions(opts ...Option) *Options {
	result := DefaultOptions()
	for _, opt := range opts {
		opt(result)
	}
	return result
}