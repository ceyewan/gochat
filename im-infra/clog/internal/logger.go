package internal

import (
	"context"
	"log/slog"

	slogx "github.com/ceyewan/gochat/im-infra/clog/internal/clogx"
)

// logger 是 Logger 接口的内部实现。
// 它包装了一个 *slog.Logger，并提供接口方法。
type logger struct {
	sl           *slog.Logger
	levelManager *slogx.LevelManager
}

// convertFieldsToArgs 将 Field 切片转换为 slog 期望的键值对
func convertFieldsToArgs(fields ...Field) []any {
	var result []any
	for _, field := range fields {
		result = append(result, field.Key, field.Value)
	}
	return result
}

// newLogger 创建一个包装给定 slog.Logger 的新日志器实例。
// 这是工厂内部使用的构造函数。
func newLogger(sl *slog.Logger, levelManager *slogx.LevelManager) Logger {
	return &logger{
		sl:           sl,
		levelManager: levelManager,
	}
}

// Debug 以 Debug 级别记录日志。
func (l *logger) Debug(msg string, fields ...Field) {
	l.sl.Debug(msg, convertFieldsToArgs(fields...)...)
}

// Info 以 Info 级别记录日志。
func (l *logger) Info(msg string, fields ...Field) {
	l.sl.Info(msg, convertFieldsToArgs(fields...)...)
}

// Warn 以 Warn 级别记录日志。
func (l *logger) Warn(msg string, fields ...Field) {
	l.sl.Warn(msg, convertFieldsToArgs(fields...)...)
}

// Error 以 Error 级别记录日志。
func (l *logger) Error(msg string, fields ...Field) {
	l.sl.Error(msg, convertFieldsToArgs(fields...)...)
}

// DebugContext 以 Debug 级别记录带 context 的日志。
func (l *logger) DebugContext(ctx context.Context, msg string, fields ...Field) {
	l.sl.DebugContext(ctx, msg, convertFieldsToArgs(fields...)...)
}

// InfoContext 以 Info 级别记录带 context 的日志。
func (l *logger) InfoContext(ctx context.Context, msg string, fields ...Field) {
	l.sl.InfoContext(ctx, msg, convertFieldsToArgs(fields...)...)
}

// WarnContext 以 Warn 级别记录带 context 的日志。
func (l *logger) WarnContext(ctx context.Context, msg string, fields ...Field) {
	l.sl.WarnContext(ctx, msg, convertFieldsToArgs(fields...)...)
}

// ErrorContext 以 Error 级别记录带 context 的日志。
func (l *logger) ErrorContext(ctx context.Context, msg string, fields ...Field) {
	l.sl.ErrorContext(ctx, msg, convertFieldsToArgs(fields...)...)
}

// With 返回一个带有指定字段的新 Logger，这些字段会添加到所有日志记录中。
func (l *logger) With(fields ...Field) Logger {
	return &logger{
		sl:           l.sl.With(convertFieldsToArgs(fields...)...),
		levelManager: l.levelManager,
	}
}

// WithGroup 返回一个带有指定分组名的新 Logger。
// 新 Logger 添加的所有属性都会嵌套在该分组下。
func (l *logger) WithGroup(name string) Logger {
	return &logger{
		sl:           l.sl.WithGroup(name),
		levelManager: l.levelManager,
	}
}

// SetLevel 动态修改最小日志级别。
// 支持："debug"、"info"、"warn"、"error"。
func (l *logger) SetLevel(level string) error {
	return l.levelManager.SetLevel(level)
}

// Module 返回一个带有指定模块名的日志器实例。
func (l *logger) Module(name string) Logger {
	return l.With(Field{Key: "module", Value: name})
}
