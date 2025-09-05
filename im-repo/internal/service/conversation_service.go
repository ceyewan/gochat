package service

import (
	"context"
	"strconv"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConversationService 会话服务实现
type ConversationService struct {
	v1.UnimplementedConversationServiceServer
	conversationRepo *repository.ConversationRepository
	messageRepo      *repository.MessageRepository
	logger           clog.Logger
}

// NewConversationService 创建会话服务
func NewConversationService(conversationRepo *repository.ConversationRepository, messageRepo *repository.MessageRepository) *ConversationService {
	return &ConversationService{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		logger:           clog.Module("conversation-service"),
	}
}

// GetUserConversations 获取用户会话列表
func (s *ConversationService) GetUserConversations(ctx context.Context, req *v1.GetUserConversationsRequest) (*v1.GetUserConversationsResponse, error) {
	s.logger.Debug("获取用户会话列表请求",
		clog.String("user_id", req.UserId),
		clog.Int32("offset", req.Offset),
		clog.Int32("limit", req.Limit))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 转换用户ID
	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	offset := int(req.Offset)
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20 // 默认限制
	}
	if limit > 100 {
		limit = 100 // 最大限制
	}

	// 获取会话列表
	conversationIDs, total, err := s.conversationRepo.GetUserConversations(ctx, userID, offset, limit, int(req.TypeFilter))
	if err != nil {
		s.logger.Error("获取用户会话列表失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取用户会话列表失败")
	}

	// 构造响应
	resp := &v1.GetUserConversationsResponse{
		ConversationIds: conversationIDs,
		Total:           total,
		HasMore:         int64(offset+len(conversationIDs)) < total,
	}

	s.logger.Debug("获取用户会话列表成功",
		clog.String("user_id", req.UserId),
		clog.Int64("total", total),
		clog.Int("returned", len(conversationIDs)))

	return resp, nil
}

// UpdateReadPointer 更新已读位置
func (s *ConversationService) UpdateReadPointer(ctx context.Context, req *v1.UpdateReadPointerRequest) (*v1.UpdateReadPointerResponse, error) {
	s.logger.Debug("更新已读位置请求",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId),
		clog.Int64("seq_id", req.SeqId))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "会话ID不能为空")
	}
	if req.SeqId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "序列号必须大于0")
	}

	// 转换用户ID
	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	// 更新已读位置
	err = s.conversationRepo.UpdateReadPointer(ctx, userID, req.ConversationId, uint64(req.SeqId))
	if err != nil {
		s.logger.Error("更新已读位置失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "更新已读位置失败")
	}

	// 构造响应
	resp := &v1.UpdateReadPointerResponse{
		Success: true,
	}

	s.logger.Debug("已读位置更新成功",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId),
		clog.Int64("seq_id", req.SeqId))

	return resp, nil
}

// GetUnreadCount 获取未读消息数
func (s *ConversationService) GetUnreadCount(ctx context.Context, req *v1.GetUnreadCountRequest) (*v1.GetUnreadCountResponse, error) {
	s.logger.Debug("获取未读消息数请求",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "会话ID不能为空")
	}

	// 转换用户ID
	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	// 获取未读消息数
	unreadCount, err := s.conversationRepo.GetUnreadCount(ctx, userID, req.ConversationId)
	if err != nil {
		s.logger.Error("获取未读消息数失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取未读消息数失败")
	}

	// 构造响应
	resp := &v1.GetUnreadCountResponse{
		UnreadCount: unreadCount,
	}

	s.logger.Debug("获取未读消息数成功",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId),
		clog.Int64("unread_count", unreadCount))

	return resp, nil
}

// BatchGetUnreadCounts 批量获取未读消息数
func (s *ConversationService) BatchGetUnreadCounts(ctx context.Context, req *v1.BatchGetUnreadCountsRequest) (*v1.BatchGetUnreadCountsResponse, error) {
	s.logger.Debug("批量获取未读消息数请求",
		clog.String("user_id", req.UserId),
		clog.Int("conversation_count", len(req.ConversationIds)))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}
	if len(req.ConversationIds) == 0 {
		return &v1.BatchGetUnreadCountsResponse{
			UnreadCounts: make(map[string]int64),
		}, nil
	}

	// 转换用户ID
	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	// 批量获取未读消息数
	unreadCounts, err := s.conversationRepo.BatchGetUnreadCounts(ctx, userID, req.ConversationIds)
	if err != nil {
		s.logger.Error("批量获取未读消息数失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "批量获取未读消息数失败")
	}

	// 构造响应
	resp := &v1.BatchGetUnreadCountsResponse{
		UnreadCounts: unreadCounts,
	}

	s.logger.Debug("批量获取未读消息数成功",
		clog.String("user_id", req.UserId),
		clog.Int("requested", len(req.ConversationIds)),
		clog.Int("successful", len(unreadCounts)))

	return resp, nil
}

