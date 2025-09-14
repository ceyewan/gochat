package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/model"
	"gorm.io/gorm"
)

// GroupRepository 群组数据仓储
type GroupRepository struct {
	db     *Database
	cache  *CacheManager
	logger clog.Logger
}

// NewGroupRepository 创建群组数据仓储
func NewGroupRepository(db *Database, cache *CacheManager) *GroupRepository {
	return &GroupRepository{
		db:     db,
		cache:  cache,
		logger: clog.Namespace("group-repository"),
	}
}

// CreateGroup 创建群组
func (r *GroupRepository) CreateGroup(ctx context.Context, group *model.Group) error {
	r.logger.Info("创建群组",
		clog.String("group_id", fmt.Sprintf("%d", group.ID)),
		clog.String("name", group.Name))

	// 设置时间戳
	now := time.Now()
	group.CreatedAt = now
	group.UpdatedAt = now
	group.MemberCount = 1 // 创建者自动成为第一个成员

	// 使用事务创建群组和群主成员记录
	err := r.db.Transaction(ctx, func(tx *gorm.DB) error {
		// 创建群组
		if err := tx.Create(group).Error; err != nil {
			return err
		}

		// 添加群主为成员
		member := &model.GroupMember{
			GroupID:  group.ID,
			UserID:   group.OwnerID,
			Role:     3, // 群主角色
			JoinedAt: now,
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		r.logger.Error("创建群组失败", clog.Err(err))
		return fmt.Errorf("创建群组失败: %w", err)
	}

	// 初始化群组成员缓存
	ownerIDStr := fmt.Sprintf("%d", group.OwnerID)
	if err := r.cache.SetGroupMembers(ctx, fmt.Sprintf("%d", group.ID), []string{ownerIDStr}); err != nil {
		r.logger.Warn("初始化群组成员缓存失败", clog.Err(err))
	}

	r.logger.Info("群组创建成功", clog.String("group_id", fmt.Sprintf("%d", group.ID)))
	return nil
}

// GetGroup 获取群组信息
func (r *GroupRepository) GetGroup(ctx context.Context, groupID uint64) (*model.Group, error) {
	groupIDStr := fmt.Sprintf("%d", groupID)
	r.logger.Debug("获取群组信息", clog.String("group_id", groupIDStr))

	group := &model.Group{}
	err := r.db.GetDB().WithContext(ctx).Where("id = ?", groupID).First(group).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 群组不存在
		}
		r.logger.Error("获取群组信息失败", clog.Err(err))
		return nil, fmt.Errorf("获取群组信息失败: %w", err)
	}

	return group, nil
}

// AddGroupMember 添加群组成员
func (r *GroupRepository) AddGroupMember(ctx context.Context, groupID, userID uint64, role int) error {
	groupIDStr := fmt.Sprintf("%d", groupID)
	userIDStr := fmt.Sprintf("%d", userID)

	r.logger.Info("添加群组成员",
		clog.String("group_id", groupIDStr),
		clog.String("user_id", userIDStr),
		clog.Int("role", role))

	// 使用事务添加成员并更新成员数量
	err := r.db.Transaction(ctx, func(tx *gorm.DB) error {
		// 检查是否已经是成员
		var count int64
		err := tx.Model(&model.GroupMember{}).
			Where("group_id = ? AND user_id = ?", groupID, userID).
			Count(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("用户已经是群组成员")
		}

		// 添加成员记录
		member := &model.GroupMember{
			GroupID:  groupID,
			UserID:   userID,
			Role:     role,
			JoinedAt: time.Now(),
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}

		// 更新群组成员数量
		return tx.Model(&model.Group{}).
			Where("id = ?", groupID).
			UpdateColumn("member_count", gorm.Expr("member_count + 1")).Error
	})

	if err != nil {
		r.logger.Error("添加群组成员失败", clog.Err(err))
		return fmt.Errorf("添加群组成员失败: %w", err)
	}

	// 更新群组成员缓存
	if err := r.cache.AddGroupMember(ctx, groupIDStr, userIDStr); err != nil {
		r.logger.Warn("更新群组成员缓存失败", clog.Err(err))
	}

	r.logger.Info("群组成员添加成功")
	return nil
}

// RemoveGroupMember 移除群组成员
func (r *GroupRepository) RemoveGroupMember(ctx context.Context, groupID, userID uint64) error {
	groupIDStr := fmt.Sprintf("%d", groupID)
	userIDStr := fmt.Sprintf("%d", userID)

	r.logger.Info("移除群组成员",
		clog.String("group_id", groupIDStr),
		clog.String("user_id", userIDStr))

	// 使用事务移除成员并更新成员数量
	err := r.db.Transaction(ctx, func(tx *gorm.DB) error {
		// 删除成员记录
		result := tx.Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&model.GroupMember{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("用户不是群组成员")
		}

		// 更新群组成员数量
		return tx.Model(&model.Group{}).
			Where("id = ?", groupID).
			UpdateColumn("member_count", gorm.Expr("member_count - 1")).Error
	})

	if err != nil {
		r.logger.Error("移除群组成员失败", clog.Err(err))
		return fmt.Errorf("移除群组成员失败: %w", err)
	}

	// 更新群组成员缓存
	if err := r.cache.RemoveGroupMember(ctx, groupIDStr, userIDStr); err != nil {
		r.logger.Warn("更新群组成员缓存失败", clog.Err(err))
	}

	r.logger.Info("群组成员移除成功")
	return nil
}

