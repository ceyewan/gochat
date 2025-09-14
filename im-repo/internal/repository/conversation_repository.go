package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/model"
	"gorm.io/gorm"
)

// ConversationRepository 会话数据仓储
type ConversationRepository struct {
	db     *Database
	cache  *CacheManager
	logger clog.Logger
}

// NewConversationRepository 创建会话数据仓储
func NewConversationRepository(db *Database, cache *CacheManager) *ConversationRepository {
	return &ConversationRepository{
		db:     db,
		cache:  cache,
		logger: clog.Namespace("conversation-repository"),
	}
}

// GetUserConversations 获取用户会话列表
func (r *ConversationRepository) GetUserConversations(ctx context.Context, userID uint64, offset, limit int, typeFilter int) ([]string, int64, error) {
	r.logger.Debug("获取用户会话列表",
		clog.String("user_id", fmt.Sprintf("%d", userID)),
		clog.Int("offset", offset),
		clog.Int("limit", limit))

	// 这里简化实现，实际项目中可能需要一个专门的用户会话关系表
	// 目前通过已读位置表来获取用户参与的会话
	var conversationIDs []string
	var total int64

	query := r.db.GetDB().WithContext(ctx).
		Model(&model.UserReadPointer{}).
		Where("user_id = ?", userID)

	// 统计总数
	err := query.Count(&total).Error
	if err != nil {
		r.logger.Error("统计用户会话数量失败", clog.Err(err))
		return nil, 0, fmt.Errorf("统计用户会话数量失败: %w", err)
	}

	// 获取会话 ID 列表
	err = query.
		Select("conversation_id").
		Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Pluck("conversation_id", &conversationIDs).Error

	if err != nil {
		r.logger.Error("获取用户会话列表失败", clog.Err(err))
		return nil, 0, fmt.Errorf("获取用户会话列表失败: %w", err)
	}

	r.logger.Debug("获取用户会话列表成功",
		clog.String("user_id", fmt.Sprintf("%d", userID)),
		clog.Int64("total", total),
		clog.Int("returned", len(conversationIDs)))

	return conversationIDs, total, nil
}

// UpdateReadPointer 更新已读位置
func (r *ConversationRepository) UpdateReadPointer(ctx context.Context, userID uint64, conversationID string, seqID uint64) error {
	r.logger.Debug("更新已读位置",
		clog.String("user_id", fmt.Sprintf("%d", userID)),
		clog.String("conversation_id", conversationID),
		clog.Int64("seq_id", int64(seqID)))

	now := time.Now()

	// 使用 UPSERT 操作：如果记录存在则更新，不存在则插入
	readPointer := &model.UserReadPointer{
		UserID:         userID,
		ConversationID: conversationID,
		LastReadSeqID:  seqID,
		UpdatedAt:      now,
	}

	// 先尝试更新
	result := r.db.GetDB().WithContext(ctx).
		Model(&model.UserReadPointer{}).
		Where("user_id = ? AND conversation_id = ?", userID, conversationID).
		Updates(map[string]interface{}{
			"last_read_seq_id": seqID,
			"updated_at":       now,
		})

	if result.Error != nil {
		r.logger.Error("更新已读位置失败", clog.Err(result.Error))
		return fmt.Errorf("更新已读位置失败: %w", result.Error)
	}

	// 如果没有更新任何记录，说明记录不存在，需要插入
	if result.RowsAffected == 0 {
		err := r.db.GetDB().WithContext(ctx).Create(readPointer).Error
		if err != nil {
			r.logger.Error("插入已读位置失败", clog.Err(err))
			return fmt.Errorf("插入已读位置失败: %w", err)
		}
	}

	// 更新缓存中的未读消息数（设置为0，因为已读到最新位置）
	err := r.cache.SetUnreadCount(ctx, conversationID, fmt.Sprintf("%d", userID), 0)
	if err != nil {
		r.logger.Warn("更新未读消息数缓存失败", clog.Err(err))
	}

	r.logger.Debug("已读位置更新成功",
		clog.String("user_id", fmt.Sprintf("%d", userID)),
		clog.String("conversation_id", conversationID),
		clog.Int64("seq_id", int64(seqID)))

	return nil
}

// GetUnreadCount 获取未读消息数
func (r *ConversationRepository) GetUnreadCount(ctx context.Context, userID uint64, conversationID string) (int64, error) {
	userIDStr := fmt.Sprintf("%d", userID)

	r.logger.Debug("获取未读消息数",
		clog.String("user_id", userIDStr),
		clog.String("conversation_id", conversationID))

	// 先尝试从缓存获取
	cachedCount, err := r.cache.GetUnreadCount(ctx, conversationID, userIDStr)
	if err != nil {
		r.logger.Warn("从缓存获取未读消息数失败", clog.Err(err))
	} else if cachedCount >= 0 {
		r.logger.Debug("从缓存获取未读消息数成功",
			clog.String("user_id", userIDStr),
			clog.String("conversation_id", conversationID),
			clog.Int64("count", cachedCount))
		return cachedCount, nil
	}

	// 缓存未命中，从数据库计算
	unreadCount, err := r.calculateUnreadCount(ctx, userID, conversationID)
	if err != nil {
		return 0, err
	}

	// 写入缓存
	err = r.cache.SetUnreadCount(ctx, conversationID, userIDStr, unreadCount)
	if err != nil {
		r.logger.Warn("缓存未读消息数失败", clog.Err(err))
	}

	r.logger.Debug("从数据库计算未读消息数成功",
		clog.String("user_id", userIDStr),
		clog.String("conversation_id", conversationID),
		clog.Int64("count", unreadCount))

	return unreadCount, nil
}

