package server

import (
	"context"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-gateway/internal/config"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/mq"
)

// Server 定义 im-gateway 服务器接口
type Server interface {
	// Start 启动服务器
	Start() error

	// Shutdown 优雅关闭服务器
	Shutdown(ctx context.Context) error
}

// server 服务器实现
type server struct {
	config     *config.Config
	httpServer *HTTPServer
	logger     clog.Logger

	// 组件
	coordinator coord.Provider
	grpcClient  *GRPCClient
	mqProducer  mq.Producer
	mqConsumer  *KafkaConsumer
	wsManager   *WebSocketManager

	// 等待组
	wg sync.WaitGroup
}

// New 创建新的服务器实例
func New(cfg *config.Config) (Server, error) {
	logger := clog.Module("server")

	s := &server{
		config: cfg,
		logger: logger,
	}

	// 初始化各个组件
	if err := s.initComponents(); err != nil {
		return nil, err
	}

	// 初始化 HTTP 服务器
	s.httpServer = NewHTTPServer(cfg)
	s.httpServer.RegisterRoutes()

	// 设置 WebSocket 路由
	s.httpServer.Engine().GET(cfg.Server.WSPath, s.wsManager.HandleWebSocket)

	return s, nil
}

// initComponents 初始化所有组件
func (s *server) initComponents() error {
	var err error

	// 初始化协调器
	s.coordinator, err = coord.New(context.Background(), s.config.Coordinator,
		coord.WithLogger(s.logger))
	if err != nil {
		return err
	}

	// 初始化 Kafka 生产者
	s.mqProducer, err = mq.NewProducer(s.config.Kafka)
	if err != nil {
		return err
	}

	// 初始化 gRPC 客户端
	s.grpcClient, err = NewGRPCClient(s.config, s.coordinator)
	if err != nil {
		return err
	}

	// 初始化 WebSocket 管理器
	s.wsManager = NewWebSocketManager(s.config, s.mqProducer)

	// 初始化 Kafka 消费者
	s.mqConsumer, err = NewKafkaConsumer(s.config, s.mqProducer, s.wsManager)
	if err != nil {
		return err
	}

	return nil
}

// Start 启动服务器
func (s *server) Start() error {
	s.logger.Info("启动 im-gateway 服务器")

	// 启动 WebSocket 管理器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.wsManager.Start()
	}()

	// 启动 Kafka 消费者
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.mqConsumer.Start(context.Background()); err != nil {
			s.logger.Error("Kafka 消费器启动失败", clog.Err(err))
		}
	}()

	// 注册服务
	if err := s.registerService(); err != nil {
		return err
	}

	// 启动 HTTP 服务器
	s.logger.Info("HTTP 服务器启动", clog.String("addr", s.config.Server.HTTPAddr))
	return s.httpServer.Engine().Run(s.config.Server.HTTPAddr)
}

// Shutdown 优雅关闭服务器
func (s *server) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭服务器...")

	// 注销服务
	if err := s.deregisterService(); err != nil {
		s.logger.Error("注销服务失败", clog.Err(err))
	}

	// 关闭 Kafka 消费者
	if s.mqConsumer != nil {
		if err := s.mqConsumer.Stop(); err != nil {
			s.logger.Error("关闭 Kafka 消费者失败", clog.Err(err))
		}
	}

	// 关闭 WebSocket 管理器
	if s.wsManager != nil {
		s.wsManager.Stop()
	}

	// 关闭 gRPC 客户端
	if s.grpcClient != nil {
		if err := s.grpcClient.Close(); err != nil {
			s.logger.Error("关闭 gRPC 客户端失败", clog.Err(err))
		}
	}

	// 关闭 Kafka 生产者
	if s.mqProducer != nil {
		if err := s.mqProducer.Close(); err != nil {
			s.logger.Error("关闭 Kafka 生产者失败", clog.Err(err))
		}
	}

	// 关闭协调器
	if s.coordinator != nil {
		if err := s.coordinator.Close(); err != nil {
			s.logger.Error("关闭协调器失败", clog.Err(err))
		}
	}

	// 等待所有协程结束
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("服务器已优雅关闭")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// registerService 注册服务到服务发现
func (s *server) registerService() error {
	service := coord.ServiceInfo{
		ID:      s.wsManager.GetGatewayID(),
		Name:    "im-gateway",
		Address: s.config.Server.HTTPAddr,
		Port:    8080, // 从配置中解析端口
	}

	if err := s.coordinator.Registry().Register(context.Background(), service, 30*time.Second); err != nil {
		return err
	}

	s.logger.Info("服务注册成功", clog.String("service_id", service.ID))
	return nil
}

// deregisterService 注销服务
func (s *server) deregisterService() error {
	serviceID := s.wsManager.GetGatewayID()
	if err := s.coordinator.Registry().Unregister(context.Background(), serviceID); err != nil {
		return err
	}

	s.logger.Info("服务注销成功", clog.String("service_id", serviceID))
	return nil
}
