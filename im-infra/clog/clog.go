package clog

import (
	"context"
	"log"
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

// 默认的 TraceID 提取函数
func defaultTraceIDHook(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	// 遍历一个统一的 key 列表来查找 TraceID，更简洁高效
	commonKeys := []string{
		"traceID", // 将最常见的放在首位
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
// 如果没有提供配置，将优先从配置管理器获取，如果配置管理器不可用则使用 DefaultConfig。
//
// 两阶段初始化支持：
// - 阶段一（降级启动）：使用默认配置或传入配置
// - 阶段二（功能完备）：从配置中心获取配置
func New(config ...Config) (Logger, error) {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	} else {
		// 从配置管理器获取当前配置（支持配置中心和降级）
		cfg = *getConfigFromManager()
	}

	logger, err := internal.NewLogger(cfg, internal.WithHook(traceIDHook))
	if err != nil {
		// 返回一个备用的 fallback logger 和原始错误
		return internal.NewFallbackLogger(), err
	}
	return logger, nil
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

func Error(msg string, fields ...Field) {
	getDefaultLogger().WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// Init 初始化或重新配置全局日志器。
// 这个函数主要用于应用程序启动时设置全局日志，或在运行时进行热重载。
// 它通过调用 New() 来创建新的 logger，然后安全地更新全局实例。
//
// 两阶段初始化支持：
// - 阶段一（降级启动）：Init() 使用默认配置，确保基础日志功能可用
// - 阶段二（功能完备）：Init() 从配置中心重新加载配置
//
// 对于可测试和可维护的代码，推荐使用 New() 创建 logger 并通过依赖注入传递，而不是依赖此全局函数。
func Init(config ...Config) error {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	} else {
		// 从配置管理器获取当前配置（支持配置中心和降级）
		cfg = *getConfigFromManager()
	}

	logger, err := New(cfg)
	if err != nil {
		return err
	}

	// 原子地替换全局日志器
	defaultLogger.Store(logger)

	// 安全地清空模块日志器缓存，强制它们使用新的 logger 配置重新创建
	moduleLoggers.Range(func(key, _ any) bool {
		moduleLoggers.Delete(key)
		return true
	})

	return nil
}

// C 返回一个带 context 的 logger，用于链式调用
// 使用示例：clog.C(ctx).Info("message", fields...)
func C(ctx context.Context) Logger {
	return WithContext(ctx)
}
