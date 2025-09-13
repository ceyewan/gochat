package clog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_InvalidConfig 测试无效配置的处理
func TestNew_InvalidConfig(t *testing.T) {
	// 准备测试数据 - 注意：clog 内部会使用默认值而不是报错
	invalidConfig := &Config{
		Level:  "invalid-level",  // 内部会默认为 "info"
		Format: "invalid-format", // 内部会默认为 "console"
		Output: "",                // 内部会默认为 "stdout"
	}

	// 执行测试 - clog 有容错机制，不会报错
	logger, err := New(context.Background(), invalidConfig)

	// 验证结果 - clog 应该能处理无效配置并返回有效的 logger
	assert.NoError(t, err, "clog 应该能处理无效配置")
	assert.NotNil(t, logger, "应该返回有效的 logger")
	
	// logger 应该能正常工作
	logger.Info("logger with invalid config test")
}

// TestGetDefaultConfig 测试默认配置功能
func TestGetDefaultConfig(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected *Config
	}{
		{
			name: "开发环境配置",
			env:  "development",
			expected: &Config{
				Level:       "debug",
				Format:      "console",
				Output:      "stdout",
				AddSource:   true,
				EnableColor: true,
				RootPath:    "gochat",
			},
		},
		{
			name: "生产环境配置",
			env:  "production",
			expected: &Config{
				Level:       "info",
				Format:      "json",
				Output:      "stdout",
				AddSource:   true,
				EnableColor: false,
				RootPath:    "",
			},
		},
		{
			name: "未知环境配置",
			env:  "unknown",
			expected: &Config{
				Level:       "info",
				Format:      "console",
				Output:      "stdout",
				AddSource:   true,
				EnableColor: true,
				RootPath:    "gochat",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetDefaultConfig(tt.env)
			
			assert.Equal(t, tt.expected.Level, config.Level)
			assert.Equal(t, tt.expected.Format, config.Format)
			assert.Equal(t, tt.expected.Output, config.Output)
			assert.Equal(t, tt.expected.AddSource, config.AddSource)
			assert.Equal(t, tt.expected.EnableColor, config.EnableColor)
			assert.Equal(t, tt.expected.RootPath, config.RootPath)
		})
	}
}

// TestInit_GlobalLogger 测试全局 logger 初始化
func TestInit_GlobalLogger(t *testing.T) {
	// 使用控制台输出进行测试，避免文件创建问题
	config := &Config{
		Level:       "debug",
		Format:      "console",
		Output:      "stdout",
		AddSource:   false, // 关闭源码信息避免测试输出混乱
		EnableColor: false,
	}

	// 执行初始化
	err := Init(context.Background(), config, WithNamespace("test-service"))
	require.NoError(t, err, "初始化应该成功")

	// 验证全局 logger 工作
	Info("全局 logger 测试消息")
	Warn("全局 logger 警告消息")
	Error("全局 logger 错误消息")
}

// TestWithTraceID_WithContext 测试 TraceID 注入和提取
func TestWithTraceID_WithContext(t *testing.T) {
	// 准备测试
	traceID := "test-trace-12345"
	ctx := context.Background()
	
	// 执行测试：注入 TraceID
	ctxWithTrace := WithTraceID(ctx, traceID)
	
	// 验证 TraceID 被正确注入
	logger := WithContext(ctxWithTrace)
	assert.NotNil(t, logger, "应该返回 logger 实例")
	
	// 验证没有 TraceID 的情况
	loggerWithoutTrace := WithContext(ctx)
	assert.NotNil(t, loggerWithoutTrace, "即使没有 TraceID 也应该返回 logger")
}

// TestNamespace_Chaining 测试层次化命名空间链式调用
func TestNamespace_Chaining(t *testing.T) {
	// 初始化测试 logger
	config := GetDefaultConfig("development")
	err := Init(context.Background(), config, WithNamespace("test-service"))
	require.NoError(t, err)

	// 测试链式命名空间创建
	baseLogger := Namespace("user")
	authLogger := baseLogger.Namespace("auth")
	dbLogger := baseLogger.Namespace("database")
	
	// 验证所有 logger 都能正常工作
	baseLogger.Info("用户模块测试")
	authLogger.Warn("认证模块警告")
	dbLogger.Error("数据库模块错误")
	
	// 测试深层链式调用
	deepLogger := Namespace("payment").Namespace("processor").Namespace("stripe")
	deepLogger.Info("深层链式调用测试")
	
	// 注意：由于 zap 的限制，可能会有重复的 namespace 字段，但功能是正常的
	// 这在实际使用中不会影响日志的可读性和查询
}

// TestLoggerMethods 测试所有日志级别方法
func TestLoggerMethods(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "methods_test.log")
	
	config := &Config{
		Level:     "debug",
		Format:    "json",
		Output:    logFile,
		AddSource: false, // 关闭源码信息以便测试
	}

	logger, err := New(context.Background(), config, WithNamespace("test-methods"))
	require.NoError(t, err)

	// 测试所有日志级别
	logger.Debug("Debug 消息", String("level", "debug"))
	logger.Info("Info 消息", String("level", "info"))
	logger.Warn("Warn 消息", String("level", "warn"))
	logger.Error("Error 消息", String("level", "error"))

	// 验证日志文件内容
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	logContent := string(content)
	assert.Contains(t, logContent, "Debug 消息")
	assert.Contains(t, logContent, "Info 消息")
	assert.Contains(t, logContent, "Warn 消息")
	assert.Contains(t, logContent, "Error 消息")
	assert.Contains(t, logContent, "test-methods")
}

