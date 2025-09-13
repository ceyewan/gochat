package clog

import "fmt"

// Config 是 clog 组件的配置结构体
type Config struct {
	// Level 日志级别: "debug", "info", "warn", "error", "fatal"
	Level string `json:"level" yaml:"level"`
	
	// Format 输出格式: "json" (生产环境推荐) 或 "console" (开发环境推荐)
	Format string `json:"format" yaml:"format"`
	
	// Output 输出目标: "stdout", "stderr", 或文件路径
	Output string `json:"output" yaml:"output"`
	
	// AddSource 控制日志是否包含源码文件名和行号
	AddSource bool `json:"addSource" yaml:"addSource"`
	
	// EnableColor 是否启用颜色（仅 console 格式）
	EnableColor bool `json:"enableColor" yaml:"enableColor"`
	
	// RootPath 项目根目录，用于控制文件路径显示
	RootPath string `json:"rootPath,omitempty" yaml:"rootPath,omitempty"`
	
	// Rotation 日志轮转配置（仅文件输出）
	Rotation *RotationConfig `json:"rotation,omitempty" yaml:"rotation,omitempty"`
}

// RotationConfig 定义日志文件轮转设置
type RotationConfig struct {
	MaxSize    int  `json:"maxSize"`    // 单个日志文件最大尺寸(MB)
	MaxBackups int  `json:"maxBackups"` // 最多保留文件个数
	MaxAge     int  `json:"maxAge"`     // 日志保留天数
	Compress   bool `json:"compress"`   // 是否压缩轮转文件
}

// GetDefaultConfig 返回默认的日志配置
// 开发环境：console 格式，debug 级别，带颜色
// 生产环境：json 格式，info 级别，无颜色
func GetDefaultConfig(env string) *Config {
	switch env {
	case "development":
		return &Config{
			Level:       "debug",
			Format:      "console",
			Output:      "stdout",
			AddSource:   true,
			EnableColor: true,
			RootPath:    "gochat",
		}
	case "production":
		return &Config{
			Level:       "info",
			Format:      "json",
			Output:      "stdout",
			AddSource:   true,
			EnableColor: false,
			RootPath:    "",
		}
	default:
		return &Config{
			Level:       "info",
			Format:      "console",
			Output:      "stdout",
			AddSource:   true,
			EnableColor: true,
			RootPath:    "gochat",
		}
	}
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证日志级别
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLevels[c.Level] {
		return fmt.Errorf("invalid log level: %s", c.Level)
	}

	// 验证日志格式
	if c.Format != "json" && c.Format != "console" {
		return fmt.Errorf("invalid log format: %s", c.Format)
	}

	// 验证输出目标
	if c.Output == "" {
		return fmt.Errorf("log output cannot be empty")
	}

	// 验证轮转配置
	if c.Rotation != nil {
		if c.Rotation.MaxSize < 0 {
			return fmt.Errorf("rotation maxSize cannot be negative")
		}
		if c.Rotation.MaxBackups < 0 {
			return fmt.Errorf("rotation maxBackups cannot be negative")
		}
		if c.Rotation.MaxAge < 0 {
			return fmt.Errorf("rotation maxAge cannot be negative")
		}
	}

	return nil
}

