package internal

// Config 是 clog 的主配置结构体。
// 用于声明式地定义日志行为。
type Config struct {
	// Level 控制输出的最小日志级别。
	// 支持："debug"、"info"、"warn"、"error"。默认："info"。
	Level string `json:"level" yaml:"level"`

	// Outputs 定义一个或多个日志输出目标。
	// 每个输出可设置不同格式和写入方式。
	Outputs []OutputConfig `json:"outputs" yaml:"outputs"`

	// EnableTraceID 控制是否自动从 context.Context 注入 TraceID。
	// 启用后，日志器会从 context 查找 TraceID 并添加到日志记录中。
	EnableTraceID bool `json:"enableTraceID" yaml:"enableTraceID"`

	// TraceIDKey 用于从 context 提取 TraceID 的 key。
	// 默认："traceID"。
	TraceIDKey any `json:"traceIDKey" yaml:"traceIDKey"`

	// AddSource 控制日志是否包含源码文件名和行号。
	// 启用后，日志会包含源码信息，便于调试。
	AddSource bool `json:"addSource" yaml:"addSource"`
}

// OutputConfig 定义单个输出目标的配置。
type OutputConfig struct {
	// Format 日志输出格式。
	// 支持："json"、"text"。
	Format string `json:"format" yaml:"format"`

	// Writer 日志写入位置。
	// 支持："stdout"、"stderr"、"file"。
	Writer string `json:"writer" yaml:"writer"`

	// FileRotation 配置 "file" 写入方式的日志滚动。
	// Writer 非 "file" 时此配置无效。
	FileRotation *FileRotationConfig `json:"fileRotation,omitempty" yaml:"fileRotation,omitempty"`
}

// FileRotationConfig 配置日志文件滚动。
// 基于 lumberjack.v2，可靠的文件滚动方案。
type FileRotationConfig struct {
	// Filename 日志写入的文件路径。
	Filename string `json:"filename" yaml:"filename"`

	// MaxSize 单个日志文件最大 MB，超过则滚动。
	// 默认：100 MB。
	MaxSize int `json:"maxSize" yaml:"maxSize"`

	// MaxAge 日志文件最大保存天数。
	// 默认：30 天。
	MaxAge int `json:"maxAge" yaml:"maxAge"`

	// MaxBackups 最大保留的旧日志文件数。
	// 默认：10 个文件。
	MaxBackups int `json:"maxBackups" yaml:"maxBackups"`

	// LocalTime 备份文件时间戳是否使用本地时间。
	// 默认：false（UTC）。
	LocalTime bool `json:"localTime" yaml:"localTime"`

	// Compress 滚动后的日志文件是否使用 gzip 压缩。
	// 默认：false。
	Compress bool `json:"compress" yaml:"compress"`
}

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置：
//   - Level: "info"
//   - Format: "text"
//   - Writer: "stdout"
//   - TraceID: 关闭
//   - AddSource: false
func DefaultConfig() Config {
	return Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "text",
				Writer: "stdout",
			},
		},
		EnableTraceID: false,
		TraceIDKey:    "traceID",
		AddSource:     false,
	}
}