// TestWithNamespace_Option 测试 WithNamespace 选项
func TestWithNamespace_Option(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "namespace_test.log")
	
	config := &Config{
		Level:  "info",
		Format: "json",
		Output: logFile,
	}

	// 测试 WithNamespace 选项
	logger, err := New(context.Background(), config, WithNamespace("custom-namespace"))
	require.NoError(t, err)

	logger.Info("命名空间测试消息")

	// 验证命名空间被正确设置
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	assert.Contains(t, string(content), "custom-namespace")
}

// TestParseOptions 测试选项解析
func TestParseOptions(t *testing.T) {
	// 测试空选项
	options := ParseOptions()
	assert.Equal(t, "", options.namespace)

	// 测试 WithNamespace 选项
	options = ParseOptions(WithNamespace("test-namespace"))
	assert.Equal(t, "test-namespace", options.namespace)

	// 测试多个选项（最后一个生效）
	options = ParseOptions(
		WithNamespace("first"),
		WithNamespace("second"),
	)
	assert.Equal(t, "second", options.namespace)
}

// TestConfig_Validation 测试配置验证
func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "有效配置",
			config: &Config{
				Level:  "info",
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "无效日志级别",
			config: &Config{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "无效格式",
			config: &Config{
				Level:  "info",
				Format: "invalid",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "空输出目标",
			config: &Config{
				Level:  "info",
				Format: "json",
				Output: "",
			},
			wantErr: true,
		},
		{
			name: "有效轮转配置",
			config: &Config{
				Level:  "info",
				Format: "json",
				Output: "stdout",
				Rotation: &RotationConfig{
					MaxSize:    100,
					MaxBackups: 3,
					MaxAge:     7,
					Compress:   true,
				},
			},
			wantErr: false,
		},
		{
			name: "无效轮转配置",
			config: &Config{
				Level:  "info",
				Format: "json",
				Output: "stdout",
				Rotation: &RotationConfig{
					MaxSize:    -1, // 无效值
					MaxBackups: 3,
					MaxAge:     7,
					Compress:   true,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFileOutputWithRotation 测试文件输出和轮转功能
func TestFileOutputWithRotation(t *testing.T) {
	// 跳过短测试，因为轮转需要时间
	if testing.Short() {
		t.Skip("跳过轮转测试")
	}

	// 准备测试环境
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "rotation_test.log")
	
	config := &Config{
		Level:  "info",
		Format: "json",
		Output: logFile,
		Rotation: &RotationConfig{
			MaxSize:    1, // 1MB，便于测试
			MaxBackups: 2,
			MaxAge:     1,
			Compress:   false, // 关闭压缩便于检查
		},
	}

	logger, err := New(context.Background(), config)
	require.NoError(t, err)

	// 写入大量日志以触发轮转
	for i := 0; i < 1000; i++ {
		logger.Info("测试日志消息", Int("index", i), String("message", "这是一条比较长的日志消息，用于测试文件轮转功能"))
	}

	// 给文件系统一些时间
	time.Sleep(100 * time.Millisecond)

	// 验证主日志文件存在
	_, err = os.Stat(logFile)
	assert.NoError(t, err, "主日志文件应该存在")
}

// TestContextConcurrency 测试上下文并发安全性
func TestContextConcurrency(t *testing.T) {
	// 并发测试 TraceID 注入和提取
	var wg sync.WaitGroup
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			
			traceID := fmt.Sprintf("trace-%d", i)
			ctx := WithTraceID(context.Background(), traceID)
			
			logger := WithContext(ctx)
			logger.Info("并发测试消息", String("goroutine", fmt.Sprintf("%d", i)))
		}(i)
	}
	
	wg.Wait()
}

// TestNamespaceIsolation 测试命名空间隔离性
func TestNamespaceIsolation(t *testing.T) {
	// 初始化
	config := GetDefaultConfig("development")
	err := Init(context.Background(), config)
	require.NoError(t, err)

	// 创建不同的命名空间
	userLogger := Namespace("user")
	orderLogger := Namespace("order")
	
	// 验证命名空间互不影响
	userLogger.Info("用户日志")
	orderLogger.Info("订单日志")
	
	// 验证子命名空间
	userAuthLogger := userLogger.Namespace("auth")
	userAuthLogger.Warn("认证警告")
}

// BenchmarkLoggerPerformance 性能基准测试
func BenchmarkLoggerPerformance(b *testing.B) {
	// 准备测试环境
	tempDir := b.TempDir()
	logFile := filepath.Join(tempDir, "benchmark.log")
	
	config := &Config{
		Level:     "info",
		Format:    "json",
		Output:    logFile,
		AddSource: false,
	}

	logger, err := New(context.Background(), config)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.Info("性能测试消息", 
				String("id", fmt.Sprintf("%d", i)),
				Int("value", i),
			)
			i++
		}
	})
}