// GetGroupMembers 获取群组成员列表
func (r *GroupRepository) GetGroupMembers(ctx context.Context, groupID uint64, offset, limit int, roleFilter int) ([]*model.GroupMember, int64, bool, error) {
	groupIDStr := fmt.Sprintf("%d", groupID)
	r.logger.Debug("获取群组成员列表",
		clog.String("group_id", groupIDStr),
		clog.Int("offset", offset),
		clog.Int("limit", limit))

	// 先尝试从缓存获取成员ID列表（仅用于快速检查）
	cachedMemberIDs, err := r.cache.GetGroupMembers(ctx, groupIDStr)
	if err != nil {
		r.logger.Warn("从缓存获取群组成员失败", clog.Err(err))
	}

	// 构建查询
	query := r.db.GetDB().WithContext(ctx).Where("group_id = ?", groupID)
	if roleFilter > 0 {
		query = query.Where("role = ?", roleFilter)
	}

	// 获取总数
	var total int64
	err = query.Model(&model.GroupMember{}).Count(&total).Error
	if err != nil {
		r.logger.Error("获取群组成员总数失败", clog.Err(err))
		return nil, 0, false, fmt.Errorf("获取群组成员总数失败: %w", err)
	}

	// 分页查询成员
	var members []*model.GroupMember
	queryLimit := limit + 1 // 多查询一条用于判断是否还有更多
	err = query.Offset(offset).Limit(queryLimit).Order("joined_at ASC").Find(&members).Error
	if err != nil {
		r.logger.Error("获取群组成员列表失败", clog.Err(err))
		return nil, 0, false, fmt.Errorf("获取群组成员列表失败: %w", err)
	}

	// 判断是否还有更多
	hasMore := len(members) > limit
	if hasMore {
		members = members[:limit]
	}

	// 如果缓存为空或不一致，更新缓存
	if len(cachedMemberIDs) == 0 && len(members) > 0 {
		memberIDs := make([]string, len(members))
		for i, member := range members {
			memberIDs[i] = fmt.Sprintf("%d", member.UserID)
		}
		if err := r.cache.SetGroupMembers(ctx, groupIDStr, memberIDs); err != nil {
			r.logger.Warn("更新群组成员缓存失败", clog.Err(err))
		}
	}

	r.logger.Debug("获取群组成员列表完成",
		clog.Int("count", len(members)),
		clog.Int64("total", total),
		clog.Bool("has_more", hasMore))

	return members, total, hasMore, nil
}

// UpdateMemberRole 更新成员角色
func (r *GroupRepository) UpdateMemberRole(ctx context.Context, groupID, userID uint64, newRole int) error {
	groupIDStr := fmt.Sprintf("%d", groupID)
	userIDStr := fmt.Sprintf("%d", userID)

	r.logger.Info("更新成员角色",
		clog.String("group_id", groupIDStr),
		clog.String("user_id", userIDStr),
		clog.Int("new_role", newRole))

	err := r.db.GetDB().WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("role", newRole).Error

	if err != nil {
		r.logger.Error("更新成员角色失败", clog.Err(err))
		return fmt.Errorf("更新成员角色失败: %w", err)
	}

	r.logger.Info("成员角色更新成功")
	return nil
}

// IsGroupMember 检查用户是否为群组成员
func (r *GroupRepository) IsGroupMember(ctx context.Context, groupID, userID uint64) (bool, int, error) {
	groupIDStr := fmt.Sprintf("%d", groupID)
	userIDStr := fmt.Sprintf("%d", userID)

	// 先尝试从缓存快速检查
	cachedMembers, err := r.cache.GetGroupMembers(ctx, groupIDStr)
	if err == nil && len(cachedMembers) > 0 {
		for _, memberID := range cachedMembers {
			if memberID == userIDStr {
				// 从数据库获取角色信息
				var member model.GroupMember
				err := r.db.GetDB().WithContext(ctx).
					Where("group_id = ? AND user_id = ?", groupID, userID).
					First(&member).Error
				if err == nil {
					return true, member.Role, nil
				}
				break
			}
		}
		return false, 0, nil
	}

	// 缓存未命中，从数据库查询
	var member model.GroupMember
	err = r.db.GetDB().WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, 0, nil // 不是群组成员
		}
		r.logger.Error("检查群组成员失败", clog.Err(err))
		return false, 0, fmt.Errorf("检查群组成员失败: %w", err)
	}

	return true, member.Role, nil
}
