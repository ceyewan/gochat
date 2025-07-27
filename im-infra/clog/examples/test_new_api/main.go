package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 新 API 测试 ===")

	// 1. 基本日志功能
	fmt.Println("1. 基本日志功能:")
	clog.Info("服务启动", clog.String("version", "1.0.0"))
	clog.Warn("这是一个警告", clog.Int("code", 1001))
	clog.Error("这是一个错误", clog.String("error", "test error"))

	// 2. 带 Context 的日志 - 使用 C(ctx) 方法
	fmt.Println("\n2. 带 Context 的日志:")
	ctx := context.WithValue(context.Background(), "traceID", "test-trace-123")
	clog.C(ctx).Info("处理请求", clog.String("user", "alice"))
	clog.C(ctx).Warn("处理较慢", clog.Duration("duration", time.Second))

	// 3. 模块日志
	fmt.Println("\n3. 模块日志:")
	userModule := clog.Module("user")
	userModule.Info("用户模块初始化")
	userModule.With(clog.String("userID", "123")).Info("用户登录")

	// 4. 链式调用 - 展示 API 的简洁性
	fmt.Println("\n4. 链式调用:")
	clog.C(ctx).Module("order").With(clog.String("orderID", "order-456")).Info("订单创建")

	// 5. 自定义 logger
	fmt.Println("\n5. 自定义 logger:")
	customLogger := clog.New()
	customLogger.Info("自定义日志器", clog.String("type", "custom"))

	// 6. 测试 TraceID 自动注入
	fmt.Println("\n6. TraceID 自动注入测试:")
	ctx2 := context.WithValue(context.Background(), "trace_id", "auto-inject-456")
	clog.C(ctx2).Info("应该自动注入 TraceID", clog.String("action", "test"))

	// 7. 错误处理
	fmt.Println("\n7. 错误处理:")
	err := fmt.Errorf("数据库连接失败")
	clog.C(ctx).Error("操作失败", clog.Err(err), clog.String("operation", "connect"))

	// 8. 各种字段类型
	fmt.Println("\n8. 各种字段类型:")
	clog.Info("字段类型展示",
		clog.String("string", "测试"),
		clog.Int("int", 42),
		clog.Bool("bool", true),
		clog.Float64("float", 3.14),
		clog.Duration("duration", 100*time.Millisecond),
		clog.Time("time", time.Now()),
		clog.Strings("array", []string{"a", "b", "c"}))

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("✅ 新 API 设计验证:")
	fmt.Println("  - 使用 C(ctx) 替代 XxxContext 方法")
	fmt.Println("  - 支持链式调用: C(ctx).Module().With().Info()")
	fmt.Println("  - 自动 TraceID 注入")
	fmt.Println("  - 简洁的模块日志")
	fmt.Println("  - 直接使用 zap.Field，避免重复定义")
}
