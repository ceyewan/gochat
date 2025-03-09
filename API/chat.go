package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"gochat/api/router"
	"gochat/api/rpc"
	"gochat/clog"
	"gochat/config"
	"gochat/tools"

	"github.com/gin-gonic/gin"
)

// Chat API服务实例
type Chat struct {
	server *http.Server
	quit   chan os.Signal
}

// New 创建API服务实例
func New() *Chat {
	return &Chat{
		quit: make(chan os.Signal, 1),
	}
}

// Run 启动API服务
func (c *Chat) Run() error {
	// 初始化依赖服务
	tools.InitEtcdClient()
	rpc.InitLogicRPCClient()

	// 配置Gin
	runMode := config.GetGinRunMode()
	gin.SetMode(runMode)
	clog.Info("[API] Server running in %s mode", runMode)

	// 初始化路由
	r := router.Register()

	// 配置HTTP服务
	c.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Conf.APIConfig.Port),
		Handler: r,
	}

	// 启动服务器
	go func() {
		clog.Info("[API] Server starting on port %d", config.Conf.APIConfig.Port)
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			clog.Error("[API] Server failed to start: %v", err)
			os.Exit(1)
		}
	}()
	return nil
}

// Shutdown 优雅关闭API服务
func (c *Chat) Shutdown(ctx context.Context) error {
	if c.server != nil {
		// 关闭HTTP服务
		if err := c.server.Shutdown(ctx); err != nil {
			clog.Error("[API] Server shutdown error: %v", err)
			return err
		}
	}

	clog.Info("[API] Server shutdown complete")

	// 关闭依赖服务
	tools.CloseEtcdClient()
	clog.Info("[API] Etcd client closed")
	return nil
}
