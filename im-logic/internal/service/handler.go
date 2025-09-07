package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/api/kafka"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/server/grpc"
	"github.com/ceyewan/gochat/im-logic/internal/server/kafka"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	config   *config.Config
	logger   clog.Logger
	client   *grpc.Client
	producer *kafka.Producer
	services *Services
}

// Services 业务服务集合
type Services struct {
	Auth         *AuthService
	Conversation *ConversationService
	Message      *MessageService
	Group        *GroupService
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(cfg *config.Config, client *grpc.Client, producer *kafka.Producer, services *Services) *MessageHandler {
	logger := clog.Module("message-handler")

	return &MessageHandler{
		config:   cfg,
		logger:   logger,
		client:   client,
		producer: producer,
		services: services,
	}
}

// HandleUpstreamMessage 处理上行消息
func (h *MessageHandler) HandleUpstreamMessage(ctx context.Context, upstream *kafka.UpstreamMessage) error {
	h.logger.Info("处理上行消息",
		clog.String("trace_id", upstream.TraceID),
		clog.String("user_id", upstream.UserID),
		clog.String("conversation_id", upstream.ConversationID),
		clog.Int("message_type", upstream.MessageType))

	// 验证消息参数
	if err := h.validateUpstreamMessage(upstream); err != nil {
		h.logger.Error("上行消息验证失败", clog.Err(err))
		return err
	}

	// 生成消息 ID 和序列号
	messageID := fmt.Sprintf("msg-%d", time.Now().UnixNano())

	// 获取序列号
	messageRepo := h.client.GetMessageServiceClient()
	seqReq := &repopb.GenerateSeqIDRequest{
		ConversationId: upstream.ConversationID,
	}
	seqResp, err := messageRepo.GenerateSeqID(ctx, seqReq)
	if err != nil {
		h.logger.Error("生成序列号失败", clog.Err(err))
		return fmt.Errorf("生成序列号失败: %w", err)
	}

	// 保存消息
	saveReq := &repopb.SaveMessageRequest{
		MessageId:      messageID,
		ConversationId: upstream.ConversationID,
		SenderId:       upstream.UserID,
		MessageType:    int32(upstream.MessageType),
		Content:        upstream.Content,
		SeqId:          seqResp.SeqId,
		ClientMsgId:    upstream.ClientMsgID,
		Extra:          upstream.Extra,
	}

	_, err = messageRepo.SaveMessage(ctx, saveReq)
	if err != nil {
		h.logger.Error("保存消息失败", clog.Err(err))
		return fmt.Errorf("保存消息失败: %w", err)
	}

	// 构建下行消息
	downstream := h.producer.CreateDownstreamMessage(upstream, messageID, seqResp.SeqId)

	// 发送下行消息到目标网关
	if err := h.routeDownstreamMessage(ctx, downstream); err != nil {
		h.logger.Error("路由下行消息失败", clog.Err(err))
		return fmt.Errorf("路由下行消息失败: %w", err)
	}

	// 如果是群组消息，创建扩散任务
	if h.isGroupConversation(upstream.ConversationID) {
		task := h.producer.CreateFanoutTask(
			upstream.ConversationID,
			messageID,
			upstream.UserID,
			[]string{upstream.UserID}, // 排除发送者
		)
		if err := h.producer.ProduceTaskMessage(ctx, task); err != nil {
			h.logger.Error("创建群组扩散任务失败", clog.Err(err))
			// 不影响主要流程，只记录错误
		}
	}

	h.logger.Info("上行消息处理成功",
		clog.String("trace_id", upstream.TraceID),
		clog.String("message_id", messageID))

	return nil
}

// HandleDownstreamMessage 处理下行消息
func (h *MessageHandler) HandleDownstreamMessage(ctx context.Context, downstream *kafka.DownstreamMessage) error {
	h.logger.Info("处理下行消息",
		clog.String("trace_id", downstream.TraceID),
		clog.String("message_id", downstream.MessageID),
		clog.String("target_user_id", downstream.TargetUserID))

	// 下行消息通常由网关处理，这里主要做日志记录
	// 可以在这里实现一些额外的业务逻辑，如消息审计、统计等

	h.logger.Debug("下行消息处理完成",
		clog.String("trace_id", downstream.TraceID))

	return nil
}

// HandleTaskMessage 处理任务消息
func (h *MessageHandler) HandleTaskMessage(ctx context.Context, task *kafka.TaskMessage) error {
	h.logger.Info("处理任务消息",
		clog.String("trace_id", task.TraceID),
		clog.String("task_id", task.TaskID),
		clog.String("task_type", string(task.TaskType)))

	switch task.TaskType {
	case kafka.TaskTypeFanout:
		return h.handleFanoutTask(ctx, task)
	case kafka.TaskTypePush:
		return h.handlePushTask(ctx, task)
	case kafka.TaskTypeAudit:
		return h.handleAuditTask(ctx, task)
	case kafka.TaskTypeIndex:
		return h.handleIndexTask(ctx, task)
	default:
		h.logger.Warn("未知的任务类型", clog.String("task_type", string(task.TaskType)))
		return fmt.Errorf("未知的任务类型: %s", task.TaskType)
	}
}

// handleFanoutTask 处理大群消息扩散任务
func (h *MessageHandler) handleFanoutTask(ctx context.Context, task *kafka.TaskMessage) error {
	var taskData kafka.FanoutTaskData
	if err := json.Unmarshal(task.Data, &taskData); err != nil {
		h.logger.Error("解析扩散任务数据失败", clog.Err(err))
		return fmt.Errorf("解析扩散任务数据失败: %w", err)
	}

	h.logger.Info("处理群组消息扩散任务",
		clog.String("group_id", taskData.GroupID),
		clog.String("message_id", taskData.MessageID))

	// 获取群组成员
	groupRepo := h.client.GetGroupServiceClient()
	membersReq := &repopb.GetGroupMembersRequest{
		GroupId: taskData.GroupID,
		Offset:  0,
		Limit:   1000,
	}
	membersResp, err := groupRepo.GetGroupMembers(ctx, membersReq)
	if err != nil {
		h.logger.Error("获取群组成员失败", clog.Err(err))
		return fmt.Errorf("获取群组成员失败: %w", err)
	}

	// 获取消息详情
	messageRepo := h.client.GetMessageServiceClient()
	messageReq := &repopb.GetMessageRequest{
		MessageId: taskData.MessageID,
	}
	messageResp, err := messageRepo.GetMessage(ctx, messageReq)
	if err != nil {
		h.logger.Error("获取消息详情失败", clog.Err(err))
		return fmt.Errorf("获取消息详情失败: %w", err)
	}

	// 为每个成员创建下行消息
	for _, member := range membersResp.Members {
		// 跳过排除的用户
		if h.shouldExcludeUser(member.UserId, taskData.ExcludeUserIDs) {
			continue
		}

		// 获取用户所在网关
		onlineStatusRepo := h.client.GetOnlineStatusServiceClient()
		statusReq := &repopb.GetUserOnlineStatusRequest{
			UserId: member.UserId,
		}
		statusResp, err := onlineStatusRepo.GetUserOnlineStatus(ctx, statusReq)
		if err != nil {
			h.logger.Error("获取用户在线状态失败", clog.Err(err))
			continue
		}

		// 如果用户在线，发送下行消息
		if statusResp.Status.IsOnline && statusResp.Status.GatewayId != "" {
			downstream := &kafka.DownstreamMessage{
				TraceID:        task.TraceID,
				TargetUserID:   member.UserId,
				MessageID:      messageResp.Message.Id,
				ConversationID: messageResp.Message.ConversationId,
				SenderID:       messageResp.Message.SenderId,
				MessageType:    int(messageResp.Message.MessageType),
				Content:        messageResp.Message.Content,
				SeqID:          messageResp.Message.SeqId,
				Timestamp:      messageResp.Message.CreatedAt,
				Headers:        task.Headers,
				Extra:          messageResp.Message.Extra,
			}

			topic := h.config.GetDownstreamTopic(statusResp.Status.GatewayId)
			if err := h.producer.ProduceDownstreamMessage(ctx, statusResp.Status.GatewayId, downstream); err != nil {
				h.logger.Error("发送下行消息失败",
					clog.String("user_id", member.UserId),
					clog.String("gateway_id", statusResp.Status.GatewayId),
					clog.Err(err))
			}
		}
	}

	h.logger.Info("群组消息扩散任务处理完成",
		clog.String("group_id", taskData.GroupID),
		clog.String("message_id", taskData.MessageID))

	return nil
}

// handlePushTask 处理离线推送任务
func (h *MessageHandler) handlePushTask(ctx context.Context, task *kafka.TaskMessage) error {
	var taskData kafka.PushTaskData
	if err := json.Unmarshal(task.Data, &taskData); err != nil {
		h.logger.Error("解析推送任务数据失败", clog.Err(err))
		return fmt.Errorf("解析推送任务数据失败: %w", err)
	}

	h.logger.Info("处理离线推送任务",
		clog.String("title", taskData.Title),
		clog.Int("user_count", len(taskData.UserIDs)))

	// TODO: 实现离线推送逻辑
	// 这里可以集成第三方推送服务，如 APNs、Firebase Cloud Messaging 等
	// 或者发送邮件、短信等通知

	h.logger.Info("离线推送任务处理完成",
		clog.String("title", taskData.Title))

	return nil
}

// handleAuditTask 处理内容审核任务
func (h *MessageHandler) handleAuditTask(ctx context.Context, task *kafka.TaskMessage) error {
	h.logger.Info("处理内容审核任务", clog.String("task_id", task.TaskID))

	// TODO: 实现内容审核逻辑
	// 可以集成第三方内容审核服务，或者使用本地敏感词过滤

	h.logger.Info("内容审核任务处理完成", clog.String("task_id", task.TaskID))
	return nil
}

// handleIndexTask 处理索引任务
func (h *MessageHandler) handleIndexTask(ctx context.Context, task *kafka.TaskMessage) error {
	h.logger.Info("处理索引任务", clog.String("task_id", task.TaskID))

	// TODO: 实现索引更新逻辑
	// 可以集成 Elasticsearch 等搜索引擎服务

	h.logger.Info("索引任务处理完成", clog.String("task_id", task.TaskID))
	return nil
}

// validateUpstreamMessage 验证上行消息
func (h *MessageHandler) validateUpstreamMessage(upstream *kafka.UpstreamMessage) error {
	if upstream.UserID == "" {
		return fmt.Errorf("用户 ID 不能为空")
	}
	if upstream.ConversationID == "" {
		return fmt.Errorf("会话 ID 不能为空")
	}
	if upstream.Content == "" {
		return fmt.Errorf("消息内容不能为空")
	}
	if upstream.MessageType <= 0 || upstream.MessageType > 6 {
		return fmt.Errorf("无效的消息类型")
	}
	return nil
}

// routeDownstreamMessage 路由下行消息
func (h *MessageHandler) routeDownstreamMessage(ctx context.Context, downstream *kafka.DownstreamMessage) error {
	// 根据会话类型决定路由策略
	if h.isGroupConversation(downstream.ConversationID) {
		// 群组消息通过扩散任务处理
		return nil
	}

	// 单聊消息直接发送给目标用户
	// 这里需要根据会话 ID 找到目标用户
	// 简化处理，假设会话 ID 格式为 "user1-user2"
	targetUserID := h.extractTargetUserID(downstream.ConversationID, downstream.SenderID)
	if targetUserID != "" {
		// 获取用户在线状态
		onlineStatusRepo := h.client.GetOnlineStatusServiceClient()
		statusReq := &repopb.GetUserOnlineStatusRequest{
			UserId: targetUserID,
		}
		statusResp, err := onlineStatusRepo.GetUserOnlineStatus(ctx, statusReq)
		if err != nil {
			h.logger.Error("获取用户在线状态失败", clog.Err(err))
			return err
		}

		// 如果用户在线，发送下行消息
		if statusResp.Status.IsOnline && statusResp.Status.GatewayId != "" {
			downstream.TargetUserID = targetUserID
			return h.producer.ProduceDownstreamMessage(ctx, statusResp.Status.GatewayId, downstream)
		}
	}

	return nil
}

// isGroupConversation 判断是否为群组会话
func (h *MessageHandler) isGroupConversation(conversationID string) bool {
	// 简化处理，假设群组会话 ID 以 "group-" 开头
	return len(conversationID) > 6 && conversationID[:6] == "group-"
}

// extractTargetUserID 从会话 ID 中提取目标用户 ID
func (h *MessageHandler) extractTargetUserID(conversationID, senderID string) string {
	// 简化处理，假设单聊会话 ID 格式为 "user1-user2"
	// 实际需要根据具体的会话 ID 生成规则来解析
	// 这里返回空，表示需要更复杂的逻辑来处理
	return ""
}

// shouldExcludeUser 判断是否应该排除用户
func (h *MessageHandler) shouldExcludeUser(userID string, excludeUserIDs []string) bool {
	for _, excludeID := range excludeUserIDs {
		if userID == excludeID {
			return true
		}
	}
	return false
}
