package internal

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

// DB 定义数据库操作的内部接口。
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

	// transactionInternal 执行事务操作（内部方法）
	transactionInternal(fn func(tx *gorm.DB) error) error

	// autoMigrateInternal 自动迁移数据库表结构
	autoMigrateInternal(dst ...interface{}) error

	// CreateDatabaseIfNotExists 如果数据库不存在则创建
	CreateDatabaseIfNotExists(dbName string) error
}

// Provider 提供了访问数据库的能力。
// 这是用户应该使用的接口，符合文档设计。
type Provider interface {
	// DB 从当前请求的上下文中获取一个 gorm.DB 实例用于执行查询。
	// 返回的 *gorm.DB 实例是轻量级且无状态的，应在需要时调用此方法获取，不要长期持有。
	DB(ctx context.Context) *gorm.DB

	// Transaction 执行一个数据库事务。
	// 传入的 ctx 会被自动应用到事务实例 tx 上，使用者无需再次调用 tx.WithContext(ctx)。
	// 回调函数中的任何 error 都会导致事务回滚。
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error

	// AutoMigrate 自动迁移数据库表结构，能正确处理分片表的创建。
	AutoMigrate(ctx context.Context, dst ...interface{}) error

	// Ping 检查数据库连接。
	Ping(ctx context.Context) error

	// Close 关闭数据库连接池。
	Close() error
}
