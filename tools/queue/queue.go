package queue

import "time"

type QueueMsg struct {
	Op           int               `json:"op"`             // 操作码
	ServerId     string            `json:"server_id"`      // 服务器ID
	Msg          []byte            `json:"msg"`            // 消息内容
	UserId       int               `json:"user_id"`        // 用户ID
	RoomId       int               `json:"room_id"`        // 房间ID
	Count        int               `json:"count"`          // 在线用户数
	RoomUserInfo map[string]string `json:"room_user_info"` // 房间用户信息
}

// MessageQueue 定义消息队列的抽象接口
// 任何实现此接口的结构体都可以作为消息队列使用
type MessageQueue interface {
	// Initialize 初始化队列连接
	Initialize() error
	// Close 关闭队列连接
	Close() error
	// PublishMessage 发布消息到队列
	PublishMessage(message *QueueMsg) error
	// ConsumeMessages 从队列中消费消息，并通过回调函数处理
	// timeout: 等待消息的超时时间
	// callback: 处理消息的回调函数，接收消息并返回处理结果
	ConsumeMessages(timeout time.Duration, callback func(*QueueMsg) error) error
}

// 全局共享的RedisQueue实例
var DefaultQueue MessageQueue
