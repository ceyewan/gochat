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
	// TopicConversationEvents 会话事件 (例如: 成员变更、会话更新)
	TopicConversationEvents = "gochat.conversation-events"
	// TopicMessageEvents 消息事件 (例如: 已读、撤回)
	TopicMessageEvents = "gochat.message-events"
	// TopicFriendEvents 好友事件 (例如: 好友申请、添加好友)
	TopicFriendEvents = "gochat.friend-events"
	// TopicSystemNotifications 系统通知 (例如: 被拉入群、好友申请通知)
	TopicSystemNotifications = "gochat.system-notifications"
)

// ConversationType 会话类型常量
const (
	ConversationTypeSingle = 1 // 单聊
	ConversationTypeGroup  = 2 // 群聊
	ConversationTypeWorld  = 3 // 世界聊天室
)

// MessageType 消息类型常量
const (
	MessageTypeText   = 1 // 文本消息
	MessageTypeImage  = 2 // 图片消息
	MessageTypeFile   = 3 // 文件消息
	MessageTypeSystem = 4 // 系统消息
	MessageTypeVoice  = 5 // 语音消息
	MessageTypeVideo  = 6 // 视频消息
)

// MemberRole 成员角色常量
const (
	MemberRoleMember = 1 // 普通成员
	MemberRoleAdmin  = 2 // 管理员
	MemberRoleOwner  = 3 // 群主/所有者
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
	ClientMsgID    string    `json:"client_msg_id"`
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
	ClientMsgID    string    `json:"client_msg_id"`
	Timestamp      int64     `json:"timestamp"`
	Extra          string    `json:"extra,omitempty"`
}

// FanoutTask 大群消息扩散任务
type FanoutTask struct {
	TraceID        string   `json:"trace_id"`
	ConversationID string   `json:"conversation_id"`
	MessageID      int64    `json:"message_id"` // 使用已持久化消息的ID
	ExcludeUserIDs []string `json:"exclude_user_ids,omitempty"`
	BatchSize      int      `json:"batch_size,omitempty"` // 批处理大小
}

// UserInfo 用户信息（轻量级）
type UserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	IsGuest   bool   `json:"is_guest"`
}

// ConversationInfo 会话信息（轻量级）
type ConversationInfo struct {
	ID          string `json:"id"`
	Type        int    `json:"type"`
	Name        string `json:"name"`
	AvatarURL   string `json:"avatar_url"`
	Description string `json:"description,omitempty"`
	OwnerID     string `json:"owner_id,omitempty"`
}

// SystemSignalMessage 系统信令消息
// 用于后端服务之间（如 im-task -> im-gateway）传递需要实时推送给客户端的非聊天类信息
type SystemSignalMessage struct {
	TraceID       string          `json:"trace_id"`
	TargetUserIDs []string        `json:"target_user_ids"` // 目标用户列表
	SignalType    string          `json:"signal_type"`     // 信令类型
	Payload       json.RawMessage `json:"payload"`         // 具体信令的载荷
	Timestamp     int64           `json:"timestamp"`
}

// SystemSignalType 系统信令类型常量
const (
	SignalTypeUserOnline              = "user.online"               // 用户上线
	SignalTypeUserOffline             = "user.offline"              // 用户下线
	SignalTypeMessageRead             = "message.read"              // 消息已读
	SignalTypeConversationMemberAdded = "conversation.member.added" // 会话成员添加
	SignalTypeConversationUpdated     = "conversation.updated"      // 会话信息更新
	SignalTypeFriendRequestReceived   = "friend.request.received"   // 收到好友申请
	SignalTypeFriendAdded             = "friend.added"              // 好友添加成功
)

// --- 领域事件消息结构 ---

// UserEvent 用户事件
type UserEvent struct {
	EventID   string          `json:"event_id"`   // 事件唯一标识
	EventType string          `json:"event_type"` // 事件类型
	Timestamp int64           `json:"timestamp"`  // 事件时间戳
	UserID    string          `json:"user_id"`    // 用户ID
	Payload   json.RawMessage `json:"payload"`    // 事件载荷
}

// UserEventType 用户事件类型常量
const (
	UserEventTypeOnline         = "user.online"          // 用户上线
	UserEventTypeOffline        = "user.offline"         // 用户下线
	UserEventTypeProfileUpdated = "user.profile.updated" // 用户资料更新
)

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

// ConversationEvent 会话事件
type ConversationEvent struct {
	EventID        string          `json:"event_id"`        // 事件唯一标识
	EventType      string          `json:"event_type"`      // 事件类型
	Timestamp      int64           `json:"timestamp"`       // 事件时间戳
	ConversationID string          `json:"conversation_id"` // 会话ID
	OperatorID     string          `json:"operator_id"`     // 执行操作的用户ID
	Payload        json.RawMessage `json:"payload"`         // 事件载荷
}

// ConversationEventType 会话事件类型常量
const (
	ConversationEventTypeMemberAdded       = "conversation.member.added"        // 成员添加
	ConversationEventTypeMemberRemoved     = "conversation.member.removed"      // 成员移除
	ConversationEventTypeMemberRoleUpdated = "conversation.member.role.updated" // 成员角色更新
	ConversationEventTypeInfoUpdated       = "conversation.info.updated"        // 会话信息更新
	ConversationEventTypeCreated           = "conversation.created"             // 会话创建
)

// ConversationMemberAddedPayload 成员添加事件载荷
type ConversationMemberAddedPayload struct {
	AddedUserIDs []string `json:"added_user_ids"` // 新增的用户ID列表
	AddedBy      string   `json:"added_by"`       // 添加者用户ID
}

