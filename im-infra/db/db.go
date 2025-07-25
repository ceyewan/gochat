package db

import (
	"context"
	"database/sql"
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db/internal"
	"gorm.io/gorm"
)

// DB 定义数据库操作的核心接口。
// 提供对 GORM 实例的访问和基础连接管理功能。
type DB = internal.DB

// Config 是 db 的主配置结构体。
// 用于声明式地定义数据库连接和行为参数。
type Config = internal.Config

// ShardingConfig 分库分表配置
type ShardingConfig = internal.ShardingConfig

// TableShardingConfig 表分片配置
type TableShardingConfig = internal.TableShardingConfig

var (
	// 全局默认数据库实例
	defaultDB DB
	// 确保默认数据库只初始化一次
	defaultDBOnce sync.Once
	// 模块日志器
	logger = clog.Module("db")
)

// getDefaultDB 获取全局默认数据库实例，使用懒加载和单例模式
func getDefaultDB() DB {
	defaultDBOnce.Do(func() {
		cfg := DefaultConfig()
		var err error
		defaultDB, err = internal.NewDB(cfg)
		if err != nil {
			logger.Error("创建默认数据库实例失败", clog.Err(err))
			panic(err)
		}
	})
	return defaultDB
}

// New 根据提供的配置创建一个新的 DB 实例。
// 用于自定义数据库实例的主要入口。
//
// 示例：
//
//	// MySQL 配置
//	cfg := db.Config{
//		DSN:             "root:mysql@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local",
//		Driver:          "mysql",
//		MaxOpenConns:    25,
//		MaxIdleConns:    10,
//		ConnMaxLifetime: time.Hour,
//		LogLevel:        "warn",
//		SlowThreshold:   200 * time.Millisecond,
//	}
//
//	database, err := db.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer database.Close()
//
//	// 获取 GORM 实例进行数据库操作
//	gormDB := database.GetDB()
//
//	// 自动迁移
//	err = database.AutoMigrate(&User{})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 创建记录
//	user := &User{Name: "Alice", Email: "alice@example.com"}
//	result := gormDB.Create(user)
//	if result.Error != nil {
//		log.Fatal(result.Error)
//	}
func New(cfg Config) (DB, error) {
	return internal.NewDB(cfg)
}

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置适用于大多数开发和测试场景。
//
// 示例：
//
//	// 使用默认配置
//	cfg := db.DefaultConfig()
//
//	// 可以根据需要修改特定配置
//	cfg.DSN = "root:mysql@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local"
//	cfg.LogLevel = "info"
//	cfg.MaxOpenConns = 50
//
//	database, err := db.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// GetDB 使用全局默认数据库实例获取 GORM 数据库对象
func GetDB() *gorm.DB {
	return getDefaultDB().GetDB()
}

// Ping 使用全局默认数据库实例检查数据库连接
func Ping(ctx context.Context) error {
	return getDefaultDB().Ping(ctx)
}

// Close 关闭全局默认数据库连接
func Close() error {
	if defaultDB != nil {
		return defaultDB.Close()
	}
	return nil
}

// Stats 获取全局默认数据库连接池统计信息
func Stats() sql.DBStats {
	return getDefaultDB().Stats()
}

// WithContext 使用全局默认数据库实例返回带有指定上下文的数据库实例
func WithContext(ctx context.Context) *gorm.DB {
	return getDefaultDB().WithContext(ctx)
}

// Transaction 使用全局默认数据库实例执行事务操作
//
// 示例：
//
//	// 在事务中执行多个操作
//	err := db.Transaction(func(tx *gorm.DB) error {
//		// 创建用户
//		user := &User{Name: "Alice", Email: "alice@example.com"}
//		if err := tx.Create(user).Error; err != nil {
//			return err // 事务会自动回滚
//		}
//
//		// 创建用户资料
//		profile := &Profile{UserID: user.ID, Bio: "Software Engineer"}
//		if err := tx.Create(profile).Error; err != nil {
//			return err // 事务会自动回滚
//		}
//
//		return nil // 事务提交
//	})
//
//	if err != nil {
//		log.Printf("事务执行失败: %v", err)
//	}
func Transaction(fn func(tx *gorm.DB) error) error {
	return getDefaultDB().Transaction(fn)
}

// AutoMigrate 使用全局默认数据库实例自动迁移数据库表结构
//
// 示例：
//
//	// 定义模型
//	type User struct {
//		ID        uint      `gorm:"primaryKey"`
//		Name      string    `gorm:"size:100;not null"`
//		Email     string    `gorm:"uniqueIndex;size:100"`
//		CreatedAt time.Time
//		UpdatedAt time.Time
//	}
//
//	type Profile struct {
//		ID     uint   `gorm:"primaryKey"`
//		UserID uint   `gorm:"not null"`
//		Bio    string `gorm:"type:text"`
//		User   User   `gorm:"foreignKey:UserID"`
//	}
//
//	// 自动迁移多个模型
//	err := db.AutoMigrate(&User{}, &Profile{})
//	if err != nil {
//		log.Fatal("数据库迁移失败:", err)
//	}
func AutoMigrate(dst ...interface{}) error {
	return getDefaultDB().AutoMigrate(dst...)
}

// CreateDatabaseIfNotExists 使用全局默认数据库实例创建数据库（如果不存在）
// 注意：这个方法要求目标数据库已经存在才能工作
func CreateDatabaseIfNotExists(dbName string) error {
	return getDefaultDB().CreateDatabaseIfNotExists(dbName)
}
