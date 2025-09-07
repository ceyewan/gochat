package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/config"
	"github.com/ceyewan/gochat/im-repo/internal/model"
)

// CacheManager 缓存管理器
type CacheManager struct {
	cache  cache.Cache
	config *config.Config
	logger clog.Logger
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(cfg *config.Config) (*CacheManager, error) {
	logger := clog.Module("cache-manager")

	// 创建缓存连接
	cacheClient, err := cache.New(context.Background(), cfg.Cache, cache.WithLogger(logger))
	if err != nil {
		logger.Error("创建缓存连接失败", clog.Err(err))
		return nil, fmt.Errorf("创建缓存连接失败: %w", err)
	}

	manager := &CacheManager{
		cache:  cacheClient,
		config: cfg,
		logger: logger,
	}

	logger.Info("缓存连接创建成功")
	return manager, nil
}

// Redis 缓存键定义
const (
	// 用户信息缓存 - HASH
	UserInfoKey = "user_info:%s"

	// 用户在线状态 - HASH
	UserSessionKey = "user_session:%s"

	// 会话序列号 - STRING
	ConvSeqKey = "conv_seq:%s"

	// 群组成员列表 - SET
	GroupMembersKey = "group_members:%s"

	// 消息去重 - STRING with TTL
	MsgDedupKey = "msg_dedup:%s"

	// 热点消息缓存 - ZSET (暂未实现)
	HotMessagesKey = "hot_messages:%s"

	// 未读消息数 - STRING
	UnreadCountKey = "unread:%s:%s"
)

// === 用户信息缓存 ===

// GetUserInfo 获取用户信息缓存
func (c *CacheManager) GetUserInfo(ctx context.Context, userID string) (*model.User, error) {
	key := fmt.Sprintf(UserInfoKey, userID)

	userMap, err := c.cache.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(userMap) == 0 {
		return nil, nil // 缓存未命中
	}

	// 将 map 转换为 User 结构体
	user := &model.User{}
	if err := c.mapToUser(userMap, user); err != nil {
		c.logger.Error("用户信息反序列化失败", clog.Err(err))
		return nil, err
	}

	return user, nil
}

// SetUserInfo 设置用户信息缓存
func (c *CacheManager) SetUserInfo(ctx context.Context, user *model.User) error {
	key := fmt.Sprintf(UserInfoKey, fmt.Sprintf("%d", user.ID))

	// 将 User 结构体转换为 map
	userMap := c.userToMap(user)

	// 使用 pipeline 批量设置
	for field, value := range userMap {
		if err := c.cache.HSet(ctx, key, field, value); err != nil {
			return err
		}
	}

	// 设置过期时间
	ttl := c.config.Business.Cache.UserInfoTTL
	return c.cache.Expire(ctx, key, ttl)
}

// DelUserInfo 删除用户信息缓存
func (c *CacheManager) DelUserInfo(ctx context.Context, userID string) error {
	key := fmt.Sprintf(UserInfoKey, userID)
	return c.cache.Del(ctx, key)
}

// === 在线状态管理 ===

// SetUserOnlineStatus 设置用户在线状态
func (c *CacheManager) SetUserOnlineStatus(ctx context.Context, userID string, onlineData map[string]interface{}, ttl time.Duration) error {
	key := fmt.Sprintf(UserSessionKey, userID)

	// 设置在线状态信息
	for field, value := range onlineData {
		if err := c.cache.HSet(ctx, key, field, fmt.Sprintf("%v", value)); err != nil {
			return err
		}
	}

	// 设置过期时间
	return c.cache.Expire(ctx, key, ttl)
}

// DeleteUserOnlineStatus 删除用户在线状态
func (c *CacheManager) DeleteUserOnlineStatus(ctx context.Context, userID string) error {
	key := fmt.Sprintf(UserSessionKey, userID)
	return c.cache.Del(ctx, key)
}

// GetUserOnlineStatus 获取用户在线状态
func (c *CacheManager) GetUserOnlineStatus(ctx context.Context, userID string) (map[string]interface{}, error) {
	key := fmt.Sprintf(UserSessionKey, userID)
	result, err := c.cache.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	// 转换为 map[string]interface{}
	onlineData := make(map[string]interface{})
	for k, v := range result {
		onlineData[k] = v
	}

	return onlineData, nil
}

// BatchGetUserOnlineStatus 批量获取用户在线状态
func (c *CacheManager) BatchGetUserOnlineStatus(ctx context.Context, userIDs []string) (map[string]map[string]interface{}, error) {
	result := make(map[string]map[string]interface{})

	for _, userID := range userIDs {
		onlineData, err := c.GetUserOnlineStatus(ctx, userID)
		if err != nil {
			c.logger.Warn("获取用户在线状态失败",
				clog.String("user_id", userID),
				clog.Err(err))
			continue
		}
		result[userID] = onlineData
	}

	return result, nil
}

// RefreshUserOnlineStatus 刷新用户在线状态TTL
func (c *CacheManager) RefreshUserOnlineStatus(ctx context.Context, userID string, ttl time.Duration) error {
	key := fmt.Sprintf(UserSessionKey, userID)
	return c.cache.Expire(ctx, key, ttl)
}

// GetOnlineUsersByGateway 获取指定网关的在线用户列表
func (c *CacheManager) GetOnlineUsersByGateway(ctx context.Context, gatewayID string) ([]string, error) {
	// 这里需要扫描所有在线用户，实际项目中可能需要更高效的实现
	// 简化实现：返回空列表
	c.logger.Warn("GetOnlineUsersByGateway 方法需要更高效的实现")
	return []string{}, nil
}

// GetTotalOnlineUserCount 获取总在线用户数
func (c *CacheManager) GetTotalOnlineUserCount(ctx context.Context) (int64, error) {
	// 简化实现：返回0
	c.logger.Warn("GetTotalOnlineUserCount 方法需要更高效的实现")
	return 0, nil
}

// CleanupExpiredOnlineStatus 清理过期的在线状态
func (c *CacheManager) CleanupExpiredOnlineStatus(ctx context.Context) (int64, error) {
	// 简化实现：返回0
	c.logger.Warn("CleanupExpiredOnlineStatus 方法需要更高效的实现")
	return 0, nil
}

// SetUserLastOnlineTime 设置用户最后在线时间
func (c *CacheManager) SetUserLastOnlineTime(ctx context.Context, userID string, timestamp int64) error {
	key := fmt.Sprintf("last_online:%s", userID)
	return c.cache.Set(ctx, key, timestamp, 24*time.Hour) // 保存24小时
}

// GetUserLastOnlineTime 获取用户最后在线时间
func (c *CacheManager) GetUserLastOnlineTime(ctx context.Context, userID string) (int64, error) {
	key := fmt.Sprintf("last_online:%s", userID)
	timestampStr, err := c.cache.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	if timestampStr == "" {
		return 0, nil
	}

	var timestamp int64
	fmt.Sscanf(timestampStr, "%d", &timestamp)
	return timestamp, nil
}

// SetUserOnline 设置用户在线状态（兼容旧方法）
func (c *CacheManager) SetUserOnline(ctx context.Context, userID, gatewayID string) error {
	onlineData := map[string]interface{}{
		"gateway_id": gatewayID,
		"timestamp":  time.Now().Unix(),
	}
	ttl := c.config.Business.Cache.OnlineStatusTTL
	return c.SetUserOnlineStatus(ctx, userID, onlineData, ttl)
}

// SetUserOffline 设置用户离线状态（兼容旧方法）
func (c *CacheManager) SetUserOffline(ctx context.Context, userID string) error {
	return c.DeleteUserOnlineStatus(ctx, userID)
}

// === 序列号生成 ===

// GenerateSeqID 生成会话序列号
func (c *CacheManager) GenerateSeqID(ctx context.Context, conversationID string) (int64, error) {
	key := fmt.Sprintf(ConvSeqKey, conversationID)
	return c.cache.Incr(ctx, key)
}

// === 消息去重 ===

// CheckAndSetMessageDedup 检查并设置消息去重
func (c *CacheManager) CheckAndSetMessageDedup(ctx context.Context, clientMsgID string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf(MsgDedupKey, clientMsgID)

	// 尝试设置，如果已存在则返回 false
	success, err := c.cache.SetNX(ctx, key, "1", ttl)
	if err != nil {
		return false, err
	}

	return success, nil
}

// === 群组成员缓存 ===

// GetGroupMembers 获取群组成员列表
func (c *CacheManager) GetGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	key := fmt.Sprintf(GroupMembersKey, groupID)
	return c.cache.SMembers(ctx, key)
}

