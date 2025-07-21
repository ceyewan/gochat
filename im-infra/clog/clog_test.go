package clog

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// setupTestLogger 是一个测试辅助函数，用于创建一个带有临时文件和指定选项的日志记录器。
// 它返回创建的日志记录器、临时文件名以及一个用于在测试结束时进行清理的函数。
func setupTestLogger(t *testing.T, opts ...Option) (Logger, string, func()) {
	t.Helper()

	// 创建一个临时文件
	tempFile, err := os.CreateTemp("", "test_log_*.log")
	if err != nil {
		t.Fatalf("无法创建临时文件: %v", err)
	}
	tempFilename := tempFile.Name()
	tempFile.Close() // 关闭文件，因为logger会重新打开它

	// 添加文件名选项
	finalOpts := append([]Option{WithFilename(tempFilename), WithConsoleOutput(false)}, opts...)

	// 初始化日志记录器
	logger, err := NewLogger(finalOpts...)
	if err != nil {
		os.Remove(tempFilename) // 清理以防失败
		t.Fatalf("初始化日志器失败: %v", err)
	}

	// 定义清理函数
	cleanup := func() {
		logger.Close()
		os.Remove(tempFilename)
		// 如果测试修改了全局状态，也在这里重置
		resetDefaultLogger()
	}

	return logger, tempFilename, cleanup
}

// captureOutput 捕获指定io.Writer的输出
func captureOutput(t *testing.T, writer *os.File, action func()) string {
	t.Helper()
	old := *writer
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道失败: %v", err)
	}
	*writer = *w

	action()

	w.Close()
	*writer = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("读取捕获的输出失败: %v", err)
	}
	r.Close()
	return buf.String()
}

// TestConsoleHumanReadableOutput 测试控制台输出是否为人类可读格式
func TestConsoleHumanReadableOutput(t *testing.T) {
	// 捕获标准输出
	var output string
	action := func() {
		logger, err := NewLogger(
			WithLevel("info"),
			WithFormat(FormatConsole),
			WithConsoleOutput(true),
			WithFilename(""), // 禁用文件输出
			WithEnableColor(false),
		)
		if err != nil {
			t.Fatalf("初始化日志器失败: %v", err)
		}
		logger.Info("这是一条控制台测试消息", String("user", "test"))
		logger.Close()
	}

	output = captureOutput(t, os.Stdout, action)

	// 验证输出是人类可读的 (非JSON)
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Errorf("期望是人类可读的控制台格式, 但似乎是JSON格式: %s", output)
	}

	// 验证包含关键信息
	// 例如: 2023-10-27T10:00:00.000+0800    INFO    default    logger/clog_test.go:86    这是一条控制台测试消息    {"user": "test"}
	// The regex now optionally matches the caller info, which is enabled by default.
	match, err := regexp.MatchString(`\sINFO\s+default\s+.*logger/clog_test.go:\d+\s+这是一条控制台测试消息\s+{"user": "test"}`, output)
	if err != nil {
		t.Fatalf("正则表达式匹配失败: %v", err)
	}
	if !match {
		t.Errorf("控制台输出格式不符合预期, 实际输出: %s", output)
	}
}

// TestFileFormatSwitching 测试文件输出格式可根据配置切换
func TestFileFormatSwitching(t *testing.T) {
	t.Run("FileAsConsoleFormat", func(t *testing.T) {
		logger, tempFile, cleanup := setupTestLogger(t, WithFormat(FormatConsole))
		defer cleanup()

		logger.Info("文件中的控制台格式日志")
		logger.Sync()

		content, err := os.ReadFile(tempFile)
		if err != nil {
			t.Fatalf("读取日志文件失败: %v", err)
		}

		if strings.HasPrefix(strings.TrimSpace(string(content)), "{") {
			t.Errorf("期望文件内容为console格式, 但似乎是JSON格式: %s", string(content))
		}
		if !strings.Contains(string(content), "文件中的控制台格式日志") {
			t.Errorf("文件内容缺少期望的日志消息, 实际内容: %s", string(content))
		}
	})

	t.Run("FileAsJSONFormat", func(t *testing.T) {
		logger, tempFile, cleanup := setupTestLogger(t, WithFormat(FormatJSON))
		defer cleanup()

		logger.Info("文件中的JSON格式日志")
		logger.Sync()

		content, err := os.ReadFile(tempFile)
		if err != nil {
			t.Fatalf("读取日志文件失败: %v", err)
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal(content, &logEntry); err != nil {
			t.Fatalf("期望文件内容为JSON格式, 但解析失败: %v\n内容: %s", err, string(content))
		}
		if msg, _ := logEntry["message"].(string); msg != "文件中的JSON格式日志" {
			t.Errorf("期望的message不匹配, 期望: '文件中的JSON格式日志', 实际: %q", msg)
		}
	})
}

// TestJSONFileOutput 测试文件日志输出是否为合法的JSON格式
func TestJSONFileOutput(t *testing.T) {
	logger, tempFile, cleanup := setupTestLogger(t,
		WithLevel("info"),
		WithName("test-module"),
		WithFormat(FormatJSON), // Explicitly set format to JSON
	)
	defer cleanup()

	testMessage := "测试JSON格式日志"
	logger.Info(testMessage, String("test_key", "test_value"))
	logger.Sync()

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Fatalf("日志内容不是合法JSON: %v\n内容: %s", err, string(content))
	}

	requiredFields := []string{"timestamp", "level", "message", "module"}
	for _, field := range requiredFields {
		if _, exists := logEntry[field]; !exists {
			t.Errorf("缺少必需字段: %s", field)
		}
	}

	if logEntry["level"] != "info" {
		t.Errorf("期望level为info，实际为: %v", logEntry["level"])
	}
	if logEntry["message"] != testMessage {
		t.Errorf("期望message为%q，实际为: %v", testMessage, logEntry["message"])
	}
	if logEntry["module"] != "test-module" {
		t.Errorf("期望module为test-module，实际为: %v", logEntry["module"])
	}
	if logEntry["test_key"] != "test_value" {
		t.Errorf("期望test_key为test_value，实际为: %v", logEntry["test_key"])
	}
}

