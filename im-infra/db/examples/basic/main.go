package main

import (
	"context"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
)

// User 用户模型
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"uniqueIndex;size:100;not null"`
	Email     string `gorm:"size:100"`
	Age       int    `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	ctx := context.Background()

	// 创建自定义日志器
	logger := clog.Module("db-basic-example")

	// 创建 MySQL 配置
	cfg := db.MySQLConfig("gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local")

	// 优化配置
	cfg.MaxOpenConns = 25
	cfg.MaxIdleConns = 10
	cfg.ConnMaxLifetime = time.Hour
	cfg.ConnMaxIdleTime = 30 * time.Minute

	// 使用 New 函数创建数据库实例，并注入 Logger
	database, err := db.New(ctx, cfg, db.WithLogger(logger), db.WithComponentName("basic-example"))
	if err != nil {
		log.Fatalf("创建数据库实例失败: %v", err)
	}
	defer database.Close()

	logger.Info("数据库连接创建成功")

	// 检查连接
	if err := database.Ping(ctx); err != nil {
		log.Fatalf("数据库连接检查失败: %v", err)
	}
	logger.Info("数据库连接检查通过")

	// 自动迁移
	if err := database.AutoMigrate(&User{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	logger.Info("数据库迁移完成")

	// 获取 GORM 实例进行操作
	gormDB := database.GetDB()

	// 创建用户
	user := &User{
		Username: "alice",
		Email:    "alice@example.com",
		Age:      25,
	}

	if err := gormDB.WithContext(ctx).Create(user).Error; err != nil {
		logger.Error("创建用户失败", clog.Err(err))
	} else {
		logger.Info("用户创建成功", clog.Uint("userID", user.ID))
	}

	// 查询用户
	var foundUser User
	if err := gormDB.WithContext(ctx).Where("username = ?", "alice").First(&foundUser).Error; err != nil {
		logger.Error("查询用户失败", clog.Err(err))
	} else {
		logger.Info("查询到用户",
			clog.Uint("id", foundUser.ID),
			clog.String("username", foundUser.Username),
			clog.String("email", foundUser.Email))
	}

	// 更新用户
	if err := gormDB.WithContext(ctx).Model(&foundUser).Update("age", 26).Error; err != nil {
		logger.Error("更新用户失败", clog.Err(err))
	} else {
		logger.Info("用户更新成功")
	}

	// 查询所有用户
	var users []User
	if err := gormDB.WithContext(ctx).Find(&users).Error; err != nil {
		logger.Error("查询所有用户失败", clog.Err(err))
	} else {
		logger.Info("查询到用户列表", clog.Int("count", len(users)))
		for _, u := range users {
			logger.Info("用户信息",
				clog.Uint("id", u.ID),
				clog.String("username", u.Username),
				clog.Int("age", u.Age))
		}
	}

	// 删除用户
	if err := gormDB.WithContext(ctx).Delete(&foundUser).Error; err != nil {
		logger.Error("删除用户失败", clog.Err(err))
	} else {
		logger.Info("用户删除成功")
	}

	// 获取连接池统计信息
	stats := database.Stats()
	logger.Info("连接池统计信息",
		clog.Int("openConnections", stats.OpenConnections),
		clog.Int("inUse", stats.InUse),
		clog.Int("idle", stats.Idle),
		clog.Int64("waitCount", stats.WaitCount))

	logger.Info("基础示例运行完成")
}
