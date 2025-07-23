// Package clog 提供了一个现代化、高性能的 Go 结构化日志库。
//
// clog 基于 Go 1.21+ 标准库 `log/slog` 构建，专为内部库使用而设计。
// 它提供了简化的 API 接口，内置最佳实践配置，特别适合微服务架构。
//
// # 核心设计原则
//
//   - 简化接口：只暴露核心必需的API，降低学习成本
//   - 内置配置：使用生产环境优化的默认配置
//   - 模块化：通过 Module() 方法支持模块化日志，适配微服务架构
//   - 高性能：模块日志器缓存，避免重复创建
//
// # 日志输出格式
//
// 日志输出包含以下信息：
//   - timestamp: 时间戳（自动添加）
//   - level: 日志级别
//   - module: 模块名称（通过 Module() 添加）
//   - msg: 日志消息
//   - fields: 结构化字段信息
//   - source: 源码文件和行号信息
package clog

import (
	"context"
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog/internal"
)

// Logger 定义了结构化日志操作的核心接口。
// 专注于生产环境常用的日志级别，简化API复杂度。
type Logger = internal.Logger

// Field 表示一个结构化日志字段（键值对）。
type Field = internal.Field

var (
	// defaultLogger 是全局单例日志器实例
	defaultLogger     internal.Logger
	defaultLoggerOnce sync.Once

	// moduleLoggers 缓存模块日志器，避免重复创建，提升性能
	moduleLoggers sync.Map
)

// getDefaultLogger 以线程安全的方式获取全局默认日志器实例。
// 使用内置的最佳实践配置：JSON格式、Info级别、启用TraceID、包含source信息。
func getDefaultLogger() internal.Logger {
	defaultLoggerOnce.Do(func() {
		cfg := internal.DefaultConfig()
		logger, err := internal.NewLogger(cfg)
		if err != nil {
			// 这不应该发生，但如果发生了，创建一个基本的备用日志器
			logger = internal.NewFallbackLogger()
		}
		defaultLogger = logger
	})
	return defaultLogger
}

// New 创建一个使用最佳实践配置的新 Logger 实例。
// 这是唯一的日志器创建入口，使用内置的生产环境优化配置：
// - JSON格式输出到stdout（便于日志收集系统处理）
// - Info级别（平衡性能和信息量）
// - 启用TraceID自动注入（微服务追踪必需）
// - 包含source信息（便于调试）
func New() internal.Logger {
	cfg := internal.DefaultConfig()
	logger, err := internal.NewLogger(cfg)
	if err != nil {
		// 返回默认logger作为备用
		return getDefaultLogger()
	}
	return logger
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
func DebugContext(ctx context.Context, msg string, fields ...Field) {
	getDefaultLogger().DebugContext(ctx, msg, fields...)
}

// InfoContext 使用全局默认日志器以 Info 级别记录一条带上下文的消息。
// 自动从 context 中提取 TraceID 并添加到日志中。
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

// Module 返回一个带有指定模块名的模块化日志器。
// 这是微服务架构中进行日志分类的最佳实践。
//
// 特性：
// - 自动为每条日志添加 "module=<name>" 字段
// - 相同模块名返回缓存的实例，避免重复创建，提升性能
// - 便于日志查询和分析
//
// 用法示例：
//
//	// 在包级别缓存模块日志器以获得最佳性能
//	var dbLogger = clog.Module("database")
//	var apiLogger = clog.Module("api")
//
//	func someFunction() {
//	    dbLogger.Info("数据库连接成功", clog.String("host", "localhost"))
//	    apiLogger.Info("处理API请求", clog.String("path", "/users"))
//	}
func Module(name string) internal.Logger {
	// 尝试从缓存获取
	if cached, ok := moduleLoggers.Load(name); ok {
		return cached.(internal.Logger)
	}

	// 创建新的模块日志器并缓存
	moduleLogger := getDefaultLogger().Module(name)
	moduleLoggers.Store(name, moduleLogger)
	return moduleLogger
}
