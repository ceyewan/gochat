package clog

import (
	"context"
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog/internal"
	"go.uber.org/zap"
)

// Logger 是内部 logger 的别名
type Logger = internal.Logger

// TraceIDKey 定义从 context 中提取 TraceID 的 key 类型
type TraceIDKey string

// 默认的 TraceID key
const DefaultTraceIDKey TraceIDKey = "traceID"

var (
	defaultLogger     internal.Logger
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

// 默认的 TraceID 提取函数
func defaultTraceIDHook(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	// 首先尝试默认的 TraceID key
	if val := ctx.Value(DefaultTraceIDKey); val != nil {
		if str, ok := val.(string); ok && str != "" {
			return str, true
		}
	}

	// 然后尝试常见的包含 "trace" 的 key
	commonKeys := []any{
		"traceID",
		"trace_id",
		"TraceID",
		"X-Trace-ID",
		"trace-id",
		"TRACE_ID",
	}

	for _, key := range commonKeys {
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
// 调用链：main -> clog.C -> WithContext -> logger.With -> zapLogger.Info
// 需要跳过：clog.C + WithContext + internal.Logger.Info 三层，所以 skip = 3
func WithContext(ctx context.Context) Logger {
	logger := getDefaultLogger()

	// 关键：使用 WithOptions 设置正确的 CallerSkip
	// C(ctx).Info() 比直接 Info() 多了 C() 这一层
	logger = logger.WithOptions(zap.AddCallerSkip(3))

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
			logger = internal.NewFallbackLogger()
		}
		defaultLogger = logger
	})
	return defaultLogger
}

// New 创建新的 Logger 实例
func New(config ...Config) Logger {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultConfig()
	}

	logger, err := internal.NewLogger(cfg, internal.WithHook(traceIDHook))
	if err != nil {
		return getDefaultLogger()
	}
	return logger
}

// Module 返回模块化日志器
// 调用链：main -> clog.Module -> Module -> logger.With -> zapLogger.Info
// 需要跳过：clog.Module + internal.Logger.Info 两层，所以 skip = 3
func Module(name string) Logger {
	if cached, ok := moduleLoggers.Load(name); ok {
		return cached.(Logger)
	}

	// 关键：使用 WithOptions 设置正确的 CallerSkip
	moduleLogger := getDefaultLogger().WithOptions(zap.AddCallerSkip(3)).With(String("module", name))
	moduleLoggers.Store(name, moduleLogger)
	return moduleLogger
}

// 全局日志方法
func Debug(msg string, fields ...Field) {
	getDefaultLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...Field) {
	getDefaultLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...Field) {
	getDefaultLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...Field) {
	getDefaultLogger().Error(msg, fields...)
}

// Init 初始化或重新配置全局日志器
// 用于配置重载，支持两阶段初始化模式
func Init(config Config) error {
	logger, err := internal.NewLogger(config, internal.WithHook(traceIDHook))
	if err != nil {
		return err
	}

	// 原子替换全局日志器
	defaultLogger = logger

	// 清空模块日志器缓存，强制重新创建
	moduleLoggers = sync.Map{}

	return nil
}

// C 返回一个带 context 的 logger，用于链式调用
// 使用示例：clog.C(ctx).Info("message", fields...)
func C(ctx context.Context) Logger {
	return WithContext(ctx)
}
