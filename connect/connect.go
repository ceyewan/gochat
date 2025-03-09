package connect

import (
	"context"
	"fmt"
	"gochat/clog"
	"gochat/config"
	"gochat/tools"
	"runtime"
)

// Connect 是连接服务的主要入口结构
// 负责初始化和管理整个连接层
type Connect struct {
	ServerID string // 服务器唯一标识
}

// New 创建一个新的Connect实例
func New() *Connect {
	return &Connect{}
}

// Run 启动Connect服务
// 初始化所有必要的组件并开始服务
func (c *Connect) Run() error {
	// 优化CPU使用
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 生成唯一实例ID
	ip, err := tools.GetLocalIP()
	if err != nil {
		clog.Error("failed to get local IP: %v", err)
	}
	c.ServerID = fmt.Sprintf("connect-server-%d-%s", config.Conf.RPC.Port, ip)
	clog.Info("服务ID初始化: %v", c.ServerID)

	// 设置服务器ID给WebSocket服务器
	DefaultWSServer.ServerID = c.ServerID

	// 初始化连接管理器(在RPC服务器初始化之前)
	connectionManager = NewConnectionManager()

	// 初始化逻辑服务的 RPC 客户端
	InitLogicRPCClient()

	// 初始化 Etcd 客户端
	tools.InitEtcdClient()

	// 初始化RPC服务器并将服务注册到etcd
	go InitRPCServer()

	// 初始化并启动WebSocket服务器
	// 这是阻塞调用
	clog.Info("启动WebSocket服务...")
	if err := InitWebSocket(); err != nil {
		clog.Error("WebSocket服务启动失败: %v", err)
	}
	return nil
}

// Shutdown 实现优雅退出
func (c *Connect) Shutdown(ctx context.Context) error {
	return nil
}
