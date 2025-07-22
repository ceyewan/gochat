package internal

import (
	"context"
	"log/slog"

	"github.com/ceyewan/gochat/im-infra/xlog/internal/slogx"
)

// logger is the internal implementation of Logger interface.
// It wraps a *slog.Logger and provides the interface methods.
type logger struct {
	sl           *slog.Logger
	levelManager *slogx.LevelManager
}

// newLogger creates a new logger instance wrapping the given slog.Logger.
// This is an internal constructor used by the factory.
func newLogger(sl *slog.Logger, levelManager *slogx.LevelManager) Logger {
	return &logger{
		sl:           sl,
		levelManager: levelManager,
	}
}

// Debug logs a message at Debug level.
func (l *logger) Debug(msg string, args ...any) {
	l.sl.Debug(msg, args...)
}

// Info logs a message at Info level.
func (l *logger) Info(msg string, args ...any) {
	l.sl.Info(msg, args...)
}

// Warn logs a message at Warn level.
func (l *logger) Warn(msg string, args ...any) {
	l.sl.Warn(msg, args...)
}

// Error logs a message at Error level.
func (l *logger) Error(msg string, args ...any) {
	l.sl.Error(msg, args...)
}

// DebugContext logs a message at Debug level with context.
func (l *logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.sl.DebugContext(ctx, msg, args...)
}

// InfoContext logs a message at Info level with context.
func (l *logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.sl.InfoContext(ctx, msg, args...)
}

// WarnContext logs a message at Warn level with context.
func (l *logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.sl.WarnContext(ctx, msg, args...)
}

// ErrorContext logs a message at Error level with context.
func (l *logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.sl.ErrorContext(ctx, msg, args...)
}

// With returns a new Logger that includes the given attributes
// in all of its output.
func (l *logger) With(args ...any) Logger {
	return &logger{
		sl:           l.sl.With(args...),
		levelManager: l.levelManager,
	}
}

// WithGroup returns a new Logger that starts a group, if name is non-empty.
// The keys of all attributes added to the Logger will be qualified by the given name.
func (l *logger) WithGroup(name string) Logger {
	return &logger{
		sl:           l.sl.WithGroup(name),
		levelManager: l.levelManager,
	}
}

// SetLevel dynamically changes the minimum log level.
// Supported levels: "debug", "info", "warn", "error".
func (l *logger) SetLevel(level string) error {
	return l.levelManager.SetLevel(level)
}
