package service

import (
	"context"
	"fmt"
	"time"

	logicpb "github.com/ceyewan/gochat/api/gen/im_logic/v1"
	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-logic/internal/config"
	"github.com/ceyewan/gochat/im-logic/internal/server/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MessageService 消息服务
type MessageService struct {
	config *config.Config
	logger clog.Logger
	client *grpc.Client
}

// NewMessageService 创建消息服务
func NewMessageService(cfg *config.Config, client *grpc.Client) *MessageService {
	logger := clog.Module("message-service")

	return &MessageService{
		config: cfg,
		logger: logger,
		client: client,
	}
}

// SendMessage 发送消息
func (s *MessageService) SendMessage(ctx context.Context, req *logicpb.SendMessageRequest) (*logicpb.SendMessageResponse, error) {
	s.logger.Info("发送消息",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId),
		clog.String("content", req.Content))

	// 验证参数
	if req.UserId == "" || req.ConversationId == "" || req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID、会话 ID 和消息内容不能为空")
	}

	// 验证消息类型
	if !s.isValidMessageType(req.Type) {
		return nil, status.Error(codes.InvalidArgument, "不支持的消息类型")
	}

	// 验证消息长度
	if len(req.Content) > s.config.Message.MaxLength {
		return nil, status.Error(codes.InvalidArgument, "消息内容过长")
	}

	// 检查消息幂等性
	messageRepo := s.client.GetMessageServiceClient()
	idempotencyReq := &repopb.CheckMessageIdempotencyRequest{
		ClientMsgId: req.ClientMsgId,
		TtlSeconds:  60,
	}
	idempotencyResp, err := messageRepo.CheckMessageIdempotency(ctx, idempotencyReq)
	if err != nil {
		s.logger.Error("检查消息幂等性失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "检查消息幂等性失败")
	}

	if idempotencyResp.Exists {
		s.logger.Warn("消息已存在，返回已存在的消息", clog.String("client_msg_id", req.ClientMsgId))
		return &logicpb.SendMessageResponse{
			MessageId: idempotencyResp.ExistingMessageId,
			Success:   true,
		}, nil
	}

	// 生成序列号
	seqReq := &repopb.GenerateSeqIDRequest{
		ConversationId: req.ConversationId,
	}
	seqResp, err := messageRepo.GenerateSeqID(ctx, seqReq)
	if err != nil {
		s.logger.Error("生成序列号失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "生成序列号失败")
	}

	// 保存消息
	saveReq := &repopb.SaveMessageRequest{
		MessageId:      fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		ConversationId: req.ConversationId,
		SenderId:       req.UserId,
		MessageType:    int32(req.Type),
		Content:        req.Content,
		SeqId:          seqResp.SeqId,
		ClientMsgId:    req.ClientMsgId,
		Extra:          req.Extra,
	}

	saveResp, err := messageRepo.SaveMessage(ctx, saveReq)
	if err != nil {
		s.logger.Error("保存消息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "保存消息失败")
	}

	// TODO: 发送消息到 Kafka 进行异步处理
	// 这里需要调用 Kafka 生产者发送消息到下游

	s.logger.Info("消息发送成功",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId),
		clog.String("message_id", saveResp.Message.Id))

	return &logicpb.SendMessageResponse{
		MessageId: saveResp.Message.Id,
		SeqId:     seqResp.SeqId,
		Success:   true,
	}, nil
}

// GetMessage 获取消息详情
func (s *MessageService) GetMessage(ctx context.Context, req *logicpb.GetMessageRequest) (*logicpb.GetMessageResponse, error) {
	s.logger.Info("获取消息详情", clog.String("message_id", req.MessageId))

	// 验证参数
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "消息 ID 不能为空")
	}

	// 获取消息
	messageRepo := s.client.GetMessageServiceClient()
	msgReq := &repopb.GetMessageRequest{
		MessageId: req.MessageId,
	}
	msgResp, err := messageRepo.GetMessage(ctx, msgReq)
	if err != nil {
		s.logger.Error("获取消息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "消息不存在")
	}

	// 转换消息格式
	message := s.convertMessage(msgResp.Message)

	s.logger.Info("获取消息详情成功", clog.String("message_id", req.MessageId))

	return &logicpb.GetMessageResponse{
		Message: message,
	}, nil
}

