package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// OnlineStatusRepository 在线状态数据仓储
type OnlineStatusRepository struct {
	cache  *CacheManager
	logger clog.Logger
}

// NewOnlineStatusRepository 创建在线状态数据仓储
func NewOnlineStatusRepository(cache *CacheManager) *OnlineStatusRepository {
	return &OnlineStatusRepository{
		cache:  cache,
		logger: clog.Namespace("online-status-repository"),
	}
}

// SetUserOnline 设置用户在线状态
func (r *OnlineStatusRepository) SetUserOnline(ctx context.Context, userID uint64, gatewayID string, ttlSeconds int) error {
	userIDStr := fmt.Sprintf("%d", userID)
	r.logger.Debug("设置用户在线状态",
		clog.String("user_id", userIDStr),
		clog.String("gateway_id", gatewayID),
		clog.Int("ttl_seconds", ttlSeconds))

	if ttlSeconds <= 0 {
		ttlSeconds = 300 // 默认5分钟TTL
	}

	ttl := time.Duration(ttlSeconds) * time.Second

	// 设置用户在线状态，包含网关信息
	onlineData := map[string]interface{}{
		"gateway_id": gatewayID,
		"timestamp":  time.Now().Unix(),
	}

	err := r.cache.SetUserOnlineStatus(ctx, userIDStr, onlineData, ttl)
	if err != nil {
		r.logger.Error("设置用户在线状态失败", clog.Err(err))
		return fmt.Errorf("设置用户在线状态失败: %w", err)
	}

	r.logger.Debug("用户在线状态设置成功",
		clog.String("user_id", userIDStr),
		clog.String("gateway_id", gatewayID))

	return nil
}

// SetUserOffline 设置用户离线状态
func (r *OnlineStatusRepository) SetUserOffline(ctx context.Context, userID uint64) error {
	userIDStr := fmt.Sprintf("%d", userID)
	r.logger.Debug("设置用户离线状态", clog.String("user_id", userIDStr))

	err := r.cache.DeleteUserOnlineStatus(ctx, userIDStr)
	if err != nil {
		r.logger.Error("设置用户离线状态失败", clog.Err(err))
		return fmt.Errorf("设置用户离线状态失败: %w", err)
	}

	r.logger.Debug("用户离线状态设置成功", clog.String("user_id", userIDStr))
	return nil
}

// IsUserOnline 检查用户是否在线
func (r *OnlineStatusRepository) IsUserOnline(ctx context.Context, userID uint64) (bool, string, error) {
	userIDStr := fmt.Sprintf("%d", userID)
	r.logger.Debug("检查用户在线状态", clog.String("user_id", userIDStr))

	onlineData, err := r.cache.GetUserOnlineStatus(ctx, userIDStr)
	if err != nil {
		r.logger.Error("检查用户在线状态失败", clog.Err(err))
		return false, "", fmt.Errorf("检查用户在线状态失败: %w", err)
	}

	if onlineData == nil {
		return false, "", nil // 用户离线
	}

	// 提取网关ID
	gatewayID := ""
	if gid, ok := onlineData["gateway_id"].(string); ok {
		gatewayID = gid
	}

	r.logger.Debug("用户在线状态检查完成",
		clog.String("user_id", userIDStr),
		clog.Bool("online", true),
		clog.String("gateway_id", gatewayID))

	return true, gatewayID, nil
}

// BatchGetOnlineStatus 批量获取用户在线状态
func (r *OnlineStatusRepository) BatchGetOnlineStatus(ctx context.Context, userIDs []uint64) (map[uint64]bool, map[uint64]string, error) {
	if len(userIDs) == 0 {
		return make(map[uint64]bool), make(map[uint64]string), nil
	}

	r.logger.Debug("批量获取用户在线状态", clog.Int("user_count", len(userIDs)))

	onlineStatus := make(map[uint64]bool)
	gatewayMap := make(map[uint64]string)

	// 转换用户ID为字符串
	userIDStrs := make([]string, len(userIDs))
	userIDMap := make(map[string]uint64)
	for i, userID := range userIDs {
		userIDStr := fmt.Sprintf("%d", userID)
		userIDStrs[i] = userIDStr
		userIDMap[userIDStr] = userID
	}

	// 批量获取在线状态
	onlineDataMap, err := r.cache.BatchGetUserOnlineStatus(ctx, userIDStrs)
	if err != nil {
		r.logger.Error("批量获取用户在线状态失败", clog.Err(err))
		return nil, nil, fmt.Errorf("批量获取用户在线状态失败: %w", err)
	}

	// 处理结果
	for userIDStr, onlineData := range onlineDataMap {
		userID := userIDMap[userIDStr]

		if onlineData != nil {
			onlineStatus[userID] = true
			// 提取网关ID
			if gid, ok := onlineData["gateway_id"].(string); ok {
				gatewayMap[userID] = gid
			}
		} else {
			onlineStatus[userID] = false
		}
	}

	// 对于没有返回数据的用户，设置为离线
	for _, userID := range userIDs {
		if _, exists := onlineStatus[userID]; !exists {
			onlineStatus[userID] = false
		}
	}

	r.logger.Debug("批量获取用户在线状态完成",
		clog.Int("requested", len(userIDs)),
		clog.Int("online_count", len(gatewayMap)))

	return onlineStatus, gatewayMap, nil
}

