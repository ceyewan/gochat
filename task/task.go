package task

import (
	"context"
	"gochat/tools/queue"
	"runtime"
	"time"
)

type Task struct {
}

func New() *Task {
	return new(Task)
}

func (t *Task) Run() error {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 启动消息队列
	queue.InitDefaultQueue()
	// 消费消息
	queue.DefaultQueue.ConsumeMessages(5*time.Second, Push)
	return nil
}

func (t *Task) Shutdown(ctx context.Context) error {
	return nil
}
