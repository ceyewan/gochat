package clog

import (
	"context"
	"sync"

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

var (
	// 全局默认日志器实例
	defaultLogger Logger
	// 确保默认日志器只初始化一次
	defaultLoggerOnce sync.Once
	// 模块日志器缓存
	moduleLoggers = make(map[string]Logger)
	// 保护模块日志器缓存的读写锁
	moduleLoggersMutex sync.RWMutex
)

// getDefaultLogger 获取全局默认日志器实例，使用懒加载和单例模式
func getDefaultLogger() Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = internal.NewDefaultLogger()
	})
	return defaultLogger
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

// Debug 使用全局默认日志器以 Debug 级别记录日志
func Debug(msg string, args ...any) {
	getDefaultLogger().Debug(msg, args...)
}

// Info 使用全局默认日志器以 Info 级别记录日志
func Info(msg string, args ...any) {
	getDefaultLogger().Info(msg, args...)
}

// Warn 使用全局默认日志器以 Warn 级别记录日志
func Warn(msg string, args ...any) {
	getDefaultLogger().Warn(msg, args...)
}

// Error 使用全局默认日志器以 Error 级别记录日志
func Error(msg string, args ...any) {
	getDefaultLogger().Error(msg, args...)
}

// DebugContext 使用全局默认日志器以 Debug 级别记录带 context 的日志
func DebugContext(ctx context.Context, msg string, args ...any) {
	getDefaultLogger().DebugContext(ctx, msg, args...)
}

// InfoContext 使用全局默认日志器以 Info 级别记录带 context 的日志
func InfoContext(ctx context.Context, msg string, args ...any) {
	getDefaultLogger().InfoContext(ctx, msg, args...)
}

// WarnContext 使用全局默认日志器以 Warn 级别记录带 context 的日志
func WarnContext(ctx context.Context, msg string, args ...any) {
	getDefaultLogger().WarnContext(ctx, msg, args...)
}

// ErrorContext 使用全局默认日志器以 Error 级别记录带 context 的日志
func ErrorContext(ctx context.Context, msg string, args ...any) {
	getDefaultLogger().ErrorContext(ctx, msg, args...)
}

// Module 返回一个带有指定模块名的日志器实例。
// 对于相同的模块名，返回相同的日志器实例（单例模式）。
// 模块日志器继承默认日志器的配置，并添加 "module" 字段。
//
// 示例：
//
//	logger := clog.Module("database")
//	logger.Info("连接已建立", "host", "localhost")
//	// 输出: {"level":"info","msg":"连接已建立","module":"database","host":"localhost"}
func Module(name string) Logger {
	// 先尝试读锁获取已存在的模块日志器
	moduleLoggersMutex.RLock()
	if logger, exists := moduleLoggers[name]; exists {
		moduleLoggersMutex.RUnlock()
		return logger
	}
	moduleLoggersMutex.RUnlock()

	// 如果不存在，使用写锁创建新的模块日志器
	moduleLoggersMutex.Lock()
	defer moduleLoggersMutex.Unlock()

	// 双重检查，防止在获取写锁期间其他 goroutine 已经创建了
	if logger, exists := moduleLoggers[name]; exists {
		return logger
	}

	// 基于默认日志器创建模块日志器，添加 module 字段
	moduleLogger := getDefaultLogger().With("module", name)
	moduleLoggers[name] = moduleLogger
	return moduleLogger
}
