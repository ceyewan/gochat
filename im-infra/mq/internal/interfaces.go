package internal

import (
	"context"
	"time"
)

// Producer 定义消息生产者的核心接口。
// 提供同步和异步消息发布、幂等性保证、消息批处理等功能。
type Producer interface {
	// SendSync 同步发送单条消息
	// 返回错误表示发送失败，nil表示发送成功
	SendSync(ctx context.Context, topic string, message []byte) error

	// SendSyncWithKey 同步发送带键的消息（用于分区路由）
	SendSyncWithKey(ctx context.Context, topic string, key []byte, message []byte) error

	// SendSyncWithHeaders 同步发送带头部信息的消息
	SendSyncWithHeaders(ctx context.Context, topic string, key []byte, message []byte, headers map[string][]byte) error

	// SendAsync 异步发送单条消息
	// callback 在消息发送完成后被调用，参数为发送错误（nil表示成功）
	SendAsync(ctx context.Context, topic string, message []byte, callback func(error))

	// SendAsyncWithKey 异步发送带键的消息
	SendAsyncWithKey(ctx context.Context, topic string, key []byte, message []byte, callback func(error))

	// SendAsyncWithHeaders 异步发送带头部信息的消息
	SendAsyncWithHeaders(ctx context.Context, topic string, key []byte, message []byte, headers map[string][]byte, callback func(error))

	// SendBatchSync 同步发送消息批次
	// 返回每条消息的发送结果
	SendBatchSync(ctx context.Context, batch MessageBatch) ([]ProduceResult, error)

	// SendBatchAsync 异步发送消息批次
	// callback 在批次发送完成后被调用
	SendBatchAsync(ctx context.Context, batch MessageBatch, callback func([]ProduceResult, error))

	// Flush 刷新所有待发送的消息
	// 等待所有异步消息发送完成
	Flush(ctx context.Context) error

	// Close 关闭生产者，释放资源
	Close() error

	// GetMetrics 获取生产者性能指标
	GetMetrics() ProducerMetrics
}

// Consumer 定义消息消费者的核心接口。
// 提供可配置的消费者组、偏移量管理、回调式消息处理等功能。
type Consumer interface {
	// Subscribe 订阅主题列表
	// callback 在接收到消息时被调用
	Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error

	// SubscribePattern 使用正则表达式订阅主题
	SubscribePattern(ctx context.Context, pattern string, callback ConsumeCallback) error

	// Unsubscribe 取消订阅指定主题
	Unsubscribe(topics []string) error

	// UnsubscribeAll 取消所有订阅
	UnsubscribeAll() error

	// Pause 暂停指定主题分区的消费
	Pause(topicPartitions []TopicPartition) error

	// Resume 恢复指定主题分区的消费
	Resume(topicPartitions []TopicPartition) error

	// CommitOffset 手动提交偏移量
	CommitOffset(ctx context.Context, topic string, partition int32, offset int64) error

	// CommitOffsets 批量提交偏移量
	CommitOffsets(ctx context.Context, offsets map[TopicPartition]int64) error

	// GetCommittedOffset 获取已提交的偏移量
	GetCommittedOffset(ctx context.Context, topic string, partition int32) (int64, error)

	// GetCurrentOffset 获取当前消费偏移量
	GetCurrentOffset(topic string, partition int32) (int64, error)

	// Seek 设置消费位置到指定偏移量
	Seek(topic string, partition int32, offset int64) error

	// SeekToBeginning 设置消费位置到最早偏移量
	SeekToBeginning(topicPartitions []TopicPartition) error

	// SeekToEnd 设置消费位置到最新偏移量
	SeekToEnd(topicPartitions []TopicPartition) error

	// Close 关闭消费者，释放资源
	Close() error

	// GetMetrics 获取消费者性能指标
	GetMetrics() ConsumerMetrics
}

// ConnectionPool 定义连接池管理器的接口。
// 提供连接复用、健康检查、自动重连等功能。
type ConnectionPool interface {
	// GetConnection 从连接池获取连接
	GetConnection(ctx context.Context) (interface{}, error)

	// ReleaseConnection 将连接归还到连接池
	ReleaseConnection(conn interface{}) error

	// GetStats 获取连接池统计信息
	GetStats() PoolStats

	// HealthCheck 执行连接健康检查
	HealthCheck(ctx context.Context) error

	// Close 关闭连接池，释放所有连接
	Close() error
}

