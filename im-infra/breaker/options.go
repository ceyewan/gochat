package breaker

// WithLogger 为 breaker Provider 设置一个 Logger 实例
func WithLogger(logger Logger) Option {
	return func(opts *providerOptions) {
		opts.logger = logger
	}
}

// WithCoordProvider 为 breaker Provider 设置一个 CoordProvider 实例
// 这是动态加载和更新熔断策略所必需的
func WithCoordProvider(coordProvider CoordProvider) Option {
	return func(opts *providerOptions) {
		opts.coordProvider = coordProvider
	}
}