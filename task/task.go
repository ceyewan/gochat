package task

import (
	"gochat/clog"
	"gochat/tools/queue"
	"runtime"
	"time"
)

type Task struct {
}

func New() *Task {
	return new(Task)
}

func (t *Task) Run() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 启动消息队列
	queue.InitDefaultQueue()
	// 消费消息
	queue.DefaultQueue.ConsumeMessages(5*time.Second, Push)
}

func Push(msg *queue.QueueMsg) error {
	// 将消息使用 clog.Info 打印出来
	clog.Info("Push: %v", msg)
	return nil
}
