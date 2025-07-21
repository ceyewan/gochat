package clog

// DefaultConfigMerger 是 ConfigMerger 接口的默认实现
type DefaultConfigMerger struct {
	defaultConfig *Config
}

// NewDefaultConfigMerger 创建一个新的默认配置合并器
func NewDefaultConfigMerger() ConfigMerger {
	return &DefaultConfigMerger{
		defaultConfig: DefaultConfig(),
	}
}

// Merge 合并两个配置，override 中的非零值会覆盖 base 中的值
func (m *DefaultConfigMerger) Merge(base, override *Config) *Config {
	if base == nil {
		base = DefaultConfig()
	}

	if override == nil {
		return base
	}

	result := *base // 复制 base 配置

	// 合并非零值
	if override.Level != "" {
		result.Level = override.Level
	}
	if override.Format != "" {
		result.Format = override.Format
	}
	if override.FileFormat != "" {
		result.FileFormat = override.FileFormat
	}
	if override.Filename != "" {
		result.Filename = override.Filename
	}
	if override.Name != "" {
		result.Name = override.Name
	}

	// 对于布尔值，需要特殊处理因为 false 也是有效值
	// 这里我们假设用户通过选项函数明确设置了这些值
	result.ConsoleOutput = override.ConsoleOutput
	result.EnableCaller = override.EnableCaller
	result.EnableColor = override.EnableColor

	if override.FileRotation != nil {
		if result.FileRotation == nil {
			result.FileRotation = &FileRotationConfig{}
		}
		// 合并文件轮转配置
		if override.FileRotation.MaxSize > 0 {
			result.FileRotation.MaxSize = override.FileRotation.MaxSize
		}
		if override.FileRotation.MaxBackups >= 0 {
			result.FileRotation.MaxBackups = override.FileRotation.MaxBackups
		}
		if override.FileRotation.MaxAge >= 0 {
			result.FileRotation.MaxAge = override.FileRotation.MaxAge
		}
		result.FileRotation.Compress = override.FileRotation.Compress
	}

	if override.InitialFields != nil {
		result.InitialFields = append(result.InitialFields, override.InitialFields...)
	}

	if override.TraceID != "" {
		result.TraceID = override.TraceID
	}

	return &result
}

// MergeWithDefault 将选项应用到默认配置上
func (m *DefaultConfigMerger) MergeWithDefault(opts ...Option) *Config {
	config := DefaultConfig()

	// 应用所有选项
	for _, opt := range opts {
		opt(config)
	}

	// 填充默认值
	config.fillDefaultConfig()

	return config
}

// SetDefaultConfig 设置默认配置
func (m *DefaultConfigMerger) SetDefaultConfig(config *Config) {
	if config != nil {
		m.defaultConfig = config
	}
}

// GetDefaultConfig 获取默认配置的副本
func (m *DefaultConfigMerger) GetDefaultConfig() *Config {
	result := *m.defaultConfig
	return &result
}
