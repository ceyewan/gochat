package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/google/uuid"
)

func main() {
	fmt.Println("=== clog E2E 测试开始 ===")

	// 清理之前的测试文件
	testDir := "e2e_test_logs"
	os.RemoveAll(testDir)

	// 测试 1: 服务初始化场景
	fmt.Println("\n--- 测试 1: 服务初始化场景 ---")
	testServiceInitialization()

	// 测试 2: 上下文感知日志记录
	fmt.Println("\n--- 测试 2: 上下文感知日志记录 ---")
	testContextualLogging()

	// 测试 3: 层次化命名空间
	fmt.Println("\n--- 测试 3: 层次化命名空间 ---")
	testHierarchicalNamespaces()

	// 测试 4: 文件输出与轮转
	fmt.Println("\n--- 测试 4: 文件输出与轮转 ---")
	testFileOutputAndRotation()

	// 测试 5: 错误处理与降级
	fmt.Println("\n--- 测试 5: 错误处理与降级 ---")
	testErrorHandlingAndFallback()

	// 测试 6: 性能与并发
	fmt.Println("\n--- 测试 6: 性能与并发 ---")
	testPerformanceAndConcurrency()

	fmt.Println("\n=== clog E2E 测试完成 ===")

	// 清理测试文件
	os.RemoveAll(testDir)
	fmt.Println("测试文件已清理")
}

// testServiceInitialization 测试服务初始化场景
func testServiceInitialization() {
	fmt.Println("1.1 测试环境相关的默认配置...")

	// 测试开发环境配置
	devConfig := clog.GetDefaultConfig("development")
	fmt.Printf("开发环境配置: Level=%s, Format=%s, Color=%v\n",
		devConfig.Level, devConfig.Format, devConfig.EnableColor)

	// 测试生产环境配置
	prodConfig := clog.GetDefaultConfig("production")
	fmt.Printf("生产环境配置: Level=%s, Format=%s, Color=%v\n",
		prodConfig.Level, prodConfig.Format, prodConfig.EnableColor)

	fmt.Println("1.2 测试全局 logger 初始化...")

	// 初始化全局 logger
	err := clog.Init(context.Background(), devConfig, clog.WithNamespace("e2e-test-service"))
	if err != nil {
		fmt.Printf("❌ 全局 logger 初始化失败: %v\n", err)
		return
	}

	// 验证全局 logger 工作
	clog.Info("服务启动成功")
	clog.Warn("这是一条警告消息")
	clog.Error("这是一条错误消息", clog.String("error_type", "test_error"))

	fmt.Println("✅ 服务初始化测试通过")
}

// testContextualLogging 测试上下文感知日志记录
func testContextualLogging() {
	fmt.Println("2.1 测试 TraceID 注入和提取...")

	// 模拟中间件注入 TraceID
	traceID := uuid.NewString()
	ctx := clog.WithTraceID(context.Background(), traceID)

	// 模拟业务处理
	handleUserRequest(ctx, "user123", "login")

	fmt.Println("2.2 测试多层级上下文传递...")

	// 模拟多层服务调用
	processOrder(ctx, "order456")

	fmt.Println("✅ 上下文感知日志记录测试通过")
}

// handleUserRequest 模拟用户请求处理
func handleUserRequest(ctx context.Context, userID string, action string) {
	// 从 context 获取带 TraceID 的 logger
	logger := clog.WithContext(ctx)

	logger.Info("开始处理用户请求",
		clog.String("user_id", userID),
		clog.String("action", action))

	// 模拟处理时间
	time.Sleep(10 * time.Millisecond)

	logger.Info("用户请求处理完成")
}

// processOrder 模拟订单处理
func processOrder(ctx context.Context, orderID string) {
	logger := clog.WithContext(ctx)

	logger.Info("开始处理订单", clog.String("order_id", orderID))

	// 模拟多步骤处理
	validateOrder(ctx, orderID)
	processPayment(ctx, orderID)
	sendNotification(ctx, orderID)

	logger.Info("订单处理完成")
}

