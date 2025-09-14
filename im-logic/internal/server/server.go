package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/server/grpc"
	"github.com/ceyewan/gochat/im-logic/internal/server/kafka"
	"github.com/ceyewan/gochat/im-logic/internal/service"
)

// Server 服务器接口
type Server interface {
	// Start 启动服务器
	Start() error

	// Stop 停止服务器
	Stop() error

	// IsHealthy 检查服务器健康状态
	IsHealthy() bool
}

// server 服务器实现
type server struct {
	config *config.Config
	logger clog.Logger

	// 组件
	grpcClient    *grpc.Client
	grpcServer    *grpc.Server
	kafkaProducer *kafka.Producer
	kafkaConsumer *kafka.Consumer
	services      *service.Services
	handler       *service.MessageHandler

	// 健康检查
	healthCheck *HealthChecker
}

// New 创建新的服务器实例
func New(cfg *config.Config) (Server, error) {
	logger := clog.Namespace("logic-server")

	s := &server{
		config:      cfg,
		logger:      logger,
		healthCheck: NewHealthChecker(),
	}

	// 1. 创建 gRPC 客户端
	logger.Info("创建 gRPC 客户端...")
	grpcClient, err := grpc.NewClient(cfg)
	if err != nil {
		logger.Error("创建 gRPC 客户端失败", clog.Err(err))
		return nil, fmt.Errorf("创建 gRPC 客户端失败: %w", err)
	}
	s.grpcClient = grpcClient

	// 2. 创建业务服务
	logger.Info("创建业务服务...")
	s.services = service.NewServices(cfg, grpcClient)

	// 3. 创建 Kafka 生产者
	logger.Info("创建 Kafka 生产者...")
	kafkaProducer, err := kafka.NewProducer(cfg)
	if err != nil {
		logger.Error("创建 Kafka 生产者失败", clog.Err(err))
		return nil, fmt.Errorf("创建 Kafka 生产者失败: %w", err)
	}
	s.kafkaProducer = kafkaProducer

	// 4. 创建消息处理器
	logger.Info("创建消息处理器...")
	s.handler = service.NewMessageHandler(cfg, grpcClient, kafkaProducer, s.services)

	// 5. 创建 Kafka 消费者
	logger.Info("创建 Kafka 消费者...")
	kafkaConsumer, err := kafka.NewConsumer(cfg, s.handler)
	if err != nil {
		logger.Error("创建 Kafka 消费者失败", clog.Err(err))
		return nil, fmt.Errorf("创建 Kafka 消费者失败: %w", err)
	}
	s.kafkaConsumer = kafkaConsumer

	// 6. 创建 gRPC 服务器
	logger.Info("创建 gRPC 服务器...")
	grpcServer, err := grpc.New(cfg, s.services)
	if err != nil {
		logger.Error("创建 gRPC 服务器失败", clog.Err(err))
		return nil, fmt.Errorf("创建 gRPC 服务器失败: %w", err)
	}
	s.grpcServer = grpcServer

	// 7. 注册健康检查
	s.healthCheck.RegisterComponent("grpc_client", s.grpcClient)
	s.healthCheck.RegisterComponent("kafka_producer", s.kafkaProducer)
	s.healthCheck.RegisterComponent("kafka_consumer", s.kafkaConsumer)

	logger.Info("im-logic 服务器创建成功")
	return s, nil
}

// Start 启动服务器
func (s *server) Start() error {
	s.logger.Info("启动 im-logic 服务器...")

	// 1. 启动 Kafka 消费者
	s.logger.Info("启动 Kafka 消费者...")
	if err := s.kafkaConsumer.Start(); err != nil {
		s.logger.Error("启动 Kafka 消费者失败", clog.Err(err))
		return fmt.Errorf("启动 Kafka 消费者失败: %w", err)
	}

	// 2. 启动 gRPC 服务器
	s.logger.Info("启动 gRPC 服务器...")
	go func() {
		if err := s.grpcServer.Start(); err != nil {
			s.logger.Error("gRPC 服务器启动失败", clog.Err(err))
		}
	}()

	// 3. 启动 HTTP 健康检查服务
	s.logger.Info("启动 HTTP 健康检查服务...")
	go s.startHTTPServer()

	// 等待服务器启动完成
	time.Sleep(1 * time.Second)

	s.logger.Info("im-logic 服务器启动成功")
	return nil
}

// Stop 停止服务器
func (s *server) Stop() error {
	s.logger.Info("停止 im-logic 服务器...")

	// 1. 停止 gRPC 服务器
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}

	// 2. 停止 Kafka 消费者
	if s.kafkaConsumer != nil {
		s.kafkaConsumer.Close()
	}

	// 3. 停止 Kafka 生产者
	if s.kafkaProducer != nil {
		s.kafkaProducer.Close()
	}

	// 4. 停止 gRPC 客户端
	if s.grpcClient != nil {
		s.grpcClient.Close()
	}

	s.logger.Info("im-logic 服务器已停止")
	return nil
}

// IsHealthy 检查服务器健康状态
func (s *server) IsHealthy() bool {
	return s.healthCheck.IsHealthy()
}

// startHTTPServer 启动 HTTP 健康检查服务
func (s *server) startHTTPServer() {
	mux := http.NewServeMux()

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if s.IsHealthy() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "OK")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, "Service Unhealthy")
		}
	})

	// 就绪检查端点
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if s.IsHealthy() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Ready")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, "Not Ready")
		}
	})

	// 存活检查端点
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Alive")
	})

	// 指标端点（如果启用）
	if s.config.Monitoring.MetricsEnabled {
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			// TODO: 实现 Prometheus 指标收集
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "# TODO: Implement Prometheus metrics")
		})
	}

	addr := s.config.GetHTTPAddr()
	s.logger.Info("HTTP 健康检查服务启动", clog.String("addr", addr))

	if err := http.ListenAndServe(addr, mux); err != nil {
		s.logger.Error("HTTP 健康检查服务启动失败", clog.Err(err))
	}
}
