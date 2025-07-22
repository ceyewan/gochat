package main

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog Basic Example ===")

	// 1. 全局日志方法（推荐方式）
	fmt.Println("1. 全局日志方法:")
	clog.Info("Hello from global logger!")
	clog.Warn("This is a warning message", clog.String("component", "example"))
	clog.Error("This is an error message", clog.Int("error_code", 500))

	// 2. 带 Context 的全局日志方法
	fmt.Println("\n2. 带 Context 的全局日志:")
	ctx := context.Background()
	clog.InfoContext(ctx, "Processing request", clog.String("request_id", "req-123"))
	clog.WarnContext(ctx, "Request timeout warning", clog.String("timeout", "30s"))

	// 3. 模块日志器（推荐用于不同组件）
	fmt.Println("\n3. 模块日志器:")
	dbLogger := clog.Module("database")
	apiLogger := clog.Module("api")
	authLogger := clog.Module("auth")

	dbLogger.Info("Database connection established",
		clog.String("host", "localhost"),
		clog.Int("port", 5432))
	apiLogger.Info("API server started",
		clog.Int("port", 8080),
		clog.String("version", "v1.0"))
	authLogger.Info("User authenticated",
		clog.Int("user_id", 12345),
		clog.String("username", "alice"))

	// 4. 模块日志器的性能优势（缓存使用）
	fmt.Println("\n4. 模块日志器缓存演示:")
	// 相同模块名返回相同实例，无额外开销
	db1 := clog.Module("database")
	db2 := clog.Module("database")
	db1.Info("First database logger")
	db2.Info("Second database logger (same instance)")

	// 5. 传统方式（兼容性）
	fmt.Println("\n5. 传统方式（兼容性）:")
	logger := clog.Default()
	logger.Info("Using traditional Default() method")
	logger.Warn("This still works for backward compatibility")

	// 6. 自定义配置示例
	fmt.Println("\n6. 自定义配置:")
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

	customLogger, err := clog.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create custom logger: %v\n", err)
		return
	}

	customLogger.Debug("Debug message with custom config")
	customLogger.Info("Info message",
		clog.Int("user_id", 12345),
		clog.String("action", "login"))

	// 7. 结构化日志
	fmt.Println("\n7. 结构化日志:")
	serviceLogger := customLogger.With(
		clog.String("service", "user-service"),
		clog.String("version", "1.2.3"))
	serviceLogger.Info("Service started successfully", clog.Int("port", 8080))

	fmt.Println("\n=== Example Complete ===")
}
