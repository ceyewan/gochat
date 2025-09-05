package server

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-task/internal/config"
)

// Server 定义 im-task 服务器接口
type Server interface {
	// StartTaskProcessor 启动任务处理器
	StartTaskProcessor() error

	// Shutdown 优雅关闭服务器
	Shutdown(ctx context.Context) error
}

// server 服务器实现
type server struct {
	config *config.Config
	logger clog.Logger

	// TODO: 添加其他组件
	// repoClient   *grpc.ClientConn
	// mqConsumer   mq.Consumer
	// mqProducer   mq.Producer
	// dispatcher   *dispatcher.TaskDispatcher
	// processors   map[string]processor.TaskProcessor
}

// New 创建新的服务器实例
func New(cfg *config.Config) (Server, error) {
	logger := clog.Module("task-server")

	s := &server{
		config: cfg,
		logger: logger,
	}

	// TODO: 初始化组件
	// 1. 初始化 gRPC 客户端（im-repo）
	// 2. 初始化 Kafka 消费者和生产者
	// 3. 创建任务分发器
	// 4. 注册任务处理器
	// 5. 初始化外部服务客户端
	// 6. 服务注册

	logger.Info("im-task 服务器创建成功")
	return s, nil
}

// StartTaskProcessor 启动任务处理器
func (s *server) StartTaskProcessor() error {
	s.logger.Info("启动任务处理器...")

	// TODO: 启动任务处理逻辑
	// 1. 启动 Kafka 消费者
	// 2. 启动工作协程池
	// 3. 启动任务分发器
	// 4. 启动监控和指标收集

	s.logger.Info("任务处理器启动成功")
	return nil
}

// Shutdown 优雅关闭服务器
func (s *server) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭 im-task 服务器...")

	// TODO: 关闭各个组件
	// 1. 停止任务处理器
	// 2. 停止 Kafka 消费者
	// 3. 关闭 gRPC 客户端连接
	// 4. 关闭外部服务连接
	// 5. 服务注销

	s.logger.Info("im-task 服务器已关闭")
	return nil
}
