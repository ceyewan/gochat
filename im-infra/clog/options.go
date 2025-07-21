package clog

import "go.uber.org/zap"

// Option 是一个函数，用于设置日志记录器的配置选项。
type Option func(*Config)

// WithLevel 设置日志记录级别。
func WithLevel(level string) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithFormat 设置日志输出格式 ("json" 或 "console")。
func WithFormat(format string) Option {
	return func(c *Config) {
		c.Format = format
	}
}

// WithFileFormat 设置文件日志的输出格式 ("json" 或 "console")。
// 如果不设置，文件格式将默认使用全局的 Format 设置。
func WithFileFormat(format string) Option {
	return func(c *Config) {
		c.FileFormat = format
	}
}

// WithFilename 设置日志输出的文件名。
func WithFilename(filename string) Option {
	return func(c *Config) {
		c.Filename = filename
	}
}

// WithName 设置日志记录器的名称。
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithConsoleOutput 控制是否将日志同时输出到控制台。
func WithConsoleOutput(enable bool) Option {
	return func(c *Config) {
		c.ConsoleOutput = enable
	}
}

// WithEnableCaller 控制是否在日志中记录调用者信息。
func WithEnableCaller(enable bool) Option {
	return func(c *Config) {
		c.EnableCaller = enable
	}
}

// WithEnableColor 控制是否为控制台输出启用颜色。
func WithEnableColor(enable bool) Option {
	return func(c *Config) {
		c.EnableColor = enable
	}
}

// WithFileRotation 设置日志文件的轮转配置。
func WithFileRotation(rotationConfig *FileRotationConfig) Option {
	return func(c *Config) {
		c.FileRotation = rotationConfig
	}
}

// WithInitialFields 添加在日志记录器初始化时始终包含的字段。
func WithInitialFields(fields ...Field) Option {
	return func(c *Config) {
		c.InitialFields = append(c.InitialFields, fields...)
	}
}

// WithTraceID 添加一个跟踪ID到日志记录器的初始字段中。
func WithTraceID(traceID string) Option {
	return func(c *Config) {
		if traceID != "" {
			c.TraceID = traceID
			c.InitialFields = append(c.InitialFields, zap.String("traceID", traceID))
		}
	}
}
