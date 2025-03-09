package logic

import (
	"context"
	"gochat/tools"
	"gochat/tools/queue"
	"runtime"

	"google.golang.org/grpc"
)

type Logic struct {
	rpcServer *grpc.Server
}

func New() *Logic {
	return new(Logic)
}

func (l *Logic) Run() error {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 初始化 Etcd 客户端
	tools.InitEtcdClient()
	// 启动消息队列
	queue.InitDefaultQueue()
	// 初始化Redis
	InitRedisClient()
	// 初始化RPC服务器
	var err error
	l.rpcServer, err = InitRPCServer(context.Background())
	if err != nil {
		return err
	}
	return nil
}

// Shutdown 实现优雅退出
func (l *Logic) Shutdown(ctx context.Context) error {
	if l.rpcServer != nil {
		l.rpcServer.GracefulStop()
	}
	tools.CloseAllDBConnections()
	tools.CloseEtcdClient()
	queue.DefaultQueue.Close()
	return nil
}
