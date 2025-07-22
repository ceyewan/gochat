package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ceyewan/gochat/im-infra/xlog"
)

func main() {
	fmt.Println("=== xlog Advanced Example ===\n")

	// Create logs directory
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		fmt.Printf("Failed to create logs directory: %v\n", err)
		return
	}

	fmt.Println("1. Multi-Output Logger (Console + File):")
	// 1. Multi-output configuration
	cfg := xlog.Config{
		Level: "debug",
		Outputs: []xlog.OutputConfig{
			// Console output in text format
			{
				Format: "text",
				Writer: "stdout",
			},
			// File output in JSON format with rotation
			{
				Format: "json",
				Writer: "file",
				FileRotation: &xlog.FileRotationConfig{
					Filename:   filepath.Join(logsDir, "app.log"),
					MaxSize:    10, // 10MB
					MaxAge:     7,  // 7 days
					MaxBackups: 5,  // 5 backup files
					LocalTime:  true,
					Compress:   true,
				},
			},
		},
		EnableTraceID: true,
		TraceIDKey:    "trace_id",
		AddSource:     true,
	}

	logger, err := xlog.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}

	// 2. Application startup logging
	fmt.Println("\n2. Application Startup:")
	appLogger := logger.WithGroup("app")
	appLogger.Info("Application starting", "version", "1.0.0", "env", "production")

	// 3. Service initialization
	fmt.Println("\n3. Service Initialization:")
	services := []string{"database", "redis", "message-queue", "http-server"}
	for i, service := range services {
		serviceLogger := logger.WithGroup("init").With("service", service)

		// Simulate initialization time
		start := time.Now()
		time.Sleep(time.Duration(10+i*5) * time.Millisecond)
		duration := time.Since(start)

		serviceLogger.Info("Service initialized",
			"duration_ms", duration.Milliseconds(),
			"status", "success",
		)
	}

	// 4. Request processing simulation
	fmt.Println("\n4. Request Processing:")
	requests := []struct {
		id       string
		endpoint string
		method   string
		userID   int
	}{
		{"req-001", "/api/users", "GET", 12345},
		{"req-002", "/api/orders", "POST", 67890},
		{"req-003", "/api/products", "GET", 12345},
	}

	for _, req := range requests {
		// Create context with trace ID
		ctx := context.WithValue(context.Background(), "trace_id", req.id)

		// Request logger with common attributes
		reqLogger := logger.With(
			"endpoint", req.endpoint,
			"method", req.method,
			"user_id", req.userID,
		)

		reqLogger.InfoContext(ctx, "Request started")

		// Simulate request processing
		start := time.Now()

		// Database operation
		dbLogger := reqLogger.WithGroup("database")
		dbLogger.DebugContext(ctx, "Executing query", "table", "users", "operation", "SELECT")
		time.Sleep(20 * time.Millisecond)

		// Business logic
		bizLogger := reqLogger.WithGroup("business")
		bizLogger.DebugContext(ctx, "Processing business logic", "step", "validation")
		time.Sleep(15 * time.Millisecond)

		// Response
		duration := time.Since(start)
		reqLogger.InfoContext(ctx, "Request completed",
			"status_code", 200,
			"duration_ms", duration.Milliseconds(),
			"response_size", 1024,
		)
	}

	// 5. Error handling example
	fmt.Println("\n5. Error Handling:")
	errorLogger := logger.WithGroup("error")

	// Simulate different types of errors
	errors := []struct {
		level   string
		message string
		details map[string]interface{}
	}{
		{
			"warn",
			"Rate limit approaching",
			map[string]interface{}{
				"current_rate": 850,
				"limit":        1000,
				"window":       "1m",
			},
		},
		{
			"error",
			"Database connection failed",
			map[string]interface{}{
				"host":        "db.example.com",
				"port":        5432,
				"error":       "connection timeout",
				"retry_count": 3,
			},
		},
	}

	for _, errInfo := range errors {
		ctx := context.WithValue(context.Background(), "trace_id", fmt.Sprintf("err-%d", time.Now().UnixNano()))

		var args []interface{}
		for k, v := range errInfo.details {
			args = append(args, k, v)
		}

		switch errInfo.level {
		case "warn":
			errorLogger.WarnContext(ctx, errInfo.message, args...)
		case "error":
			errorLogger.ErrorContext(ctx, errInfo.message, args...)
		}
	}

	// 6. Performance monitoring
	fmt.Println("\n6. Performance Monitoring:")
	perfLogger := logger.WithGroup("performance")

	operations := []string{"cache_lookup", "db_query", "api_call", "file_processing"}
	for _, op := range operations {
		start := time.Now()

		// Simulate operation
		time.Sleep(time.Duration(20+len(op)*2) * time.Millisecond)

		duration := time.Since(start)
		perfLogger.Info("Operation metrics",
			"operation", op,
			"duration_ms", duration.Milliseconds(),
			"success", true,
			"timestamp", start.Unix(),
		)
	}

	// 7. Dynamic level change demonstration
	fmt.Println("\n7. Dynamic Level Changes:")
	logger.Info("Current level: debug - all messages show")
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warn message")

	logger.SetLevel("warn")
	fmt.Println("Changed level to warn:")
	logger.Debug("Debug message (won't show)")
	logger.Info("Info message (won't show)")
	logger.Warn("Warn message (will show)")
	logger.Error("Error message (will show)")

	// 8. Application shutdown
	fmt.Println("\n8. Application Shutdown:")
	appLogger.Info("Application shutting down gracefully",
		"uptime_seconds", 5,
		"requests_processed", len(requests),
		"errors_encountered", len(errors),
	)

	fmt.Printf("\n=== Logs written to: %s ===\n", filepath.Join(logsDir, "app.log"))
	fmt.Println("=== Advanced Example Complete ===")
}
