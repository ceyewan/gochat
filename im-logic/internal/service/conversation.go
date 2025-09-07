package service

import (
	"context"
	"time"

	logicpb "github.com/ceyewan/gochat/api/gen/im_logic/v1"
	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/server/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConversationService 会话服务
type ConversationService struct {
	config *config.Config
	logger clog.Logger
	client *grpc.Client
}

// NewConversationService 创建会话服务
func NewConversationService(cfg *config.Config, client *grpc.Client) *ConversationService {
	logger := clog.Module("conversation-service")

	return &ConversationService{
		config: cfg,
		logger: logger,
		client: client,
	}
}

// GetConversations 获取用户会话列表
func (s *ConversationService) GetConversations(ctx context.Context, req *logicpb.GetConversationsRequest) (*logicpb.GetConversationsResponse, error) {
	s.logger.Info("获取用户会话列表", clog.String("user_id", req.UserId))

	// 验证参数
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 不能为空")
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 获取会话 ID 列表
	conversationRepo := s.client.GetConversationServiceClient()
	offset := (req.Page - 1) * req.PageSize
	convReq := &repopb.GetUserConversationsRequest{
		UserId:     req.UserId,
		Offset:     int32(offset),
		Limit:      int32(req.PageSize),
		TypeFilter: int32(req.Type),
	}

	convResp, err := conversationRepo.GetUserConversations(ctx, convReq)
	if err != nil {
		s.logger.Error("获取用户会话列表失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取会话列表失败")
	}

	// 如果没有会话，返回空列表
	if len(convResp.ConversationIds) == 0 {
		return &logicpb.GetConversationsResponse{
			Conversations: []*logicpb.Conversation{},
			Total:         0,
			Page:          req.Page,
			PageSize:      req.PageSize,
		}, nil
	}

	// 批量获取会话的未读消息数
	unreadReq := &repopb.BatchGetUnreadCountsRequest{
		UserId:          req.UserId,
		ConversationIds: convResp.ConversationIds,
	}
	unreadResp, err := conversationRepo.BatchGetUnreadCounts(ctx, unreadReq)
	if err != nil {
		s.logger.Error("批量获取未读消息数失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取未读消息数失败")
	}

	// 获取最新消息
	messageRepo := s.client.GetMessageServiceClient()
	latestMsgReq := &repopb.GetLatestMessagesRequest{
		ConversationIds:      convResp.ConversationIds,
		LimitPerConversation: 1,
	}
	latestMsgResp, err := messageRepo.GetLatestMessages(ctx, latestMsgReq)
	if err != nil {
		s.logger.Error("获取最新消息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取最新消息失败")
	}

	// 构建会话列表
	conversations := make([]*logicpb.Conversation, 0, len(convResp.ConversationIds))
	for _, convID := range convResp.ConversationIds {
		// 获取会话信息
		conversation, err := s.buildConversation(ctx, convID, req.UserId, unreadResp, latestMsgResp)
		if err != nil {
			s.logger.Error("构建会话信息失败", clog.String("conversation_id", convID), clog.Err(err))
			continue
		}

		conversations = append(conversations, conversation)
	}

	s.logger.Info("获取用户会话列表成功", clog.String("user_id", req.UserId), clog.Int("count", len(conversations)))

	return &logicpb.GetConversationsResponse{
		Conversations: conversations,
		Total:         int(convResp.Total),
		Page:          req.Page,
		PageSize:      req.PageSize,
	}, nil
}

// GetConversation 获取单个会话详情
func (s *ConversationService) GetConversation(ctx context.Context, req *logicpb.GetConversationRequest) (*logicpb.GetConversationResponse, error) {
	s.logger.Info("获取会话详情", clog.String("user_id", req.UserId), clog.String("conversation_id", req.ConversationId))

	// 验证参数
	if req.UserId == "" || req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 和会话 ID 不能为空")
	}

	// 获取未读消息数
	conversationRepo := s.client.GetConversationServiceClient()
	unreadReq := &repopb.GetUnreadCountRequest{
		UserId:         req.UserId,
		ConversationId: req.ConversationId,
	}
	unreadResp, err := conversationRepo.GetUnreadCount(ctx, unreadReq)
	if err != nil {
		s.logger.Error("获取未读消息数失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取未读消息数失败")
	}

	// 获取最新消息
	messageRepo := s.client.GetMessageServiceClient()
	latestMsgReq := &repopb.GetLatestMessagesRequest{
		ConversationIds:      []string{req.ConversationId},
		LimitPerConversation: 1,
	}
	latestMsgResp, err := messageRepo.GetLatestMessages(ctx, latestMsgReq)
	if err != nil {
		s.logger.Error("获取最新消息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取最新消息失败")
	}

	// 构建会话信息
	conversation, err := s.buildConversation(ctx, req.ConversationId, req.UserId,
		&repopb.BatchGetUnreadCountsResponse{UnreadCounts: map[string]int64{req.ConversationId: unreadResp.UnreadCount}},
		latestMsgResp)
	if err != nil {
		s.logger.Error("构建会话信息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "构建会话信息失败")
	}

	s.logger.Info("获取会话详情成功", clog.String("user_id", req.UserId), clog.String("conversation_id", req.ConversationId))

	return &logicpb.GetConversationResponse{
		Conversation: conversation,
	}, nil
}

// GetMessages 获取会话历史消息
func (s *ConversationService) GetMessages(ctx context.Context, req *logicpb.GetMessagesRequest) (*logicpb.GetMessagesResponse, error) {
	s.logger.Info("获取会话历史消息",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId))

	// 验证参数
	if req.UserId == "" || req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 和会话 ID 不能为空")
	}

	// 设置默认参数
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// 获取消息列表
	messageRepo := s.client.GetMessageServiceClient()
	msgReq := &repopb.GetConversationMessagesRequest{
		ConversationId: req.ConversationId,
		StartSeqId:     req.StartSeqId,
		EndSeqId:       req.EndSeqId,
		Limit:          req.Limit,
		Ascending:      req.Ascending,
	}

	msgResp, err := messageRepo.GetConversationMessages(ctx, msgReq)
	if err != nil {
		s.logger.Error("获取消息列表失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取消息列表失败")
	}

	// 转换消息格式
	messages := make([]*logicpb.Message, 0, len(msgResp.Messages))
	for _, msg := range msgResp.Messages {
		logicMsg := s.convertMessage(msg)
		messages = append(messages, logicMsg)
	}

	s.logger.Info("获取会话历史消息成功",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId),
		clog.Int("count", len(messages)))

	return &logicpb.GetMessagesResponse{
		Messages:  messages,
		HasMore:   msgResp.HasMore,
		NextSeqId: msgResp.NextSeqId,
	}, nil
}

