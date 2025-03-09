package api

import (
	"context"
	"fmt"
	"gochat/api/router"
	"gochat/api/rpc"
	"gochat/clog"
	"gochat/config"
	"gochat/tools"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
)

// Chat 结构体作为整个API服务的实例
type Chat struct{}

// NewChat 创建并返回一个新的 Chat 实例
//
// 返回:
//   - *Chat: 初始化好的 Chat 对象实例
func New() *Chat {
	return &Chat{}
}

// Run 启动 Chat 服务
func (c *Chat) Run() {
	// 初始化 etcd 服务
	tools.InitEtcdClient(config.Conf.Etcd.Addrs, 5*time.Second)
	// 初始化 LogicRPC 客户端
	rpc.InitLogicRPCClient()

	// 初始化并注册路由
	r := router.Register()
	runMode := config.GetGinRunMode()
	clog.Info("API Server Running in %s mode", runMode)
	gin.SetMode(runMode)

	// 配置服务端口
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Conf.APIConfig.Port),
		Handler: r,
	}

	// 启动服务
	go func() {
		if err := server.ListenAndServe(); err != nil {
			clog.Error("API Server start failed: %v", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	clog.Info("API Server Shutting down...")
	// 设置关闭超时时间，确保所有连接都能正常关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		clog.Error("API Server Shutdown failed: %v", err)
	}
	clog.Info("API Server exiting")
	os.Exit(0)
}
