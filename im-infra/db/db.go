package db

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db/internal"
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

// New 根据提供的配置创建一个新的 DB 实例。
// 这是创建数据库实例的唯一入口，移除了全局方法以推动依赖注入。
//
// 参数：
//   - ctx: 上下文，用于超时控制和取消操作
//   - cfg: 数据库配置
//   - opts: 可选配置项，支持注入 Logger 等依赖
//
// 示例：
//
// // 基础配置
// cfg := db.Config{
// DSN:             "root:mysql@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local",
// Driver:          "mysql",
// MaxOpenConns:    25,
// MaxIdleConns:    10,
// ConnMaxLifetime: time.Hour,
// LogLevel:        "warn",
// SlowThreshold:   200 * time.Millisecond,
// }
//
// database, err := db.New(ctx, cfg)
// if err != nil {
// log.Fatal(err)
// }
// defer database.Close()
//
// // 带 Logger 的配置
// logger := clog.Namespace("my-app")
// database, err := db.New(ctx, cfg, db.WithLogger(logger))
//
// // 带组件名称的配置
// database, err := db.New(ctx, cfg, db.WithComponentName("user-db"))
//
// // 获取 GORM 实例进行数据库操作
// gormDB := database.GetDB()
//
// // 自动迁移
// err = database.AutoMigrate(&User{})
// if err != nil {
// log.Fatal(err)
// }
//
// // 创建记录
// user := &User{Name: "Alice", Email: "alice@example.com"}
// result := gormDB.Create(user)
// if result.Error != nil {
// log.Fatal(result.Error)
// }
//
// // 分片配置示例
// shardingConfig := &db.ShardingConfig{
// ShardingKey:    "user_id",
// NumberOfShards: 16,
// Tables: map[string]*db.TableShardingConfig{
// "users":  {},
// "orders": {},
// },
// }
//
// cfg.Sharding = shardingConfig
// database, err := db.New(ctx, cfg)
func New(ctx context.Context, cfg Config, opts ...Option) (DB, error) {
	// 应用选项
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	// 设置默认 Logger
	var componentLogger clog.Logger
	if options.Logger != nil {
		if options.ComponentName != "" {
			componentLogger = options.Logger.With(clog.String("component", options.ComponentName))
		} else {
			componentLogger = options.Logger.With(clog.String("component", "db"))
		}
	} else {
		if options.ComponentName != "" {
			componentLogger = clog.Namespace("db").With(clog.String("name", options.ComponentName))
		} else {
			componentLogger = clog.Namespace("db")
		}
	}

	componentLogger.Info("创建数据库实例",
		clog.String("driver", cfg.Driver),
		clog.Int("maxOpenConns", cfg.MaxOpenConns),
		clog.Int("maxIdleConns", cfg.MaxIdleConns),
	)

	return internal.NewDB(cfg, componentLogger)
}

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置专门为 MySQL 优化，适用于大多数开发和生产场景。
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
//	database, err := db.New(ctx, cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
func DefaultConfig() Config {
	return internal.DefaultConfig()
}
