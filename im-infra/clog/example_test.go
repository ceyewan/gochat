package clog_test

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func ExampleDefault() {
	// Create a default logger
	logger := clog.Default()

	// Log some messages
	logger.Info("Hello, world!")
	logger.Debug("This won't show with default info level")
	logger.Warn("This is a warning")
	logger.Error("This is an error")

	// Use structured logging
	logger.Info("User logged in", "userID", 12345, "username", "alice")

	// Use context-aware logging
	ctx := context.Background()
	logger.InfoContext(ctx, "Processing request", "requestID", "req-123")
}

func ExampleNew() {
	// Create a custom configuration
	cfg := clog.Config{
		Level: "debug",
		Outputs: []clog.OutputConfig{
			{
				Format: "json",
				Writer: "stdout",
			},
		},
		EnableTraceID: true,
		TraceIDKey:    "trace_id",
		AddSource:     true,
	}

	// Create logger with custom config
	logger, err := clog.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}

	// Now debug messages will show
	logger.Debug("Debug message with custom config")
	logger.Info("Info message", "key", "value")

	// Create a child logger with additional attributes
	childLogger := logger.With("component", "auth", "version", "1.0")
	childLogger.Info("Authentication successful")

	// Create a grouped logger
	groupedLogger := logger.WithGroup("database")
	groupedLogger.Info("Connection established", "host", "localhost", "port", 5432)
}

func ExampleLogger_SetLevel() {
	logger := clog.Default()

	// Initially at info level
	logger.Debug("This won't show")
	logger.Info("This will show")

	// Change to debug level
	err := logger.SetLevel("debug")
	if err != nil {
		fmt.Printf("Failed to set level: %v\n", err)
		return
	}

	// Now debug messages will show
	logger.Debug("Now this will show")
	logger.Info("This still shows")
}
