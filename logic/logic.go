package logic

import (
	"gochat/config"
	"gochat/tools"
	"gochat/tools/queue"
	"runtime"
	"time"
)

type Logic struct {
}

func New() *Logic {
	return new(Logic)
}

func (l *Logic) Run() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 初始化 Etcd 客户端
	tools.InitEtcdClient(config.Conf.Etcd.Addrs, 5*time.Second)
	// 启动消息队列
	queue.InitDefaultQueue()
	// 初始化Redis
	InitRedisClient()
	// 初始化RPC服务器
	InitRPCServer()
}
