package kafka

import (
	"encoding/json"
	"time"
)

// Kafka Topic 定义
const (
	// TopicUpstream 上行消息 Topic
	// 客户端发送的消息通过 im-gateway 投递到此 Topic
	TopicUpstream = "im-upstream-topic"

	// TopicDownstreamPrefix 下行消息 Topic 前缀
	// 实际 Topic 格式为: im-downstream-topic-{gateway_id}
	TopicDownstreamPrefix = "im-downstream-topic-"

	// TopicTask 异步任务 Topic
	// im-logic 投递给 im-task 的异步任务
	TopicTask = "im-task-topic"
)

// UpstreamMessage 上行消息结构
// 从客户端通过 im-gateway 发送到 im-logic 的消息
type UpstreamMessage struct {
	// 请求追踪 ID，用于全链路追踪
	TraceID string `json:"trace_id"`

	// 发送消息的用户 ID
	UserID string `json:"user_id"`

	// 用户所在的网关 ID
	GatewayID string `json:"gateway_id"`

	// 目标会话 ID
	ConversationID string `json:"conversation_id"`

	// 消息类型 (1:文本, 2:图片, 3:文件, 4:系统消息)
	MessageType int `json:"message_type"`

	// 消息内容
	Content string `json:"content"`

	// 客户端消息 ID，用于幂等性检查
	ClientMsgID string `json:"client_msg_id"`

	// 消息发送时间戳
	Timestamp int64 `json:"timestamp"`

	// 扩展头部信息
	Headers map[string]string `json:"headers"`

	// 消息扩展信息（JSON 格式）
	Extra string `json:"extra,omitempty"`
}

// DownstreamMessage 下行消息结构
// 从 im-logic 发送到 im-gateway 再推送给客户端的消息
type DownstreamMessage struct {
	// 请求追踪 ID
	TraceID string `json:"trace_id"`

	// 目标用户 ID
	TargetUserID string `json:"target_user_id"`

	// 消息 ID
	MessageID string `json:"message_id"`

	// 会话 ID
	ConversationID string `json:"conversation_id"`

	// 发送者 ID
	SenderID string `json:"sender_id"`

	// 消息类型
	MessageType int `json:"message_type"`

	// 消息内容
	Content string `json:"content"`

	// 会话内序列号
	SeqID int64 `json:"seq_id"`

	// 消息创建时间戳
	Timestamp int64 `json:"timestamp"`

	// 扩展头部信息
	Headers map[string]string `json:"headers"`

	// 发送者信息（可选，用于客户端显示）
	SenderInfo *UserInfo `json:"sender_info,omitempty"`

	// 消息扩展信息
	Extra string `json:"extra,omitempty"`
}

// TaskMessage 异步任务消息结构
// im-logic 发送给 im-task 的异步任务
type TaskMessage struct {
	// 请求追踪 ID
	TraceID string `json:"trace_id"`

	// 任务类型
	TaskType TaskType `json:"task_type"`

	// 任务 ID（唯一标识）
	TaskID string `json:"task_id"`

	// 任务数据（JSON 格式）
	Data json.RawMessage `json:"data"`

	// 扩展头部信息
	Headers map[string]string `json:"headers"`

	// 任务创建时间戳
	CreatedAt int64 `json:"created_at"`

	// 任务优先级 (1:低, 2:普通, 3:高, 4:紧急)
	Priority int `json:"priority"`

	// 最大重试次数
	MaxRetries int `json:"max_retries"`

	// 任务超时时间（秒）
	TimeoutSeconds int `json:"timeout_seconds"`
}

// TaskType 任务类型
type TaskType string

const (
	// TaskTypeFanout 大群消息扩散任务
	TaskTypeFanout TaskType = "fanout"

	// TaskTypePush 离线推送任务
	TaskTypePush TaskType = "push"

	// TaskTypeAudit 内容审核任务
	TaskTypeAudit TaskType = "audit"

	// TaskTypeIndex 数据索引任务
	TaskTypeIndex TaskType = "index"

	// TaskTypeRecommend 推荐更新任务
	TaskTypeRecommend TaskType = "recommend"

	// TaskTypeArchive 数据归档任务
	TaskTypeArchive TaskType = "archive"
)

