package db

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db/internal"
)

// Provider 提供了访问数据库的能力。
// 这是用户应该使用的接口，符合文档设计。
type Provider = internal.Provider

// Config 是 db 的主配置结构体。
// 用于声明式地定义数据库连接和行为参数。
type Config = internal.Config

// ShardingConfig 分库分表配置
type ShardingConfig = internal.ShardingConfig

// TableShardingConfig 表分片配置
type TableShardingConfig = internal.TableShardingConfig

// New 根据提供的配置创建一个新的 Provider 实例。
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
// gormDB := database.DB(ctx)
//
// // 自动迁移
// err = database.AutoMigrate(ctx, &User{})
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
func New(ctx context.Context, cfg Config, opts ...Option) (Provider, error) {
	// 应用选项
	p := &provider{
		logger:       clog.Namespace("db"),
		componentName: "db",
	}

	for _, opt := range opts {
		opt(p)
	}

	// 设置组件日志器
	componentLogger := p.logger
	if p.componentName != "" && p.componentName != "db" {
		componentLogger = p.logger.With(clog.String("component", p.componentName))
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

// GetDefaultConfig 返回默认的数据库配置。
// 开发环境：较少连接数，较详细的日志级别，较短的超时时间
// 生产环境：较多连接数，较少的日志输出，较长的连接生命周期
func GetDefaultConfig(env string) Config {
	return internal.GetDefaultConfig(env)
}
