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
	moduleLoggers     sync.Map

	// 全局 TraceID Hook
	traceIDHook internal.Hook
)

// SetTraceIDHook 设置全局 TraceID 提取钩子
// 这允许用户自定义如何从 context 中提取 TraceID
func SetTraceIDHook(hook func(context.Context) (string, bool)) {
	traceIDHook = hook
}

// 预定义的 TraceID 键列表，避免每次调用时重新创建
var commonTraceIDKeys = []string{
	"traceID", // 将最常见的放在首位
	"trace_id",
	"TraceID",
	"X-Trace-ID",
	"trace-id",
	"TRACE_ID",
}

// 默认的 TraceID 提取函数
func defaultTraceIDHook(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	// 使用预定义的 key 列表来查找 TraceID，避免重复分配
	for _, key := range commonTraceIDKeys {
		if val := ctx.Value(key); val != nil {
			if str, ok := val.(string); ok && str != "" {
				return str, true
			}
		}
	}

	return "", false
}

func init() {
	// 设置默认的 TraceID 提取函数
	SetTraceIDHook(defaultTraceIDHook)
}

// WithContext 为日志操作添加 context
// 如果设置了 TraceID Hook，会自动提取并添加 TraceID
func WithContext(ctx context.Context) Logger {
	logger := getDefaultLogger()

	// 不需要额外跳过层数，因为C(ctx).Info()是直接调用
	// logger = logger.WithOptions(zap.AddCallerSkip(1))

	if traceIDHook != nil {
		if traceID, ok := traceIDHook(ctx); ok {
			return logger.With(String("traceID", traceID))
		}
	}

	return logger
}

// getDefaultLogger 获取默认日志器
func getDefaultLogger() internal.Logger {
	defaultLoggerOnce.Do(func() {
		cfg := DefaultConfig()
		logger, err := internal.NewLogger(cfg, internal.WithHook(traceIDHook))
		if err != nil {
			// 当初始化失败时，至少应在标准错误中打印一条日志
			log.Printf("clog: failed to initialize default logger: %v", err)
			logger = internal.NewFallbackLogger()
		}
		defaultLogger.Store(logger)
	})
	return defaultLogger.Load().(internal.Logger)
}

// New 根据传入的配置创建一个新的、独立的 Logger 实例。
// 这个函数是创建 logger 的主要入口，推荐在需要日志记录的组件中通过依赖注入使用它返回的 logger。
// 如果没有提供配置，使用 DefaultConfig。
func New(cfg Config) (Logger, error) {
	logger, err := internal.NewLogger(cfg, internal.WithHook(traceIDHook))
	if err != nil {
		// 返回一个备用的 fallback logger 和原始错误
		return internal.NewFallbackLogger(), err
	}
	return logger, nil
}

// Init 根据传入的配置重新初始化全局默认 logger。
// 这个函数用于运行时重新配置全局 logger，通常在从配置中心获取到最终配置后调用。
// Init 会原子性地替换全局 logger，所有后续的日志调用都会使用新的配置。
func Init(cfg Config) error {
	logger, err := internal.NewLogger(cfg, internal.WithHook(traceIDHook))
	if err != nil {
		// 返回错误，但不替换现有 logger，保持系统可用性
		return err
	}
	// 原子替换全局 logger
	defaultLogger.Store(logger)
	return nil
}

// Module 返回模块化日志器
func Module(name string) Logger {
	if cached, ok := moduleLoggers.Load(name); ok {
		return cached.(Logger)
	}

	// 创建新的模块logger，不需要额外跳过层数，因为Module().Info()是直接调用
	moduleLogger := getDefaultLogger().With(String("module", name))
	moduleLoggers.Store(name, moduleLogger)
	return moduleLogger
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
	os.Exit(1)
}

// C 返回一个带 context 的 logger，用于链式调用
// 使用示例：clog.C(ctx).Info("message", fields...)
func C(ctx context.Context) Logger {
	return WithContext(ctx)
}
