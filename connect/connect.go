package connect

import (
	"fmt"
	"gochat/clog"
	"gochat/config"
	"gochat/tools"
	"runtime"
	"time"
)

type Connect struct {
	ServerID string
}

func New() *Connect {
	return &Connect{}
}

func (c *Connect) Run() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 生成唯一实例ID
	ip, err := tools.GetLocalIP()
	if err != nil {
		clog.Error("failed to get local IP: %v", err)
	}
	c.ServerID = fmt.Sprintf("connect-server-%d-%s", config.Conf.RPC.Port, ip)
	clog.Info("c.ServerID: %v", c.ServerID)

	// 初始化逻辑服务的 RPC 客户端
	InitLogicRPCClient()

	// 初始化 Etcd 客户端
	tools.InitEtcdClient(config.Conf.Etcd.Addrs, 5*time.Second)

	// 初始化RPC服务器并将服务注册到etcd
	InitRPCServer()

	// 初始化 WebSocket 服务器
	InitWebSocket()
}
