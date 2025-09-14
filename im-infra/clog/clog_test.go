package clog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCoreFeatures tests core clog functionality: config, levels, fields, namespace, traceid, caller, rotation
func TestCoreFeatures(t *testing.T) {
	t.Run("Environment Defaults", testEnvDefaults)
	t.Run("Log Levels", testLogLevels)
	t.Run("All Fields", testAllFields)
	t.Run("Hierarchical Namespace", testNamespace)
	t.Run("Context TraceID", testTraceID)
	t.Run("Caller Info", testCaller)
	t.Run("File Rotation", testRotation)
}

// testEnvDefaults verifies GetDefaultConfig
func testEnvDefaults(t *testing.T) {
	dev := GetDefaultConfig("development")
	if dev.Level != "debug" || dev.Format != "console" || !dev.EnableColor {
		t.Errorf("Dev config mismatch: %+v", dev)
	}

	prod := GetDefaultConfig("production")
	if prod.Level != "info" || prod.Format != "json" || prod.EnableColor {
		t.Errorf("Prod config mismatch: %+v", prod)
	}
}

// testLogLevels captures output for all levels
func testLogLevels(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{Level: "debug", Format: "console", Output: "stdout"}
	if err := Init(context.Background(), config); err != nil {
		t.Fatal(err)
	}

	ctx := WithTraceID(context.Background(), "test-trace")

	Debug("debug msg", String("k", "v"))
	Info("info msg")
	Warn("warn msg")
	Error("error msg")

	// Fatal in subtest to avoid exit
	t.Run("Fatal", func(t *testing.T) {
		originalExit := exitFunc
		exitCalled := false
		SetExitFunc(func(code int) {
			exitCalled = true
			// Don't call originalExit to avoid actually exiting
		})
		defer func() {
			SetExitFunc(originalExit)
		}()

		WithContext(ctx).Fatal("fatal msg")

		if !exitCalled {
			t.Error("Fatal did not call exit function")
		}
	})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !contains(output, "debug msg") || !contains(output, "info msg") || !contains(output, "warn msg") || !contains(output, "error msg") {
		t.Errorf("Missing log levels in output: %s", output)
	}
	// Fatal log captured in subtest mock
}

// testAllFields verifies all field types in output
func testAllFields(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{Level: "debug", Format: "json", Output: "stdout"}
	if err := Init(context.Background(), config); err != nil {
		t.Fatal(err)
	}

	err := errors.New("test err")
	now := time.Now()
	dur := 5 * time.Second

	Info("fields test",
		String("string", "hello"),
		Int("int", 42),
		Bool("bool", true),
		Float64("float64", 3.14),
		Duration("duration", dur),
		Time("time", now),
		Err(err),
		Any("any", map[string]int{"k": 1}),
	)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	var logs []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logs); err != nil {
		t.Fatal("Invalid JSON output:", err)
	}
	if len(logs) == 0 {
		t.Fatal("No logs")
	}
	log := logs[0]

	if log["string"] != "hello" || log["int"] != float64(42) || log["bool"] != true || log["float64"] != float64(3.14) {
		t.Errorf("Field mismatch: %+v", log)
	}
	if log["duration"] != dur.String() || log["time"].(string)[:10] != now.Format("2006-01-02") {
		t.Errorf("Time/Duration mismatch: %+v", log)
	}
	if log["error"] != "test err" {
		t.Errorf("Err field mismatch: %v", log["error"])
	}
	if m, ok := log["any"].(map[string]interface{}); !ok || m["k"] != float64(1) {
		t.Errorf("Any field mismatch: %+v", log["any"])
	}
}

// testNamespace verifies hierarchical namespace
func testNamespace(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{Level: "info", Format: "json", Output: "stdout", AddSource: true}
	if err := Init(context.Background(), config, WithNamespace("root")); err != nil {
		t.Fatal(err)
	}

	// Chain namespaces
	ns := Namespace("a").Namespace("b").Namespace("c")
	ns.Info("namespace test")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	var logs []map[string]interface{}
	json.Unmarshal(buf.Bytes(), &logs)
	if len(logs) == 0 {
		t.Fatal("No logs")
	}
	log := logs[0]
	if log["namespace"] != "root.a.b.c" {
		t.Errorf("Namespace mismatch: %v", log["namespace"])
	}
}

// testTraceID verifies traceid in context
func testTraceID(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{Level: "info", Format: "json", Output: "stdout"}
	if err := Init(context.Background(), config); err != nil {
		t.Fatal(err)
	}

	traceID := "test-trace-123"
	ctx := WithTraceID(context.Background(), traceID)
	WithContext(ctx).Info("traceid test")
	C(ctx).Namespace("test").Info("alias test")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	var logs []map[string]interface{}
	json.Unmarshal(buf.Bytes(), &logs)
	if len(logs) < 2 {
		t.Fatal("Insufficient logs")
	}
	for _, log := range logs {
		if log["trace_id"] != traceID {
			t.Errorf("TraceID mismatch: %v", log["trace_id"])
		}
	}
}

// testCaller verifies AddSource/caller
func testCaller(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{Level: "info", Format: "json", Output: "stdout", AddSource: true}
	if err := Init(context.Background(), config); err != nil {
		t.Fatal(err)
	}

	Info("caller test")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	var logs []map[string]interface{}
	json.Unmarshal(buf.Bytes(), &logs)
	if len(logs) == 0 {
		t.Fatal("No logs")
	}
	log := logs[0]
	caller, ok := log["caller"]
	if !ok || caller == "" {
		t.Errorf("No caller: %+v", log)
	}
	// Caller should be like "clog_test.go:XXX"
	if callerStr, ok := caller.(string); ok && !contains(callerStr, "clog_test.go") {
		t.Errorf("Invalid caller format: %s", callerStr)
	}
}

// testRotation verifies RotationConfig
func testRotation(t *testing.T) {
	dir, err := os.MkdirTemp("", "clog-test-rotation")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logFile := filepath.Join(dir, "app.log")
	config := &Config{
		Level:     "info",
		Format:    "json",
		Output:    logFile,
		AddSource: true,
		Rotation: &RotationConfig{
			MaxSize:    1024, // 1KB small for test
			MaxBackups: 2,
			MaxAge:     1,
			Compress:   false, // No compress for simplicity
		},
	}
	if err := Init(context.Background(), config); err != nil {
		t.Fatal(err)
	}

	// Write many logs to trigger rotation (each log ~100-200 bytes)
	for i := 0; i < 20; i++ { // Enough to exceed 1KB
		Info(fmt.Sprintf("rotation log %d", i),
			String("id", fmt.Sprintf("%d", i)),
			Int("size", i*50),
		)
		time.Sleep(10 * time.Millisecond)
	}

	// Check files
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	current := 0
	backups := 0
	for _, f := range files {
		name := f.Name()
		if name == "app.log" {
			current++
		} else if name == "app.log.1" || name == "app.log.2" {
			backups++
		}
	}

	if current != 1 {
		t.Errorf("Expected 1 current log, got %d", current)
	}
	if backups < 1 { // At least one backup
		t.Errorf("Expected at least 1 backup, got %d", backups)
	}

	// Verify content in current log
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("\"msg\":\"rotation log")) {
		t.Errorf("Invalid log content")
	}
}

// Helper: contains for byte slices
func contains(s string, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
