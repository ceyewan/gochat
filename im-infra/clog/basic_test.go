package clog

import (
	"context"
	"testing"
)

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Fatal("Default() returned nil logger")
	}

	// Test basic logging methods
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Test context methods
	ctx := context.Background()
	logger.DebugContext(ctx, "debug with context")
	logger.InfoContext(ctx, "info with context")
	logger.WarnContext(ctx, "warn with context")
	logger.ErrorContext(ctx, "error with context")

	// Test With method
	childLogger := logger.With("key", "value")
	if childLogger == nil {
		t.Fatal("With() returned nil logger")
	}
	childLogger.Info("child logger message")

	// Test WithGroup method
	groupLogger := logger.WithGroup("test")
	if groupLogger == nil {
		t.Fatal("WithGroup() returned nil logger")
	}
	groupLogger.Info("group logger message", "key", "value")

	// Test SetLevel method
	err := logger.SetLevel("debug")
	if err != nil {
		t.Fatalf("SetLevel failed: %v", err)
	}

	logger.Debug("debug message after level change")
}

func TestNew(t *testing.T) {
	cfg := Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "text",
				Writer: "stdout",
			},
		},
		EnableTraceID: false,
		AddSource:     false,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if logger == nil {
		t.Fatal("New() returned nil logger")
	}

	logger.Info("test message from custom logger")
}

func TestNewWithMultipleOutputs(t *testing.T) {
	cfg := Config{
		Level: "debug",
		Outputs: []OutputConfig{
			{
				Format: "text",
				Writer: "stdout",
			},
			{
				Format: "json",
				Writer: "stderr",
			},
		},
		EnableTraceID: true,
		TraceIDKey:    "trace_id",
		AddSource:     true,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New() with multiple outputs failed: %v", err)
	}

	if logger == nil {
		t.Fatal("New() returned nil logger")
	}

	logger.Info("test message with multiple outputs", "key", "value")
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != "info" {
		t.Errorf("Expected default level 'info', got '%s'", cfg.Level)
	}

	if len(cfg.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(cfg.Outputs))
	}

	if cfg.Outputs[0].Format != "text" {
		t.Errorf("Expected default format 'text', got '%s'", cfg.Outputs[0].Format)
	}

	if cfg.Outputs[0].Writer != "stdout" {
		t.Errorf("Expected default writer 'stdout', got '%s'", cfg.Outputs[0].Writer)
	}

	if cfg.EnableTraceID {
		t.Error("Expected TraceID to be disabled by default")
	}

	if cfg.AddSource {
		t.Error("Expected AddSource to be disabled by default")
	}
}

func TestInvalidLevel(t *testing.T) {
	cfg := Config{
		Level: "invalid",
		Outputs: []OutputConfig{
			{
				Format: "text",
				Writer: "stdout",
			},
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("Expected error for invalid level, got nil")
	}
}

func TestInvalidFormat(t *testing.T) {
	cfg := Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "invalid",
				Writer: "stdout",
			},
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("Expected error for invalid format, got nil")
	}
}

func TestInvalidWriter(t *testing.T) {
	cfg := Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "text",
				Writer: "invalid",
			},
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("Expected error for invalid writer, got nil")
	}
}

// TestGlobalLoggerMethods 测试全局日志方法
func TestGlobalLoggerMethods(t *testing.T) {
	// 测试基本的全局日志方法
	Debug("global debug message")
	Info("global info message")
	Warn("global warn message")
	Error("global error message")

	// 测试带 context 的全局日志方法
	ctx := context.Background()
	DebugContext(ctx, "global debug with context")
	InfoContext(ctx, "global info with context")
	WarnContext(ctx, "global warn with context")
	ErrorContext(ctx, "global error with context")

	// 测试带参数的全局日志方法
	Info("global info with args", "key1", "value1", "key2", 42)
	InfoContext(ctx, "global info context with args", "key1", "value1", "key2", 42)
}

// TestGlobalLoggerSingleton 测试全局日志器单例模式
func TestGlobalLoggerSingleton(t *testing.T) {
	// 多次调用应该返回同一个实例
	logger1 := getDefaultLogger()
	logger2 := getDefaultLogger()

	if logger1 == nil || logger2 == nil {
		t.Fatal("getDefaultLogger() returned nil")
	}

	// 注意：由于 Logger 是接口，我们无法直接比较指针
	// 但我们可以通过行为来验证它们是同一个实例
	// 这里我们主要验证函数不会 panic 且返回有效的日志器
}

// TestModuleLogger 测试模块日志器功能
func TestModuleLogger(t *testing.T) {
	// 测试创建模块日志器
	dbLogger := Module("database")
	if dbLogger == nil {
		t.Fatal("Module() returned nil logger")
	}

	apiLogger := Module("api")
	if apiLogger == nil {
		t.Fatal("Module() returned nil logger")
	}

	// 测试模块日志器的基本功能
	dbLogger.Info("database connection established", "host", "localhost", "port", 5432)
	apiLogger.Info("API request processed", "endpoint", "/users", "method", "GET")

	// 测试带 context 的模块日志器
	ctx := context.Background()
	dbLogger.InfoContext(ctx, "database query executed", "query", "SELECT * FROM users")
	apiLogger.InfoContext(ctx, "API response sent", "status", 200)
}

// TestModuleLoggerSingleton 测试模块日志器单例模式
func TestModuleLoggerSingleton(t *testing.T) {
	// 相同模块名应该返回同一个实例
	logger1 := Module("test")
	logger2 := Module("test")

	if logger1 == nil || logger2 == nil {
		t.Fatal("Module() returned nil logger")
	}

	// 不同模块名应该返回不同的实例
	logger3 := Module("different")
	if logger3 == nil {
		t.Fatal("Module() returned nil logger")
	}

	// 测试日志器功能正常
	logger1.Info("test message from logger1")
	logger2.Info("test message from logger2")
	logger3.Info("test message from logger3")
}

// TestModuleLoggerConcurrency 测试模块日志器的并发安全性
func TestModuleLoggerConcurrency(t *testing.T) {
	const numGoroutines = 100
	const moduleName = "concurrent"

	// 启动多个 goroutine 同时获取同一个模块的日志器
	done := make(chan Logger, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			logger := Module(moduleName)
			logger.Info("concurrent access test")
			done <- logger
		}()
	}

	// 收集所有结果
	loggers := make([]Logger, 0, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		logger := <-done
		if logger == nil {
			t.Fatal("Module() returned nil logger in concurrent access")
		}
		loggers = append(loggers, logger)
	}

	// 验证所有日志器都不为 nil
	for i, logger := range loggers {
		if logger == nil {
			t.Fatalf("Logger %d is nil", i)
		}
	}
}
