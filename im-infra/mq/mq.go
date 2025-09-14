package mq

import (
	"context"
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/mq/internal"
)

// Producer 定义消息生产者的核心接口。
// 提供同步和异步消息发布、幂等性保证、消息批处理等功能。
type Producer = internal.Producer

// Consumer 定义消息消费者的核心接口。
// 提供可配置的消费者组、偏移量管理、回调式消息处理等功能。
type Consumer = internal.Consumer

// ConnectionPool 定义连接池管理器的接口。
// 提供连接复用、健康检查、自动重连等功能。
type ConnectionPool = internal.ConnectionPool

// Config 是 mq 的主配置结构体。
// 用于声明式地定义Kafka连接和行为参数。
type Config = internal.Config

// ProducerConfig 生产者配置
type ProducerConfig = internal.ProducerConfig

// ConsumerConfig 消费者配置
type ConsumerConfig = internal.ConsumerConfig

// AdminClient 管理客户端接口
type AdminClient = internal.AdminClient

// TopicConfig topic 配置
type TopicConfig = internal.TopicConfig

// Message 消息结构体
type Message = internal.Message

// MessageBatch 消息批次结构体
type MessageBatch = internal.MessageBatch

// TopicPartition 主题分区信息
type TopicPartition = internal.TopicPartition

// ProduceResult 生产结果
type ProduceResult = internal.ProduceResult

// ConsumeCallback 消费回调函数类型
type ConsumeCallback = internal.ConsumeCallback

// ErrorHandler 错误处理函数类型
type ErrorHandler = internal.ErrorHandler

var (
	// 全局默认MQ实例
	defaultMQ MQ
	// 确保默认MQ只初始化一次
	defaultMQOnce sync.Once
	// 模块日志器
	logger = clog.Namespace("mq")
)

// MQ 定义消息队列操作的核心接口。
// 提供生产者、消费者和连接池的统一管理。
type MQ interface {
	// Producer 获取生产者实例
	Producer() Producer

	// Consumer 获取消费者实例
	Consumer() Consumer

	// ConnectionPool 获取连接池实例
	ConnectionPool() ConnectionPool

	// Close 关闭MQ实例，释放所有资源
	Close() error

	// Ping 检查连接健康状态
	Ping(ctx context.Context) error
}

// getDefaultMQ 获取全局默认MQ实例
func getDefaultMQ() MQ {
	defaultMQOnce.Do(func() {
		cfg := DefaultConfig()
		var err error
		defaultMQ, err = internal.NewMQ(cfg)
		if err != nil {
			logger.Error("创建默认MQ实例失败", clog.Err(err))
			panic(err)
		}
		logger.Info("默认MQ实例创建成功")
	})
	return defaultMQ
}

// New 根据提供的配置创建一个新的 MQ 实例。
// 这是核心工厂函数，按配置组装所有组件。
func New(cfg Config) (MQ, error) {
	return internal.NewMQ(cfg)
}

// Default 返回一个带有合理默认配置的 MQ 实例。
// 默认MQ连接到 localhost:9092，使用合理的性能配置。
func Default() MQ {
	return getDefaultMQ()
}

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置适用于大多数开发和测试场景。
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// DefaultProducerConfig 返回默认的生产者配置
func DefaultProducerConfig() ProducerConfig {
	return internal.DefaultProducerConfig()
}

// DefaultConsumerConfig 返回默认的消费者配置
func DefaultConsumerConfig() ConsumerConfig {
	return internal.DefaultConsumerConfig()
}

// MergeWithDefaults 将用户配置与默认配置合并
// 用户未设置的字段将使用默认值，这样用户只需要设置需要自定义的字段
func MergeWithDefaults(cfg Config) Config {
	return internal.MergeWithDefaults(cfg)
}

// ===== 全局生产者方法 =====

// SendSync 使用全局默认MQ同步发送消息
func SendSync(ctx context.Context, topic string, message []byte) error {
	return getDefaultMQ().Producer().SendSync(ctx, topic, message)
}

// SendAsync 使用全局默认MQ异步发送消息
func SendAsync(ctx context.Context, topic string, message []byte, callback func(error)) {
	getDefaultMQ().Producer().SendAsync(ctx, topic, message, callback)
}

// SendBatchSync 使用全局默认MQ同步发送消息批次
func SendBatchSync(ctx context.Context, batch MessageBatch) ([]ProduceResult, error) {
	return getDefaultMQ().Producer().SendBatchSync(ctx, batch)
}

// SendBatchAsync 使用全局默认MQ异步发送消息批次
func SendBatchAsync(ctx context.Context, batch MessageBatch, callback func([]ProduceResult, error)) {
	getDefaultMQ().Producer().SendBatchAsync(ctx, batch, callback)
}

// ===== 全局消费者方法 =====

// Subscribe 使用全局默认MQ订阅主题
func Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error {
	return getDefaultMQ().Consumer().Subscribe(ctx, topics, callback)
}

// Unsubscribe 使用全局默认MQ取消订阅主题
func Unsubscribe(topics []string) error {
	return getDefaultMQ().Consumer().Unsubscribe(topics)
}

// CommitOffset 使用全局默认MQ提交偏移量
func CommitOffset(ctx context.Context, topic string, partition int32, offset int64) error {
	return getDefaultMQ().Consumer().CommitOffset(ctx, topic, partition, offset)
}

// ===== 全局连接池方法 =====

// GetConnection 使用全局默认MQ获取连接
func GetConnection(ctx context.Context) (interface{}, error) {
	return getDefaultMQ().ConnectionPool().GetConnection(ctx)
}

// ReleaseConnection 使用全局默认MQ释放连接
func ReleaseConnection(conn interface{}) error {
	return getDefaultMQ().ConnectionPool().ReleaseConnection(conn)
}

// Ping 使用全局默认MQ检查连接健康状态
func Ping(ctx context.Context) error {
	return getDefaultMQ().Ping(ctx)
}

// ===== 高级工厂方法 =====

// NewProducer 创建独立的生产者实例
func NewProducer(cfg ProducerConfig) (Producer, error) {
	return internal.NewProducer(cfg)
}

// NewConsumer 创建独立的消费者实例
func NewConsumer(cfg ConsumerConfig) (Consumer, error) {
	return internal.NewConsumer(cfg)
}

// NewConnectionPool 创建独立的连接池实例
func NewConnectionPool(cfg Config) (ConnectionPool, error) {
	return internal.NewConnectionPool(cfg)
}

// NewAdminClient 创建管理客户端
// 用于创建、删除、列出 topic 等管理操作
func NewAdminClient(cfg Config) (AdminClient, error) {
	return internal.NewAdminClient(cfg)
}

// ===== 错误处理工具函数 =====

// IsRetryableError 判断错误是否可重试
func IsRetryableError(err error) bool {
	return internal.IsRetryableError(err)
}

// IsFatalError 判断错误是否为致命错误（不可恢复）
func IsFatalError(err error) bool {
	return internal.IsFatalError(err)
}

// NewMQError 创建新的MQ错误
func NewMQError(code, message string, cause error) error {
	return internal.NewMQError(code, message, cause)
}

// ===== 序列化和压缩工具 =====

// NewMessageUtils 创建消息工具类
func NewMessageUtils(serializerType, compressionType string) *internal.MessageUtils {
	return internal.NewMessageUtils(serializerType, compressionType)
}

// NewCompressionCodec 创建压缩编解码器
func NewCompressionCodec(compressionType string) internal.CompressionCodec {
	return internal.NewCompressionCodec(compressionType)
}
