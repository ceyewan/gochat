package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 高级用法示例 ===")

	// 1. With 方法的高级用法
	demonstrateWithMethod()

	// 2. 嵌套模块日志器
	demonstrateNestedModules()

	// 3. 并发安全性测试
	demonstrateConcurrencySafety()

	// 4. 性能优化最佳实践
	demonstratePerformanceOptimization()

	// 5. 复杂数据结构记录
	demonstrateComplexDataStructures()

	// 6. Context 链式传递
	demonstrateContextChaining()

	// 7. 错误处理模式
	demonstrateErrorHandlingPatterns()

	fmt.Println("\n=== 高级示例完成 ===")
}

// demonstrateWithMethod 演示 With 方法的高级用法
func demonstrateWithMethod() {
	fmt.Println("\n1. With 方法高级用法:")

	// 创建基础日志器
	baseLogger := clog.New()

	// 添加服务级别的通用字段
	serviceLogger := baseLogger.With(
		clog.String("service", "order-service"),
		clog.String("version", "2.1.0"),
		clog.String("environment", "production"),
		clog.String("region", "us-west-2"))

	serviceLogger.Info("服务启动完成")

	// 在服务日志器基础上添加更多字段
	requestLogger := serviceLogger.With(
		clog.String("request_id", "req-12345"),
		clog.String("user_id", "user-789"),
		clog.String("session_id", "sess-abc"))

	requestLogger.Info("开始处理用户请求")
	requestLogger.Info("请求验证通过")

	// 进一步细化到具体操作
	operationLogger := requestLogger.With(
		clog.String("operation", "create_order"),
		clog.String("order_type", "premium"))

	operationLogger.Info("订单创建开始")
	operationLogger.Info("订单创建成功", clog.String("order_id", "order-2024-001"))
}

// demonstrateNestedModules 演示嵌套模块日志器
func demonstrateNestedModules() {
	fmt.Println("\n2. 嵌套模块日志器:")

	// 创建主服务日志器
	mainService := clog.New().With(
		clog.String("service", "e-commerce-platform"),
		clog.String("instance", "platform-001"))

	// 从主服务创建子模块
	userModule := mainService.Module("user_management")
	orderModule := mainService.Module("order_processing")
	paymentModule := mainService.Module("payment_gateway")

	// 从子模块创建更细粒度的模块
	userAuthModule := userModule.Module("authentication")
	userProfileModule := userModule.Module("profile")

	orderCreationModule := orderModule.Module("creation")
	orderFulfillmentModule := orderModule.Module("fulfillment")

	// 使用嵌套模块记录日志
	userAuthModule.Info("用户认证模块初始化")
	userProfileModule.Info("用户资料模块初始化")

	orderCreationModule.Info("订单创建模块初始化")
	orderFulfillmentModule.Info("订单履行模块初始化")

	paymentModule.Info("支付网关模块初始化")

	// 模拟业务操作
	ctx := context.WithValue(context.Background(), "trace_id", "nested-op-001")
	userAuthModule.InfoContext(ctx, "用户登录成功",
		clog.String("username", "alice"),
		clog.String("login_method", "oauth"))

	orderCreationModule.InfoContext(ctx, "订单创建请求",
		clog.String("user_id", "user-123"),
		clog.Int("item_count", 3))
}

