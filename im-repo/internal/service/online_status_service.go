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
		logger:           clog.Namespace("online-status-service"),
	}
}

// SetUserOnline 设置用户在线状态
func (s *OnlineStatusService) SetUserOnline(ctx context.Context, req *repopb.SetUserOnlineRequest) (*repopb.SetUserOnlineResponse, error) {
	s.logger.Debug("设置用户在线状态请求",
		clog.String("user_id", req.UserId),
		clog.String("gateway_id", req.GatewayId))

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

	// 设置用户在线状态
	err = s.onlineStatusRepo.SetUserOnline(ctx, userID, req.GatewayId, 300) // 默认5分钟
	if err != nil {
		s.logger.Error("设置用户在线状态失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "设置用户在线状态失败")
	}

	// 构造响应
	resp := &repopb.SetUserOnlineResponse{
		Success: true,
		Status: &repopb.OnlineStatus{
			UserId:    req.UserId,
			IsOnline:  true,
			GatewayId: req.GatewayId,
		},
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

// GetUserOnlineStatus 获取用户在线状态
func (s *OnlineStatusService) GetUserOnlineStatus(ctx context.Context, req *repopb.GetUserOnlineStatusRequest) (*repopb.GetUserOnlineStatusResponse, error) {
	s.logger.Debug("获取用户在线状态请求", clog.String("user_id", req.UserId))

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
	resp := &repopb.GetUserOnlineStatusResponse{
		Status: &repopb.OnlineStatus{
			UserId:    req.UserId,
			IsOnline:  isOnline,
			GatewayId: gatewayID,
		},
	}

	s.logger.Debug("用户在线状态检查完成",
		clog.String("user_id", req.UserId),
		clog.Bool("is_online", isOnline),
		clog.String("gateway_id", gatewayID))

	return resp, nil
}

// GetUsersOnlineStatus 批量获取用户在线状态
func (s *OnlineStatusService) GetUsersOnlineStatus(ctx context.Context, req *repopb.GetUsersOnlineStatusRequest) (*repopb.GetUsersOnlineStatusResponse, error) {
	s.logger.Debug("批量获取用户在线状态请求", clog.Int("user_count", len(req.UserIds)))

	// 参数验证
	if len(req.UserIds) == 0 {
		return &repopb.GetUsersOnlineStatusResponse{
			Statuses: []*repopb.OnlineStatus{},
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
	statuses := make([]*repopb.OnlineStatus, 0, len(userIDs))
	for _, userID := range userIDs {
		userIDStr := fmt.Sprintf("%d", userID)
		statuses = append(statuses, &repopb.OnlineStatus{
			UserId:    userIDStr,
			IsOnline:  onlineStatusMap[userID],
			GatewayId: gatewayMap[userID],
		})
	}

	// 构造响应
	resp := &repopb.GetUsersOnlineStatusResponse{
		Statuses: statuses,
	}

	s.logger.Debug("批量获取用户在线状态完成",
		clog.Int("requested", len(req.UserIds)),
		clog.Int("online_count", len(gatewayMap)))

	return resp, nil
}

// UpdateHeartbeat 更新心跳
func (s *OnlineStatusService) UpdateHeartbeat(ctx context.Context, req *repopb.UpdateHeartbeatRequest) (*repopb.UpdateHeartbeatResponse, error) {
	s.logger.Debug("更新心跳请求",
		clog.String("user_id", req.UserId))

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

	// 刷新用户在线状态TTL
	err = s.onlineStatusRepo.RefreshUserOnline(ctx, userID, 300) // 默认5分钟
	if err != nil {
		s.logger.Error("刷新用户在线状态TTL失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "刷新用户在线状态TTL失败")
	}

	// 构造响应
	resp := &repopb.UpdateHeartbeatResponse{
		Success: true,
	}

	s.logger.Debug("用户心跳更新成功", clog.String("user_id", req.UserId))
	return resp, nil
}

// CleanupExpiredStatus 清理过期的在线状态
func (s *OnlineStatusService) CleanupExpiredStatus(ctx context.Context, req *repopb.CleanupExpiredStatusRequest) (*repopb.CleanupExpiredStatusResponse, error) {
	s.logger.Debug("清理过期在线状态请求")

	// 清理过期的在线状态
	// 注意：当前 repository 实现不使用 req 中的参数，但保留以备将来扩展
	cleanedCount, err := s.onlineStatusRepo.CleanupExpiredOnlineStatus(ctx)
	if err != nil {
		s.logger.Error("清理过期在线状态失败", clog.Err(err))
		return nil, status.Error(codes.Internal, "清理过期在线状态失败")
	}

	// 构造响应
	resp := &repopb.CleanupExpiredStatusResponse{
		CleanedCount: cleanedCount,
	}

	s.logger.Debug("清理过期在线状态完成", clog.Int64("cleaned_count", cleanedCount))
	return resp, nil
}
