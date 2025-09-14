package grpc

import (
	"fmt"
	"net"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Server gRPC 服务器
type Server struct {
	config      *config.Config
	logger      clog.Logger
	grpcServer  *grpc.Server
	listener    net.Listener
	healthCheck *health.Server

	// 业务服务
	authService         *service.AuthService
	conversationService *service.ConversationService
	groupService        *service.GroupService
}

// New 创建 gRPC 服务器
func New(cfg *config.Config, services *service.Services) (*Server, error) {
	logger := clog.Namespace("grpc-server")

	// 创建 TCP 监听器
	listener, err := net.Listen("tcp", cfg.GetGRPCAddr())
	if err != nil {
		logger.Error("创建 TCP 监听器失败", clog.Err(err))
		return nil, fmt.Errorf("创建 TCP 监听器失败: %w", err)
	}

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(uint32(cfg.Server.GRPC.MaxConn)),
		grpc.MaxRecvMsgSize(cfg.Server.GRPC.MaxMsgSize*1024*1024),
		grpc.MaxSendMsgSize(cfg.Server.GRPC.MaxMsgSize*1024*1024),
		grpc.ConnectionTimeout(time.Duration(cfg.Server.GRPC.Timeout)*time.Second),
	)

	// 创建健康检查服务
	healthCheck := health.NewServer()

	server := &Server{
		config:              cfg,
		logger:              logger,
		grpcServer:          grpcServer,
		listener:            listener,
		healthCheck:         healthCheck,
		authService:         services.Auth,
		conversationService: services.Conversation,
		groupService:        services.Group,
	}

	// 注册 gRPC 服务
	server.registerServices()

	logger.Info("gRPC 服务器创建成功", clog.String("addr", cfg.GetGRPCAddr()))
	return server, nil
}

// registerServices 注册 gRPC 服务
func (s *Server) registerServices() {
	s.logger.Info("注册 gRPC 服务...")

	// 注册业务服务
	logicpb.RegisterAuthServiceServer(s.grpcServer, s.authService)
	logicpb.RegisterConversationServiceServer(s.grpcServer, s.conversationService)
	logicpb.RegisterGroupServiceServer(s.grpcServer, s.groupService)

	// 注册健康检查服务
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthCheck)

	s.logger.Info("gRPC 服务注册完成")
}

// Start 启动 gRPC 服务器
func (s *Server) Start() error {
	s.logger.Info("启动 gRPC 服务器...", clog.String("addr", s.config.GetGRPCAddr()))

	// 设置健康检查状态
	s.healthCheck.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// 启动服务器
	if err := s.grpcServer.Serve(s.listener); err != nil {
		s.logger.Error("gRPC 服务器启动失败", clog.Err(err))
		return fmt.Errorf("gRPC 服务器启动失败: %w", err)
	}

	return nil
}

// Stop 停止 gRPC 服务器
func (s *Server) Stop() {
	s.logger.Info("停止 gRPC 服务器...")

	// 设置健康检查状态
	s.healthCheck.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// 优雅关闭
	s.grpcServer.GracefulStop()

	s.logger.Info("gRPC 服务器已停止")
}

// GetServer 获取 gRPC 服务器实例
func (s *Server) GetServer() *grpc.Server {
	return s.grpcServer
}

// UpdateHealthStatus 更新健康检查状态
func (s *Server) UpdateHealthStatus(status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.healthCheck.SetServingStatus("", status)
}
