package clog

import (
	"github.com/ceyewan/gochat/im-infra/clog/internal"
)

// Logger 定义结构化日志操作的接口。
// 提供不同级别的结构化日志方法。
type Logger = internal.Logger

// Config 是 clog 的主配置结构体。
// 用于声明式地定义日志行为。
type Config = internal.Config

// OutputConfig 定义单个输出目标的配置。
type OutputConfig = internal.OutputConfig

// FileRotationConfig 配置日志文件滚动。
// 基于 lumberjack.v2，可靠的文件滚动方案。
type FileRotationConfig = internal.FileRotationConfig

// Field 表示一个结构化日志字段
type Field struct {
	Key   string
	Value any
}

// New 根据提供的配置创建一个新的 Logger 实例。
// 用于自定义日志器的主要入口。
//
// 示例：
//
//	cfg := clog.Config{
//	  Level: "info",
//	  Outputs: []clog.OutputConfig{
//	    {Format: "json", Writer: "stdout"},
//	  },
//	}
//	logger, err := clog.New(cfg)
//	if err != nil {
//	  log.Fatal(err)
//	}
//	logger.Info("Hello world!")
func New(cfg Config) (Logger, error) {
	return internal.NewLogger(cfg)
}

// Default 返回一个带有合理默认配置的 Logger。
// 默认日志器以 Info 级别输出到 stdout，文本格式。
//
// 等价于：
//
//	cfg := clog.Config{
//	  Level: "info",
//	  Outputs: []clog.OutputConfig{
//	    {Format: "text", Writer: "stdout"},
//	  },
//	}
//	logger, _ := clog.New(cfg)
//
// 示例：
//
//	logger := clog.Default()
//	logger.Info("Hello world!")
func Default() Logger {
	return internal.NewDefaultLogger()
}

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置：
//   - Level: "info"
//   - Format: "text"
//   - Writer: "stdout"
//   - TraceID: disabled
//   - AddSource: false
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// Err 创建一个 error 类型的日志字段，使用 "error" 作为键名
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}
