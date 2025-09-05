package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/model"
	"github.com/ceyewan/gochat/im-repo/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MessageService 消息服务实现
type MessageService struct {
	repopb.UnimplementedMessageServiceServer
	messageRepo      *repository.MessageRepository
	conversationRepo *repository.ConversationRepository
	logger           clog.Logger
}

// NewMessageService 创建消息服务
func NewMessageService(messageRepo *repository.MessageRepository, conversationRepo *repository.ConversationRepository) *MessageService {
	return &MessageService{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		logger:           clog.Module("message-service"),
	}
}

// SaveMessage 保存消息
func (s *MessageService) SaveMessage(ctx context.Context, req *repopb.SaveMessageRequest) (*repopb.SaveMessageResponse, error) {
	s.logger.Info("保存消息请求",
		clog.String("message_id", req.MessageId),
		clog.String("conversation_id", req.ConversationId))

	// 参数验证
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "消息ID不能为空")
	}
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "会话ID不能为空")
	}
	if req.SenderId == "" {
		return nil, status.Error(codes.InvalidArgument, "发送者ID不能为空")
	}

	// 转换消息ID
	messageID, err := strconv.ParseUint(req.MessageId, 10, 64)
	if err != nil {
		s.logger.Error("消息ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "消息ID格式错误")
	}

	// 转换发送者ID
	senderID, err := strconv.ParseUint(req.SenderId, 10, 64)
	if err != nil {
		s.logger.Error("发送者ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "发送者ID格式错误")
	}

	// 创建消息模型
	message := &model.Message{
		ID:             messageID,
		ConversationID: req.ConversationId,
		SenderID:       senderID,
		MessageType:    int(req.MessageType),
		Content:        req.Content,
		SeqID:          uint64(req.SeqId),
		Extra:          req.Extra,
	}

	// 保存消息
	err = s.messageRepo.SaveMessage(ctx, message)
	if err != nil {
		s.logger.Error("保存消息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "保存消息失败")
	}

	// 构造响应
	resp := &repopb.SaveMessageResponse{
		Message: s.modelToProto(message),
	}

	s.logger.Info("消息保存成功", clog.String("message_id", req.MessageId))
	return resp, nil
}

// GetMessage 获取消息
func (s *MessageService) GetMessage(ctx context.Context, req *repopb.GetMessageRequest) (*repopb.GetMessageResponse, error) {
	s.logger.Debug("获取消息请求", clog.String("message_id", req.MessageId))

	// 参数验证
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "消息ID不能为空")
	}

	// 转换消息ID
	messageID, err := strconv.ParseUint(req.MessageId, 10, 64)
	if err != nil {
		s.logger.Error("消息ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "消息ID格式错误")
	}

	// 获取消息
	message, err := s.messageRepo.GetMessage(ctx, messageID)
	if err != nil {
		s.logger.Error("获取消息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取消息失败")
	}

	if message == nil {
		return nil, status.Error(codes.NotFound, "消息不存在")
	}

	// 构造响应
	resp := &repopb.GetMessageResponse{
		Message: s.modelToProto(message),
	}

	return resp, nil
}

// GetConversationMessages 获取会话消息列表
func (s *MessageService) GetConversationMessages(ctx context.Context, req *repopb.GetConversationMessagesRequest) (*repopb.GetConversationMessagesResponse, error) {
	s.logger.Debug("获取会话消息列表请求",
		clog.String("conversation_id", req.ConversationId),
		clog.Int32("limit", req.Limit))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "会话ID不能为空")
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20 // 默认限制
	}
	if limit > 100 {
		limit = 100 // 最大限制
	}

	// 获取消息列表
	messages, hasMore, nextSeqID, err := s.messageRepo.GetConversationMessages(
		ctx,
		req.ConversationId,
		req.StartSeqId,
		req.EndSeqId,
		limit,
		req.Ascending,
	)
	if err != nil {
		s.logger.Error("获取会话消息列表失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取会话消息列表失败")
	}

	// 转换为 protobuf 格式
	protoMessages := make([]*repopb.Message, len(messages))
	for i, message := range messages {
		protoMessages[i] = s.modelToProto(message)
	}

	// 构造响应
	resp := &repopb.GetConversationMessagesResponse{
		Messages:  protoMessages,
		HasMore:   hasMore,
		NextSeqId: nextSeqID,
	}

	s.logger.Debug("获取会话消息列表成功",
		clog.String("conversation_id", req.ConversationId),
		clog.Int("count", len(messages)))

	return resp, nil
}

// GenerateSeqID 生成序列号
func (s *MessageService) GenerateSeqID(ctx context.Context, req *repopb.GenerateSeqIDRequest) (*repopb.GenerateSeqIDResponse, error) {
	s.logger.Debug("生成序列号请求", clog.String("conversation_id", req.ConversationId))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "会话ID不能为空")
	}

	// 生成序列号
	seqID, err := s.messageRepo.GenerateSeqID(ctx, req.ConversationId)
	if err != nil {
		s.logger.Error("生成序列号失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "生成序列号失败")
	}

	// 构造响应
	resp := &repopb.GenerateSeqIDResponse{
		SeqId: int64(seqID),
	}

	s.logger.Debug("序列号生成成功",
		clog.String("conversation_id", req.ConversationId),
		clog.Int64("seq_id", int64(seqID)))

	return resp, nil
}

