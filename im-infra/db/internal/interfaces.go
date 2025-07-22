package internal

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

// DB 定义数据库操作的核心接口。
// 提供对 GORM 实例的访问和基础连接管理功能。
type DB interface {
	// GetDB 返回原生的 GORM 数据库实例
	// 用户通过此方法获取 *gorm.DB 来编写具体的 ORM 代码
	GetDB() *gorm.DB

	// Ping 检查数据库连接是否正常
	Ping(ctx context.Context) error

	// Close 关闭数据库连接
	Close() error

	// Stats 返回数据库连接池统计信息
	Stats() sql.DBStats

	// WithContext 返回一个带有指定上下文的数据库实例
	WithContext(ctx context.Context) *gorm.DB

	// Transaction 执行事务操作
	Transaction(fn func(tx *gorm.DB) error) error

	// AutoMigrate 自动迁移数据库表结构
	AutoMigrate(dst ...interface{}) error

	// CreateDatabaseIfNotExists 如果数据库不存在则创建
	CreateDatabaseIfNotExists(dbName string) error
}