// demonstrateConcurrencySafety 演示并发安全性
func demonstrateConcurrencySafety() {
	fmt.Println("\n3. 并发安全性测试:")

	const goroutineCount = 100
	const messagesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutineCount)

	// 共享的模块日志器
	sharedLogger := clog.Module("concurrent_test")

	startTime := time.Now()

	// 启动多个 goroutine 并发写入日志
	for i := 0; i < goroutineCount; i++ {
		go func(id int) {
			defer wg.Done()

			// 每个 goroutine 创建自己的带有ID的日志器
			goroutineLogger := sharedLogger.With(clog.Int("goroutine_id", id))

			for j := 0; j < messagesPerGoroutine; j++ {
				goroutineLogger.Info("并发日志测试",
					clog.Int("message_num", j),
					clog.Int64("timestamp", time.Now().UnixNano()),
					clog.String("data", fmt.Sprintf("data-%d-%d", id, j)))

				// 模拟一些处理时间
				time.Sleep(time.Microsecond * 100)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	clog.Info("并发测试完成",
		clog.Int("goroutines", goroutineCount),
		clog.Int("messages_per_goroutine", messagesPerGoroutine),
		clog.Int("total_messages", goroutineCount*messagesPerGoroutine),
		clog.Duration("total_duration", duration),
		clog.Float64("messages_per_second", float64(goroutineCount*messagesPerGoroutine)/duration.Seconds()))
}

// demonstratePerformanceOptimization 演示性能优化最佳实践
func demonstratePerformanceOptimization() {
	fmt.Println("\n4. 性能优化最佳实践:")

	// ✅ 推荐：包级别缓存模块日志器
	var cachedLogger = clog.Module("performance_test")

	// 性能测试：缓存 vs 非缓存
	const iterations = 1000

	// 测试缓存日志器性能
	start := time.Now()
	for i := 0; i < iterations; i++ {
		cachedLogger.Info("缓存日志器测试", clog.Int("iteration", i))
	}
	cachedDuration := time.Since(start)

	// 测试非缓存日志器性能（不推荐的做法）
	start = time.Now()
	for i := 0; i < iterations; i++ {
		clog.Module("performance_test").Info("非缓存日志器测试", clog.Int("iteration", i))
	}
	nonCachedDuration := time.Since(start)

	clog.Info("性能对比结果",
		clog.Int("iterations", iterations),
		clog.Duration("cached_duration", cachedDuration),
		clog.Duration("non_cached_duration", nonCachedDuration),
		clog.Float64("performance_improvement", float64(nonCachedDuration)/float64(cachedDuration)))

	// ✅ 推荐：使用 With 方法复用通用字段
	baseLogger := clog.New().With(
		clog.String("service", "optimization_demo"),
		clog.String("version", "1.0.0"))

	start = time.Now()
	for i := 0; i < iterations; i++ {
		baseLogger.Info("复用字段测试", clog.Int("iteration", i))
	}
	withMethodDuration := time.Since(start)

	// ❌ 不推荐：每次都重复添加相同字段
	start = time.Now()
	for i := 0; i < iterations; i++ {
		clog.Info("重复字段测试",
			clog.String("service", "optimization_demo"),
			clog.String("version", "1.0.0"),
			clog.Int("iteration", i))
	}
	repeatedFieldsDuration := time.Since(start)

	clog.Info("字段复用性能对比",
		clog.Duration("with_method_duration", withMethodDuration),
		clog.Duration("repeated_fields_duration", repeatedFieldsDuration),
		clog.Float64("field_reuse_improvement", float64(repeatedFieldsDuration)/float64(withMethodDuration)))
}

// demonstrateComplexDataStructures 演示复杂数据结构的记录
func demonstrateComplexDataStructures() {
	fmt.Println("\n5. 复杂数据结构记录:")

	logger := clog.Module("complex_data")

	// 记录各种复杂的数据类型
	logger.Info("用户订单详情",
		clog.String("order_id", "ORDER-2024-001"),
		clog.String("user_id", "USER-789"),
		clog.Strings("product_ids", []string{"PROD-001", "PROD-002", "PROD-003"}),
		clog.Ints("quantities", []int{2, 1, 3}),
		clog.Int("total_amount", 15999),
		clog.Bool("is_premium_user", true),
		clog.Time("order_time", time.Now()),
		clog.Duration("processing_time", 1250*time.Millisecond))

	// 使用 Any 类型记录复杂结构
	orderData := map[string]interface{}{
		"order_id":    "ORDER-2024-002",
		"items":       []string{"laptop", "mouse", "keyboard"},
		"total":       2999.99,
		"shipping":    map[string]string{"method": "express", "address": "123 Main St"},
		"customer_id": 12345,
	}

	logger.Info("复杂订单数据", clog.Any("order_data", orderData))

	// 记录切片数据
	userRoles := []string{"admin", "user", "moderator"}
	permissions := []int{1, 2, 4, 8, 16}

	logger.Info("用户权限信息",
		clog.String("user_id", "USER-456"),
		clog.Strings("roles", userRoles),
		clog.Ints("permission_flags", permissions))

	// 记录二进制数据
	binaryData := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x57, 0x6f, 0x72, 0x6c, 0x64}
	logger.Info("二进制数据记录", clog.Binary("data", binaryData))
}

// demonstrateContextChaining 演示 Context 链式传递
func demonstrateContextChaining() {
	fmt.Println("\n6. Context 链式传递:")

	// 模拟一个完整的请求处理链
	ctx := context.WithValue(context.Background(), "trace_id", "chain-demo-001")

	handleHTTPRequest(ctx)
}

func handleHTTPRequest(ctx context.Context) {
	logger := clog.Module("http_handler")
	logger.InfoContext(ctx, "接收 HTTP 请求",
		clog.String("method", "POST"),
		clog.String("path", "/api/users"),
		clog.String("remote_addr", "192.168.1.100"))

	// 传递到认证层
	authenticateUser(ctx, "alice")
}

func authenticateUser(ctx context.Context, username string) {
	logger := clog.Module("auth")
	logger.InfoContext(ctx, "开始用户认证", clog.String("username", username))

	// 模拟认证处理
	time.Sleep(50 * time.Millisecond)

	logger.InfoContext(ctx, "用户认证成功",
		clog.String("username", username),
		clog.String("auth_method", "jwt"))

	// 传递到业务逻辑层
	processBusinessLogic(ctx, username)
}

func processBusinessLogic(ctx context.Context, username string) {
	logger := clog.Module("business")
	logger.InfoContext(ctx, "开始业务逻辑处理", clog.String("username", username))

	// 调用数据库层
	queryDatabase(ctx, "users", username)

	logger.InfoContext(ctx, "业务逻辑处理完成", clog.String("username", username))
}

func queryDatabase(ctx context.Context, table, username string) {
	logger := clog.Module("database")
	logger.InfoContext(ctx, "执行数据库查询",
		clog.String("table", table),
		clog.String("username", username),
		clog.String("query", "SELECT * FROM users WHERE username = ?"))

	// 模拟数据库查询
	time.Sleep(30 * time.Millisecond)

	logger.InfoContext(ctx, "数据库查询完成",
		clog.String("table", table),
		clog.Int("rows_returned", 1),
		clog.Duration("query_time", 28*time.Millisecond))
}

// demonstrateErrorHandlingPatterns 演示错误处理模式
func demonstrateErrorHandlingPatterns() {
	fmt.Println("\n7. 错误处理模式:")

	logger := clog.Module("error_handling")

	// 1. 基本错误记录
	err := fmt.Errorf("数据库连接失败: connection timeout")
	logger.Error("操作失败", clog.Err(err), clog.String("operation", "user_query"))

	// 2. 包装错误
	originalErr := fmt.Errorf("网络不可达")
	wrappedErr := fmt.Errorf("无法连接到支付网关: %w", originalErr)
	logger.Error("支付处理失败",
		clog.Err(wrappedErr),
		clog.String("payment_id", "PAY-123"),
		clog.String("gateway", "alipay"))

	// 3. 带上下文的错误记录
	ctx := context.WithValue(context.Background(), "trace_id", "error-demo-001")
	serviceErr := fmt.Errorf("服务暂时不可用")
	logger.ErrorContext(ctx, "下游服务调用失败",
		clog.Err(serviceErr),
		clog.String("service", "inventory-service"),
		clog.String("endpoint", "/api/inventory/check"),
		clog.Int("retry_count", 3))

	// 4. 错误恢复记录
	logger.InfoContext(ctx, "开始错误恢复流程",
		clog.String("recovery_strategy", "fallback_cache"))

	logger.InfoContext(ctx, "错误恢复成功",
		clog.String("data_source", "local_cache"),
		clog.Duration("recovery_time", 150*time.Millisecond))

	// 5. 性能相关的警告
	slowQueryTime := 2500 * time.Millisecond
	if slowQueryTime > 2*time.Second {
		logger.Warn("慢查询检测",
			clog.Duration("query_time", slowQueryTime),
			clog.String("query", "SELECT * FROM orders WHERE created_at > ?"),
			clog.String("suggestion", "考虑添加索引"))
	}

	// 6. 资源相关的错误
	logger.Error("资源耗尽",
		clog.String("resource_type", "memory"),
		clog.Int("current_usage_mb", 1024),
		clog.Int("limit_mb", 1024),
		clog.Float64("usage_percentage", 100.0))

	// 7. 业务逻辑错误
	logger.Warn("业务规则违反",
		clog.String("rule", "用户每日下单限制"),
		clog.String("user_id", "USER-789"),
		clog.Int("daily_orders", 10),
		clog.Int("limit", 10))
}
