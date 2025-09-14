package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 基础功能演示 ===")

	// 清理之前的日志文件
	logDir := "logs"
	os.RemoveAll(logDir)

	// 示例1: 环境默认配置比较 (dev vs prod)
	fmt.Println("\n--- 示例1: 环境默认配置 ---")
	demoEnvDefaults()

	// 示例2: Console 输出 (带颜色, AddSource, RootPath)
	fmt.Println("\n--- 示例2: Console 输出 ---")
	testConsoleOutput()

	// 示例3: JSON 文件输出 (stderr, AddSource)
	fmt.Println("\n--- 示例3: JSON 文件输出 (stderr) ---")
	testJSONStderrOutput()

	// 示例4: 所有日志级别 + Fatal (隔离演示)
	fmt.Println("\n--- 示例4: 日志级别 ---")
	testLogLevels()

	// 示例5: 所有结构化字段类型
	fmt.Println("\n--- 示例5: 结构化字段 ---")
	testAllFields()

	fmt.Println("\n=== 基础演示完成 ===")
}

// demoEnvDefaults 演示环境默认配置
func demoEnvDefaults() {
	devConfig := clog.GetDefaultConfig("development")
	fmt.Printf("开发环境配置: Level=%s, Format=%s, EnableColor=%t\n", devConfig.Level, devConfig.Format, devConfig.EnableColor)

	prodConfig := clog.GetDefaultConfig("production")
	fmt.Printf("生产环境配置: Level=%s, Format=%s, EnableColor=%t\n", prodConfig.Level, prodConfig.Format, prodConfig.EnableColor)

	// 使用 dev 配置初始化临时 logger
	if err := clog.Init(context.Background(), devConfig); err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		return
	}
	clog.Info("使用开发默认配置")
}

// testConsoleOutput Console 输出演示 (EnableColor, AddSource, RootPath)
func testConsoleOutput() {
	config := &clog.Config{
		Level:       "debug",
		Format:      "console",
		Output:      "stdout",
		AddSource:   true,
		EnableColor: true,
		RootPath:    filepath.Join(os.Getenv("PWD"), "im-infra", "clog"), // 示例 RootPath
	}

	logger, err := clog.New(context.Background(), config)
	if err != nil {
		fmt.Printf("创建 logger 失败: %v\n", err)
		return
	}

	ctx := clog.WithTraceID(context.Background(), "console-trace-123")

	// 基本日志
	logger.Info("Console Info 消息")
	logger.Debug("Console Debug 消息")

	// 命名空间
	userLogger := logger.Namespace("user")
	userLogger.Warn("Console 用户警告", clog.String("id", "123"))

	// Context
	clog.C(ctx).Error("Console Context 错误")

	// RootPath 效果在 AddSource 中显示相对路径
	logger.Info("带 RootPath 的源码位置")
}

// testJSONStderrOutput JSON stderr 输出
func testJSONStderrOutput() {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		return
	}

	logFile := filepath.Join(logDir, "basic_stderr.log")

	config := &clog.Config{
		Level:     "debug",
		Format:    "json",
		Output:    "stderr", // stderr 输出 (可重定向到文件)
		AddSource: true,
		RootPath:  filepath.Join(os.Getenv("PWD"), "im-infra", "clog"),
	}

	logger, err := clog.New(context.Background(), config)
	if err != nil {
		fmt.Printf("创建 logger 失败: %v\n", err)
		return
	}

	ctx := clog.WithTraceID(context.Background(), "stderr-trace-456")

	logger.Info("JSON stderr Info")
	clog.WithContext(ctx).Namespace("api").Warn("stderr API 警告")

	// 模拟 stderr 到文件 (实际运行时可重定向)
	fmt.Printf("日志输出到 stderr (可重定向: %s 2> %s)\n", config.Output, logFile)
}

// testLogLevels 所有日志级别 (Fatal 在隔离 func)
func testLogLevels() {
	config := &clog.Config{Level: "debug", Format: "console", Output: "stdout"}
	if err := clog.Init(context.Background(), config); err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		return
	}

	ctx := clog.WithTraceID(context.Background(), "levels-trace-789")

	clog.Debug("Debug 消息")
	clog.Info("Info 消息")
	clog.Warn("Warn 消息")
	clog.Error("Error 消息")

	// Fatal: 在隔离 func 以避免退出 main
	go func() {
		clog.WithContext(ctx).Fatal("Fatal 消息 (隔离演示, 程序退出)")
	}()

	time.Sleep(100 * time.Millisecond) // 允许 Fatal 执行
	fmt.Println("Fatal 演示后继续 (实际 Fatal 会退出)")
}

// testAllFields 所有字段类型
func testAllFields() {
	config := &clog.Config{Level: "debug", Format: "console", Output: "stdout"}
	if err := clog.Init(context.Background(), config); err != nil {
		return
	}

	ctx := clog.WithTraceID(context.Background(), "fields-trace-101")

	logger := clog.WithContext(ctx).Namespace("fields")

	logger.Info("所有字段演示",
		clog.String("string", "hello"),
		clog.Int("int", 42),
		clog.Bool("bool", true),
		clog.Float64("float64", 3.14),
		clog.Duration("duration", 5*time.Second),
		clog.Time("time", time.Now()),
		clog.Err(errors.New("example error")),
		clog.Any("any", map[string]int{"key": 1}),
	)
}
