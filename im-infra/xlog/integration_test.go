package xlog

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileOutput(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := ioutil.TempDir("", "xlog_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "json",
				Writer: "file",
				FileRotation: &FileRotationConfig{
					Filename:   logFile,
					MaxSize:    1, // 1MB
					MaxAge:     1, // 1 day
					MaxBackups: 3,
					LocalTime:  true,
					Compress:   false,
				},
			},
		},
		EnableTraceID: false,
		AddSource:     true,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write some log messages
	logger.Info("test message 1", "key1", "value1")
	logger.Warn("test message 2", "key2", "value2")
	logger.Error("test message 3", "key3", "value3")

	// Give some time for the file to be written
	time.Sleep(100 * time.Millisecond)

	// Read the log file
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 log lines, got %d", len(lines))
	}

	// Verify each line is valid JSON and contains expected fields
	for i, line := range lines {
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Fatalf("Line %d is not valid JSON: %v", i+1, err)
		}

		// Check required fields
		if _, ok := logEntry["time"]; !ok {
			t.Errorf("Line %d missing 'time' field", i+1)
		}
		if _, ok := logEntry["level"]; !ok {
			t.Errorf("Line %d missing 'level' field", i+1)
		}
		if _, ok := logEntry["msg"]; !ok {
			t.Errorf("Line %d missing 'msg' field", i+1)
		}
		if _, ok := logEntry["source"]; !ok {
			t.Errorf("Line %d missing 'source' field", i+1)
		}
	}
}

func TestTraceIDIntegration(t *testing.T) {
	cfg := Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "json",
				Writer: "stdout",
			},
		},
		EnableTraceID: true,
		TraceIDKey:    "trace_id",
		AddSource:     false,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test with context containing trace ID
	ctx := context.WithValue(context.Background(), "trace_id", "test-trace-123")
	logger.InfoContext(ctx, "message with trace ID", "key", "value")

	// Test with context without trace ID
	ctx2 := context.Background()
	logger.InfoContext(ctx2, "message without trace ID", "key", "value")
}

func TestMultipleOutputsIntegration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := ioutil.TempDir("", "xlog_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logFile := filepath.Join(tmpDir, "multi_test.log")

	cfg := Config{
		Level: "debug",
		Outputs: []OutputConfig{
			{
				Format: "text",
				Writer: "stdout",
			},
			{
				Format: "json",
				Writer: "file",
				FileRotation: &FileRotationConfig{
					Filename:   logFile,
					MaxSize:    1,
					MaxAge:     1,
					MaxBackups: 3,
					LocalTime:  true,
					Compress:   false,
				},
			},
		},
		EnableTraceID: true,
		TraceIDKey:    "request_id",
		AddSource:     true,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test with context
	ctx := context.WithValue(context.Background(), "request_id", "req-456")
	logger.InfoContext(ctx, "multi-output test", "component", "integration_test")

	// Give some time for the file to be written
	time.Sleep(100 * time.Millisecond)

	// Verify file was created and contains content
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Fatal("Log file was not created")
	}

	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("Log file is empty")
	}

	// Verify it's valid JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Fatalf("Log file content is not valid JSON: %v", err)
	}
}

func TestDynamicLevelChange(t *testing.T) {
	logger := Default()

	// Initially at info level - debug should not show
	logger.Debug("debug message 1 - should not show")
	logger.Info("info message 1 - should show")

	// Change to debug level
	err := logger.SetLevel("debug")
	if err != nil {
		t.Fatalf("Failed to set level to debug: %v", err)
	}

	// Now debug should show
	logger.Debug("debug message 2 - should show")
	logger.Info("info message 2 - should show")

	// Change to error level
	err = logger.SetLevel("error")
	if err != nil {
		t.Fatalf("Failed to set level to error: %v", err)
	}

	// Now only error should show
	logger.Debug("debug message 3 - should not show")
	logger.Info("info message 3 - should not show")
	logger.Warn("warn message 3 - should not show")
	logger.Error("error message 3 - should show")

	// Test invalid level
	err = logger.SetLevel("invalid")
	if err == nil {
		t.Fatal("Expected error for invalid level, got nil")
	}
}

func TestLoggerChaining(t *testing.T) {
	logger := Default()

	// Create child logger with attributes
	childLogger := logger.With("service", "auth", "version", "1.0")
	childLogger.Info("child logger message")

	// Create grandchild logger with more attributes
	grandchildLogger := childLogger.With("user_id", 12345)
	grandchildLogger.Info("grandchild logger message")

	// Create grouped logger
	groupedLogger := logger.WithGroup("database")
	groupedLogger.Info("grouped message", "table", "users", "operation", "select")

	// Chain group and attributes
	chainedLogger := logger.WithGroup("api").With("endpoint", "/users")
	chainedLogger.Info("chained logger message", "method", "GET")
}

func TestErrorHandling(t *testing.T) {
	// Test with no outputs
	cfg := Config{
		Level:   "info",
		Outputs: []OutputConfig{},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("Expected error for config with no outputs, got nil")
	}

	// Test with file output but no rotation config
	cfg = Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "json",
				Writer: "file",
				// FileRotation is nil
			},
		},
	}

	_, err = New(cfg)
	if err == nil {
		t.Fatal("Expected error for file writer without rotation config, got nil")
	}

	// Test with file output but empty filename
	cfg = Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "json",
				Writer: "file",
				FileRotation: &FileRotationConfig{
					Filename: "", // Empty filename
				},
			},
		},
	}

	_, err = New(cfg)
	if err == nil {
		t.Fatal("Expected error for file writer with empty filename, got nil")
	}
}