// ConversationMemberRemovedPayload 成员移除事件载荷
type ConversationMemberRemovedPayload struct {
	RemovedUserID string `json:"removed_user_id"` // 被移除的用户ID
	RemovedBy     string `json:"removed_by"`      // 移除者用户ID
	Reason        string `json:"reason"`          // 移除原因
}

// ConversationInfoUpdatedPayload 会话信息更新事件载荷
type ConversationInfoUpdatedPayload struct {
	UpdatedFields []string               `json:"updated_fields"` // 更新的字段列表
	OldValues     map[string]interface{} `json:"old_values"`     // 旧值
	NewValues     map[string]interface{} `json:"new_values"`     // 新值
}

// MessageEvent 消息事件
type MessageEvent struct {
	EventID        string          `json:"event_id"`        // 事件唯一标识
	EventType      string          `json:"event_type"`      // 事件类型
	Timestamp      int64           `json:"timestamp"`       // 事件时间戳
	ConversationID string          `json:"conversation_id"` // 会话ID
	OperatorID     string          `json:"operator_id"`     // 执行操作的用户ID
	Payload        json.RawMessage `json:"payload"`         // 事件载荷
}

// MessageEventType 消息事件类型常量
const (
	MessageEventTypeRead     = "message.read"     // 消息已读
	MessageEventTypeRecalled = "message.recalled" // 消息撤回
)

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

// FriendEvent 好友事件
type FriendEvent struct {
	EventID   string          `json:"event_id"`   // 事件唯一标识
	EventType string          `json:"event_type"` // 事件类型
	Timestamp int64           `json:"timestamp"`  // 事件时间戳
	UserID    string          `json:"user_id"`    // 主用户ID
	FriendID  string          `json:"friend_id"`  // 好友用户ID
	Payload   json.RawMessage `json:"payload"`    // 事件载荷
}

// FriendEventType 好友事件类型常量
const (
	FriendEventTypeRequestSent     = "friend.request.sent"     // 好友申请发送
	FriendEventTypeRequestReceived = "friend.request.received" // 好友申请接收
	FriendEventTypeRequestAccepted = "friend.request.accepted" // 好友申请接受
	FriendEventTypeRequestRejected = "friend.request.rejected" // 好友申请拒绝
	FriendEventTypeFriendAdded     = "friend.added"            // 好友添加成功
	FriendEventTypeFriendRemoved   = "friend.removed"          // 好友删除
)

// FriendRequestPayload 好友申请相关事件载荷
type FriendRequestPayload struct {
	RequestID string `json:"request_id"` // 申请ID
	Message   string `json:"message"`    // 申请消息
}

// SystemNotificationEvent 系统通知事件
type SystemNotificationEvent struct {
	EventID      string          `json:"event_id"`       // 事件唯一标识
	EventType    string          `json:"event_type"`     // 事件类型
	Timestamp    int64           `json:"timestamp"`      // 事件时间戳
	TargetUserID string          `json:"target_user_id"` // 目标用户ID
	Payload      json.RawMessage `json:"payload"`        // 事件载荷
}

// SystemNotificationEventType 系统通知事件类型常量
const (
	SystemNotificationTypeConversationInvited = "conversation.invited" // 会话邀请
	SystemNotificationTypeFriendRequestNew    = "friend.request.new"   // 新的好友申请
	SystemNotificationTypeSystemAnnouncement  = "system.announcement"  // 系统公告
)

// ConversationInvitedPayload 会话邀请通知载荷
type ConversationInvitedPayload struct {
	ConversationID   string `json:"conversation_id"`   // 会话ID
	ConversationName string `json:"conversation_name"` // 会话名称
	ConversationType int    `json:"conversation_type"` // 会话类型
	InviterID        string `json:"inviter_id"`        // 邀请人ID
	InviterName      string `json:"inviter_name"`      // 邀请人名称
}

// FriendRequestNewPayload 新好友申请通知载荷
type FriendRequestNewPayload struct {
	RequestID     string `json:"request_id"`     // 申请ID
	RequesterID   string `json:"requester_id"`   // 申请人ID
	RequesterName string `json:"requester_name"` // 申请人名称
	Message       string `json:"message"`        // 申请消息
}

// SystemAnnouncementPayload 系统公告通知载荷
type SystemAnnouncementPayload struct {
	AnnouncementID string `json:"announcement_id"` // 公告ID
	Title          string `json:"title"`           // 公告标题
	Content        string `json:"content"`         // 公告内容
	Priority       int    `json:"priority"`        // 优先级
	ExpiresAt      int64  `json:"expires_at"`      // 过期时间
}

// TaskMessage 通用异步任务消息结构
type TaskMessage struct {
	TraceID    string          `json:"trace_id"`
	TaskType   string          `json:"task_type"`
	TaskID     string          `json:"task_id"`
	Data       json.RawMessage `json:"data"`
	CreatedAt  int64           `json:"created_at"`
	Priority   int             `json:"priority,omitempty"`    // 任务优先级
	RetryCount int             `json:"retry_count,omitempty"` // 重试次数
}

// TaskType 任务类型常量
const (
	TaskTypeMessageFanout     = "message.fanout"     // 消息扇出
	TaskTypeNotificationPush  = "notification.push"  // 通知推送
	TaskTypeDataMigration     = "data.migration"     // 数据迁移
	TaskTypeStatisticsCompute = "statistics.compute" // 统计计算
	TaskTypeCleanupExpired    = "cleanup.expired"    // 清理过期数据
)