// TestTraceIDInJSON 测试TraceID在JSON日志中的正确性
func TestTraceIDInJSON(t *testing.T) {
	t.Run("WithValidTraceID", func(t *testing.T) {
		traceID := "trace-12345-xyz"
		logger, tempFile, cleanup := setupTestLogger(t, WithTraceID(traceID), WithFormat(FormatJSON))
		defer cleanup()

		logger.Info("测试有效的TraceID")
		logger.Sync()

		content, err := os.ReadFile(tempFile)
		if err != nil {
			t.Fatalf("读取日志文件失败: %v", err)
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal(content, &logEntry); err != nil {
			t.Fatalf("日志内容不是合法JSON: %v", err)
		}

		if logEntry["traceID"] != traceID {
			t.Errorf("期望traceID为%q，实际为: %v", traceID, logEntry["traceID"])
		}
	})

	t.Run("WithEmptyTraceID", func(t *testing.T) {
		logger, tempFile, cleanup := setupTestLogger(t, WithTraceID(""), WithFormat(FormatJSON))
		defer cleanup()

		logger.Info("测试空的TraceID")
		logger.Sync()

		content, err := os.ReadFile(tempFile)
		if err != nil {
			t.Fatalf("读取日志文件失败: %v", err)
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal(content, &logEntry); err != nil {
			t.Fatalf("日志内容不是合法JSON: %v", err)
		}

		if _, exists := logEntry["traceID"]; exists {
			t.Errorf("当TraceID为空时, 不应存在traceID字段, 但实际存在: %v", logEntry["traceID"])
		}
	})

	t.Run("InheritedByModule", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "test_module_inherit_*.log")
		if err != nil {
			t.Fatalf("创建临时文件失败: %v", err)
		}
		tempFilename := tempFile.Name()
		tempFile.Close()

		defer func() {
			os.Remove(tempFilename)
			resetDefaultLogger()
		}()

		traceID := "trace-abc-987"
		err = Init(
			WithFilename(tempFilename),
			WithConsoleOutput(false),
			WithTraceID(traceID),
			WithFormat(FormatJSON), // Ensure the global logger has the correct format
		)
		if err != nil {
			t.Fatalf("初始化默认日志器失败: %v", err)
		}

		moduleLogger := Module("db-module")
		moduleLogger.Info("模块日志继承TraceID")
		Sync()

		content, err := os.ReadFile(tempFilename)
		if err != nil {
			t.Fatalf("读取日志文件失败: %v", err)
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal(content, &logEntry); err != nil {
			t.Fatalf("日志内容不是合法JSON: %v\n内容: %s", err, string(content))
		}

		if val, _ := logEntry["traceID"].(string); val != traceID {
			t.Errorf("模块日志未能继承traceID, 期望: %q, 实际: %v", traceID, val)
		}
	})
}

// TestErrorLevelWithStacktrace 测试错误级别日志包含堆栈跟踪
func TestErrorLevelWithStacktrace(t *testing.T) {
	logger, tempFile, cleanup := setupTestLogger(t, WithLevel("error"), WithFormat(FormatJSON))
	defer cleanup()

	logger.Error("测试错误日志", String("error_code", "ERR_500"))
	logger.Sync()

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Fatalf("日志内容不是合法JSON: %v", err)
	}

	if _, exists := logEntry["stacktrace"]; !exists {
		t.Error("错误级别日志缺少stacktrace字段")
	}
}

// TestModuleName 测试模块名称在日志中的正确性
func TestModuleName(t *testing.T) {
	moduleName := "user-service"
	logger, tempFile, cleanup := setupTestLogger(t, WithName(moduleName), WithFormat(FormatJSON))
	defer cleanup()

	logger.Info("测试模块名称")
	logger.Sync()

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Fatalf("日志内容不是合法JSON: %v", err)
	}

	if logEntry["module"] != moduleName {
		t.Errorf("期望module为%q，实际为: %v", moduleName, logEntry["module"])
	}
}

