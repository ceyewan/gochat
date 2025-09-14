package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/metrics"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var (
	// 模块化日志器
	mainLogger    = clog.Namespace("advanced.main")
	serviceLogger = clog.Namespace("advanced.service")
	grpcLogger    = clog.Namespace("advanced.grpc")
	httpLogger    = clog.Namespace("advanced.http")

	// 自定义业务指标
	businessCounter   *metrics.Counter
	responseTimeHist  *metrics.Histogram
	messageSizeHist   *metrics.Histogram
	connectionCounter *metrics.Counter
)

// UserService 模拟用户服务
type UserService struct {
	users sync.Map // 模拟用户存储
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string
	Password string
}

// LoginResponse 登录响应
type LoginResponse struct {
	Success bool
	Token   string
	UserID  int64
}

// MessageRequest 消息请求
type MessageRequest struct {
	UserID  int64
	Content string
	ToUser  int64
}

// MessageResponse 消息响应
type MessageResponse struct {
	MessageID int64
	Timestamp int64
}

func main() {
	mainLogger.Info("启动高级 metrics 示例应用")

	// 1. 创建高级配置的 metrics provider
	cfg := createAdvancedConfig()

	provider, err := metrics.New(cfg)
	if err != nil {
		mainLogger.Error("failed to create metrics provider", clog.Err(err))
		log.Fatalf("Failed to create metrics provider: %v", err)
	}

	// 2. 初始化自定义业务指标
	if err := initCustomMetrics(); err != nil {
		mainLogger.Error("failed to initialize custom metrics", clog.Err(err))
		log.Fatalf("Failed to initialize custom metrics: %v", err)
	}

	// 3. 启动服务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// 启动 gRPC 服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startGRPCServer(ctx, provider); err != nil {
			grpcLogger.Error("gRPC server failed", clog.Err(err))
		}
	}()

	// 启动 HTTP 服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startHTTPServer(ctx, provider); err != nil {
			httpLogger.Error("HTTP server failed", clog.Err(err))
		}
	}()

	// 启动模拟业务指标生成器
	wg.Add(1)
	go func() {
		defer wg.Done()
		simulateBusinessMetrics(ctx)
	}()

	// 4. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	mainLogger.Info("收到退出信号，开始优雅关闭")
	cancel()

	// 5. 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	mainLogger.Info("关闭 metrics provider")
	if err := provider.Shutdown(shutdownCtx); err != nil {
		mainLogger.Error("failed to shutdown metrics provider", clog.Err(err))
	}

	// 等待所有 goroutine 结束
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mainLogger.Info("所有服务已优雅关闭")
	case <-shutdownCtx.Done():
		mainLogger.Warn("关闭超时，强制退出")
	}
}

// createAdvancedConfig 创建高级配置
func createAdvancedConfig() *metrics.Config {
	cfg := metrics.DefaultConfig()

	// 从环境变量读取配置，提供默认值
	cfg.ServiceName = getEnvOrDefault("SERVICE_NAME", "advanced-metrics-demo")
	cfg.ExporterType = getEnvOrDefault("EXPORTER_TYPE", "stdout")
	cfg.ExporterEndpoint = getEnvOrDefault("EXPORTER_ENDPOINT", "http://localhost:14268/api/traces")
	cfg.PrometheusListenAddr = getEnvOrDefault("PROMETHEUS_ADDR", ":9091")
	cfg.SamplerType = getEnvOrDefault("SAMPLER_TYPE", "trace_id_ratio")

	// 在生产环境中使用较低的采样率
	if samplerRatio := getEnvOrDefault("SAMPLER_RATIO", "0.1"); samplerRatio != "" {
		if ratio, err := parseFloat(samplerRatio); err == nil {
			cfg.SamplerRatio = ratio
		}
	}

	// 根据服务类型调整慢请求阈值
	cfg.SlowRequestThreshold = 200 * time.Millisecond

	mainLogger.Info("使用高级配置创建 metrics provider",
		clog.String("service_name", cfg.ServiceName),
		clog.String("exporter_type", cfg.ExporterType),
		clog.String("prometheus_addr", cfg.PrometheusListenAddr),
		clog.String("sampler_type", cfg.SamplerType),
		clog.Float64("sampler_ratio", cfg.SamplerRatio),
		clog.Duration("slow_threshold", cfg.SlowRequestThreshold))

	return cfg
}

