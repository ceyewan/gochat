package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/im-task/internal/server"
)

// main 是 im-task 服务的入口函数
// 负责初始化配置、启动任务处理器并处理优雅关闭
func main() {
	// 初始化日志
	logger := clog.Module("im-task")
	logger.Info("正在启动 im-task 服务...")

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

	// 启动任务处理器
	go func() {
		logger.Info("启动任务处理器...")
		if err := srv.StartTaskProcessor(); err != nil {
			logger.Fatal("启动任务处理器失败", clog.Err(err))
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
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭失败", clog.Err(err))
	} else {
		logger.Info("服务器已优雅关闭")
	}
}