// SetGroupMembers 设置群组成员列表
func (c *CacheManager) SetGroupMembers(ctx context.Context, groupID string, memberIDs []string) error {
	key := fmt.Sprintf(GroupMembersKey, groupID)

	// 先删除旧的集合
	err := c.cache.Del(ctx, key)
	if err != nil {
		return err
	}

	// 添加新成员
	if len(memberIDs) > 0 {
		members := make([]interface{}, len(memberIDs))
		for i, id := range memberIDs {
			members[i] = id
		}

		err = c.cache.SAdd(ctx, key, members...)
		if err != nil {
			return err
		}

		// 设置过期时间
		ttl := c.config.Business.Cache.GroupMembersTTL
		return c.cache.Expire(ctx, key, ttl)
	}

	return nil
}

// AddGroupMember 添加群组成员
func (c *CacheManager) AddGroupMember(ctx context.Context, groupID, userID string) error {
	key := fmt.Sprintf(GroupMembersKey, groupID)
	return c.cache.SAdd(ctx, key, userID)
}

// RemoveGroupMember 移除群组成员
func (c *CacheManager) RemoveGroupMember(ctx context.Context, groupID, userID string) error {
	key := fmt.Sprintf(GroupMembersKey, groupID)
	return c.cache.SRem(ctx, key, userID)
}

