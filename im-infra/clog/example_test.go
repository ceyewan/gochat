package clog_test

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// ExampleInfo 演示全局 Info 方法
func ExampleInfo() {
	clog.Info("Hello, World!")
	clog.Info("User action", "user_id", 12345, "action", "login")
}

// ExampleWarn 演示全局 Warn 方法
func ExampleWarn() {
	clog.Warn("This is a warning", "component", "example")
	clog.Warn("Rate limit approaching", "current", 850, "limit", 1000)
}

// ExampleError 演示全局 Error 方法
func ExampleError() {
	clog.Error("This is an error", "error_code", 500)
	clog.Error("Database connection failed", "host", "localhost", "port", 5432)
}

// ExampleInfoContext 演示带 Context 的全局方法
func ExampleInfoContext() {
	ctx := context.WithValue(context.Background(), "trace_id", "req-123")
	clog.InfoContext(ctx, "Processing request", "endpoint", "/api/users")
	clog.InfoContext(ctx, "Request completed", "status", 200, "duration_ms", 150)
}

// ExampleModule 演示模块日志器
func ExampleModule() {
	dbLogger := clog.Module("database")
	apiLogger := clog.Module("api")

	dbLogger.Info("Connection established", "host", "localhost", "port", 5432)
	apiLogger.Info("Server started", "port", 8080)
	dbLogger.Warn("Slow query detected", "duration_ms", 1500)
}

// ExampleDefault 演示传统的 Default 方法（向后兼容）
func ExampleDefault() {
	logger := clog.Default()
	logger.Info("Hello from default logger!")
	logger.Warn("This is a warning", "component", "example")
}

// ExampleLogger_With 演示结构化日志
func ExampleLogger_With() {
	logger := clog.Default()
	userLogger := logger.With("user_id", 12345, "session", "abc123")

	userLogger.Info("User logged in")
	userLogger.Warn("Invalid action attempted", "action", "delete_admin")
}

// ExampleNew 演示自定义配置
func ExampleNew() {
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
		AddSource:     false,
	}

	logger, err := clog.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create logger: %v", err)
		return
	}

	logger.Debug("Debug message")
	logger.Info("Application started", "version", "1.0.0")
}