// MarkAsRead 标记消息已读
func (s *ConversationService) MarkAsRead(ctx context.Context, req *logicpb.MarkAsReadRequest) (*logicpb.MarkAsReadResponse, error) {
	s.logger.Info("标记消息已读",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId),
		clog.Int64("seq_id", req.SeqId))

	// 验证参数
	if req.UserId == "" || req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 和会话 ID 不能为空")
	}

	// 更新已读位置
	conversationRepo := s.client.GetConversationServiceClient()
	readReq := &repopb.UpdateReadPointerRequest{
		UserId:         req.UserId,
		ConversationId: req.ConversationId,
		SeqId:          req.SeqId,
	}

	readResp, err := conversationRepo.UpdateReadPointer(ctx, readReq)
	if err != nil {
		s.logger.Error("更新已读位置失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "更新已读位置失败")
	}

	s.logger.Info("标记消息已读成功",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId))

	return &logicpb.MarkAsReadResponse{
		Success:     true,
		UnreadCount: readResp.UnreadCount,
	}, nil
}

// GetUnreadCount 获取未读消息数
func (s *ConversationService) GetUnreadCount(ctx context.Context, req *logicpb.GetUnreadCountRequest) (*logicpb.GetUnreadCountResponse, error) {
	s.logger.Info("获取未读消息数",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId))

	// 验证参数
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 不能为空")
	}

	conversationRepo := s.client.GetConversationServiceClient()

	// 如果指定了会话 ID，获取单个会话的未读数
	if req.ConversationId != "" {
		unreadReq := &repopb.GetUnreadCountRequest{
			UserId:         req.UserId,
			ConversationId: req.ConversationId,
		}
		unreadResp, err := conversationRepo.GetUnreadCount(ctx, unreadReq)
		if err != nil {
			s.logger.Error("获取未读消息数失败", clog.Err(err))
			return nil, status.Error(codes.Internal, "获取未读消息数失败")
		}

		return &logicpb.GetUnreadCountResponse{
			UnreadCount: unreadResp.UnreadCount,
		}, nil
	}

	// 获取所有会话的未读数
	// 首先获取用户的所有会话
	convReq := &repopb.GetUserConversationsRequest{
		UserId: req.UserId,
		Offset: 0,
		Limit:  1000, // 获取足够多的会话
	}
	convResp, err := conversationRepo.GetUserConversations(ctx, convReq)
	if err != nil {
		s.logger.Error("获取用户会话列表失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取会话列表失败")
	}

	// 批量获取未读数
	if len(convResp.ConversationIds) == 0 {
		return &logicpb.GetUnreadCountResponse{
			UnreadCount:        0,
			ConversationCounts: []*logicpb.ConversationUnreadCount{},
		}, nil
	}

	unreadReq := &repopb.BatchGetUnreadCountsRequest{
		UserId:          req.UserId,
		ConversationIds: convResp.ConversationIds,
	}
	unreadResp, err := conversationRepo.BatchGetUnreadCounts(ctx, unreadReq)
	if err != nil {
		s.logger.Error("批量获取未读消息数失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取未读消息数失败")
	}

	// 构建会话未读数列表
	conversationCounts := make([]*logicpb.ConversationUnreadCount, 0, len(unreadResp.UnreadCounts))
	var totalUnread int64
	for convID, count := range unreadResp.UnreadCounts {
		if count > 0 {
			conversationCounts = append(conversationCounts, &logicpb.ConversationUnreadCount{
				ConversationId: convID,
				UnreadCount:    count,
			})
			totalUnread += count
		}
	}

	s.logger.Info("获取未读消息数成功",
		clog.String("user_id", req.UserId),
		clog.Int64("total_unread", totalUnread))

	return &logicpb.GetUnreadCountResponse{
		UnreadCount:        totalUnread,
		ConversationCounts: conversationCounts,
	}, nil
}