// initCustomMetrics 初始化自定义业务指标
func initCustomMetrics() error {
	var err error

	// 业务事件计数器
	businessCounter, err = metrics.NewCounter(
		"business_events_total",
		"Total number of business events by type and status",
	)
	if err != nil {
		return fmt.Errorf("failed to create business counter: %w", err)
	}

	// 响应时间直方图
	responseTimeHist, err = metrics.NewHistogram(
		"business_response_duration_seconds",
		"Business operation response time in seconds",
		"s",
	)
	if err != nil {
		return fmt.Errorf("failed to create response time histogram: %w", err)
	}

	// 消息大小直方图
	messageSizeHist, err = metrics.NewHistogram(
		"message_size_bytes",
		"Size of processed messages in bytes",
		"bytes",
	)
	if err != nil {
		return fmt.Errorf("failed to create message size histogram: %w", err)
	}

	// 连接数计数器
	connectionCounter, err = metrics.NewCounter(
		"active_connections_total",
		"Total number of active connections by type",
	)
	if err != nil {
		return fmt.Errorf("failed to create connection counter: %w", err)
	}

	serviceLogger.Info("自定义业务指标初始化完成")
	return nil
}

// startGRPCServer 启动 gRPC 服务器
func startGRPCServer(ctx context.Context, provider metrics.Provider) error {
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		return fmt.Errorf("failed to listen on :8081: %w", err)
	}

	// 创建 gRPC 服务器，集成 metrics 拦截器
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			provider.GRPCServerInterceptor(),
			loggingInterceptor(), // 自定义日志拦截器
		),
	)

	// 注册服务（这里只是示例，实际应用中会注册真实的服务）
	reflection.Register(server)

	grpcLogger.Info("gRPC 服务器启动", clog.String("address", ":8081"))

	// 记录连接指标
	connectionCounter.Inc(ctx, attribute.String("type", "grpc_server"))

	// 启动服务器
	go func() {
		if err := server.Serve(lis); err != nil {
			grpcLogger.Error("gRPC server serve failed", clog.Err(err))
		}
	}()

	// 等待上下文取消
	<-ctx.Done()
	grpcLogger.Info("正在关闭 gRPC 服务器")
	server.GracefulStop()
	grpcLogger.Info("gRPC 服务器已关闭")

	return nil
}

// startHTTPServer 启动 HTTP 服务器
func startHTTPServer(ctx context.Context, provider metrics.Provider) error {
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// 添加 metrics 中间件和自定义中间件
	engine.Use(
		provider.HTTPMiddleware(),
		corsMiddleware(),
		recoveryMiddleware(),
	)

	// 注册路由
	setupRoutes(engine)

	server := &http.Server{
		Addr:           ":8080",
		Handler:        engine,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	httpLogger.Info("HTTP 服务器启动", clog.String("address", ":8080"))

	// 记录连接指标
	connectionCounter.Inc(ctx, attribute.String("type", "http_server"))

	// 启动服务器
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpLogger.Error("HTTP server serve failed", clog.Err(err))
		}
	}()

	// 等待上下文取消
	<-ctx.Done()
	httpLogger.Info("正在关闭 HTTP 服务器")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		httpLogger.Error("HTTP server shutdown failed", clog.Err(err))
		return err
	}

	httpLogger.Info("HTTP 服务器已关闭")
	return nil
}

