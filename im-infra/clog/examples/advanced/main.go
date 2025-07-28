package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 高级功能测试 ===")

	// 清理之前的日志文件
	logDir := "logs"
	os.RemoveAll(logDir)

	// 测试1: RootPath 功能
	fmt.Println("\n--- 测试1: RootPath 路径控制 ---")
	testRootPathFeature()

	// 测试2: 日志轮转功能
	fmt.Println("\n--- 测试2: 日志轮转功能 ---")
	testLogRotation()

	// 测试3: 自定义 TraceID Hook
	fmt.Println("\n--- 测试3: 自定义 TraceID Hook ---")
	testCustomTraceIDHook()

	// 测试4: 多种输出格式对比
	fmt.Println("\n--- 测试4: 多种输出格式对比 ---")
	testMultipleFormats()

	fmt.Println("\n=== 高级测试完成 ===")
}

func testRootPathFeature() {
	// 获取当前工作目录
	wd, _ := os.Getwd()
	projectRoot := findProjectRoot(wd)

	fmt.Printf("项目根目录: %s\n", projectRoot)

	// 测试1: 不设置 RootPath
	fmt.Println("\n1. 默认路径显示（不设置 RootPath）:")
	defaultConfig := clog.Config{
		Level:       "info",
		Format:      "console",
		Output:      "stdout",
		AddSource:   true,
		EnableColor: true,
	}

	defaultLogger, _ := clog.New(defaultConfig)
	defaultLogger.Info("默认路径显示")

	// 测试2: 设置 RootPath 为项目根目录
	fmt.Println("\n2. 设置 RootPath 为项目根目录:")
	rootPathConfig := clog.Config{
		Level:       "info",
		Format:      "console",
		Output:      "stdout",
		AddSource:   true,
		EnableColor: true,
		RootPath:    projectRoot,
	}

	rootPathLogger, _ := clog.New(rootPathConfig)
	rootPathLogger.Info("项目根目录相对路径显示")

	// 测试3: 设置 RootPath 为 "gochat"
	fmt.Println("\n3. 设置 RootPath 为 'gochat':")
	gochatConfig := clog.Config{
		Level:       "info",
		Format:      "console",
		Output:      "stdout",
		AddSource:   true,
		EnableColor: true,
		RootPath:    "gochat",
	}

	gochatLogger, _ := clog.New(gochatConfig)
	gochatLogger.Info("gochat 相对路径显示")

	// 测试4: 设置不存在的 RootPath
	fmt.Println("\n4. 设置不存在的 RootPath:")
	invalidConfig := clog.Config{
		Level:       "info",
		Format:      "console",
		Output:      "stdout",
		AddSource:   true,
		EnableColor: true,
		RootPath:    "/nonexistent/path",
	}

	invalidLogger, _ := clog.New(invalidConfig)
	invalidLogger.Info("应该显示绝对路径")
}

func testLogRotation() {
	// 确保日志目录存在
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return
	}

	logFile := filepath.Join(logDir, "rotation_test.log")

	// 配置日志轮转（设置很小的文件大小以便测试）
	rotationConfig := clog.Config{
		Level:     "debug",
		Format:    "json",
		Output:    logFile,
		AddSource: true,
		RootPath:  "gochat",
		Rotation: &clog.RotationConfig{
			MaxSize:    1,    // 1MB
			MaxBackups: 3,    // 保留3个备份
			MaxAge:     7,    // 保留7天
			Compress:   true, // 压缩旧文件
		},
	}

	logger, err := clog.New(rotationConfig)
	if err != nil {
		fmt.Printf("创建轮转 logger 失败: %v\n", err)
		return
	}

	fmt.Printf("写入轮转日志到: %s\n", logFile)

	// 写入一些日志
	for i := 0; i < 10; i++ {
		logger.Info("轮转测试消息",
			clog.Int("iteration", i),
			clog.String("timestamp", time.Now().Format(time.RFC3339)),
			clog.String("data", "这是一些测试数据用于填充日志文件"))

		logger.Debug("轮转测试调试信息",
			clog.Int("debug_id", i),
			clog.String("level", "debug"))

		if i%3 == 0 {
			logger.Warn("轮转测试警告",
				clog.Int("warning_id", i),
				clog.String("reason", "测试警告消息"))
		}
	}

	// 检查日志文件
	if _, err := os.Stat(logFile); err == nil {
		fmt.Println("✅ 日志文件创建成功")

		// 显示文件大小
		if info, err := os.Stat(logFile); err == nil {
			fmt.Printf("日志文件大小: %d bytes\n", info.Size())
		}

		// 显示部分内容
		content, _ := os.ReadFile(logFile)
		if len(content) > 500 {
			fmt.Printf("日志文件内容（前500字符）:\n%s...\n", string(content[:500]))
		} else {
			fmt.Printf("日志文件内容:\n%s\n", string(content))
		}
	} else {
		fmt.Printf("❌ 日志文件创建失败: %v\n", err)
	}
}

