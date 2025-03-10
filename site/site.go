package site

import (
	"context"
	"gochat/clog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Site 结构体实现 ModuleRunner 接口
type Site struct {
	server *http.Server
}

// New 创建一个新的 Site 实例
func New() *Site {
	return &Site{}
}

// Run 启动静态文件服务
func (s *Site) Run() error {
	// 创建文件服务器
	fs := http.FileServer(http.Dir("./site"))

	// 创建 HTTP 服务器
	s.server = &http.Server{
		Addr:    ":8082",
		Handler: fs,
	}

	// 启动服务器
	go func() {
		clog.Info("Starting static file server on port 8082")
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			clog.Error("Static file server failed: %v", err)
		}
	}()

	// 处理关闭信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	return nil
}

// Shutdown 优雅关闭服务器
func (s *Site) Shutdown(ctx context.Context) error {
	clog.Info("Shutting down static file server...")

	// 创建关闭上下文
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 关闭服务器
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		clog.Error("Static file server shutdown error: %v", err)
		return err
	}

	clog.Info("Static file server shutdown completed")
	return nil
}