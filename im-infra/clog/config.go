package clog

// Config 是 clog 的主配置结构体
type Config struct {
	// Level 控制输出的最小日志级别
	Level string `json:"level" yaml:"level"`

	// Format 日志格式："json" 或 "console"
	Format string `json:"format" yaml:"format"`

	// Output 输出目标："stdout" 或文件路径
	Output string `json:"output" yaml:"output"`

	// AddSource 控制日志是否包含源码文件名和行号
	AddSource bool `json:"addSource" yaml:"addSource"`

	// EnableColor 是否启用颜色（仅 console 格式）
	EnableColor bool `json:"enableColor" yaml:"enableColor"`

	// RootPath 项目根目录，用于控制文件路径显示
	// 如果设置，文件路径将只显示 rootPath 后的部分
	// 如果 rootPath 不在路径中，则显示绝对路径
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

// DefaultConfig 返回开发环境友好的默认配置
func DefaultConfig() Config {
	return Config{
		Level:       "info",
		Format:      "console",
		Output:      "stdout",
		AddSource:   true,
		EnableColor: true,
		RootPath:    "gochat",
	}
}
