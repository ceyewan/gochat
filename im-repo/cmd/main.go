package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/config"
	"github.com/ceyewan/gochat/im-repo/internal/server"
	"google.golang.org/grpc"
)

// main 是 im-repo 服务的入口函数
// 负责初始化配置、启动 gRPC 服务器并处理优雅关闭
func main() {
	// 初始化日志
	logger := clog.Module("im-repo")
	logger.Info("正在启动 im-repo 服务...")

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
	grpcServer := grpc.NewServer()

	// 注册服务
	srv.RegisterServices(grpcServer)

	// 启动 gRPC 服务器
	lis, err := net.Listen("tcp", cfg.Server.GRPCAddr)
	if err != nil {
		logger.Fatal("监听端口失败", clog.Err(err))
	}

	go func() {
		logger.Info("gRPC 服务器启动中", clog.String("addr", cfg.Server.GRPCAddr))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC 服务器启动失败", clog.Err(err))
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭服务器
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		srv.Shutdown(ctx)
		close(done)
	}()

	select {
	case <-done:
		logger.Info("服务器已优雅关闭")
	case <-ctx.Done():
		logger.Warn("关闭超时，强制关闭")
		grpcServer.Stop()
	}
}
