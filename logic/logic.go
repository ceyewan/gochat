package logic

import "runtime"

type Logic struct {
}

func New() *Logic {
	return new(Logic)
}

func (l *Logic) Run() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 初始化Redis
	InitRedisClient()
	// 初始化RPC服务器
	InitRPCServer()
}
