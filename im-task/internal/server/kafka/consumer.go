package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	taskpb "github.com/ceyewan/gochat/api/gen/im_task/v1"
	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/internal/kafka"
	"github.com/ceyewan/gochat/internal/proto"
	"github.com/ceyewan/gochat/pkg/log"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Consumer struct {
	config         *config.Config
	logger         *log.Logger
	client         *kgo.Client
	taskSvc        TaskService
	persistenceSvc PersistenceService
}

type TaskService interface {
	HandleTaskMessage(ctx context.Context, msg *taskpb.TaskMessage) error
}

type PersistenceService interface {
	HandlePersistenceMessage(ctx context.Context, msg *proto.Message) error
}

func NewConsumer(cfg *config.Config, logger *log.Logger, taskSvc TaskService, persistenceSvc PersistenceService) (*Consumer, error) {
	kafkaClient, err := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.ConsumerGroup, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	c := &Consumer{
		config:         cfg,
		logger:         logger,
		client:         kafkaClient,
		taskSvc:        taskSvc,
		persistenceSvc: persistenceSvc,
	}

	return c, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumers")

	taskOpts := kgo.ConsumeTopics(c.config.Kafka.TaskTopic)
	persistenceOpts := kgo.ConsumeTopics(c.config.Kafka.PersistenceTopic)

	c.client.AddConsumeTopics(taskOpts, c.config.Kafka.ConsumerGroup)
	c.client.AddConsumeTopics(persistenceOpts, c.config.Kafka.PersistenceGroup)

	go c.consumeMessages(ctx)

	c.logger.Info("Kafka consumers started")
	return nil
}

func (c *Consumer) consumeMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping Kafka consumers")
			return
		default:
			fetches := c.client.PollFetches(ctx)
			if fetches.IsClientTimeout() {
				continue
			}
			if fetches.Err() != nil {
				c.logger.Error("Kafka poll error", "error", fetches.Err())
				continue
			}

			for _, fetch := range fetches.Records() {
				c.processMessage(ctx, fetch)
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, record *kgo.Record) {
	startTime := time.Now()
	requestID := uuid.New().String()

	c.logger.Debug("Processing Kafka message",
		"request_id", requestID,
		"topic", record.Topic,
		"partition", record.Partition,
		"offset", record.Offset,
		"key", string(record.Key),
	)

	switch record.Topic {
	case c.config.Kafka.TaskTopic:
		c.processTaskMessage(ctx, record, requestID)
	case c.config.Kafka.PersistenceTopic:
		c.processPersistenceMessage(ctx, record, requestID)
	default:
		c.logger.Warn("Unknown topic", "topic", record.Topic)
	}

	c.logger.Debug("Message processed",
		"request_id", requestID,
		"topic", record.Topic,
		"duration", time.Since(startTime),
	)
}

func (c *Consumer) processTaskMessage(ctx context.Context, record *kgo.Record, requestID string) {
	var taskMsg taskpb.TaskMessage
	if err := json.Unmarshal(record.Value, &taskMsg); err != nil {
		c.logger.Error("Failed to unmarshal task message",
			"request_id", requestID,
			"error", err,
		)
		return
	}

	ctx = context.WithValue(ctx, "request_id", requestID)

	if err := c.taskSvc.HandleTaskMessage(ctx, &taskMsg); err != nil {
		c.logger.Error("Failed to handle task message",
			"request_id", requestID,
			"task_type", taskMsg.Type.String(),
			"error", err,
		)
		return
	}

	c.logger.Info("Task message processed successfully",
		"request_id", requestID,
		"task_type", taskMsg.Type.String(),
		"message_id", taskMsg.MessageId,
	)
}

func (c *Consumer) processPersistenceMessage(ctx context.Context, record *kgo.Record, requestID string) {
	var message proto.Message
	if err := json.Unmarshal(record.Value, &message); err != nil {
		c.logger.Error("Failed to unmarshal persistence message",
			"request_id", requestID,
			"error", err,
		)
		return
	}

	ctx = context.WithValue(ctx, "request_id", requestID)

	if err := c.persistenceSvc.HandlePersistenceMessage(ctx, &message); err != nil {
		c.logger.Error("Failed to handle persistence message",
			"request_id", requestID,
			"message_id", message.MessageID,
			"error", err,
		)
		return
	}

	c.logger.Info("Persistence message processed successfully",
		"request_id", requestID,
		"message_id", message.MessageID,
		"conversation_id", message.ConversationID,
	)
}

func (c *Consumer) Close() error {
	if c.client != nil {
		c.client.Close()
	}
	return nil
}
