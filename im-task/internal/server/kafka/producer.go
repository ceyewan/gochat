package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/internal/kafka"
	"github.com/ceyewan/gochat/internal/proto"
	"github.com/ceyewan/gochat/pkg/log"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	config *config.Config
	logger *log.Logger
	client *kgo.Client
}

func NewProducer(cfg *config.Config, logger *log.Logger) (*Producer, error) {
	kafkaClient, err := kafka.NewProducer(cfg.Kafka.Brokers, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	p := &Producer{
		config: cfg,
		logger: logger,
		client: kafkaClient,
	}

	return p, nil
}

func (p *Producer) ProduceGatewayMessage(ctx context.Context, gatewayID string, message *proto.DownstreamMessage) error {
	startTime := time.Now()
	requestID := uuid.New().String()

	topic := fmt.Sprintf("im-downstream-topic-%s", gatewayID)

	value, err := json.Marshal(message)
	if err != nil {
		p.logger.Error("Failed to marshal downstream message",
			"request_id", requestID,
			"gateway_id", gatewayID,
			"error", err,
		)
		return fmt.Errorf("failed to marshal downstream message: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(message.MessageID),
		Value: value,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		p.logger.Error("Failed to produce gateway message",
			"request_id", requestID,
			"gateway_id", gatewayID,
			"topic", topic,
			"error", err,
		)
		return fmt.Errorf("failed to produce gateway message: %w", err)
	}

	p.logger.Info("Gateway message produced successfully",
		"request_id", requestID,
		"gateway_id", gatewayID,
		"topic", topic,
		"message_id", message.MessageID,
		"duration", time.Since(startTime),
	)

	return nil
}

func (p *Producer) ProducePushNotification(ctx context.Context, userID string, notification *proto.PushNotification) error {
	startTime := time.Now()
	requestID := uuid.New().String()

	topic := "im-push-topic"

	value, err := json.Marshal(notification)
	if err != nil {
		p.logger.Error("Failed to marshal push notification",
			"request_id", requestID,
			"user_id", userID,
			"error", err,
		)
		return fmt.Errorf("failed to marshal push notification: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(userID),
		Value: value,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		p.logger.Error("Failed to produce push notification",
			"request_id", requestID,
			"user_id", userID,
			"topic", topic,
			"error", err,
		)
		return fmt.Errorf("failed to produce push notification: %w", err)
	}

	p.logger.Info("Push notification produced successfully",
		"request_id", requestID,
		"user_id", userID,
		"topic", topic,
		"notification_id", notification.NotificationID,
		"duration", time.Since(startTime),
	)

	return nil
}

func (p *Producer) Close() error {
	if p.client != nil {
		p.client.Close()
	}
	return nil
}
