package internal

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	slogx "github.com/ceyewan/gochat/im-infra/clog/internal/clogx"
)

// Field 表示一个结构化日志字段
type Field struct {
	Key   string
	Value any
}

// Logger 定义简化的结构化日志操作接口。
// 专注于生产环境常用的日志级别，提供类型安全的 Field 参数。
type Logger interface {
	// Debug 以 Debug 级别记录日志，使用类型安全的字段。
	Debug(msg string, fields ...Field)

	// Info 以 Info 级别记录日志，使用类型安全的字段。
	Info(msg string, fields ...Field)

	// Warn 以 Warn 级别记录日志，使用类型安全的字段。
	Warn(msg string, fields ...Field)

	// Error 以 Error 级别记录日志，使用类型安全的字段。
	Error(msg string, fields ...Field)

	// DebugContext 以 Debug 级别记录带 context 的日志，自动注入 TraceID。
	DebugContext(ctx context.Context, msg string, fields ...Field)

	// InfoContext 以 Info 级别记录带 context 的日志，自动注入 TraceID。
	InfoContext(ctx context.Context, msg string, fields ...Field)

	// WarnContext 以 Warn 级别记录带 context 的日志，自动注入 TraceID。
	WarnContext(ctx context.Context, msg string, fields ...Field)

	// ErrorContext 以 Error 级别记录带 context 的日志，自动注入 TraceID。
	ErrorContext(ctx context.Context, msg string, fields ...Field)

	// With 返回一个带有指定字段的新 Logger，这些字段会添加到所有日志记录中。
	With(fields ...Field) Logger

	// Module 返回一个带有指定模块名的日志器实例。
	Module(name string) Logger
}

// NewLogger 根据提供的配置创建一个新的 Logger 实例。
// 这是核心工厂函数，按配置组装所有组件。
func NewLogger(cfg Config) (Logger, error) {
	// 1. 创建 LevelManager
	levelManager, err := slogx.NewLevelManager(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("failed to create level manager: %w", err)
	}

	// 2. 为每个输出创建 handler
	var handlers []slog.Handler
	for i, outCfg := range cfg.Outputs {
		handler, err := createOutputHandler(outCfg, cfg.AddSource, levelManager)
		if err != nil {
			return nil, fmt.Errorf("failed to create handler for output %d: %w", i, err)
		}
		handlers = append(handlers, handler)
	}

	// 处理没有有效输出的情况
	if len(handlers) == 0 {
		return nil, fmt.Errorf("no valid outputs configured")
	}

	// 如果只有一个 handler，直接使用它，否则使用 TeeHandler
	var coreHandler slog.Handler
	if len(handlers) == 1 {
		coreHandler = handlers[0]
	} else {
		coreHandler = slogx.NewTeeHandler(handlers...)
	}

	// 4. 如需 TraceID 中间件则包装
	if cfg.EnableTraceID {
		coreHandler = slogx.NewContextHandler(coreHandler, cfg.TraceIDKey)
	}

	// 5. 创建 slog.Logger 并包装
	sl := slog.New(coreHandler)
	return newLogger(sl, levelManager), nil
}

// createOutputHandler 根据输出配置创建单个输出 handler。
func createOutputHandler(outCfg OutputConfig, addSource bool, levelManager *slogx.LevelManager) (slog.Handler, error) {
	// 创建 writer（只支持 stdout 和 stderr，保持零依赖）
	writer, err := slogx.NewWriter(outCfg.Writer)
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}

	// 根据格式创建 handler
	handler, err := createBaseHandler(writer, outCfg.Format, addSource, levelManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create base handler: %w", err)
	}

	return handler, nil
}

// createBaseHandler 根据指定配置创建基础 slog.Handler。
func createBaseHandler(writer io.Writer, format string, addSource bool, levelManager *slogx.LevelManager) (slog.Handler, error) {
	// 创建 handler 选项
	opts := &slog.HandlerOptions{
		Level:     levelManager.Leveler(),
		AddSource: addSource,
	}

	// 根据格式创建 handler
	switch format {
	case "json":
		return slog.NewJSONHandler(writer, opts), nil
	case "text":
		return slog.NewTextHandler(writer, opts), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// NewFallbackLogger 创建一个基本的备用日志器。
// 当优化配置创建失败时使用。
func NewFallbackLogger() Logger {
	sl := slog.Default()
	levelManager, _ := slogx.NewLevelManager("info")
	return newLogger(sl, levelManager)
}