// === 未读消息数 ===

// IncrUnreadCount 增加未读消息数
func (c *CacheManager) IncrUnreadCount(ctx context.Context, conversationID, userID string) (int64, error) {
	key := fmt.Sprintf(UnreadCountKey, conversationID, userID)
	return c.cache.Incr(ctx, key)
}

// SetUnreadCount 设置未读消息数
func (c *CacheManager) SetUnreadCount(ctx context.Context, conversationID, userID string, count int64) error {
	key := fmt.Sprintf(UnreadCountKey, conversationID, userID)
	return c.cache.Set(ctx, key, count, 0) // 不设置过期时间
}

// GetUnreadCount 获取未读消息数
func (c *CacheManager) GetUnreadCount(ctx context.Context, conversationID, userID string) (int64, error) {
	key := fmt.Sprintf(UnreadCountKey, conversationID, userID)

	countStr, err := c.cache.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	if countStr == "" {
		return 0, nil
	}

	// 简单的字符串转数字
	var count int64
	fmt.Sscanf(countStr, "%d", &count)
	return count, nil
}

// === 工具方法 ===

// userToMap 将 User 结构体转换为 map
func (c *CacheManager) userToMap(user *model.User) map[string]interface{} {
	return map[string]interface{}{
		"id":         fmt.Sprintf("%d", user.ID),
		"username":   user.Username,
		"nickname":   user.Nickname,
		"avatar_url": user.AvatarURL,
		"created_at": fmt.Sprintf("%d", user.CreatedAt.Unix()),
		"updated_at": fmt.Sprintf("%d", user.UpdatedAt.Unix()),
	}
}

// mapToUser 将 map 转换为 User 结构体
func (c *CacheManager) mapToUser(userMap map[string]string, user *model.User) error {
	// 简单的字符串转换，实际项目中可能需要更严格的验证
	fmt.Sscanf(userMap["id"], "%d", &user.ID)
	user.Username = userMap["username"]
	user.Nickname = userMap["nickname"]
	user.AvatarURL = userMap["avatar_url"]

	var createdAt, updatedAt int64
	fmt.Sscanf(userMap["created_at"], "%d", &createdAt)
	fmt.Sscanf(userMap["updated_at"], "%d", &updatedAt)

	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)

	return nil
}

// Close 关闭缓存连接
func (c *CacheManager) Close() error {
	c.logger.Info("关闭缓存连接")
	return c.cache.Close()
}

// Ping 检查缓存连接
func (c *CacheManager) Ping(ctx context.Context) error {
	return c.cache.Ping(ctx)
}
