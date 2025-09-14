package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	// 场景 1: 在服务启动时初始化全局 Logger
	fmt.Println("=== 场景 1: 服务启动时初始化全局 Logger ===")

	// 使用默认配置（推荐）
	config := clog.GetDefaultConfig("production")

	// 初始化全局 logger
	if err := clog.Init(context.Background(), config, clog.WithNamespace("im-gateway")); err != nil {
		fmt.Printf("初始化 clog 失败: %v\n", err)
		return
	}

	clog.Info("服务启动成功") // 输出: {"namespace": "im-gateway", "msg": "服务启动成功"}

	// 场景 2: 层次化命名空间的使用
	fmt.Println("\n=== 场景 2: 层次化命名空间的使用 ===")
	demonstrateHierarchicalNamespaces()

	// 场景 3: 链式命名空间创建
	fmt.Println("\n=== 场景 3: 链式命名空间创建 ===")
	demonstrateChainedNamespaces()

	// 场景 4: Gin 中间件集成
	fmt.Println("\n=== 场景 4: Gin 中间件集成 ===")
	demonstrateGinMiddleware()

	// 场景 5: 上下文感知日志记录
	fmt.Println("\n=== 场景 5: 上下文感知日志记录 ===")
	demonstrateContextualLogging()

	fmt.Println("\n=== 所有示例演示完成 ===")
}

// demonstrateHierarchicalNamespaces 演示层次化命名空间的使用
func demonstrateHierarchicalNamespaces() {
	userLogger := clog.Namespace("user")
	authLogger := userLogger.Namespace("auth")
	dbLogger := userLogger.Namespace("database")

	userLogger.Info("开始用户注册流程", clog.String("email", "user@example.com"))

	authLogger.Info("验证用户密码强度")
	if !validatePassword("weak") {
		authLogger.Warn("密码强度不足")
	}

	dbLogger.Info("检查用户是否已存在")
	exists, err := checkUserExists("user@example.com")
	if err != nil {
		dbLogger.Error("查询用户失败", clog.Err(err))
		return
	}

	if exists {
		userLogger.Warn("用户已存在", clog.String("email", "user@example.com"))
	} else {
		userLogger.Info("用户注册成功")
	}
}

// demonstrateChainedNamespaces 演示链式命名空间创建
func demonstrateChainedNamespaces() {
	// 链式创建深层命名空间
	paymentLogger := clog.Namespace("payment").Namespace("processor").Namespace("stripe")

	paymentLogger.Info("开始处理支付请求",
		clog.String("order_id", "order_123"),
		clog.Float64("amount", 99.99))
}

// demonstrateGinMiddleware 演示 Gin 中间件集成
func demonstrateGinMiddleware() {
	// 创建 Gin 引擎
	r := gin.New()

	// 添加 Trace 中间件
	r.Use(TraceMiddleware())

	// 添加业务路由
	r.POST("/api/users", createUserHandler)

	// 模拟 HTTP 请求
	// 在实际应用中，这些会由真实的 HTTP 服务器处理
	fmt.Println("模拟 HTTP 请求处理...")

	// 创建模拟请求上下文
	ctx := context.Background()
	traceID := uuid.NewString()
	ctx = clog.WithTraceID(ctx, traceID)

	// 模拟业务处理
	handleUserCreation(ctx)
}

// TraceMiddleware 处理链路追踪 ID
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取或生成 traceID
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}

		// 2. 注入到 context（clog 核心用法）
		ctx := clog.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Header("X-Trace-ID", traceID)
		c.Next()
	}
}