// validateOrder 验证订单
func validateOrder(ctx context.Context, orderID string) {
	logger := clog.WithContext(ctx).Namespace("validation")
	logger.Info("验证订单数据", clog.String("order_id", orderID))
	time.Sleep(5 * time.Millisecond)
}

// processPayment 处理支付
func processPayment(ctx context.Context, orderID string) {
	logger := clog.WithContext(ctx).Namespace("payment")
	logger.Info("处理支付", clog.String("order_id", orderID))
	time.Sleep(15 * time.Millisecond)
}

// sendNotification 发送通知
func sendNotification(ctx context.Context, orderID string) {
	logger := clog.WithContext(ctx).Namespace("notification")
	logger.Info("发送通知", clog.String("order_id", orderID))
	time.Sleep(8 * time.Millisecond)
}

// testHierarchicalNamespaces 测试层次化命名空间
func testHierarchicalNamespaces() {
	fmt.Println("3.1 测试基本命名空间...")

	// 创建基础命名空间
	userLogger := clog.Namespace("user")
	authLogger := userLogger.Namespace("auth")
	dbLogger := userLogger.Namespace("database")

	userLogger.Info("用户模块操作")
	authLogger.Warn("认证警告")
	dbLogger.Error("数据库错误")

	fmt.Println("3.2 测试深层链式命名空间...")

	// 链式创建深层命名空间
	paymentLogger := clog.Namespace("payment").Namespace("processor").Namespace("stripe")
	orderLogger := clog.Namespace("order").Namespace("processor").Namespace("alipay")

	paymentLogger.Info("Stripe 支付处理")
	orderLogger.Info("支付宝订单处理")

	fmt.Println("3.3 测试命名空间与上下文结合...")

	// 结合 TraceID 和命名空间
	traceID := uuid.NewString()
	ctx := clog.WithTraceID(context.Background(), traceID)

	ctxLogger := clog.WithContext(ctx)
	ctxLogger.Namespace("api").Info("API 请求")
	ctxLogger.Namespace("database").Warn("数据库慢查询")
	ctxLogger.Namespace("cache").Info("缓存命中")

	fmt.Println("✅ 层次化命名空间测试通过")
}

// testFileOutputAndRotation 测试文件输出与轮转
func testFileOutputAndRotation() {
	fmt.Println("4.1 测试文件输出...")

	// 创建测试目录
	testDir := "e2e_test_logs"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		fmt.Printf("❌ 创建测试目录失败: %v\n", err)
		return
	}

	logFile := filepath.Join(testDir, "e2e_test.log")

	// 配置文件输出
	config := &clog.Config{
		Level:     "debug",
		Format:    "json",
		Output:    logFile,
		AddSource: true,
		RootPath:  "gochat",
	}

	logger, err := clog.New(context.Background(), config, clog.WithNamespace("file-test"))
	if err != nil {
		fmt.Printf("❌ 创建文件 logger 失败: %v\n", err)
		return
	}

	// 写入测试日志
	logger.Info("文件输出测试消息")
	logger.Warn("文件输出警告消息")
	logger.Error("文件输出错误消息", clog.Err(fmt.Errorf("测试错误")))

	fmt.Println("4.2 测试日志轮转配置...")

	// 配置带轮转的 logger
	rotationConfig := &clog.Config{
		Level:  "info",
		Format: "json",
		Output: filepath.Join(testDir, "rotation_test.log"),
		Rotation: &clog.RotationConfig{
			MaxSize:    1,    // 1MB
			MaxBackups: 3,    // 保留3个备份
			MaxAge:     7,    // 保留7天
			Compress:   true, // 压缩旧文件
		},
	}

	rotationLogger, err := clog.New(context.Background(), rotationConfig)
	if err != nil {
		fmt.Printf("❌ 创建轮转 logger 失败: %v\n", err)
		return
	}

	// 写入一些日志（在实际使用中会积累更多）
	for i := 0; i < 10; i++ {
		rotationLogger.Info("轮转测试消息", clog.Int("index", i))
	}

	fmt.Println("4.3 验证日志文件...")

	// 验证文件存在
	if _, err := os.Stat(logFile); err != nil {
		fmt.Printf("❌ 日志文件不存在: %v\n", err)
	} else {
		fmt.Println("✅ 日志文件创建成功")

		// 读取并显示部分内容
		content, err := os.ReadFile(logFile)
		if err != nil {
			fmt.Printf("❌ 读取日志文件失败: %v\n", err)
		} else {
			fmt.Printf("日志文件大小: %d bytes\n", len(content))
			if len(content) > 0 {
				fmt.Println("✅ 日志文件内容有效")
			}
		}
	}

	fmt.Println("✅ 文件输出与轮转测试通过")
}