// MQ a high-level interface for message queue operations.
// It encapsulates the core functionalities like producer, consumer, and connection pool management.
type MQ interface {
	// Producer returns the message producer instance.
	Producer() Producer

	// Consumer returns the message consumer instance.
	Consumer() Consumer

	// ConnectionPool returns the connection pool manager.
	ConnectionPool() ConnectionPool

	// Ping checks the health of the connection to the message broker.
	Ping(ctx context.Context) error

	// Close shuts down the MQ instance, releasing all resources.
	Close() error
}

// MessageSerializer 定义消息序列化接口
type MessageSerializer interface {
	// Serialize 序列化消息
	Serialize(message interface{}) ([]byte, error)

	// Deserialize 反序列化消息
	Deserialize(data []byte, target interface{}) error

	// ContentType 返回序列化格式的内容类型
	ContentType() string
}

// CompressionCodec 定义压缩编解码器接口
type CompressionCodec interface {
	// Compress 压缩数据
	Compress(data []byte) ([]byte, error)

	// Decompress 解压数据
	Decompress(data []byte) ([]byte, error)

	// Type 返回压缩类型
	Type() string
}

// ErrorHandler 错误处理函数类型
type ErrorHandler func(error)

// ConsumeCallback 消费回调函数类型
// 参数：消息、主题分区信息、错误（如果有）
// 返回值：是否继续消费（false表示停止消费）
type ConsumeCallback func(message *Message, partition TopicPartition, err error) bool

// Message 消息结构体
type Message struct {
	// Topic 主题名称
	Topic string

	// Partition 分区号
	Partition int32

	// Offset 偏移量
	Offset int64

	// Key 消息键（可选，用于分区路由）
	Key []byte

	// Value 消息值
	Value []byte

	// Headers 消息头部信息
	Headers map[string][]byte

	// Timestamp 消息时间戳
	Timestamp time.Time
}

// MessageBatch 消息批次结构体
type MessageBatch struct {
	// Messages 消息列表
	Messages []*Message

	// MaxBatchSize 最大批次大小（字节）
	MaxBatchSize int

	// MaxBatchCount 最大批次消息数量
	MaxBatchCount int

	// LingerMs 批次等待时间（毫秒）
	LingerMs int
}

// TopicPartition 主题分区信息
type TopicPartition struct {
	// Topic 主题名称
	Topic string

	// Partition 分区号
	Partition int32
}

// ProduceResult 生产结果
type ProduceResult struct {
	// Topic 主题名称
	Topic string

	// Partition 分区号
	Partition int32

	// Offset 消息偏移量
	Offset int64

	// Error 发送错误（nil表示成功）
	Error error

	// Latency 发送延迟
	Latency time.Duration
}

// ProducerMetrics 生产者性能指标
type ProducerMetrics struct {
	// TotalMessages 总发送消息数
	TotalMessages int64

	// TotalBytes 总发送字节数
	TotalBytes int64

	// SuccessMessages 成功发送消息数
	SuccessMessages int64

	// FailedMessages 失败发送消息数
	FailedMessages int64

	// AverageLatency 平均延迟
	AverageLatency time.Duration

	// MaxLatency 最大延迟
	MaxLatency time.Duration

	// MinLatency 最小延迟
	MinLatency time.Duration

	// MessagesPerSecond 每秒消息数
	MessagesPerSecond float64

	// BytesPerSecond 每秒字节数
	BytesPerSecond float64
}

// ConsumerMetrics 消费者性能指标
type ConsumerMetrics struct {
	// TotalMessages 总消费消息数
	TotalMessages int64

	// TotalBytes 总消费字节数
	TotalBytes int64

	// MessagesPerSecond 每秒消息数
	MessagesPerSecond float64

	// BytesPerSecond 每秒字节数
	BytesPerSecond float64

	// Lag 消费延迟（未消费的消息数）
	Lag int64

	// LastCommittedOffset 最后提交的偏移量
	LastCommittedOffset map[TopicPartition]int64

	// CurrentOffset 当前消费偏移量
	CurrentOffset map[TopicPartition]int64
}

// PoolStats 连接池统计信息
type PoolStats struct {
	// TotalConnections 总连接数
	TotalConnections int

	// ActiveConnections 活跃连接数
	ActiveConnections int

	// IdleConnections 空闲连接数
	IdleConnections int

	// MaxConnections 最大连接数
	MaxConnections int

	// ConnectionsCreated 已创建连接数
	ConnectionsCreated int64

	// ConnectionsClosed 已关闭连接数
	ConnectionsClosed int64

	// ConnectionErrors 连接错误数
	ConnectionErrors int64
}
