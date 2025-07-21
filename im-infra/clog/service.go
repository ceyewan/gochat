package clog

// DefaultLoggerService 是 LoggerService 接口的默认实现
type DefaultLoggerService struct {
	registry LoggerRegistry
	merger   ConfigMerger
}

// NewDefaultLoggerService 创建一个新的默认日志服务
func NewDefaultLoggerService() LoggerService {
	return &DefaultLoggerService{
		registry: NewMemoryLoggerRegistry(),
		merger:   NewDefaultConfigMerger(),
	}
}

// NewLoggerServiceWithDeps 使用指定的依赖创建日志服务
func NewLoggerServiceWithDeps(registry LoggerRegistry, merger ConfigMerger) LoggerService {
	return &DefaultLoggerService{
		registry: registry,
		merger:   merger,
	}
}

// Init 初始化默认日志器
func (s *DefaultLoggerService) Init(opts ...Option) error {
	// 确保有 name 选项
	var nameSet bool
	for _, opt := range opts {
		tempCfg := &Config{}
		opt(tempCfg)
		if tempCfg.Name != "" {
			nameSet = true
			break
		}
	}
	if !nameSet {
		opts = append(opts, WithName("default"))
	}

	logger, err := NewLogger(opts...)
	if err != nil {
		return NewLoggerError("Init", "default", err)
	}

	return s.registry.Register("default", logger)
}

// GetOrCreateModule 获取或创建模块日志器
func (s *DefaultLoggerService) GetOrCreateModule(name string, opts ...Option) (Logger, error) {
	// 首先检查是否已存在
	if logger, exists := s.registry.Get(name); exists {
		return logger, nil
	}

	// 创建新的模块日志器
	return s.createModuleLogger(name, opts...)
}

// createModuleLogger 创建新的模块日志器，包含配置继承逻辑
func (s *DefaultLoggerService) createModuleLogger(name string, opts ...Option) (Logger, error) {
	// 添加模块名选项
	finalOpts := append([]Option{WithName(name)}, opts...)

	// 如果有默认日志器，继承其配置
	if defaultLogger := s.registry.GetDefault(); defaultLogger != nil {
		inheritedOpts := s.inheritConfigFromDefault(defaultLogger, finalOpts...)
		finalOpts = inheritedOpts
	}

	// 创建日志器
	logger, err := NewLogger(finalOpts...)
	if err != nil {
		// 创建失败时记录错误
		if defaultLogger := s.registry.GetDefault(); defaultLogger != nil {
			defaultLogger.Error("创建模块日志器失败",
				String("module", name),
				Err(err),
			)
		}
		return nil, NewLoggerError("CreateModule", name, err)
	}

	// 注册新的日志器
	if err := s.registry.Register(name, logger); err != nil {
		return nil, err
	}

	return logger, nil
}

// inheritConfigFromDefault 从默认日志器继承配置
func (s *DefaultLoggerService) inheritConfigFromDefault(defaultLogger Logger, opts ...Option) []Option {
	defaultConfig := defaultLogger.GetConfig()

	// 分析用户提供的选项，确定哪些配置用户已明确设置
	userConfig := &Config{}
	for _, opt := range opts {
		opt(userConfig)
	}

	var inheritedOpts []Option = make([]Option, len(opts))
	copy(inheritedOpts, opts)

	// 继承未被用户明确设置的配置
	if userConfig.Filename == "" && defaultConfig.Filename != "" {
		inheritedOpts = append(inheritedOpts, WithFilename(defaultConfig.Filename))
	}
	if userConfig.Level == "" && defaultConfig.Level != "" {
		inheritedOpts = append(inheritedOpts, WithLevel(defaultConfig.Level))
	}
	if userConfig.Format == "" && defaultConfig.Format != "" {
		inheritedOpts = append(inheritedOpts, WithFormat(defaultConfig.Format))
	}
	if userConfig.FileFormat == "" && defaultConfig.FileFormat != "" {
		inheritedOpts = append(inheritedOpts, WithFileFormat(defaultConfig.FileFormat))
	}

	// 继承 TraceID
	if userConfig.TraceID == "" && defaultConfig.TraceID != "" {
		inheritedOpts = append(inheritedOpts, WithTraceID(defaultConfig.TraceID))
	}

	// 继承初始字段（除了用户已经设置的）
	if len(userConfig.InitialFields) == 0 && len(defaultConfig.InitialFields) > 0 {
		inheritedOpts = append(inheritedOpts, WithInitialFields(defaultConfig.InitialFields...))
	}

	// 继承文件轮转配置
	if userConfig.FileRotation == nil && defaultConfig.FileRotation != nil {
		inheritedOpts = append(inheritedOpts, WithFileRotation(defaultConfig.FileRotation))
	}

	// 对于布尔值，需要特殊处理
	// 这里我们采用简化策略：总是继承默认配置的布尔值，除非用户通过选项明确设置
	if !s.hasBooleanOption(opts, "ConsoleOutput") {
		inheritedOpts = append(inheritedOpts, WithConsoleOutput(defaultConfig.ConsoleOutput))
	}
	if !s.hasBooleanOption(opts, "EnableColor") {
		inheritedOpts = append(inheritedOpts, WithEnableColor(defaultConfig.EnableColor))
	}
	if !s.hasBooleanOption(opts, "EnableCaller") {
		inheritedOpts = append(inheritedOpts, WithEnableCaller(defaultConfig.EnableCaller))
	}

	return inheritedOpts
}

// hasBooleanOption 检查选项中是否包含特定的布尔选项
// 这是一个简化的实现，在实际项目中可能需要更复杂的逻辑
func (s *DefaultLoggerService) hasBooleanOption(opts []Option, optionName string) bool {
	// 这里我们无法直接检查选项类型，所以返回 false
	// 在实际实现中，可能需要重构选项模式来支持这种检查
	return false
}

// GetLogger 获取指定名称的日志器
func (s *DefaultLoggerService) GetLogger(name string) (Logger, bool) {
	return s.registry.Get(name)
}

// SetDefaultLevel 设置默认日志器的级别
func (s *DefaultLoggerService) SetDefaultLevel(level string) error {
	defaultLogger := s.registry.GetDefault()
	if defaultLogger == nil {
		return NewLoggerError("SetDefaultLevel", "default", ErrLoggerNotFound)
	}
	return defaultLogger.SetLevel(level)
}

// SyncAll 同步所有日志器
func (s *DefaultLoggerService) SyncAll() error {
	names := s.registry.List()
	var lastError error

	for _, name := range names {
		if logger, exists := s.registry.Get(name); exists {
			if err := logger.Sync(); err != nil {
				lastError = err
			}
		}
	}

	return lastError
}

// GetRegistry 获取日志器注册表（用于测试或高级用法）
func (s *DefaultLoggerService) GetRegistry() LoggerRegistry {
	return s.registry
}

// GetMerger 获取配置合并器（用于测试或高级用法）
func (s *DefaultLoggerService) GetMerger() ConfigMerger {
	return s.merger
}
