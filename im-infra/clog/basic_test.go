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