func testCustomTraceIDHook() {
	// 设置自定义 TraceID Hook
	clog.SetTraceIDHook(func(ctx context.Context) (string, bool) {
		// 优先查找自定义的 trace key
		if val := ctx.Value("custom-trace-id"); val != nil {
			if str, ok := val.(string); ok && str != "" {
				return "custom:" + str, true
			}
		}

		// 查找请求ID
		if val := ctx.Value("request-id"); val != nil {
			if str, ok := val.(string); ok && str != "" {
				return "req:" + str, true
			}
		}

		// 回退到默认行为
		if val := ctx.Value("traceID"); val != nil {
			if str, ok := val.(string); ok && str != "" {
				return str, true
			}
		}

		return "", false
	})

	// 使用全局 logger 进行测试，无需创建新实例

	// 测试不同的 context 值
	fmt.Println("\n1. 使用 custom-trace-id:")
	ctx1 := context.WithValue(context.Background(), "custom-trace-id", "abc123")
	clog.WithContext(ctx1).Info("自定义 TraceID 测试")

	fmt.Println("\n2. 使用 request-id:")
	ctx2 := context.WithValue(context.Background(), "request-id", "req456")
	clog.WithContext(ctx2).Info("请求ID 测试")

	fmt.Println("\n3. 使用默认 traceID:")
	ctx3 := context.WithValue(context.Background(), "traceID", "default789")
	clog.WithContext(ctx3).Info("默认 TraceID 测试")

	fmt.Println("\n4. 没有 TraceID:")
	ctx4 := context.Background()
	clog.WithContext(ctx4).Info("无 TraceID 测试")
}

func testMultipleFormats() {
	logDir := "logs"
	os.MkdirAll(logDir, 0755)

	// 测试数据
	testData := []struct {
		name   string
		config clog.Config
	}{
		{
			name: "Console 彩色输出",
			config: clog.Config{
				Level:       "info",
				Format:      "console",
				Output:      "stdout",
				AddSource:   true,
				EnableColor: true,
				RootPath:    "gochat",
			},
		},
		{
			name: "Console 无彩色输出",
			config: clog.Config{
				Level:       "info",
				Format:      "console",
				Output:      "stdout",
				AddSource:   true,
				EnableColor: false,
				RootPath:    "gochat",
			},
		},
		{
			name: "JSON 文件输出",
			config: clog.Config{
				Level:     "info",
				Format:    "json",
				Output:    filepath.Join(logDir, "format_test.json"),
				AddSource: true,
				RootPath:  "gochat",
			},
		},
	}

	ctx := context.WithValue(context.Background(), "traceID", "format-test-123")

	for i, test := range testData {
		fmt.Printf("\n%d. %s:\n", i+1, test.name)

		logger, err := clog.New(test.config)
		if err != nil {
			fmt.Printf("❌ 创建 logger 失败: %v\n", err)
			continue
		}

		// 测试各种日志级别和功能
		logger.Info("格式测试信息", clog.String("format", test.config.Format))
		clog.WithContext(ctx).Module("test").Warn("格式测试警告",
			clog.String("type", "format_comparison"),
			clog.Int("test_id", i+1))
	}

	// 显示 JSON 文件内容
	jsonFile := filepath.Join(logDir, "format_test.json")
	if content, err := os.ReadFile(jsonFile); err == nil {
		fmt.Printf("\nJSON 文件内容:\n%s\n", string(content))
	}
}

// findProjectRoot 查找包含 go.mod 的项目根目录
func findProjectRoot(startPath string) string {
	currentPath := startPath
	for {
		goModPath := filepath.Join(currentPath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentPath
		}

		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			// 已经到达根目录，没有找到 go.mod
			return startPath
		}
		currentPath = parentPath
	}
}