// buildConversation 构建会话信息
func (s *ConversationService) buildConversation(ctx context.Context, conversationID, userID string,
	unreadResp *repopb.BatchGetUnreadCountsResponse, latestMsgResp *repopb.GetLatestMessagesResponse) (*logicpb.Conversation, error) {

	// 获取未读数
	unreadCount := unreadResp.UnreadCounts[conversationID]

	// 获取最新消息
	var lastMessage *logicpb.Message
	if messages, exists := latestMsgResp.ConversationMessages[conversationID]; exists && len(messages.Messages) > 0 {
		lastMessage = s.convertMessage(messages.Messages[0])
	}

	// TODO: 获取会话详细信息（名称、头像等）
	// 这里需要根据会话类型获取不同的信息
	var conversationType logicpb.ConversationType
	var conversationName string
	var avatarURL string
	var memberCount int32
	var userRole int32

	// 根据会话 ID 判断类型（这里简化处理，实际需要更复杂的逻辑）
	if len(conversationID) == 36 { // UUID 格式，假设为单聊
		conversationType = logicpb.ConversationType_CONVERSATION_TYPE_SINGLE
		conversationName = "私聊"
		avatarURL = ""
		memberCount = 2
		userRole = 1
	} else {
		conversationType = logicpb.ConversationType_CONVERSATION_TYPE_GROUP
		conversationName = "群聊"
		avatarURL = ""
		memberCount = 10
		userRole = 1
	}

	return &logicpb.Conversation{
		Id:          conversationID,
		Type:        conversationType,
		Name:        conversationName,
		AvatarUrl:   avatarURL,
		LastMessage: lastMessage,
		UnreadCount: unreadCount,
		UpdatedAt:   time.Now().Unix(),
		MemberCount: memberCount,
		UserRole:    userRole,
	}, nil
}

// convertMessage 转换消息格式
func (s *ConversationService) convertMessage(msg *repopb.Message) *logicpb.Message {
	return &logicpb.Message{
		Id:             msg.Id,
		ConversationId: msg.ConversationId,
		SenderId:       msg.SenderId,
		Type:           logicpb.MessageType(msg.MessageType),
		Content:        msg.Content,
		SeqId:          msg.SeqId,
		CreatedAt:      msg.CreatedAt,
		Extra:          msg.Extra,
	}
}