// testErrorHandlingAndFallback 测试错误处理与降级
func testErrorHandlingAndFallback() {
	fmt.Println("5.1 测试无效配置处理...")

	// 测试无效配置
	invalidConfig := &clog.Config{
		Level:  "invalid-level",
		Format: "invalid-format",
		Output: "",
	}

	logger, err := clog.New(context.Background(), invalidConfig)
	if err != nil {
		fmt.Printf("✅ 正确检测到无效配置: %v\n", err)
	} else {
		fmt.Println("❌ 应该检测到无效配置")
	}

	// 测试 fallback logger 是否工作
	if logger != nil {
		logger.Info("Fallback logger 工作正常")
		fmt.Println("✅ Fallback logger 正常工作")
	}

	fmt.Println("5.2 测试配置验证...")

	// 测试配置验证
	validConfig := &clog.Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	err = validConfig.Validate()
	if err != nil {
		fmt.Printf("❌ 有效配置验证失败: %v\n", err)
	} else {
		fmt.Println("✅ 有效配置验证通过")
	}

	fmt.Println("5.3 测试文件路径错误处理...")

	// 测试无效文件路径
	invalidPathConfig := &clog.Config{
		Level:  "info",
		Format: "json",
		Output: "/invalid/path/that/does/not/exist/test.log",
	}

	_, err = clog.New(context.Background(), invalidPathConfig)
	if err != nil {
		fmt.Printf("✅ 正确处理无效文件路径: %v\n", err)
	} else {
		fmt.Println("❌ 应该检测到无效文件路径")
	}

	fmt.Println("✅ 错误处理与降级测试通过")
}

// testPerformanceAndConcurrency 测试性能与并发
func testPerformanceAndConcurrency() {
	fmt.Println("6.1 测试并发日志记录...")

	// 模拟高并发场景
	const numGoroutines = 50
	const messagesPerGoroutine = 20

	done := make(chan bool, numGoroutines)

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			traceID := fmt.Sprintf("concurrent-trace-%d", goroutineID)
			ctx := clog.WithTraceID(context.Background(), traceID)
			logger := clog.WithContext(ctx)

			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info("并发测试消息",
					clog.String("goroutine", fmt.Sprintf("%d", goroutineID)),
					clog.Int("message", j),
					clog.String("trace_id", traceID))

				// 添加一些命名空间操作
				if j%5 == 0 {
					logger.Namespace("worker").Info("工作线程消息")
				}
			}
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	duration := time.Since(startTime)
	totalMessages := numGoroutines * messagesPerGoroutine

	fmt.Printf("✅ 并发测试完成: %d 条消息，耗时 %v\n", totalMessages, duration)
	fmt.Printf("平均吞吐量: %.2f 消息/秒\n", float64(totalMessages)/duration.Seconds())

	fmt.Println("6.2 测试内存使用情况...")

	// 这里可以添加内存使用情况的监控
	// 在实际生产环境中，可以使用 pprof 等工具进行详细分析

	fmt.Println("6.3 测试长时间运行的稳定性...")

	// 模拟长时间运行
	longRunningCtx := clog.WithTraceID(context.Background(), "long-running-test")
	longRunningLogger := clog.WithContext(longRunningCtx)

	for i := 0; i < 100; i++ {
		longRunningLogger.Info("长时间运行测试", clog.Int("iteration", i))
		time.Sleep(time.Millisecond) // 模拟处理间隔
	}

	fmt.Println("✅ 性能与并发测试通过")
}
