package service

import (
	"context"
	"fmt"
	"time"

	taskpb "github.com/ceyewan/gochat/api/gen/im_task/v1"
	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/internal/proto"
	"github.com/ceyewan/gochat/pkg/log"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
)

type TaskService struct {
	config      *config.Config
	logger      *log.Logger
	repoClient  repobb.ImRepoServiceClient
	producer    Producer
	pushService PushService
}

type Producer interface {
	ProduceGatewayMessage(ctx context.Context, gatewayID string, message *proto.DownstreamMessage) error
	ProducePushNotification(ctx context.Context, userID string, notification *proto.PushNotification) error
}

type PushService interface {
	SendPushNotification(ctx context.Context, userID string, message *proto.Message) error
}

func NewTaskService(cfg *config.Config, logger *log.Logger, repoClient repobb.ImRepoServiceClient, producer Producer, pushService PushService) *TaskService {
	return &TaskService{
		config:      cfg,
		logger:      logger,
		repoClient:  repoClient,
		producer:    producer,
		pushService: pushService,
	}
}

func (s *TaskService) HandleTaskMessage(ctx context.Context, msg *taskpb.TaskMessage) error {
	startTime := time.Now()
	requestID, _ := ctx.Value("request_id").(string)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	s.logger.Info("Handling task message",
		"request_id", requestID,
		"task_type", msg.Type.String(),
		"message_id", msg.MessageId,
		"group_id", msg.GroupId,
	)

	switch msg.Type {
	case taskpb.TaskMessageType_TASK_MESSAGE_TYPE_LARGE_GROUP_FANOUT:
		return s.handleLargeGroupFanout(ctx, msg, requestID)
	case taskpb.TaskMessageType_TASK_MESSAGE_TYPE_OFFLINE_PUSH:
		return s.handleOfflinePush(ctx, msg, requestID)
	case taskpb.TaskMessageType_TASK_MESSAGE_TYPE_MESSAGE_RETRY:
		return s.handleMessageRetry(ctx, msg, requestID)
	default:
		return fmt.Errorf("unknown task message type: %s", msg.Type.String())
	}
}

func (s *TaskService) handleLargeGroupFanout(ctx context.Context, msg *taskpb.TaskMessage, requestID string) error {
	s.logger.Info("Processing large group fanout",
		"request_id", requestID,
		"group_id", msg.GroupId,
		"message_id", msg.MessageId,
	)

	groupUsers, err := s.getGroupUsers(ctx, msg.GroupId)
	if err != nil {
		s.logger.Error("Failed to get group users",
			"request_id", requestID,
			"group_id", msg.GroupId,
			"error", err,
		)
		return fmt.Errorf("failed to get group users: %w", err)
	}

	s.logger.Info("Retrieved group users for fanout",
		"request_id", requestID,
		"group_id", msg.GroupId,
		"user_count", len(groupUsers),
	)

	batchSize := s.config.Task.FanoutBatchSize
	for i := 0; i < len(groupUsers); i += batchSize {
		end := i + batchSize
		if end > len(groupUsers) {
			end = len(groupUsers)
		}

		batch := groupUsers[i:end]
		if err := s.fanoutToUsers(ctx, batch, msg, requestID); err != nil {
			s.logger.Error("Failed to fanout to users batch",
				"request_id", requestID,
				"group_id", msg.GroupId,
				"batch_start", i,
				"batch_end", end,
				"error", err,
			)
			continue
		}
	}

	s.logger.Info("Large group fanout completed",
		"request_id", requestID,
		"group_id", msg.GroupId,
		"message_id", msg.MessageId,
		"user_count", len(groupUsers),
		"duration", time.Since(startTime),
	)

	return nil
}

func (s *TaskService) getGroupUsers(ctx context.Context, groupID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.RepoService.Timeout)
	defer cancel()

	req := &repopb.GetGroupUsersRequest{
		GroupId: groupID,
	}

	resp, err := s.repoClient.GetGroupUsers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get group users: %w", err)
	}

	return resp.UserIds, nil
}

