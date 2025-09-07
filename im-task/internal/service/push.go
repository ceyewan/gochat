package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-task/internal/config"
	"github.com/ceyewan/gochat/internal/proto"
	"github.com/ceyewan/gochat/pkg/log"
	"github.com/google/uuid"
)

type PushService struct {
	config *config.Config
	logger *log.Logger
}

func NewPushService(cfg *config.Config, logger *log.Logger) *PushService {
	return &PushService{
		config: cfg,
		logger: logger,
	}
}

func (s *PushService) SendPushNotification(ctx context.Context, userID string, message *proto.Message) error {
	startTime := time.Now()
	requestID, _ := ctx.Value("request_id").(string)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	s.logger.Info("Sending push notification",
		"request_id", requestID,
		"user_id", userID,
		"message_id", message.MessageID,
		"conversation_id", message.ConversationID,
	)

	if !s.config.Task.PushNotificationEnabled {
		s.logger.Debug("Push notifications disabled",
			"request_id", requestID,
			"user_id", userID,
		)
		return nil
	}

	notification := s.buildNotification(userID, message)

	if err := s.sendNotification(ctx, notification, requestID); err != nil {
		s.logger.Error("Failed to send push notification",
			"request_id", requestID,
			"user_id", userID,
			"message_id", message.MessageID,
			"error", err,
		)
		return fmt.Errorf("failed to send push notification: %w", err)
	}

	s.logger.Info("Push notification sent successfully",
		"request_id", requestID,
		"user_id", userID,
		"message_id", message.MessageID,
		"duration", time.Since(startTime),
	)

	return nil
}

func (s *PushService) buildNotification(userID string, message *proto.Message) *proto.PushNotification {
	title := "New Message"
	body := message.Content

	if message.Type == "image" {
		title = "New Image"
		body = "Sent you an image"
	} else if message.Type == "file" {
		title = "New File"
		body = "Sent you a file"
	} else if message.Type == "voice" {
		title = "New Voice Message"
		body = "Sent you a voice message"
	}

	return &proto.PushNotification{
		NotificationID: uuid.New().String(),
		UserID:         userID,
		Title:          title,
		Body:           body,
		Data: map[string]string{
			"message_id":      message.MessageID,
			"conversation_id": message.ConversationID,
			"sender_id":       message.SenderID,
			"type":            message.Type,
			"media_url":       message.MediaURL,
			"media_type":      message.MediaType,
		},
		CreatedAt: time.Now().Unix(),
	}
}

func (s *PushService) sendNotification(ctx context.Context, notification *proto.PushNotification, requestID string) error {
	s.logger.Debug("Preparing push notification",
		"request_id", requestID,
		"notification_id", notification.NotificationID,
		"user_id", notification.UserID,
		"title", notification.Title,
		"body", notification.Body,
	)

	return nil
}
