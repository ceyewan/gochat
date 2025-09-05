package service

import (
	"context"
	"fmt"
	"strconv"

	repopb "github.com/ceyewan/gochat/api/gen/im_repo/v1"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OnlineStatusService 在线状态服务实现
type OnlineStatusService struct {
	repopb.UnimplementedOnlineStatusServiceServer
	onlineStatusRepo *repository.OnlineStatusRepository
	logger           clog.Logger
}

// NewOnlineStatusService 创建在线状态服务
func NewOnlineStatusService(onlineStatusRepo *repository.OnlineStatusRepository) *OnlineStatusService {
	return &OnlineStatusService{
		onlineStatusRepo: onlineStatusRepo,
		logger:           clog.Module("online-status-service"),
	}
}

// SetUserOnline 设置用户在线状态
func (s *OnlineStatusService) SetUserOnline(ctx context.Context, req *repopb.SetUserOnlineRequest) (*repopb.SetUserOnlineResponse, error) {
	s.logger.Debug("设置用户在线状态请求",
		clog.String("user_id", req.UserId),
		clog.String("gateway_id", req.GatewayId),
		clog.Int32("ttl_seconds", req.TtlSeconds))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}
	if req.GatewayId == "" {
		return nil, status.Error(codes.InvalidArgument, "网关ID不能为空")
	}

	// 转换用户ID
	userID, err := strconv.ParseUint(req.UserId, 10, 64)
	if err != nil {
		s.logger.Error("用户ID格式错误", clog.Err(err))
		return nil, status.Error(codes.InvalidArgument, "用户ID格式错误")
	}

	ttlSeconds := int(req.TtlSeconds)
	if ttlSeconds <= 0 {
		ttlSeconds = 300 // 默认5分钟
	}

	// 设置用户在线状态
	err = s.onlineStatusRepo.SetUserOnline(ctx, userID, req.GatewayId, ttlSeconds)
	if err != nil {
		s.logger.Error("设置用户在线状态失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "设置用户在线状态失败")
	}

	// 构造响应
	resp := &repopb.SetUserOnlineResponse{
		Success: true,
	}

	s.logger.Debug("用户在线状态设置成功",
		clog.String("user_id", req.UserId),
		clog.String("gateway_id", req.GatewayId))

	return resp, nil
}

// SetUserOffline 设置用户离线状态
func (s *OnlineStatusService) SetUserOffline(ctx context.Context, req *repopb.SetUserOfflineRequest) (*repopb.SetUserOfflineResponse, error) {
	s.logger.Debug("设置用户离线状态请求", clog.String("user_id", req.UserId))

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

	// 先记录最后在线时间
	if req.LastOnlineTime > 0 {
		err = s.onlineStatusRepo.SetUserLastOnlineTime(ctx, userID, req.LastOnlineTime)
		if err != nil {
			s.logger.Warn("设置用户最后在线时间失败", clog.Err(err))
		}
	}

	// 设置用户离线状态
	err = s.onlineStatusRepo.SetUserOffline(ctx, userID)
	if err != nil {
		s.logger.Error("设置用户离线状态失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "设置用户离线状态失败")
	}

	// 构造响应
	resp := &repopb.SetUserOfflineResponse{
		Success: true,
	}

	s.logger.Debug("用户离线状态设置成功", clog.String("user_id", req.UserId))
	return resp, nil
}

// IsUserOnline 检查用户是否在线
func (s *OnlineStatusService) IsUserOnline(ctx context.Context, req *repopb.IsUserOnlineRequest) (*repopb.IsUserOnlineResponse, error) {
	s.logger.Debug("检查用户在线状态请求", clog.String("user_id", req.UserId))

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

	// 检查用户在线状态
	isOnline, gatewayID, err := s.onlineStatusRepo.IsUserOnline(ctx, userID)
	if err != nil {
		s.logger.Error("检查用户在线状态失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "检查用户在线状态失败")
	}

	// 构造响应
	resp := &repopb.IsUserOnlineResponse{
		IsOnline:  isOnline,
		GatewayId: gatewayID,
	}

	s.logger.Debug("用户在线状态检查完成",
		clog.String("user_id", req.UserId),
		clog.Bool("is_online", isOnline),
		clog.String("gateway_id", gatewayID))

	return resp, nil
}

// BatchGetOnlineStatus 批量获取用户在线状态
func (s *OnlineStatusService) BatchGetOnlineStatus(ctx context.Context, req *repopb.BatchGetOnlineStatusRequest) (*repopb.BatchGetOnlineStatusResponse, error) {
	s.logger.Debug("批量获取用户在线状态请求", clog.Int("user_count", len(req.UserIds)))

	// 参数验证
	if len(req.UserIds) == 0 {
		return &repopb.BatchGetOnlineStatusResponse{
			OnlineStatus: make(map[string]*repopb.UserOnlineStatus),
		}, nil
	}

	// 转换用户ID列表
	userIDs := make([]uint64, len(req.UserIds))
	for i, userIDStr := range req.UserIds {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			s.logger.Error("用户ID格式错误",
				clog.String("user_id", userIDStr),
				clog.Err(err))
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("用户ID格式错误: %s", userIDStr))
		}
		userIDs[i] = userID
	}

	// 批量获取在线状态
	onlineStatusMap, gatewayMap, err := s.onlineStatusRepo.BatchGetOnlineStatus(ctx, userIDs)
	if err != nil {
		s.logger.Error("批量获取用户在线状态失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "批量获取用户在线状态失败")
	}

	// 转换为 protobuf 格式
	protoOnlineStatus := make(map[string]*repopb.UserOnlineStatus)
	for _, userID := range userIDs {
		userIDStr := fmt.Sprintf("%d", userID)
		isOnline := onlineStatusMap[userID]
		gatewayID := gatewayMap[userID]

		protoOnlineStatus[userIDStr] = &repopb.UserOnlineStatus{
			IsOnline:  isOnline,
			GatewayId: gatewayID,
		}
	}

	// 构造响应
	resp := &repopb.BatchGetOnlineStatusResponse{
		OnlineStatus: protoOnlineStatus,
	}

	s.logger.Debug("批量获取用户在线状态完成",
		clog.Int("requested", len(req.UserIds)),
		clog.Int("online_count", len(gatewayMap)))

	return resp, nil
}

// RefreshUserOnline 刷新用户在线状态TTL
func (s *OnlineStatusService) RefreshUserOnline(ctx context.Context, req *repopb.RefreshUserOnlineRequest) (*repopb.RefreshUserOnlineResponse, error) {
	s.logger.Debug("刷新用户在线状态TTL请求",
		clog.String("user_id", req.UserId),
		clog.Int32("ttl_seconds", req.TtlSeconds))

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

	ttlSeconds := int(req.TtlSeconds)
	if ttlSeconds <= 0 {
		ttlSeconds = 300 // 默认5分钟
	}

	// 刷新用户在线状态TTL
	err = s.onlineStatusRepo.RefreshUserOnline(ctx, userID, ttlSeconds)
	if err != nil {
		s.logger.Error("刷新用户在线状态TTL失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "刷新用户在线状态TTL失败")
	}

	// 构造响应
	resp := &repopb.RefreshUserOnlineResponse{
		Success: true,
	}

	s.logger.Debug("用户在线状态TTL刷新成功", clog.String("user_id", req.UserId))
	return resp, nil
}

// GetOnlineUsersByGateway 获取指定网关的在线用户列表
func (s *OnlineStatusService) GetOnlineUsersByGateway(ctx context.Context, req *repopb.GetOnlineUsersByGatewayRequest) (*repopb.GetOnlineUsersByGatewayResponse, error) {
	s.logger.Debug("获取网关在线用户列表请求", clog.String("gateway_id", req.GatewayId))

	// 参数验证
	if req.GatewayId == "" {
		return nil, status.Error(codes.InvalidArgument, "网关ID不能为空")
	}

	// 获取网关在线用户列表
	userIDs, err := s.onlineStatusRepo.GetOnlineUsersByGateway(ctx, req.GatewayId)
	if err != nil {
		s.logger.Error("获取网关在线用户列表失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取网关在线用户列表失败")
	}

	// 转换为字符串格式
	userIDStrs := make([]string, len(userIDs))
	for i, userID := range userIDs {
		userIDStrs[i] = fmt.Sprintf("%d", userID)
	}

	// 构造响应
	resp := &repopb.GetOnlineUsersByGatewayResponse{
		UserIds: userIDStrs,
	}

	s.logger.Debug("获取网关在线用户列表完成",
		clog.String("gateway_id", req.GatewayId),
		clog.Int("user_count", len(userIDs)))

	return resp, nil
}

// GetTotalOnlineCount 获取总在线用户数
func (s *OnlineStatusService) GetTotalOnlineCount(ctx context.Context, req *repopb.GetTotalOnlineCountRequest) (*repopb.GetTotalOnlineCountResponse, error) {
	s.logger.Debug("获取总在线用户数请求")

	// 获取总在线用户数
	count, err := s.onlineStatusRepo.GetTotalOnlineCount(ctx)
	if err != nil {
		s.logger.Error("获取总在线用户数失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取总在线用户数失败")
	}

	// 构造响应
	resp := &repopb.GetTotalOnlineCountResponse{
		Count: count,
	}

	s.logger.Debug("获取总在线用户数完成", clog.Int64("count", count))
	return resp, nil
}

// GetUserLastOnlineTime 获取用户最后在线时间
func (s *OnlineStatusService) GetUserLastOnlineTime(ctx context.Context, req *repopb.GetUserLastOnlineTimeRequest) (*repopb.GetUserLastOnlineTimeResponse, error) {
	s.logger.Debug("获取用户最后在线时间请求", clog.String("user_id", req.UserId))

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

	// 获取用户最后在线时间
	lastOnlineTime, err := s.onlineStatusRepo.GetUserLastOnlineTime(ctx, userID)
	if err != nil {
		s.logger.Error("获取用户最后在线时间失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "获取用户最后在线时间失败")
	}

	// 构造响应
	resp := &repopb.GetUserLastOnlineTimeResponse{
		LastOnlineTime: lastOnlineTime,
	}

	s.logger.Debug("获取用户最后在线时间完成",
		clog.String("user_id", req.UserId),
		clog.Int64("last_online_time", lastOnlineTime))

	return resp, nil
}

// CleanupExpiredOnlineStatus 清理过期的在线状态
func (s *OnlineStatusService) CleanupExpiredOnlineStatus(ctx context.Context, req *repopb.CleanupExpiredOnlineStatusRequest) (*repopb.CleanupExpiredOnlineStatusResponse, error) {
	s.logger.Debug("清理过期在线状态请求")

	// 清理过期的在线状态
	cleanedCount, err := s.onlineStatusRepo.CleanupExpiredOnlineStatus(ctx)
	if err != nil {
		s.logger.Error("清理过期在线状态失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "清理过期在线状态失败")
	}

	// 构造响应
	resp := &repopb.CleanupExpiredOnlineStatusResponse{
		CleanedCount: cleanedCount,
	}

	s.logger.Debug("清理过期在线状态完成", clog.Int64("cleaned_count", cleanedCount))
	return resp, nil
}
