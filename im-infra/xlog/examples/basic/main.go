package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/xlog"
)

func main() {
	fmt.Println("=== xlog Basic Example ===\n")

	// 1. Using Default Logger
	fmt.Println("1. Default Logger:")
	logger := xlog.Default()
	logger.Info("Hello from default logger!")
	logger.Warn("This is a warning message", "component", "example")
	logger.Error("This is an error message", "error_code", 500)

	fmt.Println("\n2. Custom Logger with JSON Output:")
	// 2. Custom Logger Configuration
	cfg := xlog.Config{
		Level: "debug",
		Outputs: []xlog.OutputConfig{
			{
				Format: "json",
				Writer: "stdout",
			},
		},
		EnableTraceID: true,
		TraceIDKey:    "request_id",
		AddSource:     true,
	}

	customLogger, err := xlog.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create custom logger: %v\n", err)
		return
	}

	customLogger.Debug("Debug message with custom config")
	customLogger.Info("Info message", "user_id", 12345, "action", "login")

	fmt.Println("\n3. Context-aware Logging with TraceID:")
	// 3. Context-aware logging
	ctx := context.WithValue(context.Background(), "request_id", "req-abc-123")
	customLogger.InfoContext(ctx, "Processing user request", "endpoint", "/api/users", "method", "GET")

	fmt.Println("\n4. Structured Logging with Attributes:")
	// 4. Structured logging with child loggers
	serviceLogger := customLogger.With("service", "user-service", "version", "1.2.3")
	serviceLogger.Info("Service started successfully", "port", 8080)

	userLogger := serviceLogger.With("user_id", 67890)
	userLogger.Info("User authenticated", "username", "alice", "role", "admin")

	fmt.Println("\n5. Grouped Logging:")
	// 5. Grouped logging
	dbLogger := customLogger.WithGroup("database")
	dbLogger.Info("Connection established", "host", "localhost", "port", 5432, "database", "users")
	dbLogger.Warn("Slow query detected", "duration_ms", 1500, "query", "SELECT * FROM users")

	fmt.Println("\n6. Dynamic Level Changes:")
	// 6. Dynamic level changes
	logger.Info("Current level allows info messages")
	logger.Debug("This debug message won't show (level is info)")

	logger.SetLevel("debug")
	logger.Debug("Now debug messages will show!")

	logger.SetLevel("error")
	logger.Info("This info message won't show (level is error)")
	logger.Error("But error messages still show")

	fmt.Println("\n7. Performance Logging Example:")
	// 7. Performance logging example
	performanceLogger := customLogger.WithGroup("performance")
	start := time.Now()

	// Simulate some work
	time.Sleep(50 * time.Millisecond)

	duration := time.Since(start)
	performanceLogger.Info("Operation completed",
		"operation", "data_processing",
		"duration_ms", duration.Milliseconds(),
		"records_processed", 1000,
		"success", true,
	)

	fmt.Println("\n=== Example Complete ===")
}