// RefreshUserOnline 刷新用户在线状态TTL
func (r *OnlineStatusRepository) RefreshUserOnline(ctx context.Context, userID uint64, ttlSeconds int) error {
	userIDStr := fmt.Sprintf("%d", userID)
	r.logger.Debug("刷新用户在线状态TTL",
		clog.String("user_id", userIDStr),
		clog.Int("ttl_seconds", ttlSeconds))

	if ttlSeconds <= 0 {
		ttlSeconds = 300 // 默认5分钟TTL
	}

	ttl := time.Duration(ttlSeconds) * time.Second

	err := r.cache.RefreshUserOnlineStatus(ctx, userIDStr, ttl)
	if err != nil {
		r.logger.Error("刷新用户在线状态TTL失败", clog.Err(err))
		return fmt.Errorf("刷新用户在线状态TTL失败: %w", err)
	}

	r.logger.Debug("用户在线状态TTL刷新成功", clog.String("user_id", userIDStr))
	return nil
}

// GetOnlineUsersByGateway 获取指定网关的在线用户列表
func (r *OnlineStatusRepository) GetOnlineUsersByGateway(ctx context.Context, gatewayID string) ([]uint64, error) {
	r.logger.Debug("获取网关在线用户列表", clog.String("gateway_id", gatewayID))

	userIDStrs, err := r.cache.GetOnlineUsersByGateway(ctx, gatewayID)
	if err != nil {
		r.logger.Error("获取网关在线用户列表失败", clog.Err(err))
		return nil, fmt.Errorf("获取网关在线用户列表失败: %w", err)
	}

	// 转换为uint64
	userIDs := make([]uint64, 0, len(userIDStrs))
	for _, userIDStr := range userIDStrs {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			r.logger.Warn("用户ID格式错误",
				clog.String("user_id", userIDStr),
				clog.Err(err))
			continue
		}
		userIDs = append(userIDs, userID)
	}

	r.logger.Debug("获取网关在线用户列表完成",
		clog.String("gateway_id", gatewayID),
		clog.Int("user_count", len(userIDs)))

	return userIDs, nil
}

// GetTotalOnlineCount 获取总在线用户数
func (r *OnlineStatusRepository) GetTotalOnlineCount(ctx context.Context) (int64, error) {
	r.logger.Debug("获取总在线用户数")

	count, err := r.cache.GetTotalOnlineUserCount(ctx)
	if err != nil {
		r.logger.Error("获取总在线用户数失败", clog.Err(err))
		return 0, fmt.Errorf("获取总在线用户数失败: %w", err)
	}

	r.logger.Debug("获取总在线用户数完成", clog.Int64("count", count))
	return count, nil
}

// CleanupExpiredOnlineStatus 清理过期的在线状态（通常由定时任务调用）
func (r *OnlineStatusRepository) CleanupExpiredOnlineStatus(ctx context.Context) (int64, error) {
	r.logger.Debug("清理过期在线状态")

	cleanedCount, err := r.cache.CleanupExpiredOnlineStatus(ctx)
	if err != nil {
		r.logger.Error("清理过期在线状态失败", clog.Err(err))
		return 0, fmt.Errorf("清理过期在线状态失败: %w", err)
	}

	r.logger.Debug("清理过期在线状态完成", clog.Int64("cleaned_count", cleanedCount))
	return cleanedCount, nil
}

// GetUserLastOnlineTime 获取用户最后在线时间
func (r *OnlineStatusRepository) GetUserLastOnlineTime(ctx context.Context, userID uint64) (int64, error) {
	userIDStr := fmt.Sprintf("%d", userID)
	r.logger.Debug("获取用户最后在线时间", clog.String("user_id", userIDStr))

	onlineData, err := r.cache.GetUserOnlineStatus(ctx, userIDStr)
	if err != nil {
		r.logger.Error("获取用户最后在线时间失败", clog.Err(err))
		return 0, fmt.Errorf("获取用户最后在线时间失败: %w", err)
	}

	if onlineData == nil {
		// 用户当前离线，尝试从历史记录获取
		lastOnlineTime, err := r.cache.GetUserLastOnlineTime(ctx, userIDStr)
		if err != nil {
			r.logger.Warn("获取用户历史在线时间失败", clog.Err(err))
			return 0, nil // 返回0表示无记录
		}
		return lastOnlineTime, nil
	}

	// 用户当前在线，返回当前时间戳
	if timestamp, ok := onlineData["timestamp"].(int64); ok {
		return timestamp, nil
	}

	// 如果没有时间戳，返回当前时间
	return time.Now().Unix(), nil
}

// SetUserLastOnlineTime 设置用户最后在线时间（用户离线时调用）
func (r *OnlineStatusRepository) SetUserLastOnlineTime(ctx context.Context, userID uint64, timestamp int64) error {
	userIDStr := fmt.Sprintf("%d", userID)
	r.logger.Debug("设置用户最后在线时间",
		clog.String("user_id", userIDStr),
		clog.Int64("timestamp", timestamp))

	if timestamp <= 0 {
		timestamp = time.Now().Unix()
	}

	err := r.cache.SetUserLastOnlineTime(ctx, userIDStr, timestamp)
	if err != nil {
		r.logger.Error("设置用户最后在线时间失败", clog.Err(err))
		return fmt.Errorf("设置用户最后在线时间失败: %w", err)
	}

	r.logger.Debug("用户最后在线时间设置成功",
		clog.String("user_id", userIDStr),
		clog.Int64("timestamp", timestamp))

	return nil
}
