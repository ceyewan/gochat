package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/config"
	"github.com/ceyewan/gochat/im-repo/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 初始化日志
	logger := clog.Module("im-repo-main")
	logger.Info("启动 im-repo 服务...")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("加载配置失败", clog.Err(err))
	}

	// 创建服务器实例
	srv, err := server.New(cfg)
	if err != nil {
		logger.Fatal("创建服务器失败", clog.Err(err))
	}

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(4*1024*1024), // 4MB
		grpc.MaxSendMsgSize(4*1024*1024), // 4MB
	)

	// 注册服务
	srv.RegisterServices(grpcServer)

	// 注册健康检查服务
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// 注册反射服务（用于调试）
	reflection.Register(grpcServer)

	// 设置所有服务为健康状态
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// 监听端口
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.Fatal("监听端口失败",
			clog.Int("port", cfg.Server.Port),
			clog.Err(err))
	}

	// 启动 gRPC 服务器
	go func() {
		logger.Info("gRPC 服务器启动",
			clog.String("address", listen.Addr().String()),
			clog.Int("port", cfg.Server.Port))

		if err := grpcServer.Serve(listen); err != nil {
			logger.Error("gRPC 服务器启动失败", clog.Err(err))
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号
	sig := <-sigChan
	logger.Info("收到关闭信号", clog.String("signal", sig.String()))

	// 优雅关闭
	logger.Info("开始优雅关闭服务...")

	// 设置健康检查为不可用
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// 创建关闭上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭 gRPC 服务器
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("gRPC 服务器已优雅关闭")
	case <-ctx.Done():
		logger.Warn("gRPC 服务器关闭超时，强制关闭")
		grpcServer.Stop()
	}

	// 关闭应用服务器
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("关闭应用服务器失败", clog.Err(err))
	}

	logger.Info("im-repo 服务已完全关闭")
}
