package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
)

func main() {
	// 初始化日志
	logger, err := clog.New()
	if err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("加载配置失败", clog.Err(err))
	}

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer()

	// 创建 im-repo 服务器
	repoServer, err := server.New(cfg)
	if err != nil {
		logger.Fatal("创建 im-repo 服务器失败", clog.Err(err))
	}

	// 注册服务
	repoServer.RegisterServices(grpcServer)

	// 启动 gRPC 服务
	go func() {
		lis, err := net.Listen("tcp", cfg.Server.GRPCAddr)
		if err != nil {
			logger.Fatal("监听 gRPC 地址失败", clog.Err(err))
		}
		logger.Info("gRPC 服务启动", clog.String("addr", cfg.Server.GRPCAddr))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC 服务启动失败", clog.Err(err))
		}
	}()

	// 启动健康检查服务
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			err := repoServer.GetHealthChecker().CheckHealth(r.Context())
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprintf(w, "Health check failed: %v", err)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "OK")
		})
		logger.Info("健康检查服务启动", clog.String("port", cfg.Server.HealthPort))
		if err := http.ListenAndServe(cfg.Server.HealthPort, nil); err != nil {
			logger.Fatal("健康检查服务启动失败", clog.Err(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 优雅地关闭服务
	logger.Info("开始关闭服务...")
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := repoServer.Shutdown(ctx); err != nil {
		logger.Error("关闭 im-repo 服务器失败", clog.Err(err))
	}
	logger.Info("服务已关闭")
}