// setupRoutes 设置 HTTP 路由
func setupRoutes(engine *gin.Engine) {
	userService := &UserService{}

	// API v1 路由组
	v1 := engine.Group("/api/v1")
	{
		// 用户相关路由
		users := v1.Group("/users")
		{
			users.POST("/login", userService.handleLogin)
			users.GET("/:id/profile", userService.handleGetProfile)
			users.PUT("/:id/profile", userService.handleUpdateProfile)
		}

		// 消息相关路由
		messages := v1.Group("/messages")
		{
			messages.POST("/send", userService.handleSendMessage)
			messages.GET("/history/:userID", userService.handleGetHistory)
		}

		// 健康检查
		v1.GET("/health", handleHealthCheck)

		// 指标端点（除了 Prometheus 自动暴露的）
		v1.GET("/metrics/custom", handleCustomMetrics)
	}

	// 静态文件服务
	engine.Static("/static", "./static")
	engine.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "advanced-metrics-demo",
			"version": "1.0.0",
			"status":  "running",
		})
	})
}

// 用户服务处理函数
func (s *UserService) handleLogin(c *gin.Context) {
	start := time.Now()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		businessCounter.Inc(c.Request.Context(),
			attribute.String("operation", "login"),
			attribute.String("status", "invalid_request"))
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	serviceLogger.Info("处理登录请求",
		clog.String("username", req.Username))

	// 模拟业务逻辑和数据库查询
	time.Sleep(50 * time.Millisecond)

	// 简单的模拟验证
	success := req.Username != "" && req.Password == "password123"

	response := LoginResponse{
		Success: success,
		Token:   fmt.Sprintf("token_%s_%d", req.Username, time.Now().Unix()),
		UserID:  12345,
	}

	// 记录业务指标
	status := "success"
	if !success {
		status = "failed"
	}

	businessCounter.Inc(c.Request.Context(),
		attribute.String("operation", "login"),
		attribute.String("status", status))

	responseTimeHist.Record(c.Request.Context(),
		time.Since(start).Seconds(),
		attribute.String("operation", "login"),
		attribute.String("status", status))

	if success {
		c.JSON(200, response)
	} else {
		c.JSON(401, gin.H{"error": "Invalid credentials"})
	}
}

func (s *UserService) handleGetProfile(c *gin.Context) {
	start := time.Now()
	userID := c.Param("id")

	serviceLogger.Debug("获取用户资料", clog.String("user_id", userID))

	// 模拟数据库查询
	time.Sleep(30 * time.Millisecond)

	profile := gin.H{
		"user_id":    userID,
		"username":   "demo_user",
		"email":      "demo@example.com",
		"created_at": time.Now().Add(-30 * 24 * time.Hour).Unix(),
	}

	businessCounter.Inc(c.Request.Context(),
		attribute.String("operation", "get_profile"),
		attribute.String("status", "success"))

	responseTimeHist.Record(c.Request.Context(),
		time.Since(start).Seconds(),
		attribute.String("operation", "get_profile"))

	c.JSON(200, profile)
}

func (s *UserService) handleUpdateProfile(c *gin.Context) {
	start := time.Now()
	userID := c.Param("id")

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		businessCounter.Inc(c.Request.Context(),
			attribute.String("operation", "update_profile"),
			attribute.String("status", "invalid_request"))
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	serviceLogger.Info("更新用户资料",
		clog.String("user_id", userID),
		clog.Int("field_count", len(updateData)))

	// 模拟数据库更新
	time.Sleep(80 * time.Millisecond)

	businessCounter.Inc(c.Request.Context(),
		attribute.String("operation", "update_profile"),
		attribute.String("status", "success"))

	responseTimeHist.Record(c.Request.Context(),
		time.Since(start).Seconds(),
		attribute.String("operation", "update_profile"))

	c.JSON(200, gin.H{"message": "Profile updated successfully"})
}

