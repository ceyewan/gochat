package repository

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
	"github.com/ceyewan/gochat/im-repo/internal/config"
	"github.com/ceyewan/gochat/im-repo/internal/model"
	"gorm.io/gorm"
)

// Database 数据库管理器
type Database struct {
	db     db.DB
	config *config.Config
	logger clog.Logger
}

// NewDatabase 创建数据库管理器
func NewDatabase(cfg *config.Config) (*Database, error) {
	logger := clog.Module("database")

	// 创建数据库连接
	database, err := db.New(context.Background(), cfg.Database, db.WithLogger(logger))
	if err != nil {
		logger.Error("创建数据库连接失败", clog.Err(err))
		return nil, fmt.Errorf("创建数据库连接失败: %w", err)
	}

	dbManager := &Database{
		db:     database,
		config: cfg,
		logger: logger,
	}

	logger.Info("数据库连接创建成功")
	return dbManager, nil
}

// GetDB 获取数据库连接
func (d *Database) GetDB() *gorm.DB {
	return d.db.GetDB()
}

// Migrate 执行数据库迁移
func (d *Database) Migrate(ctx context.Context) error {
	d.logger.Info("开始执行数据库迁移...")

	// 自动迁移所有模型
	err := d.db.AutoMigrate(
		&model.User{},
		&model.Group{},
		&model.GroupMember{},
		&model.Message{},
		&model.UserReadPointer{},
	)

	if err != nil {
		d.logger.Error("数据库迁移失败", clog.Err(err))
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	d.logger.Info("数据库迁移完成")
	return nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	d.logger.Info("关闭数据库连接")
	return d.db.Close()
}

// Ping 检查数据库连接
func (d *Database) Ping(ctx context.Context) error {
	return d.db.Ping(ctx)
}

// Transaction 执行事务
func (d *Database) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return d.db.Transaction(fn)
}
