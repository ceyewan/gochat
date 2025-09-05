package server

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"google.golang.org/grpc"
)

// Server 定义 im-logic 服务器接口
type Server interface {
	// RegisterServices 注册 gRPC 服务
	RegisterServices(grpcServer *grpc.Server)

	// StartMessageConsumer 启动消息消费者
	StartMessageConsumer() error

	// Shutdown 优雅关闭服务器
	Shutdown(ctx context.Context) error
}

// server 服务器实现
type server struct {
	config *config.Config
	logger clog.Logger

	// TODO: 添加其他组件
	// repoClient *grpc.ClientConn
	// mqConsumer mq.Consumer
	// mqProducer mq.Producer
	// authService *service.AuthService
	// conversationService *service.ConversationService
	// groupService *service.GroupService
}

// New 创建新的服务器实例
func New(cfg *config.Config) (Server, error) {
	logger := clog.Module("logic-server")

	s := &server{
		config: cfg,
		logger: logger,
	}

	// TODO: 初始化组件
	// - gRPC 客户端（im-repo）
	// - Kafka 生产者和消费者
	// - 业务服务实例
	// - 服务注册

	logger.Info("im-logic 服务器创建成功")
	return s, nil
}

// RegisterServices 注册 gRPC 服务
func (s *server) RegisterServices(grpcServer *grpc.Server) {
	s.logger.Info("注册 gRPC 服务...")

	// TODO: 注册具体的 gRPC 服务
	// logicv1.RegisterAuthServiceServer(grpcServer, s.authService)
	// logicv1.RegisterConversationServiceServer(grpcServer, s.conversationService)
	// logicv1.RegisterGroupServiceServer(grpcServer, s.groupService)

	s.logger.Info("gRPC 服务注册完成")
}

// StartMessageConsumer 启动消息消费者
func (s *server) StartMessageConsumer() error {
	s.logger.Info("启动消息消费者...")

	// TODO: 启动 Kafka 消费者
	// 1. 创建消费者实例
	// 2. 订阅上行消息 Topic
	// 3. 启动消费循环

	s.logger.Info("消息消费者启动成功")
	return nil
}

// Shutdown 优雅关闭服务器
func (s *server) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭 im-logic 服务器...")

	// TODO: 关闭各个组件
	// 1. 停止消息消费者
	// 2. 关闭 gRPC 客户端连接
	// 3. 关闭 Kafka 连接
	// 4. 服务注销

	s.logger.Info("im-logic 服务器已关闭")
	return nil
}
