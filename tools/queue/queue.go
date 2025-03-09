package queue

import "time"

// QueueMsg 定义消息结构
type QueueMsg struct {
	Op           int               `json:"op"`             // 操作码
	InstanceId     string            `json:"server_id"`      // 服务器ID
	Msg          []byte            `json:"msg"`            // 消息内容
	UserId       int               `json:"user_id"`        // 用户ID
	RoomId       int               `json:"room_id"`        // 房间ID
	Count        int               `json:"count"`          // 在线用户数
	RoomUserInfo map[string]string `json:"room_user_info"` // 房间用户信息
}

// MessageQueue 定义消息队列的抽象接口
type MessageQueue interface {
	Initialize() error
	Close() error
	PublishMessage(message *QueueMsg) error
	ConsumeMessages(timeout time.Duration, callback func(*QueueMsg) error) error
}

// 全局共享的RedisQueue实例
var DefaultQueue MessageQueue
