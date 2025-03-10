package connect

import (
	"context"
	"fmt"
	"gochat/clog"
	"gochat/config"
	"gochat/tools"
	"runtime"

	"google.golang.org/grpc"
)

// Connect 连接服务
// 负责初始化和管理连接层
type Connect struct {
	InstanceID string       // 服务器唯一标识
	rpcServer  *grpc.Server // RPC 服务
}

// New 创建 Connect 实例
func New() *Connect {
	return &Connect{}
}

// Run 启动服务
func (c *Connect) Run() error {
	// 优化 CPU 使用
	runtime.GOMAXPROCS(runtime.NumCPU())
	clog.Info("GOMAXPROCS set to: %d", runtime.NumCPU())

	// 初始化服务 ID
	if err := c.initInstanceID(); err != nil {
		return err
	}

	// 初始化 Etcd 客户端
	if err := tools.InitEtcdClient(); err != nil {
		return fmt.Errorf("failed to initialize etcd client: %w", err)
	}

	// 初始化连接管理器
	connectionManager = NewConnectionManager()

	// 初始化 Logic 服务的 RPC 客户端
	InitLogicRPCClient()

	// 初始化 RPC 服务器
	rpcServer, err := InitRPCServer(context.Background())
	if err != nil {
		return fmt.Errorf("failed to initialize RPC server: %w", err)
	}
	c.rpcServer = rpcServer

	// 初始化并启动 WebSocket 服务器
	if err := InitWebSocket(); err != nil {
		return fmt.Errorf("failed to initialize WebSocket server: %w", err)
	}
	clog.Info("WebSocket server started successfully.")

	return nil
}

// initInstanceID 初始化服务ID
func (c *Connect) initInstanceID() error {
	ip, err := tools.GetLocalIP()
	if err != nil {
		clog.Error("Failed to get local IP: %v", err)
		return err
	}

	c.InstanceID = fmt.Sprintf("connect-server-%d-%s", config.Conf.RPC.Port, ip)
	DefaultWSServer.InstanceID = c.InstanceID // 设置 WebSocket 服务器的 ID

	clog.Info("Service ID initialized: %s", c.InstanceID)
	return nil
}

// Shutdown 优雅退出
func (c *Connect) Shutdown(ctx context.Context) error {
	clog.Info("Shutting down Connect service...")

	// 关闭 RPC 服务器
	if c.rpcServer != nil {
		clog.Info("Stopping RPC server...")
		c.rpcServer.GracefulStop()
		clog.Info("RPC server stopped.")
	}

	// 关闭 Etcd 客户端
	clog.Info("Closing Etcd client...")
	tools.CloseEtcdClient()
	clog.Info("Etcd client closed.")

	clog.Info("Connect service shutdown complete.")
	return nil
}
