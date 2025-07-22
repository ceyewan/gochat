package internal

import (
	"context"
	"log/slog"

	"github.com/ceyewan/gochat/im-infra/clog/internal/slogx"
)

// logger 是 Logger 接口的内部实现。
// 它包装了一个 *slog.Logger，并提供接口方法。
type logger struct {
	sl           *slog.Logger
	levelManager *slogx.LevelManager
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
func (l *logger) Debug(msg string, args ...any) {
	l.sl.Debug(msg, args...)
}

// Info 以 Info 级别记录日志。
func (l *logger) Info(msg string, args ...any) {
	l.sl.Info(msg, args...)
}

// Warn 以 Warn 级别记录日志。
func (l *logger) Warn(msg string, args ...any) {
	l.sl.Warn(msg, args...)
}

// Error 以 Error 级别记录日志。
func (l *logger) Error(msg string, args ...any) {
	l.sl.Error(msg, args...)
}

// DebugContext 以 Debug 级别记录带 context 的日志。
func (l *logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.sl.DebugContext(ctx, msg, args...)
}

// InfoContext 以 Info 级别记录带 context 的日志。
func (l *logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.sl.InfoContext(ctx, msg, args...)
}

// WarnContext 以 Warn 级别记录带 context 的日志。
func (l *logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.sl.WarnContext(ctx, msg, args...)
}

// ErrorContext 以 Error 级别记录带 context 的日志。
func (l *logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.sl.ErrorContext(ctx, msg, args...)
}

// With 返回一个带有指定属性的新 Logger，这些属性会添加到所有日志记录中。
func (l *logger) With(args ...any) Logger {
	return &logger{
		sl:           l.sl.With(args...),
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