func (s *TaskService) fanoutToUsers(ctx context.Context, userIDs []string, msg *taskpb.TaskMessage, requestID string) error {
	gatewayUsers := make(map[string][]string)

	for _, userID := range userIDs {
		gatewayID, err := s.getUserGatewayID(ctx, userID)
		if err != nil {
			s.logger.Error("Failed to get user gateway ID",
				"request_id", requestID,
				"user_id", userID,
				"error", err,
			)
			continue
		}

		if gatewayID == "" {
			s.logger.Info("User offline, scheduling push notification",
				"request_id", requestID,
				"user_id", userID,
			)
			if s.config.Task.PushNotificationEnabled {
				s.scheduleOfflinePush(ctx, userID, msg)
			}
			continue
		}

		gatewayUsers[gatewayID] = append(gatewayUsers[gatewayID], userID)
	}

	for gatewayID, users := range gatewayUsers {
		if err := s.sendToGateway(ctx, gatewayID, users, msg, requestID); err != nil {
			s.logger.Error("Failed to send to gateway",
				"request_id", requestID,
				"gateway_id", gatewayID,
				"user_count", len(users),
				"error", err,
			)
			continue
		}
	}

	return nil
}

func (s *TaskService) getUserGatewayID(ctx context.Context, userID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.RepoService.Timeout)
	defer cancel()

	req := &repopb.GetUserGatewayRequest{
		UserId: userID,
	}

	resp, err := s.repoClient.GetUserGateway(ctx, req)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to get user gateway: %w", err)
	}

	return resp.GatewayId, nil
}

func (s *TaskService) sendToGateway(ctx context.Context, gatewayID string, userIDs []string, msg *taskpb.TaskMessage, requestID string) error {
	downstreamMsg := &proto.DownstreamMessage{
		MessageID:      msg.MessageId,
		ConversationID: msg.GroupId,
		SenderID:       msg.SenderId,
		Type:           msg.Type,
		Content:        msg.Content,
		MediaURL:       msg.MediaUrl,
		MediaType:      msg.MediaType,
		MediaSize:      msg.MediaSize,
		SentAt:         msg.SentAt,
		TargetUsers:    userIDs,
	}

	if err := s.producer.ProduceGatewayMessage(ctx, gatewayID, downstreamMsg); err != nil {
		return fmt.Errorf("failed to produce gateway message: %w", err)
	}

	s.logger.Debug("Message sent to gateway",
		"request_id", requestID,
		"gateway_id", gatewayID,
		"message_id", msg.MessageId,
		"user_count", len(userIDs),
	)

	return nil
}

func (s *TaskService) scheduleOfflinePush(ctx context.Context, userID string, msg *taskpb.TaskMessage) {
	notification := &proto.PushNotification{
		NotificationID: uuid.New().String(),
		UserID:         userID,
		Title:          "New Message",
		Body:           msg.Content,
		Data: map[string]string{
			"message_id":      msg.MessageId,
			"conversation_id": msg.GroupId,
			"sender_id":       msg.SenderId,
			"type":            msg.Type,
		},
		CreatedAt: time.Now().Unix(),
	}

	if err := s.producer.ProducePushNotification(ctx, userID, notification); err != nil {
		s.logger.Error("Failed to produce push notification",
			"user_id", userID,
			"message_id", msg.MessageId,
			"error", err,
		)
	}
}

func (s *TaskService) handleOfflinePush(ctx context.Context, msg *taskpb.TaskMessage, requestID string) error {
	s.logger.Info("Handling offline push",
		"request_id", requestID,
		"message_id", msg.MessageId,
		"target_user_id", msg.TargetUserId,
	)

	if msg.TargetUserId == "" {
		return fmt.Errorf("target user ID is required for offline push")
	}

	message := &proto.Message{
		MessageID:      msg.MessageId,
		ConversationID: msg.GroupId,
		SenderID:       msg.SenderId,
		Type:           msg.Type,
		Content:        msg.Content,
		MediaURL:       msg.MediaUrl,
		MediaType:      msg.MediaType,
		MediaSize:      msg.MediaSize,
		SentAt:         msg.SentAt,
	}

	if err := s.pushService.SendPushNotification(ctx, msg.TargetUserId, message); err != nil {
		s.logger.Error("Failed to send push notification",
			"request_id", requestID,
			"user_id", msg.TargetUserId,
			"message_id", msg.MessageId,
			"error", err,
		)
		return fmt.Errorf("failed to send push notification: %w", err)
	}

	s.logger.Info("Offline push notification sent",
		"request_id", requestID,
		"user_id", msg.TargetUserId,
		"message_id", msg.MessageId,
	)

	return nil
}

func (s *TaskService) handleMessageRetry(ctx context.Context, msg *taskpb.TaskMessage, requestID string) error {
	s.logger.Info("Handling message retry",
		"request_id", requestID,
		"message_id", msg.MessageId,
		"retry_count", msg.RetryCount,
	)

	return fmt.Errorf("message retry not implemented yet")
}