// CheckMessageIdempotency 检查消息幂等性
func (s *MessageService) CheckMessageIdempotency(ctx context.Context, req *repopb.CheckMessageIdempotencyRequest) (*repopb.CheckMessageIdempotencyResponse, error) {
	s.logger.Debug("检查消息幂等性请求", clog.String("client_msg_id", req.ClientMsgId))

	// 参数验证
	if req.ClientMsgId == "" {
		return nil, status.Error(codes.InvalidArgument, "客户端消息ID不能为空")
	}

	ttl := time.Duration(req.TtlSeconds) * time.Second
	if req.TtlSeconds <= 0 {
		ttl = 60 * time.Second // 默认60秒
	}

	// 检查幂等性
	isNew, existingMsgID, err := s.messageRepo.CheckMessageIdempotency(ctx, req.ClientMsgId, ttl)
	if err != nil {
		s.logger.Error("检查消息幂等性失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "检查消息幂等性失败")
	}

	// 构造响应
	resp := &repopb.CheckMessageIdempotencyResponse{
		Exists:            !isNew,
		ExistingMessageId: existingMsgID,
	}

	s.logger.Debug("消息幂等性检查完成",
		clog.String("client_msg_id", req.ClientMsgId),
		clog.Bool("exists", !isNew))

	return resp, nil
}

// GetLatestMessages 获取最新消息
func (s *MessageService) GetLatestMessages(ctx context.Context, req *repopb.GetLatestMessagesRequest) (*repopb.GetLatestMessagesResponse, error) {
	s.logger.Debug("获取最新消息请求", clog.Int("conversation_count", len(req.ConversationIds)))

	// 参数验证
	if len(req.ConversationIds) == 0 {
		return &repopb.GetLatestMessagesResponse{
			ConversationMessages: make(map[string]*repopb.ConversationMessages),
		}, nil
	}

	limitPerConversation := int(req.LimitPerConversation)
	if limitPerConversation <= 0 {
		limitPerConversation = 1 // 默认每个会话1条
	}

	// 获取最新消息
	messagesMap, err := s.messageRepo.GetLatestMessages(ctx, req.ConversationIds, limitPerConversation)
	if err != nil {
		s.logger.Error("获取最新消息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取最新消息失败")
	}

	// 转换为 protobuf 格式
	protoMessagesMap := make(map[string]*repopb.ConversationMessages)
	for conversationID, messages := range messagesMap {
		protoMessages := make([]*repopb.Message, len(messages))
		for i, message := range messages {
			protoMessages[i] = s.modelToProto(message)
		}
		protoMessagesMap[conversationID] = &repopb.ConversationMessages{
			Messages: protoMessages,
		}
	}

	// 构造响应
	resp := &repopb.GetLatestMessagesResponse{
		ConversationMessages: protoMessagesMap,
	}

	s.logger.Debug("获取最新消息成功",
		clog.Int("requested_conversations", len(req.ConversationIds)),
		clog.Int("successful_conversations", len(protoMessagesMap)))

	return resp, nil
}

// DeleteMessage 删除消息
func (s *MessageService) DeleteMessage(ctx context.Context, req *repopb.DeleteMessageRequest) (*repopb.DeleteMessageResponse, error) {
	s.logger.Info("删除消息请求",
		clog.String("message_id", req.MessageId),
		clog.String("operator_id", req.OperatorId))

	// 参数验证
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "消息ID不能为空")
	}
	if req.OperatorId == "" {
		return nil, status.Error(codes.InvalidArgument, "操作者ID不能为空")
	}

	// 转换ID
	messageID, err := strconv.ParseUint(req.MessageId, 10, 64)
	if err != nil {
		s.logger.Error("消息ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "消息ID格式错误")
	}

	operatorID, err := strconv.ParseUint(req.OperatorId, 10, 64)
	if err != nil {
		s.logger.Error("操作者ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "操作者ID格式错误")
	}

	// 删除消息
	err = s.messageRepo.DeleteMessage(ctx, messageID, operatorID, req.Reason)
	if err != nil {
		s.logger.Error("删除消息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "删除消息失败")
	}

	// 构造响应
	resp := &repopb.DeleteMessageResponse{
		Success: true,
	}

	s.logger.Info("消息删除成功", clog.String("message_id", req.MessageId))
	return resp, nil
}

// modelToProto 将模型转换为 protobuf 格式
func (s *MessageService) modelToProto(message *model.Message) *repopb.Message {
	return &repopb.Message{
		Id:             fmt.Sprintf("%d", message.ID),
		ConversationId: message.ConversationID,
		SenderId:       fmt.Sprintf("%d", message.SenderID),
		MessageType:    int32(message.MessageType),
		Content:        message.Content,
		SeqId:          int64(message.SeqID),
		CreatedAt:      message.CreatedAt.Unix(),
		UpdatedAt:      message.UpdatedAt.Unix(),
		Deleted:        message.Deleted,
		Extra:          message.Extra,
	}
}
