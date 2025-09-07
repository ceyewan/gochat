package kafka

import (
	"encoding/json"
)

// Kafka Topic 定义
const (
	// TopicMessagesUpstream 上行消息 (gateway -> logic)
	TopicMessagesUpstream = "gochat.messages.upstream"

	// TopicMessagesPersist 持久化消息 (logic -> task)
	TopicMessagesPersist = "gochat.messages.persist"

	// TopicMessagesDownstreamPrefix 下行推送消息前缀 (logic/task -> gateway)
	// 实际 Topic 格式为: gochat.messages.downstream.{gateway_instance_id}
	TopicMessagesDownstreamPrefix = "gochat.messages.downstream."

	// TopicTasksFanout 大群扇出任务 (logic -> task)
	TopicTasksFanout = "gochat.tasks.fanout"
)

// UpstreamMessage 上行消息结构
// 从客户端通过 im-gateway 发送到 im-logic 的消息
type UpstreamMessage struct {
	TraceID        string `json:"trace_id"`
	UserID         string `json:"user_id"`
	GatewayID      string `json:"gateway_id"`
	ConversationID string `json:"conversation_id"`
	MessageType    int    `json:"message_type"`
	Content        string `json:"content"`
	ClientMsgID    string `json:"client_msg_id"`
	Timestamp      int64  `json:"timestamp"`
	Extra          string `json:"extra,omitempty"`
}

// PersistenceMessage 持久化消息结构
// 由 im-logic 发送到 im-task，用于唯一持久化
type PersistenceMessage struct {
	TraceID        string    `json:"trace_id"`
	MessageID      int64     `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	SenderInfo     *UserInfo `json:"sender_info,omitempty"`
	MessageType    int       `json:"message_type"`
	Content        string    `json:"content"`
	SeqID          int64     `json:"seq_id"`
	Timestamp      int64     `json:"timestamp"`
	Extra          string    `json:"extra,omitempty"`
}

// DownstreamMessage 下行消息结构
// 从 im-logic 或 im-task 发送到 im-gateway 再推送给客户端的消息
type DownstreamMessage struct {
	TraceID        string    `json:"trace_id"`
	TargetUserID   string    `json:"target_user_id"`
	MessageID      int64     `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	SenderInfo     *UserInfo `json:"sender_info,omitempty"`
	MessageType    int       `json:"message_type"`
	Content        string    `json:"content"`
	SeqID          int64     `json:"seq_id"`
	Timestamp      int64     `json:"timestamp"`
	Extra          string    `json:"extra,omitempty"`
}

// FanoutTask 大群消息扩散任务
type FanoutTask struct {
	TraceID        string   `json:"trace_id"`
	GroupID        string   `json:"group_id"`
	MessageID      int64    `json:"message_id"` // 使用已持久化消息的ID
	ExcludeUserIDs []string `json:"exclude_user_ids,omitempty"`
}

// UserInfo 用户信息（轻量级）
type UserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// TaskMessage 异步任务消息结构 (保留用于未来可能的其他任务)
type TaskMessage struct {
	TraceID   string          `json:"trace_id"`
	TaskType  string          `json:"task_type"`
	TaskID    string          `json:"task_id"`
	Data      json.RawMessage `json:"data"`
	CreatedAt int64           `json:"created_at"`
}