// UserInfo 用户信息（轻量级）
type UserInfo struct {
	// 用户 ID
	ID string `json:"id"`

	// 用户名
	Username string `json:"username"`

	// 昵称
	Nickname string `json:"nickname"`

	// 头像 URL
	AvatarURL string `json:"avatar_url"`
}

// FanoutTaskData 大群消息扩散任务数据
type FanoutTaskData struct {
	// 群组 ID
	GroupID string `json:"group_id"`

	// 消息 ID
	MessageID string `json:"message_id"`

	// 发送者 ID
	SenderID string `json:"sender_id"`

	// 排除的用户 ID 列表（如发送者本人）
	ExcludeUserIDs []string `json:"exclude_user_ids,omitempty"`
}

// PushTaskData 离线推送任务数据
type PushTaskData struct {
	// 目标用户 ID 列表
	UserIDs []string `json:"user_ids"`

	// 推送标题
	Title string `json:"title"`

	// 推送内容
	Content string `json:"content"`

	// 推送数据（JSON 格式）
	Data map[string]interface{} `json:"data,omitempty"`

	// 推送类型 (1:消息通知, 2:系统通知)
	PushType int `json:"push_type"`
}

// IndexTaskData 数据索引任务数据
type IndexTaskData struct {
	// 索引类型 (message, user, group)
	IndexType string `json:"index_type"`

	// 操作类型 (create, update, delete)
	Action string `json:"action"`

	// 数据 ID
	DataID string `json:"data_id"`

	// 索引数据（JSON 格式）
	IndexData json.RawMessage `json:"index_data,omitempty"`
}

// KafkaMessageWrapper Kafka 消息包装器
// 所有 Kafka 消息的统一包装格式
type KafkaMessageWrapper struct {
	// 消息追踪 ID
	TraceID string `json:"trace_id"`

	// 消息头部
	Headers map[string]string `json:"headers"`

	// 消息体（序列化后的具体消息）
	Body json.RawMessage `json:"body"`

	// 消息类型标识
	MessageType string `json:"message_type"`

	// 消息版本
	Version string `json:"version"`

	// 消息创建时间戳
	Timestamp int64 `json:"timestamp"`
}

// NewKafkaMessageWrapper 创建 Kafka 消息包装器
func NewKafkaMessageWrapper(messageType string, body interface{}, traceID string) (*KafkaMessageWrapper, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return &KafkaMessageWrapper{
		TraceID:     traceID,
		Headers:     make(map[string]string),
		Body:        bodyBytes,
		MessageType: messageType,
		Version:     "1.0",
		Timestamp:   time.Now().Unix(),
	}, nil
}

// UnwrapMessage 解包 Kafka 消息
func (w *KafkaMessageWrapper) UnwrapMessage(target interface{}) error {
	return json.Unmarshal(w.Body, target)
}

// SetHeader 设置消息头部
func (w *KafkaMessageWrapper) SetHeader(key, value string) {
	if w.Headers == nil {
		w.Headers = make(map[string]string)
	}
	w.Headers[key] = value
}

// GetHeader 获取消息头部
func (w *KafkaMessageWrapper) GetHeader(key string) (string, bool) {
	if w.Headers == nil {
		return "", false
	}
	value, exists := w.Headers[key]
	return value, exists
}

// Marshal 序列化消息包装器
func (w *KafkaMessageWrapper) Marshal() ([]byte, error) {
	return json.Marshal(w)
}

// Unmarshal 反序列化消息包装器
func UnmarshalKafkaMessageWrapper(data []byte) (*KafkaMessageWrapper, error) {
	var wrapper KafkaMessageWrapper
	err := json.Unmarshal(data, &wrapper)
	if err != nil {
		return nil, err
	}
	return &wrapper, nil
}
