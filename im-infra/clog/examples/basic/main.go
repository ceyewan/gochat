package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 基础使用示例 ===")
	fmt.Println("使用默认配置（Console 格式，Info 级别，输出到 stdout，开发环境友好）")

	// 1. 全局日志方法（最简单的使用方式）
	fmt.Println("\n1. 全局日志方法:")
	clog.Debug("调试信息（默认不显示）", clog.String("level", "debug"))
	clog.Info("服务启动成功", clog.String("version", "1.0.0"), clog.String("env", "production"))
	clog.Warn("配置文件缺失，使用默认配置", clog.String("config", "app.yaml"))
	clog.Error("数据库连接失败", clog.String("host", "localhost"), clog.Int("port", 5432))

	// 2. 带 Context 的日志（推荐方式 - 使用 C(ctx)）
	fmt.Println("\n2. 带 Context 的日志（自动注入 TraceID）:")

	// 测试不同的 TraceID key 格式 - 使用字符串 key
	ctx1 := context.WithValue(context.Background(), "traceID", "trace-001")
	clog.C(ctx1).Info("使用 traceID key", clog.String("action", "user_login"))

	ctx2 := context.WithValue(context.Background(), "trace_id", "trace-002")
	clog.C(ctx2).Info("使用 trace_id key", clog.String("action", "order_create"))

	ctx3 := context.WithValue(context.Background(), "X-Trace-ID", "trace-003")
	clog.C(ctx3).Warn("使用 X-Trace-ID key", clog.String("warning", "slow_query"))

	// 3. 模块化日志（推荐方式）
	fmt.Println("\n3. 模块化日志:")
	userModule := clog.Module("user")
	userModule.Info("用户模块初始化完成")

	orderModule := clog.Module("order")
	orderModule.Info("订单模块初始化完成")

	// 注意：Module 方法只能基于默认 logger，不支持嵌套
	authModule := clog.Module("auth")
	authModule.Info("认证模块初始化完成")

	// 4. 链式调用（推荐方式 - 展示 API 简洁性）
	fmt.Println("\n4. 链式调用:")
	ctx := context.WithValue(context.Background(), "traceID", "chain-demo")

	// 一行代码完成复杂的日志记录
	clog.C(ctx).Module("payment").With(
		clog.String("orderID", "order-12345"),
		clog.String("userID", "user-789"),
	).Info("支付处理开始")

	// 模块日志器也支持链式调用
	orderModule.With(clog.String("status", "processing")).Info("订单状态更新")

	// 5. 使用 With 方法添加通用字段（性能优化）
	fmt.Println("\n5. 使用 With 方法（性能优化）:")

	// ✅ 推荐：缓存带有通用字段的 logger
	serviceLogger := clog.Module("user-service").With(
		clog.String("version", "2.1.0"),
		clog.String("instance", "srv-001"))

	serviceLogger.Info("服务初始化完成")
	serviceLogger.Info("配置加载完成", clog.Int("config_count", 15))
	serviceLogger.Warn("内存使用率较高", clog.Float64("memory_usage", 85.5))

	// 6. 各种 Field 类型的使用
	fmt.Println("\n6. 各种 Field 类型:")
	clog.Info("字段类型展示",
		clog.String("string", "测试值"),
		clog.Int("int", 42),
		clog.Int64("int64", 1234567890),
		clog.Bool("bool", true),
		clog.Float64("float", 3.14159),
		clog.Duration("duration", 100*time.Millisecond),
		clog.Time("timestamp", time.Now()),
		clog.Strings("array", []string{"item1", "item2", "item3"}),
		clog.Ints("numbers", []int{1, 2, 3, 4, 5}))

	// 7. 错误处理（推荐方式）
	fmt.Println("\n7. 错误处理:")
	err := fmt.Errorf("数据库连接超时")

	// 基本错误记录
	clog.Error("操作失败", clog.Err(err), clog.String("operation", "db_connect"))

	// 带 Context 的错误记录
	clog.C(ctx).Error("业务处理失败",
		clog.Err(err),
		clog.String("business", "user_registration"),
		clog.Int("retry_count", 3),
		clog.Duration("elapsed", 5*time.Second))

	// 8. 创建独立的日志器实例
	fmt.Println("\n8. 自定义日志器:")
	customLogger := clog.New() // 使用默认配置
	customLogger.Info("自定义日志器创建成功", clog.String("type", "custom"))

	// 自定义日志器也支持所有功能
	customLogger.Module("custom-module").With(
		clog.String("component", "worker"),
	).Info("自定义模块工作正常")

	// 9. 性能测试示例
	fmt.Println("\n9. 性能测试:")
	start := time.Now()

	// 测试缓存模块日志器的性能优势
	cachedLogger := clog.Module("performance")
	for i := 0; i < 10; i++ {
		cachedLogger.Info("性能测试", clog.Int("iteration", i))
	}

	elapsed := time.Since(start)
	clog.Info("性能测试完成",
		clog.Int("iterations", 10),
		clog.Duration("elapsed", elapsed),
		clog.String("recommendation", "使用缓存的模块日志器"))

	fmt.Println("\n=== 基础示例完成 ===")
	fmt.Println("🎯 API 设计亮点:")
	fmt.Println("  ✅ 使用 C(ctx) 统一处理 Context，替代多个 XxxContext 方法")
	fmt.Println("  ✅ 支持链式调用: C(ctx).Module().With().Info()")
	fmt.Println("  ✅ 自动 TraceID 注入，支持多种 key 格式")
	fmt.Println("  ✅ 模块化日志，支持嵌套模块")
	fmt.Println("  ✅ 直接使用 zap.Field，高性能零拷贝")
	fmt.Println("  ✅ Console 格式（开发友好），JSON 格式（生产环境）")
	fmt.Println("  ✅ 包含调用位置信息，便于调试")
	fmt.Println("  ✅ 支持文件轮转，智能文件管理")
}
