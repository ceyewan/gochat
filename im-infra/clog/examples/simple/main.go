package main

import (
	"context"
	"errors"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	// 1. 基本使用 - 使用默认配置
	clog.Info("应用启动", clog.String("version", "1.0.0"))
	clog.Debug("调试信息", clog.Int("debug_level", 1))
	clog.Warning("这是一个警告", clog.String("reason", "配置缺失"))
	clog.Error("这是一个错误", clog.Err(errors.New("示例错误")))

	// 2. 层次化命名空间日志
	userLogger := clog.Namespace("user-service")
	userLogger.Info("用户操作",
		clog.String("user_id", "12345"),
		clog.String("action", "login"),
		clog.Duration("duration", 150*time.Millisecond))

	authLogger := clog.Namespace("auth-service")
	authLogger.Warn("认证失败",
		clog.String("user_id", "67890"),
		clog.String("reason", "密码错误"))

	// 3. 带上下文的日志
	ctx := clog.WithTraceID(context.Background(), "abc123def456")
	clog.C(ctx).Info("处理请求",
		clog.String("method", "POST"),
		clog.String("path", "/api/users"))

	// 4. 自定义配置示例
	config := clog.Config{
		Level:       "debug",
		Format:      "console",
		Output:      "stdout",
		AddSource:   true,
		EnableColor: true,
		RootPath:    "gochat",
	}

	// 初始化全局日志器
	if err := clog.Init(context.Background(), &config); err != nil {
		clog.Error("初始化日志器失败", clog.Err(err))
		return
	}

	clog.Info("日志器重新初始化完成", clog.String("format", "console"))

	// 5. 创建独立的日志器实例
	fileConfig := clog.Config{
		Level:  "info",
		Format: "json",
		Output: "/tmp/gochat-test.log",
		Rotation: &clog.RotationConfig{
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
	}

	fileLogger, err := clog.New(context.Background(), &fileConfig)
	if err != nil {
		clog.Error("创建文件日志器失败", clog.Err(err))
		return
	}

	fileLogger.Info("这条日志会写入文件",
		clog.String("file", "/tmp/gochat-test.log"),
		clog.Bool("rotation", true))

	clog.Info("clog 功能测试完成")

	// 注意：Fatal 会退出程序，所以放在最后演示
	// clog.Fatal("这是致命错误", clog.String("reason", "演示用途"))
}
