package internal

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/ceyewan/gochat/im-infra/xlog/internal/slogx"
)

// Logger defines the interface for structured logging operations.
// It provides methods for logging at different levels with structured data.
type Logger interface {
	// Debug logs a message at Debug level with optional structured attributes.
	Debug(msg string, args ...any)

	// Info logs a message at Info level with optional structured attributes.
	Info(msg string, args ...any)

	// Warn logs a message at Warn level with optional structured attributes.
	Warn(msg string, args ...any)

	// Error logs a message at Error level with optional structured attributes.
	Error(msg string, args ...any)

	// DebugContext logs a message at Debug level with context and optional structured attributes.
	DebugContext(ctx context.Context, msg string, args ...any)

	// InfoContext logs a message at Info level with context and optional structured attributes.
	InfoContext(ctx context.Context, msg string, args ...any)

	// WarnContext logs a message at Warn level with context and optional structured attributes.
	WarnContext(ctx context.Context, msg string, args ...any)

	// ErrorContext logs a message at Error level with context and optional structured attributes.
	ErrorContext(ctx context.Context, msg string, args ...any)

	// With returns a new Logger with the given attributes added to all log records.
	With(args ...any) Logger

	// WithGroup returns a new Logger with the given group name.
	// All attributes added to the returned Logger will be nested under this group.
	WithGroup(name string) Logger

	// SetLevel dynamically changes the minimum log level.
	// Supported levels: "debug", "info", "warn", "error".
	SetLevel(level string) error
}

// NewLogger creates a new Logger instance based on the provided configuration.
// This is the core factory function that assembles all components according to the config.
func NewLogger(cfg Config) (Logger, error) {
	// 1. Create LevelManager
	levelManager, err := slogx.NewLevelManager(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("failed to create level manager: %w", err)
	}

	// 2. Create handlers for each output
	var handlers []slog.Handler
	for i, outCfg := range cfg.Outputs {
		handler, err := createOutputHandler(outCfg, cfg.AddSource, levelManager)
		if err != nil {
			return nil, fmt.Errorf("failed to create handler for output %d: %w", i, err)
		}
		handlers = append(handlers, handler)
	}

	// Handle case where no outputs are configured
	if len(handlers) == 0 {
		return nil, fmt.Errorf("no valid outputs configured")
	}

	// 3. Combine handlers using TeeHandler
	var coreHandler slog.Handler
	if len(handlers) == 1 {
		coreHandler = handlers[0]
	} else {
		coreHandler = slogx.NewTeeHandler(handlers...)
	}

	// 4. Wrap with middleware if needed
	if cfg.EnableTraceID {
		coreHandler = slogx.NewContextHandler(coreHandler, cfg.TraceIDKey)
	}

	// 5. Create slog.Logger and wrap it
	sl := slog.New(coreHandler)
	return newLogger(sl, levelManager), nil
}

// createOutputHandler creates a single output handler based on the output configuration.
func createOutputHandler(outCfg OutputConfig, addSource bool, levelManager *slogx.LevelManager) (slog.Handler, error) {
	// Convert FileRotationConfig to slogx.FileRotationConfig if needed
	var slogxFileRotation *slogx.FileRotationConfig
	if outCfg.FileRotation != nil {
		slogxFileRotation = &slogx.FileRotationConfig{
			Filename:   outCfg.FileRotation.Filename,
			MaxSize:    outCfg.FileRotation.MaxSize,
			MaxAge:     outCfg.FileRotation.MaxAge,
			MaxBackups: outCfg.FileRotation.MaxBackups,
			LocalTime:  outCfg.FileRotation.LocalTime,
			Compress:   outCfg.FileRotation.Compress,
		}
	}

	// Create writer
	writer, err := slogx.NewWriter(outCfg.Writer, slogxFileRotation)
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}

	// Create handler based on format
	handler, err := createBaseHandler(writer, outCfg.Format, addSource, levelManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create base handler: %w", err)
	}

	return handler, nil
}

// createBaseHandler creates a base slog.Handler with the specified configuration.
func createBaseHandler(writer io.Writer, format string, addSource bool, levelManager *slogx.LevelManager) (slog.Handler, error) {
	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     levelManager.Leveler(),
		AddSource: addSource,
	}

	// Create handler based on format
	switch format {
	case "json":
		return slog.NewJSONHandler(writer, opts), nil
	case "text":
		return slog.NewTextHandler(writer, opts), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// NewDefaultLogger creates a logger with default configuration.
// This is used by the Default() function in the public API.
func NewDefaultLogger() Logger {
	cfg := DefaultConfig()
	logger, err := NewLogger(cfg)
	if err != nil {
		// This should never happen with default config, but if it does,
		// we'll create a basic slog logger as fallback
		sl := slog.Default()
		levelManager, _ := slogx.NewLevelManager("info")
		return newLogger(sl, levelManager)
	}
	return logger
}