// GetReadPointer 获取已读位置
func (r *ConversationRepository) GetReadPointer(ctx context.Context, userID uint64, conversationID string) (*model.UserReadPointer, error) {
	r.logger.Debug("获取已读位置",
		clog.String("user_id", fmt.Sprintf("%d", userID)),
		clog.String("conversation_id", conversationID))

	readPointer := &model.UserReadPointer{}
	err := r.db.GetDB().WithContext(ctx).
		Where("user_id = ? AND conversation_id = ?", userID, conversationID).
		First(readPointer).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 记录不存在
		}
		r.logger.Error("获取已读位置失败", clog.Err(err))
		return nil, fmt.Errorf("获取已读位置失败: %w", err)
	}

	return readPointer, nil
}

// BatchGetUnreadCounts 批量获取未读消息数
func (r *ConversationRepository) BatchGetUnreadCounts(ctx context.Context, userID uint64, conversationIDs []string) (map[string]int64, error) {
	if len(conversationIDs) == 0 {
		return make(map[string]int64), nil
	}

	userIDStr := fmt.Sprintf("%d", userID)
	r.logger.Debug("批量获取未读消息数",
		clog.String("user_id", userIDStr),
		clog.Int("conversation_count", len(conversationIDs)))

	result := make(map[string]int64)
	cachedCounts := make(map[string]int64)
	missedConversations := make([]string, 0)

	// 先尝试从缓存批量获取
	for _, conversationID := range conversationIDs {
		cachedCount, err := r.cache.GetUnreadCount(ctx, conversationID, userIDStr)
		if err != nil {
			r.logger.Warn("从缓存获取未读消息数失败",
				clog.String("conversation_id", conversationID),
				clog.Err(err))
			missedConversations = append(missedConversations, conversationID)
			continue
		}

		cachedCounts[conversationID] = cachedCount
	}

	// 从数据库计算缓存未命中的会话
	if len(missedConversations) > 0 {
		for _, conversationID := range missedConversations {
			unreadCount, err := r.calculateUnreadCount(ctx, userID, conversationID)
			if err != nil {
				r.logger.Error("计算未读消息数失败",
					clog.String("conversation_id", conversationID),
					clog.Err(err))
				continue
			}

			cachedCounts[conversationID] = unreadCount

			// 写入缓存
			err = r.cache.SetUnreadCount(ctx, conversationID, userIDStr, unreadCount)
			if err != nil {
				r.logger.Warn("缓存未读消息数失败",
					clog.String("conversation_id", conversationID),
					clog.Err(err))
			}
		}
	}

	// 按原始顺序返回结果
	for _, conversationID := range conversationIDs {
		if count, exists := cachedCounts[conversationID]; exists {
			result[conversationID] = count
		}
	}

	r.logger.Debug("批量获取未读消息数完成",
		clog.String("user_id", userIDStr),
		clog.Int("requested", len(conversationIDs)),
		clog.Int("successful", len(result)))

	return result, nil
}

// calculateUnreadCount 计算未读消息数
func (r *ConversationRepository) calculateUnreadCount(ctx context.Context, userID uint64, conversationID string) (int64, error) {
	// 获取用户的已读位置
	readPointer, err := r.GetReadPointer(ctx, userID, conversationID)
	if err != nil {
		return 0, err
	}

	var lastReadSeqID uint64 = 0
	if readPointer != nil {
		lastReadSeqID = readPointer.LastReadSeqID
	}

	// 统计大于已读位置的消息数量
	var unreadCount int64
	err = r.db.GetDB().WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND seq_id > ? AND deleted = ?", conversationID, lastReadSeqID, false).
		Count(&unreadCount).Error

	if err != nil {
		r.logger.Error("计算未读消息数失败", clog.Err(err))
		return 0, fmt.Errorf("计算未读消息数失败: %w", err)
	}

	return unreadCount, nil
}

// IncrementUnreadCount 增加未读消息数（当有新消息时调用）
func (r *ConversationRepository) IncrementUnreadCount(ctx context.Context, conversationID string, excludeUserID uint64) error {
	r.logger.Debug("增加未读消息数",
		clog.String("conversation_id", conversationID),
		clog.String("exclude_user_id", fmt.Sprintf("%d", excludeUserID)))

	// 获取会话中的所有用户（除了发送者）
	// 这里简化实现，实际项目中可能需要从群组成员表或会话成员表获取
	var userIDs []uint64
	err := r.db.GetDB().WithContext(ctx).
		Model(&model.UserReadPointer{}).
		Where("conversation_id = ? AND user_id != ?", conversationID, excludeUserID).
		Pluck("user_id", &userIDs).Error

	if err != nil {
		r.logger.Error("获取会话用户列表失败", clog.Err(err))
		return fmt.Errorf("获取会话用户列表失败: %w", err)
	}

	// 为每个用户增加未读消息数
	for _, userID := range userIDs {
		userIDStr := fmt.Sprintf("%d", userID)
		_, err := r.cache.IncrUnreadCount(ctx, conversationID, userIDStr)
		if err != nil {
			r.logger.Warn("增加用户未读消息数失败",
				clog.String("user_id", userIDStr),
				clog.String("conversation_id", conversationID),
				clog.Err(err))
		}
	}

	r.logger.Debug("未读消息数增加完成",
		clog.String("conversation_id", conversationID),
		clog.Int("affected_users", len(userIDs)))

	return nil
}
