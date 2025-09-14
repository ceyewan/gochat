package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	// 模块化日志器
	mainLogger = clog.Namespace("basic.main")
	grpcLogger = clog.Namespace("basic.grpc")
)

func main() {
	mainLogger.Info("启动基础 metrics 示例应用")

	// 1. 配置并初始化 metrics provider
	cfg := metrics.DefaultConfig()
	cfg.ServiceName = "my-awesome-service"
	cfg.PrometheusListenAddr = ":9091" // 在此端口暴露指标
	cfg.ExporterType = "stdout"        // 演示用，将 traces 打印到控制台

	mainLogger.Info("创建 metrics provider",
		clog.String("service_name", cfg.ServiceName),
		clog.String("prometheus_addr", cfg.PrometheusListenAddr),
		clog.String("exporter_type", cfg.ExporterType))

	provider, err := metrics.New(cfg)
	if err != nil {
		mainLogger.Error("failed to create metrics provider", clog.Err(err))
		panic(err)
	}

	// 2. 延迟关闭 provider
	defer func() {
		mainLogger.Info("关闭 metrics provider")
		// 给指标导出一点时间
		time.Sleep(5 * time.Second)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := provider.Shutdown(shutdownCtx); err != nil {
			mainLogger.Error("failed to shutdown metrics provider", clog.Err(err))
		} else {
			mainLogger.Info("metrics provider 关闭完成")
		}
	}()

	// 3. 创建带有拦截器的 gRPC 服务器
	grpcLogger.Info("创建 gRPC 服务器，集成 metrics 拦截器")
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			provider.GRPCServerInterceptor(),
		),
	)

	// 在真实应用中，你会在这里注册你的 gRPC 服务。
	// 对于这个示例，我们只启用反射，这样工具就能看到服务器。
	reflection.Register(server)

	// 4. 启动 gRPC 服务器
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		mainLogger.Error("failed to listen on :8081", clog.Err(err))
		panic(err)
	}

	go func() {
		grpcLogger.Info("gRPC 服务器开始监听", clog.String("address", ":8081"))
		if err := server.Serve(lis); err != nil {
			grpcLogger.Error("failed to serve gRPC", clog.Err(err))
		}
	}()

	mainLogger.Info("服务正在运行，按 Ctrl+C 退出")
	mainLogger.Info("指标可在以下地址访问: http://localhost:9091/metrics")
	mainLogger.Info("可以使用 grpcurl 向 localhost:8081 发送任何 gRPC 调用进行追踪和测量")

	// 等待终止信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	mainLogger.Info("收到退出信号，正在关闭服务器...")
	server.GracefulStop()
	mainLogger.Info("服务器已停止")
}
