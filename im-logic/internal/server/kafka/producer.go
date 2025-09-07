package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/api/kafka"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Producer Kafka 生产者
type Producer struct {
	config *config.Config
	logger clog.Logger
	client *kgo.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// NewProducer 创建 Kafka 生产者
func NewProducer(cfg *config.Config) (*Producer, error) {
	logger := clog.Module("kafka-producer")

	ctx, cancel := context.WithCancel(context.Background())

	// 创建 Kafka 客户端
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Kafka.Brokers...),
		kgo.ProducerBatchMaxBytes(int32(cfg.Kafka.Producer.BatchSize)),
		kgo.ProducerLinger(time.Duration(cfg.Kafka.Producer.BatchTimeout)*time.Millisecond),
		kgo.ProducerAcks(kgo.All()),
		kgo.ProducerRetries(int32(cfg.Kafka.Producer.Retries)),
		kgo.ProducerRetryBackoff(time.Duration(cfg.Kafka.Producer.RetryBackoff)*time.Millisecond),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		cancel()
		logger.Error("创建 Kafka 客户端失败", clog.Err(err))
		return nil, fmt.Errorf("创建 Kafka 客户端失败: %w", err)
	}

	producer := &Producer{
		config: cfg,
		logger: logger,
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}

	logger.Info("Kafka 生产者创建成功")
	return producer, nil
}

// Close 关闭生产者
func (p *Producer) Close() error {
	p.cancel()
	p.client.Close()
	p.logger.Info("Kafka 生产者已关闭")
	return nil
}

// ProduceDownstreamMessage 发送下行消息
func (p *Producer) ProduceDownstreamMessage(ctx context.Context, gatewayID string, message *kafka.DownstreamMessage) error {
	topic := p.config.GetDownstreamTopic(gatewayID)
	return p.produceMessage(ctx, topic, message)
}

// ProduceTaskMessage 发送任务消息
func (p *Producer) ProduceTaskMessage(ctx context.Context, message *kafka.TaskMessage) error {
	return p.produceMessage(ctx, p.config.Kafka.TaskTopic, message)
}

// produceMessage 发送消息
func (p *Producer) produceMessage(ctx context.Context, topic string, message interface{}) error {
	// 创建消息包装器
	traceID := p.getTraceID(ctx)
	wrapper, err := kafka.NewKafkaMessageWrapper(topic, message, traceID)
	if err != nil {
		p.logger.Error("创建消息包装器失败", clog.Err(err))
		return fmt.Errorf("创建消息包装器失败: %w", err)
	}

	// 序列化消息
	data, err := wrapper.Marshal()
	if err != nil {
		p.logger.Error("序列化消息失败", clog.Err(err))
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 发送消息
	record := kgo.Record{
		Topic: topic,
		Key:   []byte(traceID),
		Value: data,
	}

	// 设置重试策略
	retryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = p.client.ProduceSync(retryCtx, &record).FirstErr()
	if err != nil {
		p.logger.Error("发送消息失败",
			clog.String("topic", topic),
			clog.String("trace_id", traceID),
			clog.Err(err))
		return fmt.Errorf("发送消息失败: %w", err)
	}

	p.logger.Debug("消息发送成功",
		clog.String("topic", topic),
		clog.String("trace_id", traceID))

	return nil
}

// getTraceID 获取追踪 ID
func (p *Producer) getTraceID(ctx context.Context) string {
	// 从上下文中获取追踪 ID
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}

	// 生成新的追踪 ID
	return uuid.New().String()
}

// CreateDownstreamMessage 创建下行消息
func (p *Producer) CreateDownstreamMessage(upstream *kafka.UpstreamMessage, messageID string, seqID int64) *kafka.DownstreamMessage {
	return &kafka.DownstreamMessage{
		TraceID:        upstream.TraceID,
		TargetUserID:   upstream.UserID,
		MessageID:      messageID,
		ConversationID: upstream.ConversationID,
		SenderID:       upstream.UserID,
		MessageType:    upstream.MessageType,
		Content:        upstream.Content,
		SeqID:          seqID,
		Timestamp:      time.Now().Unix(),
		Headers:        upstream.Headers,
		Extra:          upstream.Extra,
	}
}

// CreateTaskMessage 创建任务消息
func (p *Producer) CreateTaskMessage(taskType kafka.TaskType, data interface{}) *kafka.TaskMessage {
	dataBytes, _ := json.Marshal(data)

	return &kafka.TaskMessage{
		TraceID:        uuid.New().String(),
		TaskType:       taskType,
		TaskID:         uuid.New().String(),
		Data:           dataBytes,
		Headers:        make(map[string]string),
		CreatedAt:      time.Now().Unix(),
		Priority:       2, // 普通优先级
		MaxRetries:     3,
		TimeoutSeconds: 300,
	}
}

// CreateFanoutTask 创建大群消息扩散任务
func (p *Producer) CreateFanoutTask(groupID, messageID, senderID string, excludeUserIDs []string) *kafka.TaskMessage {
	taskData := &kafka.FanoutTaskData{
		GroupID:        groupID,
		MessageID:      messageID,
		SenderID:       senderID,
		ExcludeUserIDs: excludeUserIDs,
	}

	return p.CreateTaskMessage(kafka.TaskTypeFanout, taskData)
}

// CreatePushTask 创建离线推送任务
func (p *Producer) CreatePushTask(userIDs []string, title, content string, pushType int) *kafka.TaskMessage {
	taskData := &kafka.PushTaskData{
		UserIDs:  userIDs,
		Title:    title,
		Content:  content,
		PushType: pushType,
		Data:     make(map[string]interface{}),
	}

	return p.CreateTaskMessage(kafka.TaskTypePush, taskData)
}

// GetClient 获取 Kafka 客户端
func (p *Producer) GetClient() *kgo.Client {
	return p.client
}

// IsHealthy 检查生产者健康状态
func (p *Producer) IsHealthy() bool {
	// 检查连接状态
	return p.client != nil
}
