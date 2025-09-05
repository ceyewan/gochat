package server

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/config"
	"google.golang.org/grpc"
)

// Server 定义 im-repo 服务器接口
type Server interface {
	// RegisterServices 注册 gRPC 服务
	RegisterServices(grpcServer *grpc.Server)

	// Shutdown 优雅关闭服务器
	Shutdown(ctx context.Context) error
}

// server 服务器实现
type server struct {
	config *config.Config
	logger clog.Logger

	// TODO: 添加其他组件
	// db         *gorm.DB
	// cache      cache.Cache
	// userService    *service.UserService
	// messageService *service.MessageService
	// groupService   *service.GroupService
	// conversationService *service.ConversationService
	// onlineStatusService *service.OnlineStatusService
}

// New 创建新的服务器实例
func New(cfg *config.Config) (Server, error) {
	logger := clog.Module("repo-server")

	s := &server{
		config: cfg,
		logger: logger,
	}

	// TODO: 初始化组件
	// 1. 初始化数据库连接
	// 2. 初始化缓存连接
	// 3. 创建业务服务实例
	// 4. 数据库迁移
	// 5. 服务注册

	logger.Info("im-repo 服务器创建成功")
	return s, nil
}

// RegisterServices 注册 gRPC 服务
func (s *server) RegisterServices(grpcServer *grpc.Server) {
	s.logger.Info("注册 gRPC 服务...")

	// TODO: 注册具体的 gRPC 服务
	// repov1.RegisterUserServiceServer(grpcServer, s.userService)
	// repov1.RegisterMessageServiceServer(grpcServer, s.messageService)
	// repov1.RegisterGroupServiceServer(grpcServer, s.groupService)
	// repov1.RegisterConversationServiceServer(grpcServer, s.conversationService)
	// repov1.RegisterOnlineStatusServiceServer(grpcServer, s.onlineStatusService)

	s.logger.Info("gRPC 服务注册完成")
}

// Shutdown 优雅关闭服务器
func (s *server) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭 im-repo 服务器...")

	// TODO: 关闭各个组件
	// 1. 关闭数据库连接
	// 2. 关闭缓存连接
	// 3. 服务注销

	s.logger.Info("im-repo 服务器已关闭")
	return nil
}
