package task

import (
	"errors"
	"gochat/clog"
	"gochat/config"
	"gochat/tools/queue"
	"math/rand"
	"sync"
)

var (
	pushChannels []chan *queue.QueueMsg
	stopChan     chan struct{}
	wg           sync.WaitGroup
)

func init() {
	pushChannels = make([]chan *queue.QueueMsg, config.Conf.TaskConfig.ChannelSize)
	stopChan = make(chan struct{})
}

// GoPush 启动消息推送处理
func (t *Task) GoPush() {
	for i := 0; i < config.Conf.TaskConfig.ChannelSize; i++ {
		pushChannels[i] = make(chan *queue.QueueMsg, config.Conf.TaskConfig.ChannelBufferSize)
		wg.Add(1)
		go t.processPush(pushChannels[i])
	}
}

// processPush 处理推送消息
func (t *Task) processPush(c chan *queue.QueueMsg) {
	defer wg.Done()
	for {
		select {
		case msg := <-c:
			clog.Module("task").Infof("Processing message, RoomID: %d, Op: %d", msg.RoomId, msg.Op)
			switch msg.Op {
			case config.OpSingleSend:
				t.pushSingleToConnect(msg.InstanceId, msg.UserId, msg.Msg)
			case config.OpRoomSend:
				t.broadcastRoomToConnect(msg.RoomId, msg.Msg)
			case config.OpRoomInfoSend:
				t.broadcastRoomInfoToConnect(msg.RoomId, msg.Msg)
			}
		case <-stopChan:
			clog.Module("task").Infof("Stopping message processor")
			return
		}
	}
}

// Push 将消息推送到随机的通道
func Push(msg *queue.QueueMsg) error {
	select {
	case pushChannels[rand.Intn(config.Conf.TaskConfig.ChannelSize)] <- msg:
		clog.Module("task").Infof("Message pushed to channel, Op: %d", msg.Op)
		return nil
	default:
		clog.Module("task").Warnf("All push channels are full, message dropped")
		return errors.New("push channels are full")
	}
}

// StopPush 停止消息推送处理
func StopPush() {
	close(stopChan)
	wg.Wait()
	clog.Module("task").Infof("All message processors stopped")
}
