package task

import (
	"errors"
	"gochat/clog"
	"gochat/config"
	"gochat/tools/queue"
	"math/rand"
	"sync"
)

type PushParams struct {
	ServerId string // 目标服务器ID
	UserId   int    // 目标用户ID
	RoomId   int    // 房间ID(用于群发消息)
	Msg      []byte // 消息内容
	Count    int    // 在线用户数量

}

var (
	pushChannel []chan *queue.QueueMsg
	stopChan    chan struct{}
	wg          sync.WaitGroup
)

func init() {
	pushChannel = make([]chan *queue.QueueMsg, config.Conf.TaskConfig.ChannelSize)
	stopChan = make(chan struct{})
}

func (t *Task) GoPush() {
	for i := 0; i < config.Conf.TaskConfig.ChannelSize; i++ {
		pushChannel[i] = make(chan *queue.QueueMsg, 1000)
		wg.Add(1)
		go t.processPush(pushChannel[i])
	}
}

func (t *Task) processPush(c chan *queue.QueueMsg) {
	defer wg.Done()
	for {
		select {
		case msg := <-c:
			clog.Info("Push msg into: %d, op is: %d", msg.RoomId, msg.Op)
			// 通过RPC调用推送消息
			switch msg.Op {
			case config.OpSingleSend:
				// 向指定用户发送消息
				t.pushSingleToConnect(msg.ServerId, msg.UserId, msg.Msg)
			case config.OpRoomSend:
				// 向指定房间发送消息
				t.broadcastRoomToConnect(msg.RoomId, msg.Msg)
			case config.OpRoomCountSend:
				// 广播在线用户数量
				t.broadcastRoomCountToConnect(msg.RoomId, msg.Count)
			case config.OpRoomInfoSend:
				// 广播房间信息
				t.broadcastRoomInfoToConnect(msg.RoomId, msg.RoomUserInfo)
			}
		case <-stopChan:
			clog.Info("Stopping message processor")
			return
		}
	}
}

func Push(msg *queue.QueueMsg) error {
	select {
	case pushChannel[rand.Intn(config.Conf.TaskConfig.ChannelSize)] <- msg:
		clog.Info("Push msg into channel, op is: %d", msg.Op)
		return nil
	default:
		clog.Warning("All push channels are full, message dropped")
		// 可以选择重试、写入备用存储等策略
		return errors.New("push channels are full")
	}
}

// 添加优雅关闭方法
// func (t *Task) Shutdown(ctx context.Context) error {
// 	// 通知所有处理协程停止
// 	close(stopChan)

// 	// 等待所有协程退出，带超时控制
// 	doneChan := make(chan struct{})
// 	go func() {
// 		wg.Wait()
// 		close(doneChan)
// 	}()

// 	select {
// 	case <-doneChan:
// 		clog.Info("All push processors stopped gracefully")
// 		return nil
// 	case <-ctx.Done():
// 		clog.Warning("Timeout waiting for push processors to stop")
// 		return ctx.Err()
// 	}
// }