func (s *UserService) handleSendMessage(c *gin.Context) {
	start := time.Now()

	var req MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		businessCounter.Inc(c.Request.Context(),
			attribute.String("operation", "send_message"),
			attribute.String("status", "invalid_request"))
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// 记录消息大小
	messageSize := float64(len(req.Content))
	messageSizeHist.Record(c.Request.Context(), messageSize,
		attribute.String("message_type", "text"))

	serviceLogger.Info("发送消息",
		clog.Int64("from_user", req.UserID),
		clog.Int64("to_user", req.ToUser),
		clog.Int("message_size", len(req.Content)))

	// 模拟消息处理
	time.Sleep(25 * time.Millisecond)

	response := MessageResponse{
		MessageID: time.Now().UnixNano(),
		Timestamp: time.Now().Unix(),
	}

	businessCounter.Inc(c.Request.Context(),
		attribute.String("operation", "send_message"),
		attribute.String("status", "success"))

	responseTimeHist.Record(c.Request.Context(),
		time.Since(start).Seconds(),
		attribute.String("operation", "send_message"))

	c.JSON(200, response)
}

func (s *UserService) handleGetHistory(c *gin.Context) {
	start := time.Now()
	userID := c.Param("userID")

	serviceLogger.Debug("获取消息历史", clog.String("user_id", userID))

	// 模拟数据库查询
	time.Sleep(120 * time.Millisecond)

	// 模拟返回消息列表
	messages := []gin.H{
		{
			"message_id": 1001,
			"from_user":  userID,
			"to_user":    "2002",
			"content":    "Hello, world!",
			"timestamp":  time.Now().Add(-1 * time.Hour).Unix(),
		},
		{
			"message_id": 1002,
			"from_user":  "2002",
			"to_user":    userID,
			"content":    "Hi there!",
			"timestamp":  time.Now().Add(-30 * time.Minute).Unix(),
		},
	}

	businessCounter.Inc(c.Request.Context(),
		attribute.String("operation", "get_history"),
		attribute.String("status", "success"))

	responseTimeHist.Record(c.Request.Context(),
		time.Since(start).Seconds(),
		attribute.String("operation", "get_history"))

	c.JSON(200, gin.H{
		"messages": messages,
		"total":    len(messages),
	})
}

// handleHealthCheck 健康检查处理函数
func handleHealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "advanced-metrics-demo",
	})
}

// handleCustomMetrics 自定义指标端点
func handleCustomMetrics(c *gin.Context) {
	// 这里可以返回一些自定义的指标信息
	// 注意：这不会替代 Prometheus /metrics 端点
	c.JSON(200, gin.H{
		"message": "Custom metrics available at /metrics endpoint",
		"info":    "This is a demo endpoint for custom business metrics",
	})
}

// 中间件函数
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		httpLogger.Error("HTTP request panic recovered",
			clog.String("method", c.Request.Method),
			clog.String("path", c.Request.URL.Path),
			clog.Any("panic", recovered))

		businessCounter.Inc(c.Request.Context(),
			attribute.String("operation", "http_request"),
			attribute.String("status", "panic"))

		c.JSON(500, gin.H{"error": "Internal server error"})
	})
}

func loggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		grpcLogger.Debug("gRPC 请求开始", clog.String("method", info.FullMethod))

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			}
		}

		grpcLogger.Info("gRPC 请求完成",
			clog.String("method", info.FullMethod),
			clog.Duration("duration", duration),
			clog.String("status", code.String()))

		businessCounter.Inc(ctx,
			attribute.String("operation", "grpc_request"),
			attribute.String("method", info.FullMethod),
			attribute.String("status", code.String()))

		responseTimeHist.Record(ctx, duration.Seconds(),
			attribute.String("operation", "grpc_request"),
			attribute.String("method", info.FullMethod))

		return resp, err
	}
}

// simulateBusinessMetrics 模拟业务指标生成
func simulateBusinessMetrics(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	serviceLogger.Info("开始模拟业务指标生成")

	for {
		select {
		case <-ctx.Done():
			serviceLogger.Info("停止业务指标生成")
			return
		case <-ticker.C:
			// 模拟一些后台业务指标
			businessCounter.Inc(ctx,
				attribute.String("operation", "background_task"),
				attribute.String("status", "completed"))

			// 模拟消息处理指标
			messageSizeHist.Record(ctx, float64(512+time.Now().UnixNano()%1024),
				attribute.String("message_type", "background"))

			serviceLogger.Debug("生成模拟业务指标")
		}
	}
}

// 辅助函数
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
