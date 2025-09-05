package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/model"
	"gorm.io/gorm"
)

// MessageRepository 消息数据仓储
type MessageRepository struct {
	db     *Database
	cache  *CacheManager
	logger clog.Logger
}

// NewMessageRepository 创建消息数据仓储
func NewMessageRepository(db *Database, cache *CacheManager) *MessageRepository {
	return &MessageRepository{
		db:     db,
		cache:  cache,
		logger: clog.Module("message-repository"),
	}
}

// SaveMessage 保存消息
func (r *MessageRepository) SaveMessage(ctx context.Context, message *model.Message) error {
	r.logger.Info("保存消息",
		clog.String("message_id", fmt.Sprintf("%d", message.ID)),
		clog.String("conversation_id", message.ConversationID))

	// 设置时间戳
	now := time.Now()
	message.CreatedAt = now
	message.UpdatedAt = now

	// 保存到数据库
	err := r.db.GetDB().WithContext(ctx).Create(message).Error
	if err != nil {
		r.logger.Error("保存消息失败", clog.Err(err))
		return fmt.Errorf("保存消息失败: %w", err)
	}

	// TODO: 更新热点消息缓存（ZSET）
	// 这里可以将最近的消息缓存到 Redis ZSET 中，提高查询性能

	r.logger.Info("消息保存成功",
		clog.String("message_id", fmt.Sprintf("%d", message.ID)))
	return nil
}

// GetMessage 获取单条消息
func (r *MessageRepository) GetMessage(ctx context.Context, messageID uint64) (*model.Message, error) {
	messageIDStr := fmt.Sprintf("%d", messageID)
	r.logger.Debug("获取消息", clog.String("message_id", messageIDStr))

	message := &model.Message{}
	err := r.db.GetDB().WithContext(ctx).Where("id = ? AND deleted = ?", messageID, false).First(message).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 消息不存在
		}
		r.logger.Error("获取消息失败", clog.Err(err))
		return nil, fmt.Errorf("获取消息失败: %w", err)
	}

	return message, nil
}

// GetConversationMessages 获取会话消息列表
func (r *MessageRepository) GetConversationMessages(ctx context.Context, conversationID string, startSeqID, endSeqID int64, limit int, ascending bool) ([]*model.Message, bool, int64, error) {
	r.logger.Debug("获取会话消息列表",
		clog.String("conversation_id", conversationID),
		clog.Int64("start_seq_id", startSeqID),
		clog.Int64("end_seq_id", endSeqID),
		clog.Int("limit", limit),
		clog.Bool("ascending", ascending))

	query := r.db.GetDB().WithContext(ctx).Where("conversation_id = ? AND deleted = ?", conversationID, false)

	// 添加序列号范围条件
	if startSeqID > 0 {
		if ascending {
			query = query.Where("seq_id >= ?", startSeqID)
		} else {
			query = query.Where("seq_id <= ?", startSeqID)
		}
	}
	if endSeqID > 0 {
		if ascending {
			query = query.Where("seq_id <= ?", endSeqID)
		} else {
			query = query.Where("seq_id >= ?", endSeqID)
		}
	}

	// 设置排序
	if ascending {
		query = query.Order("seq_id ASC")
	} else {
		query = query.Order("seq_id DESC")
	}

	// 限制数量（多查询一条用于判断是否还有更多）
	queryLimit := limit + 1
	query = query.Limit(queryLimit)

	var messages []*model.Message
	err := query.Find(&messages).Error
	if err != nil {
		r.logger.Error("获取会话消息列表失败", clog.Err(err))
		return nil, false, 0, fmt.Errorf("获取会话消息列表失败: %w", err)
	}

	// 判断是否还有更多消息
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit] // 移除多查询的那一条
	}

	// 计算下一页的起始序列号
	var nextSeqID int64
	if len(messages) > 0 {
		if ascending {
			nextSeqID = messages[len(messages)-1].SeqID + 1
		} else {
			nextSeqID = messages[len(messages)-1].SeqID - 1
		}
	}

	r.logger.Debug("获取会话消息列表完成",
		clog.Int("count", len(messages)),
		clog.Bool("has_more", hasMore))

	return messages, hasMore, nextSeqID, nil
}

// GenerateSeqID 生成会话序列号
func (r *MessageRepository) GenerateSeqID(ctx context.Context, conversationID string) (int64, error) {
	r.logger.Debug("生成会话序列号", clog.String("conversation_id", conversationID))

	seqID, err := r.cache.GenerateSeqID(ctx, conversationID)
	if err != nil {
		r.logger.Error("生成会话序列号失败", clog.Err(err))
		return 0, fmt.Errorf("生成会话序列号失败: %w", err)
	}

	r.logger.Debug("会话序列号生成成功",
		clog.String("conversation_id", conversationID),
		clog.Int64("seq_id", seqID))

	return seqID, nil
}

