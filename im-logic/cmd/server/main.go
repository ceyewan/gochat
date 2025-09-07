package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/server"
)

func main() {
	// 初始化日志
	logger, err := clog.New(clog.DefaultConfig())
	if err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("加载配置失败", clog.Err(err))
	}

	// 创建服务器
	srv, err := server.New(cfg)
	if err != nil {
		logger.Fatal("创建服务器失败", clog.Err(err))
	}

	// 启动服务器
	if err := srv.Start(); err != nil {
		logger.Fatal("启动服务器失败", clog.Err(err))
	}

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 优雅地关闭服务
	logger.Info("开始关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 停止服务器
	if err := srv.Stop(); err != nil {
		logger.Error("关闭服务器失败", clog.Err(err))
	}

	logger.Info("服务已关闭")
}
