package clog

import (
	"path/filepath"
)

// 日志级别常量定义
const (
	DebugLevel = "debug" // 调试级别日志
	InfoLevel  = "info"  // 信息级别日志
	WarnLevel  = "warn"  // 警告级别日志
	ErrorLevel = "error" // 错误级别日志
	FatalLevel = "fatal" // 会导致程序退出的日志级别
)

// 日志输出格式定义
const (
	FormatJSON    = "json"    // JSON格式输出，适合生产环境
	FormatConsole = "console" // 控制台友好格式，适合开发环境
)

// Config 定义日志配置选项。
// 注意：这个结构体将在后续步骤中被函数式选项替代，但暂时保留用于平滑迁移。
type Config struct {
	Level         string              `json:"level"`          // 日志级别
	Format        string              `json:"format"`         // 全局日志格式 (主要用于控制台)
	FileFormat    string              `json:"file_format"`    // 文件日志格式 (如果未设置，则使用Format)
	Filename      string              `json:"filename"`       // 日志文件名
	Name          string              `json:"name"`           // 日志器名称
	ConsoleOutput bool                `json:"console_output"` // 是否同时输出到控制台
	EnableCaller  bool                `json:"enable_caller"`  // 是否记录调用者信息
	EnableColor   bool                `json:"enable_color"`   // 是否启用颜色
	FileRotation  *FileRotationConfig `json:"file_rotation"`  // 文件轮转配置
	InitialFields []Field             // 初始化时添加的固定字段
	TraceID       string              // 用于追踪的ID
}

// FileRotationConfig 定义日志文件轮转设置
type FileRotationConfig struct {
	MaxSize    int  `json:"max_size"`    // 单个日志文件最大尺寸(MB)
	MaxBackups int  `json:"max_backups"` // 最多保留文件个数
	MaxAge     int  `json:"max_age"`     // 日志保留天数
	Compress   bool `json:"compress"`    // 是否压缩轮转文件
}

// DefaultConfig 返回默认的日志配置
func DefaultConfig() *Config {
	return &Config{
		Level:         InfoLevel,
		Format:        FormatConsole,
		Filename:      filepath.Join("logs", "app.log"), // 默认日志文件路径
		Name:          "default",
		ConsoleOutput: true,
		EnableCaller:  true,
		EnableColor:   true,
		FileRotation: &FileRotationConfig{
			MaxSize:    100,
			MaxAge:     7,
			MaxBackups: 10,
			Compress:   false,
		},
	}
}

// fillDefaultConfig 用默认值填充未设置的配置项
func (c *Config) fillDefaultConfig() {
	defaultCfg := DefaultConfig()

	if c.Level == "" {
		c.Level = defaultCfg.Level
	}
	if c.Format == "" {
		c.Format = defaultCfg.Format
	}
	if c.Filename == "" {
		c.Filename = defaultCfg.Filename
	}
	if c.Name == "" {
		c.Name = defaultCfg.Name
	}

	if c.FileRotation == nil {
		c.FileRotation = defaultCfg.FileRotation
	} else {
		if c.FileRotation.MaxSize <= 0 {
			c.FileRotation.MaxSize = defaultCfg.FileRotation.MaxSize
		}
		if c.FileRotation.MaxAge <= 0 {
			c.FileRotation.MaxAge = defaultCfg.FileRotation.MaxAge
		}
		if c.FileRotation.MaxBackups <= 0 {
			c.FileRotation.MaxBackups = defaultCfg.FileRotation.MaxBackups
		}
	}
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	if c.Level != "" && !isValidLevel(c.Level) {
		return NewLoggerError("Validate", c.Name, ErrInvalidLevel)
	}

	if c.Format != "" && !isValidFormat(c.Format) {
		return NewLoggerError("Validate", c.Name, ErrInvalidFormat)
	}

	if c.FileFormat != "" && !isValidFormat(c.FileFormat) {
		return NewLoggerError("Validate", c.Name, ErrInvalidFormat)
	}

	if c.FileRotation != nil {
		if c.FileRotation.MaxSize <= 0 {
			return NewLoggerError("Validate", c.Name, ErrFileRotationConfig)
		}
		if c.FileRotation.MaxAge < 0 {
			return NewLoggerError("Validate", c.Name, ErrFileRotationConfig)
		}
		if c.FileRotation.MaxBackups < 0 {
			return NewLoggerError("Validate", c.Name, ErrFileRotationConfig)
		}
	}

	return nil
}

// isValidFormat 检查日志格式是否有效
func isValidFormat(format string) bool {
	return format == FormatJSON || format == FormatConsole
}
