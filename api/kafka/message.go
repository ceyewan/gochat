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

	// --- 领域事件 Topics ---

	// TopicUserEvents 用户事件 (例如: 上线、下线、信息变更)
	TopicUserEvents = "gochat.user-events"
	// TopicMessageEvents 消息事件 (例如: 已读、撤回)
	TopicMessageEvents = "gochat.message-events"
	// TopicNotifications 系统通知 (例如: 被拉入群、好友申请)
	TopicNotifications = "gochat.notifications"
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

// SystemSignalMessage 系统信令消息
// 用于后端服务之间（如 im-task -> im-gateway）传递需要实时推送给客户端的非聊天类信息。
type SystemSignalMessage struct {
	TraceID       string          `json:"trace_id"`
	TargetUserIDs []string        `json:"target_user_ids"` // 目标用户列表
	SignalType    string          `json:"signal_type"`     // 信令类型, e.g., "friend.status.changed", "message.receipt.updated"
	Payload       json.RawMessage `json:"payload"`         // 具体信令的载荷
	Timestamp     int64           `json:"timestamp"`
}

// --- 领域事件消息结构 ---

// UserEvent 用户事件
type UserEvent struct {
	EventID   string          `json:"event_id"`   // 事件唯一标识
	EventType string          `json:"event_type"` // 事件类型：user.online, user.offline, user.profile.updated
	Timestamp int64           `json:"timestamp"`  // 事件时间戳
	UserID    string          `json:"user_id"`    // 用户ID
	Payload   json.RawMessage `json:"payload"`    // 事件载荷
}

// UserOnlinePayload 用户上线事件载荷
type UserOnlinePayload struct {
	GatewayID string `json:"gateway_id"` // 所在网关ID
	IPAddress string `json:"ip_address"` // 客户端IP
	UserAgent string `json:"user_agent"` // 客户端信息
}

// UserOfflinePayload 用户下线事件载荷
type UserOfflinePayload struct {
	GatewayID string `json:"gateway_id"` // 原所在网关ID
	LastSeen  int64  `json:"last_seen"`  // 最后在线时间
	Reason    string `json:"reason"`     // 下线原因：disconnect, timeout, logout
}

// UserProfileUpdatedPayload 用户资料更新事件载荷
type UserProfileUpdatedPayload struct {
	UpdatedFields []string               `json:"updated_fields"` // 更新的字段列表
	OldValues     map[string]interface{} `json:"old_values"`     // 旧值
	NewValues     map[string]interface{} `json:"new_values"`     // 新值
}

// MessageEvent 消息事件
type MessageEvent struct {
	EventID        string          `json:"event_id"`        // 事件唯一标识
	EventType      string          `json:"event_type"`      // 事件类型：message.read, message.recalled
	Timestamp      int64           `json:"timestamp"`       // 事件时间戳
	ConversationID string          `json:"conversation_id"` // 会话ID
	OperatorID     string          `json:"operator_id"`     // 执行操作的用户ID
	Payload        json.RawMessage `json:"payload"`         // 事件载荷
}

// MessageReadPayload 消息已读事件载荷
type MessageReadPayload struct {
	MessageID int64 `json:"message_id"` // 消息ID
	SeqID     int64 `json:"seq_id"`     // 消息序号
}

// MessageRecalledPayload 消息撤回事件载荷
type MessageRecalledPayload struct {
	MessageID int64  `json:"message_id"` // 消息ID
	SeqID     int64  `json:"seq_id"`     // 消息序号
	Reason    string `json:"reason"`     // 撤回原因
}

// NotificationEvent 系统通知事件
type NotificationEvent struct {
	EventID      string          `json:"event_id"`       // 事件唯一标识
	EventType    string          `json:"event_type"`     // 事件类型：group.invited, friend.request, group.announcement
	Timestamp    int64           `json:"timestamp"`      // 事件时间戳
	TargetUserID string          `json:"target_user_id"` // 目标用户ID
	Payload      json.RawMessage `json:"payload"`        // 事件载荷
}

// GroupInvitedPayload 群聊邀请通知载荷
type GroupInvitedPayload struct {
	GroupID     string `json:"group_id"`     // 群ID
	GroupName   string `json:"group_name"`   // 群名称
	InviterID   string `json:"inviter_id"`   // 邀请人ID
	InviterName string `json:"inviter_name"` // 邀请人名称
}

// FriendRequestPayload 好友申请通知载荷
type FriendRequestPayload struct {
	RequestID     string `json:"request_id"`     // 申请ID
	RequesterID   string `json:"requester_id"`   // 申请人ID
	RequesterName string `json:"requester_name"` // 申请人名称
	Message       string `json:"message"`        // 申请消息
}

// GroupAnnouncementPayload 群公告通知载荷
type GroupAnnouncementPayload struct {
	GroupID        string `json:"group_id"`        // 群ID
	GroupName      string `json:"group_name"`      // 群名称
	AnnouncementID string `json:"announcement_id"` // 公告ID
	Title          string `json:"title"`           // 公告标题
	Content        string `json:"content"`         // 公告内容
	PublisherID    string `json:"publisher_id"`    // 发布人ID
	PublisherName  string `json:"publisher_name"`  // 发布人名称
}
