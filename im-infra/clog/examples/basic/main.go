package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 基础功能测试 ===")

	// 清理之前的日志文件
	logDir := "logs"
	os.RemoveAll(logDir)

	// 测试1: Console 控制台输出
	fmt.Println("\n--- 测试1: Console 控制台输出 ---")
	testConsoleOutput()

	// 测试2: JSON 文件输出
	fmt.Println("\n--- 测试2: JSON 文件输出 ---")
	testJSONFileOutput()

	fmt.Println("\n=== 基础测试完成 ===")
}

func testConsoleOutput() {
	// 配置控制台输出
	consoleConfig := &clog.Config{
		Level:       "debug",
		Format:      "console",
		Output:      "stdout",
		AddSource:   true,
		EnableColor: true,
		RootPath:    "gochat",
	}

	logger, err := clog.New(context.Background(), consoleConfig)
	if err != nil {
		fmt.Printf("创建控制台 logger 失败: %v\n", err)
		return
	}

	// 创建带 traceID 的 context
	ctx := clog.WithTraceID(context.Background(), "trace-12345")

	fmt.Println("1. 全局函数调用:")
	clog.Info("全局 Info 消息", clog.String("key", "value"))
	clog.Warn("全局 Warn 消息", clog.Int("number", 42))
	clog.Error("全局 Error 消息", clog.Bool("flag", true))

	fmt.Println("\n2. Logger 实例调用:")
	logger.Info("Logger Info 消息", clog.String("type", "instance"))
	logger.Debug("Logger Debug 消息", clog.Float64("pi", 3.14159))

	fmt.Println("\n3. 层次化命名空间 - 全局:")
	userNamespace := clog.Namespace("user")
	userNamespace.Info("用户模块消息", clog.String("userID", "123"))
	userNamespace.Warn("用户模块警告", clog.String("action", "login"))

	fmt.Println("\n4. 层次化命名空间 - Logger 实例:")
	orderNamespace := logger.Namespace("order")
	orderNamespace.Info("订单模块消息", clog.String("orderID", "order-456"))
	orderNamespace.Error("订单模块错误", clog.String("error", "payment_failed"))

	fmt.Println("\n5. Context 日志 - 全局:")
	clog.C(ctx).Info("Context 全局消息", clog.String("operation", "query"))
	clog.C(ctx).Warn("Context 全局警告", clog.String("resource", "database"))

	fmt.Println("\n6. Context 日志 - 使用全局 WithContext:")
	clog.WithContext(ctx).Info("Context 全局 WithContext 消息", clog.String("service", "api"))
	clog.WithContext(ctx).Error("Context 全局 WithContext 错误", clog.String("endpoint", "/users"))

	fmt.Println("\n7. 链式调用 - 全局:")
	clog.C(ctx).Namespace("payment").Info("支付模块链式调用", clog.String("amount", "100.00"))
	clog.C(ctx).Namespace("notification").Warn("通知模块链式调用", clog.String("type", "email"))

	fmt.Println("\n8. Logger 实例与 Context 结合:")
	// Logger 实例需要通过全局函数来使用 context
	contextLogger := clog.WithContext(ctx)
	contextLogger.Namespace("auth").Info("认证模块消息", clog.String("method", "jwt"))
	contextLogger.Namespace("cache").Debug("缓存模块消息", clog.String("key", "user:123"))
}

func testJSONFileOutput() {
	// 确保日志目录存在
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return
	}

	logFile := filepath.Join(logDir, "basic_test.log")

	// 配置 JSON 文件输出
	jsonConfig := &clog.Config{
		Level:     "debug",
		Format:    "json",
		Output:    logFile,
		AddSource: true, // 确保包含源文件信息
		RootPath:  "gochat",
	}

	logger, err := clog.New(context.Background(), jsonConfig)
	if err != nil {
		fmt.Printf("创建 JSON logger 失败: %v\n", err)
		return
	}

	// 创建带 traceID 的 context
	ctx := clog.WithTraceID(context.Background(), "json-trace-67890")

	fmt.Printf("写入日志到文件: %s\n", logFile)

	// 执行相同的测试，但输出到文件
	logger.Info("JSON Logger Info 消息", clog.String("format", "json"))
	logger.Debug("JSON Logger Debug 消息", clog.String("output", "file"))
	logger.Warn("JSON Logger Warn 消息", clog.Int("line", 100))
	logger.Error("JSON Logger Error 消息", clog.Bool("critical", true))

	// 层次化命名空间
	apiNamespace := logger.Namespace("api")
	apiNamespace.Info("API 模块消息", clog.String("endpoint", "/health"))
	apiNamespace.Error("API 模块错误", clog.String("status", "500"))

	// Context 日志 - 使用全局函数
	clog.WithContext(ctx).Info("JSON Context 消息", clog.String("request_id", "req-123"))
	clog.WithContext(ctx).Namespace("database").Warn("数据库模块警告", clog.String("query", "SELECT * FROM users"))

	// 链式调用
	clog.WithContext(ctx).Namespace("redis").Info("Redis 模块信息", clog.String("operation", "SET"))

	// 读取并显示文件内容
	fmt.Println("\n文件内容:")
	content, err := os.ReadFile(logFile)
	if err != nil {
		fmt.Printf("读取日志文件失败: %v\n", err)
		return
	}

	fmt.Println(string(content))

	// 验证文件中是否包含 caller 信息
	if string(content) == "" {
		fmt.Println("❌ 日志文件为空")
	} else if !containsCaller(string(content)) {
		fmt.Println("❌ 日志文件中缺少 caller 信息")
	} else {
		fmt.Println("✅ 日志文件包含完整的 caller 信息")
	}
}

// containsCaller 检查日志内容是否包含 caller 信息
func containsCaller(content string) bool {
	return len(content) > 0 && (
	// JSON 格式应该包含 "caller" 字段
	contains(content, `"caller"`) ||
		// 或者包含文件路径信息
		contains(content, "main.go") ||
		contains(content, ".go:"))
}

// contains 简单的字符串包含检查
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring 查找子字符串的位置
func indexOfSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
