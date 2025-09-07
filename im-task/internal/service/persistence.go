package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/internal/proto"
	"github.com/ceyewan/gochat/pkg/log"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
)

type PersistenceService struct {
	config     *config.Config
	logger     *log.Logger
	repoClient repobb.ImRepoServiceClient
}

func NewPersistenceService(cfg *config.Config, logger *log.Logger, repoClient repobb.ImRepoServiceClient) *PersistenceService {
	return &PersistenceService{
		config:     cfg,
		logger:     logger,
		repoClient: repoClient,
	}
}

func (s *PersistenceService) HandlePersistenceMessage(ctx context.Context, msg *proto.Message) error {
	startTime := time.Now()
	requestID, _ := ctx.Value("request_id").(string)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	s.logger.Info("Starting message persistence",
		"request_id", requestID,
		"message_id", msg.MessageID,
		"conversation_id", msg.ConversationID,
		"sender_id", msg.SenderID,
		"type", msg.Type,
	)

	if err := s.validateMessage(msg); err != nil {
		s.logger.Error("Message validation failed",
			"request_id", requestID,
			"message_id", msg.MessageID,
			"error", err,
		)
		return fmt.Errorf("message validation failed: %w", err)
	}

	retryCount := 0
	maxRetries := s.config.Task.MessageRetryAttempts
	backoff := s.config.Task.MessageRetryBackoff

	var lastErr error

	for retryCount <= maxRetries {
		if retryCount > 0 {
			s.logger.Info("Retrying message persistence",
				"request_id", requestID,
				"message_id", msg.MessageID,
				"attempt", retryCount,
				"max_attempts", maxRetries,
			)
			time.Sleep(backoff)
			backoff *= 2
		}

		err := s.persistMessage(ctx, msg)
		if err == nil {
			s.logger.Info("Message persisted successfully",
				"request_id", requestID,
				"message_id", msg.MessageID,
				"attempts", retryCount+1,
				"duration", time.Since(startTime),
			)
			return nil
		}

		lastErr = err
		retryCount++

		s.logger.Error("Failed to persist message",
			"request_id", requestID,
			"message_id", msg.MessageID,
			"attempt", retryCount,
			"error", err,
		)
	}

	s.logger.Error("Message persistence failed after all retries",
		"request_id", requestID,
		"message_id", msg.MessageID,
		"total_attempts", retryCount,
		"last_error", lastErr,
	)

	return fmt.Errorf("failed to persist message after %d attempts: %w", retryCount, lastErr)
}

func (s *PersistenceService) validateMessage(msg *proto.Message) error {
	if msg.MessageID == "" {
		return fmt.Errorf("message ID is required")
	}
	if msg.ConversationID == "" {
		return fmt.Errorf("conversation ID is required")
	}
	if msg.SenderID == "" {
		return fmt.Errorf("sender ID is required")
	}
	if msg.Type == "" {
		return fmt.Errorf("message type is required")
	}
	if msg.Content == "" && msg.MediaURL == "" {
		return fmt.Errorf("message content or media URL is required")
	}
	return nil
}

func (s *PersistenceService) persistMessage(ctx context.Context, msg *proto.Message) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.RepoService.Timeout)
	defer cancel()

	req := &repopb.CreateMessageRequest{
		MessageId:      msg.MessageID,
		ConversationId: msg.ConversationID,
		SenderId:       msg.SenderID,
		Type:           msg.Type,
		Content:        msg.Content,
		MediaUrl:       msg.MediaURL,
		MediaType:      msg.MediaType,
		MediaSize:      msg.MediaSize,
		SentAt:         msg.SentAt,
	}

	resp, err := s.repoClient.CreateMessage(ctx, req)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.DeadlineExceeded:
				return fmt.Errorf("repo service timeout: %w", err)
			case codes.Unavailable:
				return fmt.Errorf("repo service unavailable: %w", err)
			case codes.AlreadyExists:
				return fmt.Errorf("message already exists: %w", err)
			default:
				return fmt.Errorf("repo service error: %w", err)
			}
		}
		return fmt.Errorf("failed to create message: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to create message: %s", resp.Message)
	}

	return nil
}