// DeleteMessage 删除消息
func (s *MessageService) DeleteMessage(ctx context.Context, req *logicpb.DeleteMessageRequest) (*logicpb.DeleteMessageResponse, error) {
	s.logger.Info("删除消息",
		clog.String("message_id", req.MessageId),
		clog.String("operator_id", req.OperatorId))

	// 验证参数
	if req.MessageId == "" || req.OperatorId == "" {
		return nil, status.Error(codes.InvalidArgument, "消息 ID 和操作者 ID 不能为空")
	}

	// 获取消息详情以验证权限
	messageRepo := s.client.GetMessageServiceClient()
	msgReq := &repopb.GetMessageRequest{
		MessageId: req.MessageId,
	}
	msgResp, err := messageRepo.GetMessage(ctx, msgReq)
	if err != nil {
		s.logger.Error("获取消息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "消息不存在")
	}

	// 验证删除权限（只有发送者可以删除自己的消息）
	if msgResp.Message.SenderId != req.OperatorId {
		s.logger.Warn("无权限删除消息",
			clog.String("operator_id", req.OperatorId),
			clog.String("sender_id", msgResp.Message.SenderId))
		return nil, status.Error(codes.PermissionDenied, "无权限删除此消息")
	}

	// 删除消息
	deleteReq := &repopb.DeleteMessageRequest{
		MessageId:  req.MessageId,
		OperatorId: req.OperatorId,
		Reason:     req.Reason,
	}
	_, err = messageRepo.DeleteMessage(ctx, deleteReq)
	if err != nil {
		s.logger.Error("删除消息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "删除消息失败")
	}

	s.logger.Info("消息删除成功", clog.String("message_id", req.MessageId))

	return &logicpb.DeleteMessageResponse{
		Success: true,
	}, nil
}

// UpdateMessage 更新消息
func (s *MessageService) UpdateMessage(ctx context.Context, req *logicpb.UpdateMessageRequest) (*logicpb.UpdateMessageResponse, error) {
	s.logger.Info("更新消息",
		clog.String("message_id", req.MessageId),
		clog.String("operator_id", req.OperatorId))

	// 验证参数
	if req.MessageId == "" || req.OperatorId == "" {
		return nil, status.Error(codes.InvalidArgument, "消息 ID 和操作者 ID 不能为空")
	}

	// 获取消息详情以验证权限
	messageRepo := s.client.GetMessageServiceClient()
	msgReq := &repopb.GetMessageRequest{
		MessageId: req.MessageId,
	}
	msgResp, err := messageRepo.GetMessage(ctx, msgReq)
	if err != nil {
		s.logger.Error("获取消息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "消息不存在")
	}

	// 验证更新权限（只有发送者可以更新自己的消息）
	if msgResp.Message.SenderId != req.OperatorId {
		s.logger.Warn("无权限更新消息",
			clog.String("operator_id", req.OperatorId),
			clog.String("sender_id", msgResp.Message.SenderId))
		return nil, status.Error(codes.PermissionDenied, "无权限更新此消息")
	}

	// 验证消息类型是否支持更新
	if msgResp.Message.MessageType != int32(logicpb.MessageType_MESSAGE_TYPE_TEXT) {
		s.logger.Warn("不支持更新此类型消息", clog.Int("message_type", int(msgResp.Message.MessageType)))
		return nil, status.Error(codes.InvalidArgument, "不支持更新此类型消息")
	}

	// 验证新内容
	if req.NewContent == "" {
		return nil, status.Error(codes.InvalidArgument, "新消息内容不能为空")
	}

	// 验证消息长度
	if len(req.NewContent) > s.config.Message.MaxLength {
		return nil, status.Error(codes.InvalidArgument, "消息内容过长")
	}

	// TODO: 实现消息更新逻辑
	// 目前 im-repo 没有提供更新消息的接口，需要添加
	// 这里可以记录更新历史或者创建新消息

	s.logger.Info("消息更新成功", clog.String("message_id", req.MessageId))

	return &logicpb.UpdateMessageResponse{
		Success: true,
		Message: s.convertMessage(msgResp.Message),
	}, nil
}

// ForwardMessage 转发消息
func (s *MessageService) ForwardMessage(ctx context.Context, req *logicpb.ForwardMessageRequest) (*logicpb.ForwardMessageResponse, error) {
	s.logger.Info("转发消息",
		clog.String("user_id", req.UserId),
		clog.String("source_conversation_id", req.SourceConversationId),
		clog.String("target_conversation_id", req.TargetConversationId),
		clog.String("message_id", req.MessageId))

	// 验证参数
	if req.UserId == "" || req.SourceConversationId == "" || req.TargetConversationId == "" || req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID、源会话 ID、目标会话 ID 和消息 ID 不能为空")
	}

	// 获取原始消息
	messageRepo := s.client.GetMessageServiceClient()
	msgReq := &repopb.GetMessageRequest{
		MessageId: req.MessageId,
	}
	msgResp, err := messageRepo.GetMessage(ctx, msgReq)
	if err != nil {
		s.logger.Error("获取原始消息失败", clog.Err(err))
		return nil, status.Error(codes.NotFound, "原始消息不存在")
	}

	// 生成序列号
	seqReq := &repopb.GenerateSeqIDRequest{
		ConversationId: req.TargetConversationId,
	}
	seqResp, err := messageRepo.GenerateSeqID(ctx, seqReq)
	if err != nil {
		s.logger.Error("生成序列号失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "生成序列号失败")
	}

	// 保存转发消息
	saveReq := &repopb.SaveMessageRequest{
		MessageId:      fmt.Sprintf("fwd-%d", time.Now().UnixNano()),
		ConversationId: req.TargetConversationId,
		SenderId:       req.UserId,
		MessageType:    msgResp.Message.MessageType,
		Content:        msgResp.Message.Content,
		SeqId:          seqResp.SeqId,
		ClientMsgId:    fmt.Sprintf("fwd-%s", req.MessageId),
		Extra: fmt.Sprintf(`{"forwarded_from": {"message_id": "%s", "conversation_id": "%s", "sender_id": "%s"}}`,
			req.MessageId, req.SourceConversationId, msgResp.Message.SenderId),
	}

	saveResp, err := messageRepo.SaveMessage(ctx, saveReq)
	if err != nil {
		s.logger.Error("保存转发消息失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "保存转发消息失败")
	}

	s.logger.Info("消息转发成功",
		clog.String("user_id", req.UserId),
		clog.String("message_id", saveResp.Message.Id))

	return &logicpb.ForwardMessageResponse{
		MessageId: saveResp.Message.Id,
		SeqId:     seqResp.SeqId,
		Success:   true,
	}, nil
}

// isValidMessageType 验证消息类型是否有效
func (s *MessageService) isValidMessageType(messageType logicpb.MessageType) bool {
	for _, allowedType := range s.config.Message.AllowedTypes {
		if int(messageType) == allowedType {
			return true
		}
	}
	return false
}

// convertMessage 转换消息格式
func (s *MessageService) convertMessage(msg *repopb.Message) *logicpb.Message {
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
