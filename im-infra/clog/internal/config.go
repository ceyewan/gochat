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
	// 支持："stdout"、"stderr"。
	Writer string `json:"writer" yaml:"writer"`
}

// DefaultConfig 返回生产环境优化的配置。
// 专为内部使用设计，确保日志包含完整信息且保持零依赖。
func DefaultConfig() Config {
	return Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "json",
				Writer: "stdout",
			},
		},
		EnableTraceID: true,
		TraceIDKey:    "trace_id",
		AddSource:     true,
	}
}
