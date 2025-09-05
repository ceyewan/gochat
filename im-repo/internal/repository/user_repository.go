package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-repo/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserRepository 用户数据仓储
type UserRepository struct {
	db     *Database
	cache  *CacheManager
	logger clog.Logger
}

// NewUserRepository 创建用户数据仓储
func NewUserRepository(db *Database, cache *CacheManager) *UserRepository {
	return &UserRepository{
		db:     db,
		cache:  cache,
		logger: clog.Module("user-repository"),
	}
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	r.logger.Info("创建用户", clog.String("username", user.Username))

	// 哈希密码
	if user.PasswordHash != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
		if err != nil {
			r.logger.Error("密码哈希失败", clog.Err(err))
			return fmt.Errorf("密码哈希失败: %w", err)
		}
		user.PasswordHash = string(hashedPassword)
	}

	// 设置时间戳
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// 保存到数据库
	err := r.db.GetDB().WithContext(ctx).Create(user).Error
	if err != nil {
		r.logger.Error("创建用户失败", clog.Err(err))
		return fmt.Errorf("创建用户失败: %w", err)
	}

	// 缓存用户信息
	if err := r.cache.SetUserInfo(ctx, user); err != nil {
		r.logger.Warn("缓存用户信息失败", clog.Err(err))
		// 缓存失败不影响主流程
	}

	r.logger.Info("用户创建成功", clog.String("user_id", fmt.Sprintf("%d", user.ID)))
	return nil
}

// GetUserByID 获取用户信息
func (r *UserRepository) GetUserByID(ctx context.Context, userID uint64) (*model.User, error) {
	userIDStr := fmt.Sprintf("%d", userID)

	// 先尝试从缓存获取
	user, err := r.cache.GetUserInfo(ctx, userIDStr)
	if err != nil {
		r.logger.Warn("从缓存获取用户信息失败", clog.Err(err))
	}

	if user != nil {
		r.logger.Debug("从缓存获取用户信息成功", clog.String("user_id", userIDStr))
		return user, nil
	}

	// 缓存未命中，从数据库查询
	user = &model.User{}
	err = r.db.GetDB().WithContext(ctx).Where("id = ?", userID).First(user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 用户不存在
		}
		r.logger.Error("从数据库获取用户信息失败", clog.Err(err))
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 写入缓存
	if err := r.cache.SetUserInfo(ctx, user); err != nil {
		r.logger.Warn("缓存用户信息失败", clog.Err(err))
	}

	r.logger.Debug("从数据库获取用户信息成功", clog.String("user_id", userIDStr))
	return user, nil
}

// GetUsers 批量获取用户信息
func (r *UserRepository) GetUsers(ctx context.Context, userIDs []uint64) ([]*model.User, error) {
	if len(userIDs) == 0 {
		return []*model.User{}, nil
	}

	r.logger.Debug("批量获取用户信息", clog.Int("count", len(userIDs)))

	users := make([]*model.User, 0, len(userIDs))
	cachedUsers := make(map[uint64]*model.User)
	missedIDs := make([]uint64, 0)

	// 先尝试从缓存批量获取
	for _, userID := range userIDs {
		userIDStr := fmt.Sprintf("%d", userID)
		user, err := r.cache.GetUserInfo(ctx, userIDStr)
		if err != nil {
			r.logger.Warn("从缓存获取用户信息失败",
				clog.String("user_id", userIDStr),
				clog.Err(err))
			missedIDs = append(missedIDs, userID)
			continue
		}

		if user != nil {
			cachedUsers[userID] = user
		} else {
			missedIDs = append(missedIDs, userID)
		}
	}

	// 从数据库查询缓存未命中的用户
	if len(missedIDs) > 0 {
		var dbUsers []*model.User
		err := r.db.GetDB().WithContext(ctx).Where("id IN ?", missedIDs).Find(&dbUsers).Error
		if err != nil {
			r.logger.Error("从数据库批量获取用户信息失败", clog.Err(err))
			return nil, fmt.Errorf("批量获取用户信息失败: %w", err)
		}

		// 将数据库查询结果写入缓存
		for _, user := range dbUsers {
			cachedUsers[user.ID] = user
			if err := r.cache.SetUserInfo(ctx, user); err != nil {
				r.logger.Warn("缓存用户信息失败",
					clog.String("user_id", fmt.Sprintf("%d", user.ID)),
					clog.Err(err))
			}
		}
	}

	// 按原始顺序返回结果
	for _, userID := range userIDs {
		if user, exists := cachedUsers[userID]; exists {
			users = append(users, user)
		}
	}

	r.logger.Debug("批量获取用户信息完成",
		clog.Int("requested", len(userIDs)),
		clog.Int("found", len(users)))

	return users, nil
}

// UpdateUser 更新用户信息
func (r *UserRepository) UpdateUser(ctx context.Context, userID uint64, updates map[string]interface{}) error {
	userIDStr := fmt.Sprintf("%d", userID)
	r.logger.Info("更新用户信息", clog.String("user_id", userIDStr))

	// 添加更新时间
	updates["updated_at"] = time.Now()

	// 更新数据库
	err := r.db.GetDB().WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
	if err != nil {
		r.logger.Error("更新用户信息失败", clog.Err(err))
		return fmt.Errorf("更新用户信息失败: %w", err)
	}

	// 删除缓存，让下次查询时重新加载
	if err := r.cache.DelUserInfo(ctx, userIDStr); err != nil {
		r.logger.Warn("删除用户缓存失败", clog.Err(err))
	}

	r.logger.Info("用户信息更新成功", clog.String("user_id", userIDStr))
	return nil
}

// VerifyPassword 验证用户密码
func (r *UserRepository) VerifyPassword(ctx context.Context, userID uint64, password string) (bool, error) {
	userIDStr := fmt.Sprintf("%d", userID)
	r.logger.Debug("验证用户密码", clog.String("user_id", userIDStr))

	// 从数据库获取用户信息（包含密码哈希）
	user := &model.User{}
	err := r.db.GetDB().WithContext(ctx).Select("password_hash").Where("id = ?", userID).First(user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		r.logger.Error("获取用户密码哈希失败", clog.Err(err))
		return false, fmt.Errorf("获取用户密码哈希失败: %w", err)
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil // 密码不匹配
		}
		r.logger.Error("密码验证失败", clog.Err(err))
		return false, fmt.Errorf("密码验证失败: %w", err)
	}

	return true, nil
}

// GetUserByUsername 根据用户名获取用户
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	r.logger.Debug("根据用户名获取用户", clog.String("username", username))

	user := &model.User{}
	err := r.db.GetDB().WithContext(ctx).Where("username = ?", username).First(user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 用户不存在
		}
		r.logger.Error("根据用户名获取用户失败", clog.Err(err))
		return nil, fmt.Errorf("根据用户名获取用户失败: %w", err)
	}

	// 缓存用户信息
	if err := r.cache.SetUserInfo(ctx, user); err != nil {
		r.logger.Warn("缓存用户信息失败", clog.Err(err))
	}

	return user, nil
}
