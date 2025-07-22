package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog Advanced Example ===\n")

	// Create logs directory
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		fmt.Printf("Failed to create logs directory: %v\n", err)
		return
	}

	// 1. 多输出配置（控制台 + 文件）
	fmt.Println("1. 多输出配置:")
	cfg := clog.Config{
		Level: "debug",
		Outputs: []clog.OutputConfig{
			// Console output in text format
			{
				Format: "text",
				Writer: "stdout",
			},
			// File output in JSON format with rotation
			{
				Format: "json",
				Writer: "file",
				FileRotation: &clog.FileRotationConfig{
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

	customLogger, err := clog.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}

	customLogger.Info("Custom logger created with multi-output configuration")

	// 2. 模块日志器演示
	fmt.Println("\n2. 模块日志器演示:")
	appLogger := clog.Module("app")
	dbLogger := clog.Module("database")
	apiLogger := clog.Module("api")
	cacheLogger := clog.Module("cache")

	appLogger.Info("Application initialized", clog.Int("pid", os.Getpid()))
	dbLogger.Info("Database connection pool created", clog.Int("max_connections", 100))
	apiLogger.Info("HTTP server starting", clog.Int("port", 8080))
	cacheLogger.Info("Redis connection established", clog.String("host", "localhost:6379"))

	// 3. 请求处理演示
	fmt.Println("\n3. 请求处理演示:")
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

	requestLogger := clog.Module("request")

	for _, req := range requests {
		ctx := context.WithValue(context.Background(), "trace_id", req.id)

		requestLogger.InfoContext(ctx, "Request started",
			clog.String("endpoint", req.endpoint),
			clog.String("method", req.method),
			clog.Int("user_id", req.userID),
		)

		// Simulate processing
		start := time.Now()
		time.Sleep(time.Duration(rand.Intn(50)+10) * time.Millisecond)
		duration := time.Since(start)

		requestLogger.InfoContext(ctx, "Request completed",
			clog.Int("status_code", 200),
			clog.Int64("duration_ms", duration.Milliseconds()),
		)
	}

	// 4. 错误处理演示
	fmt.Println("\n4. 错误处理演示:")
	errorLogger := clog.Module("error")

	errors := []struct {
		errorType string
		message   string
		code      int
		severity  string
	}{
		{"validation", "Invalid user input", 400, "warn"},
		{"database", "Connection timeout", 500, "error"},
		{"external", "Third-party API failure", 502, "error"},
	}

	for _, err := range errors {
		switch err.severity {
		case "warn":
			errorLogger.Warn(err.message,
				clog.String("error_type", err.errorType),
				clog.Int("error_code", err.code),
			)
		case "error":
			errorLogger.Error(err.message,
				clog.String("error_type", err.errorType),
				clog.Int("error_code", err.code),
				clog.Bool("requires_attention", true),
			)
		}
	}

	// 5. 性能监控
	fmt.Println("\n5. 性能监控:")
	perfLogger := clog.Module("performance")

	operations := []string{"cache_lookup", "db_query", "api_call"}
	for _, op := range operations {
		start := time.Now()
		time.Sleep(time.Duration(rand.Intn(100)+10) * time.Millisecond)
		duration := time.Since(start)

		perfLogger.Info("Operation completed",
			clog.String("operation", op),
			clog.Int64("duration_ms", duration.Milliseconds()),
			clog.Bool("success", true),
		)
	}

	// 6. 动态级别变更
	fmt.Println("\n6. 动态级别变更:")
	customLogger.Info("Current level: debug")
	customLogger.Debug("Debug message (visible)")

	customLogger.SetLevel("warn")
	fmt.Println("Changed level to warn:")
	customLogger.Debug("Debug message (hidden)")
	customLogger.Info("Info message (hidden)")
	customLogger.Warn("Warn message (visible)")

	// 7. 清理
	fmt.Println("\n7. 应用关闭:")
	clog.Info("Application shutting down gracefully")

	fmt.Println("\n=== Advanced Example Complete ===")
	fmt.Printf("Check the logs directory (%s) for the generated log files.\n", logsDir)
}