// GetReadPointer 获取已读位置
func (s *ConversationService) GetReadPointer(ctx context.Context, req *v1.GetReadPointerRequest) (*v1.GetReadPointerResponse, error) {
	s.logger.Debug("获取已读位置请求",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "会话ID不能为空")
	}

	// 转换用户ID
	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	// 获取已读位置
	readPointer, err := s.conversationRepo.GetReadPointer(ctx, userID, req.ConversationId)
	if err != nil {
		s.logger.Error("获取已读位置失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取已读位置失败")
	}

	var lastReadSeqID int64 = 0
	var updatedAt int64 = 0

	if readPointer != nil {
		lastReadSeqID = int64(readPointer.LastReadSeqID)
		updatedAt = readPointer.UpdatedAt.Unix()
	}

	// 构造响应
	resp := &v1.GetReadPointerResponse{
		LastReadSeqId: lastReadSeqID,
		UpdatedAt:     updatedAt,
	}

	s.logger.Debug("获取已读位置成功",
		clog.String("user_id", req.UserId),
		clog.String("conversation_id", req.ConversationId),
		clog.Int64("last_read_seq_id", lastReadSeqID))

	return resp, nil
}

// IncrementUnreadCount 增加未读消息数（当有新消息时调用）
func (s *ConversationService) IncrementUnreadCount(ctx context.Context, req *v1.IncrementUnreadCountRequest) (*v1.IncrementUnreadCountResponse, error) {
	s.logger.Debug("增加未读消息数请求",
		clog.String("conversation_id", req.ConversationId),
		clog.String("exclude_user_id", req.ExcludeUserId))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "会话ID不能为空")
	}

	var excludeUserID uint64 = 0
	if req.ExcludeUserId != "" {
		var err error
		excludeUserID, err = strconv.ParseUint(req.ExcludeUserId, 10, 64)
		if err != nil {
			s.logger.Error("排除用户ID格式错误", clog.Err(err))
			return nil, status.Error(codes.InvalidArgument, "排除用户ID格式错误")
		}
	}

	// 增加未读消息数
	err := s.conversationRepo.IncrementUnreadCount(ctx, req.ConversationId, excludeUserID)
	if err != nil {
		s.logger.Error("增加未读消息数失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "增加未读消息数失败")
	}

	// 构造响应
	resp := &v1.IncrementUnreadCountResponse{
		Success: true,
	}

	s.logger.Debug("未读消息数增加成功",
		clog.String("conversation_id", req.ConversationId),
		clog.String("exclude_user_id", req.ExcludeUserId))

	return resp, nil
}

// GetMaxSeqID 获取会话的最大序列号
func (s *ConversationService) GetMaxSeqID(ctx context.Context, req *v1.GetMaxSeqIDRequest) (*v1.GetMaxSeqIDResponse, error) {
	s.logger.Debug("获取会话最大序列号请求", clog.String("conversation_id", req.ConversationId))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "会话ID不能为空")
	}

	// 获取最大序列号
	maxSeqID, err := s.messageRepo.GetMaxSeqID(ctx, req.ConversationId)
	if err != nil {
		s.logger.Error("获取会话最大序列号失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取会话最大序列号失败")
	}

	// 构造响应
	resp := &v1.GetMaxSeqIDResponse{
		MaxSeqId: int64(maxSeqID),
	}

	s.logger.Debug("获取会话最大序列号成功",
		clog.String("conversation_id", req.ConversationId),
		clog.Int64("max_seq_id", int64(maxSeqID)))

	return resp, nil
}

// CountMessages 统计会话消息数量
func (s *ConversationService) CountMessages(ctx context.Context, req *v1.CountMessagesRequest) (*v1.CountMessagesResponse, error) {
	s.logger.Debug("统计会话消息数量请求", clog.String("conversation_id", req.ConversationId))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "会话ID不能为空")
	}

	// 统计消息数量
	count, err := s.messageRepo.CountMessages(ctx, req.ConversationId)
	if err != nil {
		s.logger.Error("统计会话消息数量失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "统计会话消息数量失败")
	}

	// 构造响应
	resp := &v1.CountMessagesResponse{
		Count: count,
	}

	s.logger.Debug("统计会话消息数量成功",
		clog.String("conversation_id", req.ConversationId),
		clog.Int64("count", count))

	return resp, nil
}