// createUserHandler 业务处理函数
func createUserHandler(c *gin.Context) {
	// 3. 从 context 获取带 traceID 的 logger（clog 核心用法）
	logger := clog.WithContext(c.Request.Context())

	logger.Info("开始创建用户请求")

	// 调用服务层，传递 context
	user, err := createUser(c.Request.Context(), "testuser", "password123")
	if err != nil {
		logger.Error("创建用户失败", clog.Err(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	logger.Info("用户创建成功", clog.String("user_id", user.ID))
	c.JSON(200, user)
}

// handleUserCreation 模拟用户创建处理
func handleUserCreation(ctx context.Context) {
	// 从 context 获取带 traceID 的 logger
	logger := clog.WithContext(ctx)

	logger.Info("开始处理用户创建请求")

	// 模拟业务逻辑
	time.Sleep(100 * time.Millisecond)

	logger.Info("用户创建处理完成")
}

// demonstrateContextualLogging 演示上下文感知日志记录
func demonstrateContextualLogging() {
	// 创建带有 traceID 的 context
	traceID := uuid.NewString()
	ctx := clog.WithTraceID(context.Background(), traceID)

	// 模拟一个完整的业务流程
	processUserRegistration(ctx)
}

// processUserRegistration 处理用户注册流程
func processUserRegistration(ctx context.Context) {
	startTime := time.Now()
	// 从 context 获取带 traceID 的 logger
	logger := clog.WithContext(ctx)

	logger.Info("开始用户注册流程",
		clog.String("email", "newuser@example.com"),
		clog.Time("start_time", startTime))

	// 验证步骤
	if err := validateUserData(ctx, "newuser@example.com", "password123"); err != nil {
		logger.Error("用户数据验证失败",
			clog.Err(err),
			clog.Duration("duration", time.Since(startTime)),
			clog.Bool("success", false))
		return
	}

	// 数据库操作
	if err := saveUserToDatabase(ctx, "newuser@example.com"); err != nil {
		logger.Error("保存用户到数据库失败",
			clog.Err(err),
			clog.Duration("duration", time.Since(startTime)),
			clog.Any("context_data", map[string]interface{}{"email": "newuser@example.com"}))
		return
	}

	// 发送确认邮件
	if err := sendConfirmationEmail(ctx, "newuser@example.com"); err != nil {
		logger.Warn("发送确认邮件失败，但用户注册成功",
			clog.Err(err),
			clog.Duration("email_duration", time.Since(startTime)),
			clog.Bool("critical", false))
	}

	logger.Info("用户注册流程完成",
		clog.Duration("total_duration", time.Since(startTime)),
		clog.Bool("success", true))
}

// 以下是一些辅助函数，用于演示

func validatePassword(password string) bool {
	return len(password) >= 8
}

func checkUserExists(email string) (bool, error) {
	// 模拟数据库查询
	time.Sleep(50 * time.Millisecond)
	return false, nil
}

type User struct {
	ID    string
	Email string
}

func createUser(ctx context.Context, username, password string) (*User, error) {
	logger := clog.WithContext(ctx)

	logger.Info("创建用户", clog.String("username", username))

	// 模拟数据库操作
	time.Sleep(100 * time.Millisecond)

	user := &User{
		ID:    uuid.NewString(),
		Email: username + "@example.com",
	}

	return user, nil
}

func validateUserData(ctx context.Context, email, password string) error {
	logger := clog.WithContext(ctx).Namespace("validation")

	logger.Info("验证用户数据", clog.String("email", email))

	if len(password) < 8 {
		return fmt.Errorf("密码长度不足")
	}

	return nil
}

func saveUserToDatabase(ctx context.Context, email string) error {
	logger := clog.WithContext(ctx).Namespace("database")

	logger.Info("保存用户到数据库", clog.String("email", email))

	// 模拟数据库操作
	time.Sleep(150 * time.Millisecond)

	return nil
}

func sendConfirmationEmail(ctx context.Context, email string) error {
	logger := clog.WithContext(ctx).Namespace("email")

	logger.Info("发送确认邮件", clog.String("email", email))

	// 模拟邮件发送
	time.Sleep(200 * time.Millisecond)

	// 模拟偶尔失败
	if time.Now().Unix()%5 == 0 {
		return fmt.Errorf("邮件服务暂时不可用")
	}

	return nil
}
