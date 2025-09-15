package kafka

import (
	"context"
)

// TraceIDKey 是用于在 context 中传递 trace ID 的 key
const TraceIDKey = "trace-id"

// Message 是跨服务的标准消息结构。
type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string][]byte
}

// ConsumeCallback 定义了消息处理回调函数
// 返回值表示是否成功处理，成功时消费者会提交偏移量
type ConsumeCallback func(ctx context.Context, msg *Message) error

// Provider 定义了 kafka 组件提供的所有能力
type Provider interface {
	Producer() ProducerOperations
	Consumer(groupID string) ConsumerOperations
	Admin() AdminOperations

	// Ping 检查与 Kafka 集群的连接
	Ping(ctx context.Context) error
	// Close 关闭所有与 Kafka 的连接
	Close() error
}

// ProducerOperations 定义了生产者的操作接口
type ProducerOperations interface {
	// Send 异步发送消息。此方法立即返回，并通过回调函数处理发送结果。
	Send(ctx context.Context, msg *Message, callback func(error))

	// SendSync 同步发送消息。此方法将阻塞直到消息发送成功或失败。
	SendSync(ctx context.Context, msg *Message) error

	// Close 关闭生产者，并确保所有缓冲区的消息都已发送。
	Close() error

	// GetMetrics 获取性能指标
	GetMetrics() map[string]interface{}

	// Ping 健康检查
	Ping(ctx context.Context) error
}

// ConsumerOperations 定义了消费者的操作接口
type ConsumerOperations interface {
	// Subscribe 订阅消息并根据处理结果决定是否提交偏移量。
	Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error

	// Close 优雅地关闭消费者，完成当前正在处理的消息并提交最后一次偏移量。
	Close() error

	// GetMetrics 获取性能指标
	GetMetrics() map[string]interface{}

	// Ping 健康检查
	Ping(ctx context.Context) error
}

// AdminOperations 定义了管理操作接口
type AdminOperations interface {
	// CreateTopic 创建主题
	CreateTopic(ctx context.Context, topic string, partitions int32, replicationFactor int16, config map[string]string) error

	// DeleteTopic 删除主题
	DeleteTopic(ctx context.Context, topic string) error

	// ListTopics 列出所有主题
	ListTopics(ctx context.Context) (map[string]TopicDetail, error)

	// GetTopicMetadata 获取主题元数据
	GetTopicMetadata(ctx context.Context, topic string) (*TopicDetail, error)

	// CreatePartitions 增加主题分区数
	CreatePartitions(ctx context.Context, topic string, newPartitionCount int32) error
}

// TopicDetail 包含主题的详细信息
type TopicDetail struct {
	NumPartitions     int32
	ReplicationFactor int16
	Config            map[string]string
}

// Producer 是向后兼容的生产者接口
// 新代码建议使用 Provider.Producer() 获取 ProducerOperations
type Producer interface {
	// Send 异步发送消息。此方法立即返回，并通过回调函数处理发送结果。
	Send(ctx context.Context, msg *Message, callback func(error))

	// SendSync 同步发送消息。此方法将阻塞直到消息发送成功或失败。
	SendSync(ctx context.Context, msg *Message) error

	// Close 关闭生产者，并确保所有缓冲区的消息都已发送。
	Close() error

	// GetMetrics 获取性能指标
	GetMetrics() map[string]interface{}

	// Ping 健康检查
	Ping(ctx context.Context) error
}

// Consumer 是向后兼容的消费者接口
// 新代码建议使用 Provider.Consumer() 获取 ConsumerOperations
type Consumer interface {
	// Subscribe 订阅消息并根据处理结果决定是否提交偏移量。
	Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error

	// Close 优雅地关闭消费者，完成当前正在处理的消息并提交最后一次偏移量。
	Close() error

	// GetMetrics 获取性能指标
	GetMetrics() map[string]interface{}

	// Ping 健康检查
	Ping(ctx context.Context) error
}