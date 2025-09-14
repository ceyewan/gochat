package clog

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"github.com/ceyewan/gochat/im-infra/clog/internal"
	"go.uber.org/zap"
)

// Logger 是内部 logger 的别名
type Logger = internal.Logger

var (
	// 使用 atomic.Value 保证 defaultLogger 的并发安全
	defaultLogger     atomic.Value
	defaultLoggerOnce sync.Once

	// exitFunc allows mocking os.Exit in tests
	exitFunc = os.Exit

	// traceID 上下文键的类型安全封装
	traceIDKey struct{}
)

// SetExitFunc sets the exit function for testing (used in tests to mock os.Exit)
func SetExitFunc(fn func(int)) {
	exitFunc = fn
	internal.SetExitFunc(fn)
}

// WithTraceID 将一个 trace_id 注入到 context 中，并返回一个新的 context

// WithTraceID 将一个 trace_id 注入到 context 中，并返回一个新的 context
// 这个函数通常在请求入口处（如 gRPC 拦截器或 HTTP 中间件）调用
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithContext 从 context 中获取一个 Logger 实例
// 如果 ctx 中包含 trace_id，返回的 Logger 会自动在每条日志中添加 "trace_id" 字段
// 这是在处理请求的函数中进行日志记录的【首选方式】
func WithContext(ctx context.Context) Logger {
	logger := getDefaultLogger()

	if ctx != nil {
		if traceID := ctx.Value(traceIDKey); traceID != nil {
			if id, ok := traceID.(string); ok && id != "" {
				return logger.With(zap.String("trace_id", id))
			}
		}
	}

	return logger
}

// getDefaultLogger 获取默认日志器
func getDefaultLogger() Logger {
	defaultLoggerOnce.Do(func() {
		cfg := GetDefaultConfig("development")
		logger, err := internal.NewLogger(cfg, "")
		if err != nil {
			// 当初始化失败时，至少应在标准错误中打印一条日志
			log.Printf("clog: failed to initialize default logger: %v", err)
			logger = internal.NewFallbackLogger()
		}
		defaultLogger.Store(logger)
	})
	return defaultLogger.Load().(Logger)
}

// New 创建一个独立的、可自定义的 Logger 实例
// 这在需要将日志输出到不同位置或使用不同格式的特殊场景下很有用
// ctx: 仅用于控制本次初始化过程的上下文。Logger 实例本身不会持有此上下文
// opts: 一系列功能选项，如 WithNamespace()，用于定制 Logger 的行为
func New(ctx context.Context, config *Config, opts ...Option) (Logger, error) {
	// 验证配置有效性
	if err := config.Validate(); err != nil {
		// 对于明显无效的配置，直接返回错误，不创建 fallback logger
		return nil, err
	}

	// 解析选项
	options := ParseOptions(opts...)
	logger, err := internal.NewLogger(config, options.Namespace)
	if err != nil {
		// 返回一个备用的 fallback logger 和原始错误
		return internal.NewFallbackLogger(), err
	}
	return logger, nil
}

// Init 初始化全局默认的日志器
// 这是最常用的方式，通常在服务的 main 函数中调用一次
// ctx: 仅用于控制本次初始化过程的上下文。Logger 实例本身不会持有此上下文
// opts: 一系列功能选项，如 WithNamespace()，用于定制 Logger 的行为
func Init(ctx context.Context, config *Config, opts ...Option) error {
	// 验证配置有效性
	if err := config.Validate(); err != nil {
		// 对于明显无效的配置，直接返回错误
		return err
	}

	// 解析选项
	options := ParseOptions(opts...)
	logger, err := internal.NewLogger(config, options.Namespace)
	if err != nil {
		// 返回错误，但不替换现有 logger，保持系统可用性
		return err
	}
	// 原子替换全局 logger
	defaultLogger.Store(logger)
	return nil
}

// Namespace 创建一个带有层次化命名空间的 Logger 实例
// 支持链式调用来构建深层的命名空间路径，如 "service.module.component"
// 这是区分不同业务模块或分层的推荐方式
func Namespace(name string) Logger {
	return getDefaultLogger().Namespace(name)
}

// 全局日志方法
func Debug(msg string, fields ...Field) {
	getDefaultLogger().WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

func Info(msg string, fields ...Field) {
	getDefaultLogger().WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

func Warn(msg string, fields ...Field) {
	getDefaultLogger().WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

// Warning 是 Warn 的别名，提供更直观的 API
func Warning(msg string, fields ...Field) {
	getDefaultLogger().WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

func Error(msg string, fields ...Field) {
	getDefaultLogger().WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// Fatal 记录 Fatal 级别的日志并退出程序
func Fatal(msg string, fields ...Field) {
	getDefaultLogger().WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
	exitFunc(1)
}

// C 是 WithContext 的简短别名，提供更简洁的 API
// 使用示例：clog.C(ctx).Info("message", fields...)
func C(ctx context.Context) Logger {
	return WithContext(ctx)
}