// TestDifferentLogLevels 测试不同日志级别的输出
func TestDifferentLogLevels(t *testing.T) {
	logger, tempFile, cleanup := setupTestLogger(t,
		WithLevel("debug"),
		WithName("test-levels"),
		WithFormat(FormatJSON), // Explicitly set format to JSON
	)
	defer cleanup()

	logger.Debug("调试信息")
	logger.Info("普通信息")
	logger.Warn("警告信息")
	logger.Error("错误信息")
	logger.Sync()

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 4 {
		// Provide more context on failure
		t.Fatalf("期望4条日志，实际为: %d\n内容:\n%s", len(lines), string(content))
	}

	expectedLevels := []string{"debug", "info", "warn", "error"}
	for i, line := range lines {
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Fatalf("第%d行日志不是合法JSON: %v", i+1, err)
		}

		if logEntry["level"] != expectedLevels[i] {
			t.Errorf("第%d行期望level为%q，实际为: %v", i+1, expectedLevels[i], logEntry["level"])
		}
	}
}

// TestRequiredFieldsAndCaller 测试JSON日志包含所有必需字段以及调用者信息
func TestRequiredFieldsAndCaller(t *testing.T) {
	t.Run("WithCallerEnabled", func(t *testing.T) {
		logger, tempFile, cleanup := setupTestLogger(t,
			WithFormat(FormatJSON),
			WithName("caller-test"),
			WithEnableCaller(true),
		)
		defer cleanup()

		logger.Info("测试调用者信息")
		logger.Sync()

		content, err := os.ReadFile(tempFile)
		if err != nil {
			t.Fatalf("读取日志文件失败: %v", err)
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal(content, &logEntry); err != nil {
			t.Fatalf("日志内容不是合法JSON: %v", err)
		}

		requiredFields := []string{"timestamp", "level", "module", "caller", "message"}
		for _, field := range requiredFields {
			if _, exists := logEntry[field]; !exists {
				t.Errorf("缺少必需字段: %s", field)
			}
		}

		caller, ok := logEntry["caller"].(string)
		if !ok {
			t.Fatalf("caller字段不是字符串类型")
		}
		if !strings.Contains(caller, "clog_test.go:") {
			t.Errorf("caller字段的文件名不正确, 期望包含 'clog_test.go:...', 实际: %s", caller)
		}
	})

	t.Run("WithCallerDisabled", func(t *testing.T) {
		logger, tempFile, cleanup := setupTestLogger(t,
			WithFormat(FormatJSON),
			WithEnableCaller(false),
		)
		defer cleanup()

		logger.Info("测试无调用者信息")
		logger.Sync()

		content, err := os.ReadFile(tempFile)
		if err != nil {
			t.Fatalf("读取日志文件失败: %v", err)
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal(content, &logEntry); err != nil {
			t.Fatalf("日志内容不是合法JSON: %v", err)
		}

		if _, exists := logEntry["caller"]; exists {
			t.Errorf("当EnableCaller为false时, 不应存在caller字段, 但实际存在: %v", logEntry["caller"])
		}
	})
}

// TestFileRotation 测试文件轮转功能
func TestFileRotation(t *testing.T) {
	logDir := t.TempDir()
	logFile := filepath.Join(logDir, "rotate.log")

	rotationConfig := &FileRotationConfig{
		MaxSize:    1, // 1 MB
		MaxBackups: 2,
		MaxAge:     7,
		Compress:   false,
	}

	// We don't use setupTestLogger here because rotation tests are special.
	// We need to manage the directory lifecycle ourselves.
	logger, err := NewLogger(
		WithFilename(logFile),
		WithFileRotation(rotationConfig),
		WithConsoleOutput(false),
	)
	if err != nil {
		t.Fatalf("初始化日志器失败: %v", err)
	}

	longMessage := strings.Repeat("a", 1024*500) // 0.5MB
	logger.Info(longMessage)
	logger.Info(longMessage)
	logger.Info(longMessage)
	logger.Close() // Close to ensure rotation is triggered and files are written.

	files, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("读取日志目录失败: %v", err)
	}

	// We expect 3 files: the current log file and 2 backups.
	// Lumberjack might create one extra backup before cleaning up.
	if len(files) < 2 || len(files) > rotationConfig.MaxBackups+1 {
		t.Errorf("文件数量不符合预期, 期望 %d 到 %d 个, 实际找到 %d 个", 2, rotationConfig.MaxBackups+1, len(files))
	}

	var backupCount int
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "rotate-") {
			backupCount++
		}
	}
	if backupCount == 0 {
		t.Error("未找到任何备份文件")
	}
	if backupCount > rotationConfig.MaxBackups {
		t.Errorf("备份文件数量超过限制, 期望最多 %d, 实际 %d", rotationConfig.MaxBackups, backupCount)
	}
}

// resetDefaultLogger 是一个辅助函数，用于在测试后重置全局日志记录器状态。
// 这对于需要修改全局默认记录器的测试至关重要，以避免测试间的相互影响。
func resetDefaultLogger() {
	// 创建新的服务实例来重置状态
	globalLoggerService = NewDefaultLoggerService()
}
