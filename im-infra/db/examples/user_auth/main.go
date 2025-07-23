package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Email     string    `gorm:"uniqueIndex;size:100;not null" json:"email"`
	Password  string    `gorm:"size:32;not null" json:"-"` // MD5 哈希，不在 JSON 中返回
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserService 用户服务
type UserService struct {
	db     db.DB
	logger clog.Logger
}

// NewUserService 创建用户服务实例
func NewUserService(database db.DB) *UserService {
	return &UserService{
		db:     database,
		logger: clog.Module("user.service"),
	}
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, username, email, password string) (*User, error) {
	s.logger.Info("用户注册开始",
		clog.String("username", username),
		clog.String("email", email),
	)

	// 检查用户名是否已存在
	var existingUser User
	err := s.db.GetDB().WithContext(ctx).Where("username = ?", username).First(&existingUser).Error
	if err == nil {
		s.logger.Warn("用户名已存在", clog.String("username", username))
		return nil, fmt.Errorf("username already exists")
	}
	if err != gorm.ErrRecordNotFound {
		s.logger.Error("检查用户名失败", clog.Err(err))
		return nil, fmt.Errorf("failed to check username: %w", err)
	}

	// 检查邮箱是否已存在
	err = s.db.GetDB().WithContext(ctx).Where("email = ?", email).First(&existingUser).Error
	if err == nil {
		s.logger.Warn("邮箱已存在", clog.String("email", email))
		return nil, fmt.Errorf("email already exists")
	}
	if err != gorm.ErrRecordNotFound {
		s.logger.Error("检查邮箱失败", clog.Err(err))
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	// 创建新用户
	user := &User{
		Username: username,
		Email:    email,
		Password: hashPassword(password),
	}

	err = s.db.GetDB().WithContext(ctx).Create(user).Error
	if err != nil {
		s.logger.Error("创建用户失败",
			clog.Err(err),
			clog.String("username", username),
		)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("用户注册成功",
		clog.String("username", username),
		clog.Uint("userID", user.ID),
	)

	return user, nil
}

// Login 用户登录
func (s *UserService) Login(ctx context.Context, username, password string) (*User, error) {
	s.logger.Info("用户登录开始", clog.String("username", username))

	var user User
	err := s.db.GetDB().WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Warn("用户不存在", clog.String("username", username))
			return nil, fmt.Errorf("user not found")
		}
		s.logger.Error("查询用户失败", clog.Err(err))
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// 验证密码
	if !checkPassword(password, user.Password) {
		s.logger.Warn("密码错误", clog.String("username", username))
		return nil, fmt.Errorf("invalid password")
	}

	s.logger.Info("用户登录成功",
		clog.String("username", username),
		clog.Uint("userID", user.ID),
	)

	return &user, nil
}

// GetUserByID 根据 ID 获取用户
func (s *UserService) GetUserByID(ctx context.Context, id uint) (*User, error) {
	var user User
	err := s.db.GetDB().WithContext(ctx).First(&user, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		s.logger.Error("查询用户失败", clog.Err(err), clog.Uint("userID", id))
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &user, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error {
	err := s.db.GetDB().WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		s.logger.Error("更新用户失败", clog.Err(err), clog.Uint("userID", id))
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("用户更新成功", clog.Uint("userID", id))
	return nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	err := s.db.GetDB().WithContext(ctx).Delete(&User{}, id).Error
	if err != nil {
		s.logger.Error("删除用户失败", clog.Err(err), clog.Uint("userID", id))
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.logger.Info("用户删除成功", clog.Uint("userID", id))
	return nil
}

// ListUsers 获取用户列表
func (s *UserService) ListUsers(ctx context.Context, offset, limit int) ([]User, error) {
	var users []User
	err := s.db.GetDB().WithContext(ctx).Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		s.logger.Error("查询用户列表失败", clog.Err(err))
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

// hashPassword 对密码进行 MD5 哈希
func hashPassword(password string) string {
	hash := md5.Sum([]byte(password))
	return fmt.Sprintf("%x", hash)
}

// checkPassword 验证密码
func checkPassword(password, hashedPassword string) bool {
	return hashPassword(password) == hashedPassword
}

// 演示函数
func main() {
	// 初始化日志
	clog.Info("=== 用户注册登录演示开始 ===")

	// 配置数据库连接
	cfg := db.MySQLConfig("root:mysql@tcp(localhost:3306)/gochat?charset=utf8mb4&parseTime=True&loc=Local").
		MaxOpenConns(10).
		MaxIdleConns(5).
		LogLevel("info").
		TablePrefix("demo_").
		Build()

	// 创建数据库实例
	database, err := db.New(cfg)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer database.Close()

	// 自动迁移
	err = database.GetDB().AutoMigrate(&User{})
	if err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	// 创建用户服务
	userService := NewUserService(database)
	ctx := context.Background()

	// 演示用户注册
	clog.Info("--- 演示用户注册 ---")
	user1, err := userService.Register(ctx, "alice", "alice@example.com", "password123")
	if err != nil {
		clog.Error("用户注册失败", clog.Err(err))
	} else {
		clog.Info("用户注册成功", clog.String("username", user1.Username))
	}

	user2, err := userService.Register(ctx, "bob", "bob@example.com", "password456")
	if err != nil {
		clog.Error("用户注册失败", clog.Err(err))
	} else {
		clog.Info("用户注册成功", clog.String("username", user2.Username))
	}

	// 演示用户登录
	clog.Info("--- 演示用户登录 ---")
	loginUser, err := userService.Login(ctx, "alice", "password123")
	if err != nil {
		clog.Error("用户登录失败", clog.Err(err))
	} else {
		clog.Info("用户登录成功", clog.String("username", loginUser.Username))
	}

	// 演示错误密码登录
	_, err = userService.Login(ctx, "alice", "wrongpassword")
	if err != nil {
		clog.Info("预期的登录失败", clog.Err(err))
	}

	// 演示获取用户信息
	clog.Info("--- 演示获取用户信息 ---")
	if user1 != nil {
		foundUser, err := userService.GetUserByID(ctx, user1.ID)
		if err != nil {
			clog.Error("获取用户失败", clog.Err(err))
		} else {
			clog.Info("获取用户成功", clog.String("username", foundUser.Username))
		}
	}

	// 演示用户列表
	clog.Info("--- 演示用户列表 ---")
	users, err := userService.ListUsers(ctx, 0, 10)
	if err != nil {
		clog.Error("获取用户列表失败", clog.Err(err))
	} else {
		clog.Info("获取用户列表成功", clog.Int("count", len(users)))
		for _, u := range users {
			clog.Info("用户", clog.String("username", u.Username), clog.String("email", u.Email))
		}
	}

	clog.Info("=== 演示完成 ===")
}
