package xlog

import (
	"github.com/ceyewan/gochat/im-infra/xlog/internal"
)

// Logger defines the interface for structured logging operations.
// It provides methods for logging at different levels with structured data.
type Logger = internal.Logger

// Config is the main configuration structure for xlog.
// It provides a declarative way to define logging behavior.
type Config = internal.Config

// OutputConfig defines the configuration for a single output destination.
type OutputConfig = internal.OutputConfig

// FileRotationConfig configures log file rotation.
// Based on lumberjack.v2 for reliable file rotation.
type FileRotationConfig = internal.FileRotationConfig

// New creates a new Logger instance based on the provided configuration.
// This is the primary way to create a logger with custom settings.
//
// Example:
//
//	cfg := xlog.Config{
//	  Level: "info",
//	  Outputs: []xlog.OutputConfig{
//	    {Format: "json", Writer: "stdout"},
//	  },
//	}
//	logger, err := xlog.New(cfg)
//	if err != nil {
//	  log.Fatal(err)
//	}
//	logger.Info("Hello, world!")
func New(cfg Config) (Logger, error) {
	return internal.NewLogger(cfg)
}

// Default returns a Logger with sensible default configuration.
// The default logger outputs to stdout in text format at Info level.
//
// This is equivalent to:
//
//	cfg := xlog.Config{
//	  Level: "info",
//	  Outputs: []xlog.OutputConfig{
//	    {Format: "text", Writer: "stdout"},
//	  },
//	}
//	logger, _ := xlog.New(cfg)
//
// Example:
//
//	logger := xlog.Default()
//	logger.Info("Hello, world!")
func Default() Logger {
	return internal.NewDefaultLogger()
}

// DefaultConfig returns a Config with reasonable defaults.
// Default configuration:
//   - Level: "info"
//   - Format: "text"
//   - Writer: "stdout"
//   - TraceID: disabled
//   - AddSource: false
func DefaultConfig() Config {
	return internal.DefaultConfig()
}