// CheckMessageIdempotency 检查消息幂等性
func (r *MessageRepository) CheckMessageIdempotency(ctx context.Context, clientMsgID string, ttl time.Duration) (bool, string, error) {
	r.logger.Debug("检查消息幂等性", clog.String("client_msg_id", clientMsgID))

	// 检查是否已存在
	exists, err := r.cache.CheckAndSetMessageDedup(ctx, clientMsgID, ttl)
	if err != nil {
		r.logger.Error("检查消息幂等性失败", clog.Err(err))
		return false, "", fmt.Errorf("检查消息幂等性失败: %w", err)
	}

	if !exists {
		// 消息已存在，查找对应的消息ID
		// 这里可以通过查询数据库或缓存来找到已存在的消息ID
		// 为了简化，这里返回空字符串，实际项目中可能需要额外的缓存来存储映射关系
		r.logger.Debug("检测到重复消息", clog.String("client_msg_id", clientMsgID))
		return false, "", nil
	}

	r.logger.Debug("消息幂等性检查通过", clog.String("client_msg_id", clientMsgID))
	return true, "", nil
}

// GetLatestMessages 获取多个会话的最新消息
func (r *MessageRepository) GetLatestMessages(ctx context.Context, conversationIDs []string, limitPerConversation int) (map[string][]*model.Message, error) {
	if len(conversationIDs) == 0 {
		return make(map[string][]*model.Message), nil
	}

	r.logger.Debug("获取多个会话的最新消息",
		clog.Int("conversation_count", len(conversationIDs)),
		clog.Int("limit_per_conversation", limitPerConversation))

	result := make(map[string][]*model.Message)

	// 为每个会话查询最新消息
	for _, conversationID := range conversationIDs {
		var messages []*model.Message
		err := r.db.GetDB().WithContext(ctx).
			Where("conversation_id = ? AND deleted = ?", conversationID, false).
			Order("seq_id DESC").
			Limit(limitPerConversation).
			Find(&messages).Error

		if err != nil {
			r.logger.Error("获取会话最新消息失败",
				clog.String("conversation_id", conversationID),
				clog.Err(err))
			continue // 跳过失败的会话，不影响其他会话
		}

		result[conversationID] = messages
	}

	r.logger.Debug("获取多个会话的最新消息完成",
		clog.Int("success_count", len(result)))

	return result, nil
}

// DeleteMessage 软删除消息
func (r *MessageRepository) DeleteMessage(ctx context.Context, messageID uint64, operatorID uint64, reason string) error {
	messageIDStr := fmt.Sprintf("%d", messageID)
	operatorIDStr := fmt.Sprintf("%d", operatorID)

	r.logger.Info("删除消息",
		clog.String("message_id", messageIDStr),
		clog.String("operator_id", operatorIDStr))

	// 软删除：标记为已删除
	updates := map[string]interface{}{
		"deleted":    true,
		"updated_at": time.Now(),
	}

	err := r.db.GetDB().WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", messageID).
		Updates(updates).Error

	if err != nil {
		r.logger.Error("删除消息失败", clog.Err(err))
		return fmt.Errorf("删除消息失败: %w", err)
	}

	// TODO: 从热点消息缓存中移除

	r.logger.Info("消息删除成功", clog.String("message_id", messageIDStr))
	return nil
}

// GetMessagesBySeqRange 根据序列号范围获取消息
func (r *MessageRepository) GetMessagesBySeqRange(ctx context.Context, conversationID string, startSeqID, endSeqID int64) ([]*model.Message, error) {
	r.logger.Debug("根据序列号范围获取消息",
		clog.String("conversation_id", conversationID),
		clog.Int64("start_seq_id", startSeqID),
		clog.Int64("end_seq_id", endSeqID))

	var messages []*model.Message
	err := r.db.GetDB().WithContext(ctx).
		Where("conversation_id = ? AND seq_id >= ? AND seq_id <= ? AND deleted = ?",
			conversationID, startSeqID, endSeqID, false).
		Order("seq_id ASC").
		Find(&messages).Error

	if err != nil {
		r.logger.Error("根据序列号范围获取消息失败", clog.Err(err))
		return nil, fmt.Errorf("根据序列号范围获取消息失败: %w", err)
	}

	r.logger.Debug("根据序列号范围获取消息完成",
		clog.Int("count", len(messages)))

	return messages, nil
}

// GetConversationLatestSeqID 获取会话的最新序列号
func (r *MessageRepository) GetConversationLatestSeqID(ctx context.Context, conversationID string) (int64, error) {
	r.logger.Debug("获取会话最新序列号", clog.String("conversation_id", conversationID))

	var maxSeqID int64
	err := r.db.GetDB().WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND deleted = ?", conversationID, false).
		Select("COALESCE(MAX(seq_id), 0)").
		Scan(&maxSeqID).Error

	if err != nil {
		r.logger.Error("获取会话最新序列号失败", clog.Err(err))
		return 0, fmt.Errorf("获取会话最新序列号失败: %w", err)
	}

	r.logger.Debug("获取会话最新序列号完成",
		clog.String("conversation_id", conversationID),
		clog.Int64("max_seq_id", maxSeqID))

	return maxSeqID, nil
}
