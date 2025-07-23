// Package clog 提供了一个现代化、高性能的 Go 结构化日志库。
//
// clog 基于 Go 1.21+ 标准库 `log/slog` 构建，旨在为大型项目提供一个统一、
// 可扩展且易于使用的日志解决方案。它通过简洁的 API 提供了多目标输出、
// 动态级别调整、TraceID 注入和模块化日志等核心功能。
//
// # 核心设计
//
//   - 接口驱动: 所有日志操作都通过 Logger 接口完成，便于测试和扩展。
//   - 全局与实例: 提供全局日志函数（如 clog.Info）用于快速开发和简单场景，
//     同时支持通过 clog.New() 创建完全隔离和自定义配置的日志器实例。
//   - 模块化: 通过 Module() 方法，可以从任何日志器实例创建子日志器，
//     自动继承父配置并携带模块标识，是大型项目中划分日志领域的推荐方式。
package clog

import (
	"context"
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog/internal"
)

// Logger 定义了结构化日志操作的核心接口。
// 它统一了所有日志级别方法、上下文感知方法以及创建子日志器的能力。
type Logger = internal.Logger

// Config 是 clog 的主配置结构体，用于声明式地定义日志行为。
type Config = internal.Config

// OutputConfig 定义单个输出目标的配置，如格式（json/text）和位置（stdout/file）。
type OutputConfig = internal.OutputConfig

// FileRotationConfig 配置日志文件的滚动策略，如大小、时间和备份数。
type FileRotationConfig = internal.FileRotationConfig

// Field 表示一个结构化日志字段（键值对）。
type Field = internal.Field

var (
	// defaultLogger 是一个由 sync.Once 保护的全局单例日志器实例。
	// 所有全局日志方法（如 clog.Info）都通过此实例执行。
	defaultLogger     Logger
	defaultLoggerOnce sync.Once
)

// getDefaultLogger 以线程安全的方式获取全局默认日志器实例。
// 它使用 sync.Once 确保全局实例仅被创建一次，实现了懒加载。
func getDefaultLogger() Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = internal.NewDefaultLogger()
	})
	return defaultLogger
}

// New 根据提供的配置创建一个新的 Logger 实例。
// 这是创建自定义日志器的主要入口点，适用于需要独立于全局日志器的复杂场景，
// 例如将特定模块的日志输出到独立文件或使用不同的格式。
func New(cfg Config) (Logger, error) {
	return internal.NewLogger(cfg)
}

// Default 返回全局共享的默认日志器实例。
// 此函数返回的实例与所有全局日志方法（如 clog.Info）使用的实例相同。
// 多次调用将始终返回同一个实例。
func Default() Logger {
	return getDefaultLogger()
}

// DefaultConfig 返回一个带有合理默认值的配置，可作为自定义配置的基础。
// 默认配置：
//   - Level: "info"
//   - Outputs: to "stdout" with "text" format
//   - EnableTraceID: false
//   - AddSource: false
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// Debug 使用全局默认日志器以 Debug 级别记录一条消息。
func Debug(msg string, fields ...Field) {
	getDefaultLogger().Debug(msg, fields...)
}

// Info 使用全局默认日志器以 Info 级别记录一条消息。
func Info(msg string, fields ...Field) {
	getDefaultLogger().Info(msg, fields...)
}

// Warn 使用全局默认日志器以 Warn 级别记录一条消息。
func Warn(msg string, fields ...Field) {
	getDefaultLogger().Warn(msg, fields...)
}

// Error 使用全局默认日志器以 Error 级别记录一条消息。
func Error(msg string, fields ...Field) {
	getDefaultLogger().Error(msg, fields...)
}

// DebugContext 使用全局默认日志器以 Debug 级别记录一条带上下文的消息。
// 如果日志器配置为提取 TraceID，则会自动从 context 中查找并添加。
func DebugContext(ctx context.Context, msg string, fields ...Field) {
	getDefaultLogger().DebugContext(ctx, msg, fields...)
}

// InfoContext 使用全局默认日志器以 Info 级别记录一条带上下文的消息。
func InfoContext(ctx context.Context, msg string, fields ...Field) {
	getDefaultLogger().InfoContext(ctx, msg, fields...)
}

// WarnContext 使用全局默认日志器以 Warn 级别记录一条带上下文的消息。
func WarnContext(ctx context.Context, msg string, fields ...Field) {
	getDefaultLogger().WarnContext(ctx, msg, fields...)
}

// ErrorContext 使用全局默认日志器以 Error 级别记录一条带上下文的消息。
func ErrorContext(ctx context.Context, msg string, fields ...Field) {
	getDefaultLogger().ErrorContext(ctx, msg, fields...)
}

// Module 返回一个基于全局默认日志器的、带有指定模块名的模块化日志器。
// 这是 `clog.Default().Module(name)` 的便捷写法。
//
// 模块化日志器是大型项目中进行日志分类的最佳实践。它会自动为每一条日志
// 添加 "module=<name>" 字段，便于后续的日志查询和分析。
//
// 为获得最佳性能，建议在包初始化时创建并缓存模块日志器，而非在热点路径上重复调用。
//
//	// good: at package level
//	var dbLogger = clog.Module("database")
//
//	func someFunction() {
//	    dbLogger.Info("user query", clog.Int("id", 123))
//	}
func Module(name string) Logger {
	return getDefaultLogger().Module(name)
}
