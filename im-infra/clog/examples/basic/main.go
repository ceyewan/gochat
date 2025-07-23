package main

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 基础使用示例 ===")

	// 1. 全局日志方法（最简单的使用方式）
	fmt.Println("1. 全局日志方法:")
	clog.Info("服务启动成功", clog.String("version", "1.0.0"))
	clog.Warn("配置文件缺失，使用默认配置", clog.String("config", "app.yaml"))
	clog.Error("数据库连接失败", clog.String("host", "localhost"), clog.Int("port", 5432))

	// 2. 带 Context 的全局日志方法（自动注入 TraceID）
	fmt.Println("\n2. 带 Context 的全局日志:")
	ctx := context.WithValue(context.Background(), "trace_id", "req-123")
	clog.InfoContext(ctx, "处理用户请求",
		clog.String("request_id", "req-123"),
		clog.String("method", "POST"),
		clog.String("path", "/api/users"))

	clog.WarnContext(ctx, "请求处理较慢",
		clog.String("request_id", "req-123"),
		clog.Duration("duration", 2000))

	clog.ErrorContext(ctx, "请求处理失败",
		clog.String("request_id", "req-123"),
		clog.String("reason", "timeout"))

	// 3. 创建独立的日志器实例
	fmt.Println("\n3. 创建独立日志器实例:")
	logger := clog.New()
	logger.Info("独立日志器实例创建成功", clog.String("type", "custom"))

	// 4. 使用 With 方法添加通用字段
	fmt.Println("\n4. 使用 With 方法:")
	serviceLogger := logger.With(
		clog.String("service", "user-service"),
		clog.String("version", "2.1.0"))

	serviceLogger.Info("服务初始化完成")
	serviceLogger.Info("配置加载完成", clog.Int("config_count", 15))

	// 5. 各种 Field 类型的使用
	fmt.Println("\n5. 各种 Field 类型:")
	clog.Info("字段类型展示",
		clog.String("string_field", "测试值"),
		clog.Int("int_field", 42),
		clog.Bool("bool_field", true),
		clog.Float64("float_field", 3.14159),
		clog.Strings("array_field", []string{"item1", "item2", "item3"}))

	// 6. 错误处理
	fmt.Println("\n6. 错误处理:")
	err := fmt.Errorf("测试错误: 连接超时")
	clog.Error("操作失败",
		clog.Err(err),
		clog.String("operation", "database_connect"),
		clog.Int("retry_count", 3))

	fmt.Println("\n=== 基础示例完成 ===")
	fmt.Println("查看日志输出格式:")
	fmt.Println("✅ JSON 格式，便于日志收集")
	fmt.Println("✅ 包含 timestamp、level、source、msg、fields")
	fmt.Println("✅ 自动注入 TraceID（使用 Context 方法时）")
}
