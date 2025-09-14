package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/api/kafka"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/service"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Consumer Kafka 消费者
type Consumer struct {
	config  *config.Config
	logger  clog.Logger
	client  *kgo.Client
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	handler *service.MessageHandler
}

// NewConsumer 创建 Kafka 消费者
func NewConsumer(cfg *config.Config, handler *service.MessageHandler) (*Consumer, error) {
	logger := clog.Namespace("kafka-consumer")

	ctx, cancel := context.WithCancel(context.Background())

	// 创建 Kafka 客户端
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Kafka.Brokers...),
		kgo.ConsumerGroup(cfg.Kafka.ConsumerGroup),
		kgo.ConsumeTopics(cfg.Kafka.UpstreamTopic),
		kgo.Balancers(kgo.CooperativeStickyBalancer()),
		kgo.SessionTimeout(cfg.Kafka.SessionTimeout*time.Second),
		kgo.HeartbeatInterval(cfg.Kafka.HeartbeatInterval*time.Second),
		kgo.FetchMaxBytes(10*1024*1024), // 10MB
		kgo.FetchMaxWait(time.Duration(cfg.Kafka.BatchTimeout)*time.Millisecond),
		kgo.FetchMinBytes(1),
		kgo.FetchMaxRecords(int32(cfg.Kafka.BatchSize)),
		kgo.AutoCommitMarks(),
		kgo.AutoCommitInterval(5*time.Second),
		kgo.WithLogger(kgo.BasicLogger(logger.Writer(), kgo.LogLevelInfo, func() string {
			return "kafka-consumer"
		})),
	)
	if err != nil {
		cancel()
		logger.Error("创建 Kafka 客户端失败", clog.Err(err))
		return nil, fmt.Errorf("创建 Kafka 客户端失败: %w", err)
	}

	consumer := &Consumer{
		config:  cfg,
		logger:  logger,
		client:  client,
		ctx:     ctx,
		cancel:  cancel,
		handler: handler,
	}

	logger.Info("Kafka 消费者创建成功")
	return consumer, nil
}

// Start 启动消费者
func (c *Consumer) Start() error {
	c.logger.Info("启动 Kafka 消费者...")

	c.wg.Add(1)
	go c.consume()

	c.logger.Info("Kafka 消费者启动成功")
	return nil
}

// consume 消费消息
func (c *Consumer) consume() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("消费者停止消费")
			return
		default:
			// 获取消息
			fetches := c.client.PollFetches(c.ctx)
			if fetches.IsClientClosed() {
				c.logger.Info("Kafka 客户端已关闭")
				return
			}

			if fetches.Err() != nil {
				c.logger.Error("拉取消息失败", clog.Err(fetches.Err()))
				continue
			}

			// 处理消息
			fetches.EachRecord(c.processRecord)
		}
	}
}

// processRecord 处理单条消息
func (c *Consumer) processRecord(record *kgo.Record) {
	ctx := c.ctx

	// 解析消息包装器
	wrapper, err := kafka.UnmarshalKafkaMessageWrapper(record.Value)
	if err != nil {
		c.logger.Error("解析消息包装器失败", clog.Err(err))
		return
	}

	// 设置追踪 ID
	ctx = context.WithValue(ctx, "trace_id", wrapper.TraceID)

	c.logger.Debug("收到消息",
		clog.String("topic", record.Topic),
		clog.String("trace_id", wrapper.TraceID),
		clog.String("message_type", wrapper.MessageType))

	// 根据消息类型处理
	switch wrapper.MessageType {
	case "upstream":
		c.handleUpstreamMessage(ctx, wrapper)
	case "downstream":
		c.handleDownstreamMessage(ctx, wrapper)
	case "task":
		c.handleTaskMessage(ctx, wrapper)
	default:
		c.logger.Warn("未知的消息类型", clog.String("message_type", wrapper.MessageType))
	}
}

// handleUpstreamMessage 处理上行消息
func (c *Consumer) handleUpstreamMessage(ctx context.Context, wrapper *kafka.KafkaMessageWrapper) {
	var upstream kafka.UpstreamMessage
	if err := wrapper.UnwrapMessage(&upstream); err != nil {
		c.logger.Error("解析上行消息失败", clog.Err(err))
		return
	}

	// 处理上行消息
	if err := c.handler.HandleUpstreamMessage(ctx, &upstream); err != nil {
		c.logger.Error("处理上行消息失败",
			clog.String("trace_id", upstream.TraceID),
			clog.String("conversation_id", upstream.ConversationID),
			clog.Err(err))
	}
}

