package logic

import (
	"gochat/tools/queue"
	"runtime"
)

type Logic struct {
}

func New() *Logic {
	return new(Logic)
}

func (l *Logic) Run() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 启动消息队列
	queue.InitDefaultQueue()
	// 初始化Redis
	InitRedisClient()
	// 初始化RPC服务器
	InitRPCServer()
}
