package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/api/kafka"
	"github.com/ceyewan/gochat/im-gateway/internal/config"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/mq"
)

// KafkaConsumer Kafka 消费者
type KafkaConsumer struct {
	config          *config.Config
	producer        mq.Producer
	consumer        mq.ConsumerGroup
	wsManager       *WebSocketManager
	logger          clog.Logger
	downstreamTopic string
}

// NewKafkaConsumer 创建 Kafka 消费者
func NewKafkaConsumer(cfg *config.Config, producer mq.Producer, wsManager *WebSocketManager) (*KafkaConsumer, error) {
	downstreamTopic := fmt.Sprintf("%s%s", kafka.TopicDownstreamPrefix, wsManager.GetGatewayID())

	// 创建消费者配置
	consumerConfig := mq.ConsumerConfig{
		GroupID:           wsManager.GetGatewayID(),
		Brokers:           cfg.Kafka.Brokers,
		Topics:            []string{downstreamTopic},
		StartOffset:       mq.OffsetNewest,
		SessionTimeout:    30 * time.Second,
		RebalanceTimeout:  60 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		FetchMin:          1,
		FetchDefault:      1024 * 1024,      // 1MB
		FetchMax:          10 * 1024 * 1024, // 10MB
		ChannelBufferSize: 256,
		MaxWaitTime:       100 * time.Millisecond,
	}

	// 创建消费者
	consumer, err := mq.NewConsumerGroup(consumerConfig)
	if err != nil {
		return nil, err
	}

	return &KafkaConsumer{
		config:          cfg,
		producer:        producer,
		consumer:        consumer,
		wsManager:       wsManager,
		logger:          clog.Namespace("kafka-consumer"),
		downstreamTopic: downstreamTopic,
	}, nil
}

// Start 启动消费者
func (kc *KafkaConsumer) Start(ctx context.Context) error {
	kc.logger.Info("启动 Kafka 消费者", clog.String("topic", kc.downstreamTopic))

	// 消费消息
	return kc.consumer.Consume(ctx, kc.handleMessage)
}

// Stop 停止消费者
func (kc *KafkaConsumer) Stop() error {
	kc.logger.Info("停止 Kafka 消费者")
	return kc.consumer.Close()
}

// handleMessage 处理消息
func (kc *KafkaConsumer) handleMessage(ctx context.Context, msg mq.Message) error {
	// 解析消息
	var downstreamMsg kafka.DownstreamMessage
	if err := json.Unmarshal(msg.Value, &downstreamMsg); err != nil {
		kc.logger.Error("解析下行消息失败", clog.Err(err))
		return err
	}

	kc.logger.Debug("收到下行消息",
		clog.String("trace_id", downstreamMsg.TraceID),
		clog.String("target_user_id", downstreamMsg.TargetUserID),
		clog.String("message_id", downstreamMsg.MessageID))

	// 构造 WebSocket 消息
	wsMsg := &WSMessage{
		Type:      WSMsgTypeNewMessage,
		MessageID: downstreamMsg.MessageID,
		Timestamp: time.Now().Unix(),
	}

	// 构造消息数据
	messageData := NewMessageData{
		MessageID:      downstreamMsg.MessageID,
		ConversationID: downstreamMsg.ConversationID,
		SenderID:       downstreamMsg.SenderID,
		MessageType:    downstreamMsg.MessageType,
		Content:        downstreamMsg.Content,
		SeqID:          downstreamMsg.SeqID,
		Timestamp:      downstreamMsg.Timestamp,
	}

	dataBytes, err := json.Marshal(messageData)
	if err != nil {
		kc.logger.Error("序列化消息数据失败", clog.Err(err))
		return err
	}
	wsMsg.Data = dataBytes

	// 推送消息给用户
	if err := kc.wsManager.PushMessage(downstreamMsg.TargetUserID, wsMsg); err != nil {
		kc.logger.Error("推送消息失败",
			clog.String("user_id", downstreamMsg.TargetUserID),
			clog.Err(err))
		return err
	}

	kc.logger.Debug("消息推送成功",
		clog.String("user_id", downstreamMsg.TargetUserID),
		clog.String("message_id", downstreamMsg.MessageID))

	return nil
}

// SendUpstreamMessage 发送上行消息到 Kafka
func (kc *KafkaConsumer) SendUpstreamMessage(ctx context.Context, msg *kafka.UpstreamMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return kc.producer.Send(ctx, kafka.TopicUpstream, data)
}