// handleDownstreamMessage 处理下行消息
func (c *Consumer) handleDownstreamMessage(ctx context.Context, wrapper *kafka.KafkaMessageWrapper) {
	var downstream kafka.DownstreamMessage
	if err := wrapper.UnwrapMessage(&downstream); err != nil {
		c.logger.Error("解析下行消息失败", clog.Err(err))
		return
	}

	// 处理下行消息
	if err := c.handler.HandleDownstreamMessage(ctx, &downstream); err != nil {
		c.logger.Error("处理下行消息失败",
			clog.String("trace_id", downstream.TraceID),
			clog.String("message_id", downstream.MessageID),
			clog.Err(err))
	}
}

// handleTaskMessage 处理任务消息
func (c *Consumer) handleTaskMessage(ctx context.Context, wrapper *kafka.KafkaMessageWrapper) {
	var task kafka.TaskMessage
	if err := wrapper.UnwrapMessage(&task); err != nil {
		c.logger.Error("解析任务消息失败", clog.Err(err))
		return
	}

	// 处理任务消息
	if err := c.handler.HandleTaskMessage(ctx, &task); err != nil {
		c.logger.Error("处理任务消息失败",
			clog.String("trace_id", task.TraceID),
			clog.String("task_id", task.TaskID),
			clog.Err(err))
	}
}

// Close 关闭消费者
func (c *Consumer) Close() error {
	c.logger.Info("关闭 Kafka 消费者...")

	// 取消上下文
	c.cancel()

	// 等待消费协程结束
	c.wg.Wait()

	// 关闭客户端
	c.client.Close()

	c.logger.Info("Kafka 消费者已关闭")
	return nil
}

// GetClient 获取 Kafka 客户端
func (c *Consumer) GetClient() *kgo.Client {
	return c.client
}

// IsHealthy 检查消费者健康状态
func (c *Consumer) IsHealthy() bool {
	// 检查连接状态
	return c.client != nil
}

// PauseConsumption 暂停消费
func (c *Consumer) PauseConsumption() {
	c.logger.Info("暂停 Kafka 消费")
	c.client.PauseFetchRecords(c.config.Kafka.UpstreamTopic)
}

// ResumeConsumption 恢复消费
func (c *Consumer) ResumeConsumption() {
	c.logger.Info("恢复 Kafka 消费")
	c.client.ResumeFetchRecords(c.config.Kafka.UpstreamTopic)
}

// GetConsumerGroup 获取消费者组信息
func (c *Consumer) GetConsumerGroup() string {
	return c.config.Kafka.ConsumerGroup
}

// GetSubscribedTopics 获取订阅的主题
func (c *Consumer) GetSubscribedTopics() []string {
	return []string{c.config.Kafka.UpstreamTopic}
}

// CreateUpstreamMessageForTesting 创建上行消息用于测试
func (c *Consumer) CreateUpstreamMessageForTesting(userID, gatewayID, conversationID string, messageType int, content string) *kafka.UpstreamMessage {
	return &kafka.UpstreamMessage{
		TraceID:        fmt.Sprintf("test-%d", time.Now().Unix()),
		UserID:         userID,
		GatewayID:      gatewayID,
		ConversationID: conversationID,
		MessageType:    messageType,
		Content:        content,
		ClientMsgID:    fmt.Sprintf("client-%d", time.Now().Unix()),
		Timestamp:      time.Now().Unix(),
		Headers:        make(map[string]string),
	}
}

// MockProduceMessageForTesting 模拟生产消息用于测试
func (c *Consumer) MockProduceMessageForTesting(message *kafka.UpstreamMessage) error {
	// 创建消息包装器
	wrapper, err := kafka.NewKafkaMessageWrapper("upstream", message, message.TraceID)
	if err != nil {
		return err
	}

	// 序列化消息
	data, err := wrapper.Marshal()
	if err != nil {
		return err
	}

	// 发送消息
	record := kgo.Record{
		Topic: c.config.Kafka.UpstreamTopic,
		Key:   []byte(message.TraceID),
		Value: data,
	}

	// 同步发送
	err = c.client.ProduceSync(c.ctx, &record).FirstErr()
	if err != nil {
		return err
	}

	return nil
}
