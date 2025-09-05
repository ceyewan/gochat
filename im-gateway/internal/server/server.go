package server

import (
	"context"
	"net/http"

	"github.com/ceyewan/gochat/im-gateway/internal/config"
	"github.com/ceyewan/gochat/im-infra/clog"
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
	httpServer *http.Server
	logger     clog.Logger

	// TODO: 添加其他组件
	// wsManager  *websocket.Manager
	// grpcClient *grpc.Client
	// mqProducer mq.Producer
	// mqConsumer mq.Consumer
}

// New 创建新的服务器实例
func New(cfg *config.Config) (Server, error) {
	logger := clog.Module("server")

	s := &server{
		config: cfg,
		logger: logger,
	}

	// 初始化 HTTP 服务器
	if err := s.initHTTPServer(); err != nil {
		return nil, err
	}

	// TODO: 初始化其他组件
	// - WebSocket 管理器
	// - gRPC 客户端
	// - Kafka 生产者和消费者
	// - 服务注册

	return s, nil
}

// Start 启动服务器
func (s *server) Start() error {
	s.logger.Info("启动 HTTP 服务器", clog.String("addr", s.config.Server.HTTPAddr))
	return s.httpServer.ListenAndServe()
}

// Shutdown 优雅关闭服务器
func (s *server) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭服务器...")

	// 关闭 HTTP 服务器
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("关闭 HTTP 服务器失败", clog.Err(err))
		return err
	}

	// TODO: 关闭其他组件
	// - WebSocket 连接
	// - gRPC 连接
	// - Kafka 连接
	// - 服务注销

	return nil
}

// initHTTPServer 初始化 HTTP 服务器
func (s *server) initHTTPServer() error {
	// TODO: 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthCheck)

	s.httpServer = &http.Server{
		Addr:         s.config.Server.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	return nil
}

// healthCheck 健康检查端点
func (s *server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
