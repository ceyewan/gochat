package internal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var (
	// 模块日志器
	moduleLogger = clog.Module("db")
)

// client 是 DB 接口的内部实现。
// 它包装了一个 *gorm.DB，并提供接口方法。
type client struct {
	db     *gorm.DB
	config Config
	logger clog.Logger
}

// GetDB 返回原生的 GORM 数据库实例
func (c *client) GetDB() *gorm.DB {
	return c.db
}

// Ping 检查数据库连接是否正常
func (c *client) Ping(ctx context.Context) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.Debug("数据库 Ping 操作完成",
			clog.Duration("duration", duration),
		)
	}()

	sqlDB, err := c.db.DB()
	if err != nil {
		c.logger.Error("获取底层数据库连接失败", clog.ErrorValue(err))
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	err = sqlDB.PingContext(ctx)
	if err != nil {
		c.logger.Error("数据库 Ping 失败", clog.ErrorValue(err))
		return fmt.Errorf("database ping failed: %w", err)
	}

	c.logger.Debug("数据库 Ping 成功")
	return nil
}

// Close 关闭数据库连接
func (c *client) Close() error {
	c.logger.Info("正在关闭数据库连接")

	sqlDB, err := c.db.DB()
	if err != nil {
		c.logger.Error("获取底层数据库连接失败", clog.ErrorValue(err))
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	err = sqlDB.Close()
	if err != nil {
		c.logger.Error("关闭数据库连接失败", clog.ErrorValue(err))
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	c.logger.Info("数据库连接已关闭")
	return nil
}

// Stats 返回数据库连接池统计信息
func (c *client) Stats() sql.DBStats {
	sqlDB, err := c.db.DB()
	if err != nil {
		c.logger.Error("获取底层数据库连接失败", clog.ErrorValue(err))
		return sql.DBStats{}
	}

	stats := sqlDB.Stats()

	c.logger.Debug("数据库连接池统计信息",
		clog.Int("openConnections", stats.OpenConnections),
		clog.Int("inUse", stats.InUse),
		clog.Int("idle", stats.Idle),
		clog.Int64("waitCount", stats.WaitCount),
		clog.Duration("waitDuration", stats.WaitDuration),
		clog.Int64("maxIdleClosed", stats.MaxIdleClosed),
		clog.Int64("maxIdleTimeClosed", stats.MaxIdleTimeClosed),
		clog.Int64("maxLifetimeClosed", stats.MaxLifetimeClosed),
	)

	return stats
}

// WithContext 返回一个带有指定上下文的数据库实例
func (c *client) WithContext(ctx context.Context) *gorm.DB {
	return c.db.WithContext(ctx)
}

// Transaction 执行事务操作
func (c *client) Transaction(fn func(tx *gorm.DB) error) error {
	start := time.Now()

	c.logger.Debug("开始数据库事务")

	err := c.db.Transaction(fn)

	duration := time.Since(start)

	if err != nil {
		c.logger.Error("数据库事务失败",
			clog.ErrorValue(err),
			clog.Duration("duration", duration),
		)
		return err
	}

	c.logger.Debug("数据库事务成功完成",
		clog.Duration("duration", duration),
	)

	return nil
}

// AutoMigrate 自动迁移数据库表结构
func (c *client) AutoMigrate(dst ...interface{}) error {
	start := time.Now()

	c.logger.Info("开始数据库自动迁移")

	err := c.db.AutoMigrate(dst...)

	duration := time.Since(start)

	if err != nil {
		c.logger.Error("数据库自动迁移失败",
			clog.ErrorValue(err),
			clog.Duration("duration", duration),
		)
		return fmt.Errorf("auto migrate failed: %w", err)
	}

	c.logger.Info("数据库自动迁移成功完成",
		clog.Duration("duration", duration),
		clog.Int("models", len(dst)),
	)

	return nil
}

// CreateDatabaseIfNotExists 如果数据库不存在则创建
func (c *client) CreateDatabaseIfNotExists(dbName string) error {
	c.logger.Info("检查并创建数据库", clog.String("database", dbName))

	// 获取底层数据库连接
	sqlDB, err := c.db.DB()
	if err != nil {
		c.logger.Error("获取底层数据库连接失败", clog.ErrorValue(err))
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	// 根据数据库类型执行不同的创建语句
	var createSQL string
	switch c.config.Driver {
	case "mysql":
		createSQL = fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)
	case "postgres":
		// PostgreSQL 需要先检查数据库是否存在
		var exists bool
		checkSQL := "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)"
		err = sqlDB.QueryRow(checkSQL, dbName).Scan(&exists)
		if err != nil {
			c.logger.Error("检查 PostgreSQL 数据库是否存在失败", clog.ErrorValue(err))
			return fmt.Errorf("failed to check if database exists: %w", err)
		}

		if !exists {
			createSQL = fmt.Sprintf("CREATE DATABASE \"%s\"", dbName)
		} else {
			c.logger.Info("数据库已存在", clog.String("database", dbName))
			return nil
		}
	case "sqlite":
		// SQLite 不需要创建数据库，文件会自动创建
		c.logger.Info("SQLite 数据库文件会自动创建", clog.String("database", dbName))
		return nil
	default:
		return fmt.Errorf("unsupported database driver for database creation: %s", c.config.Driver)
	}

	// 执行创建数据库语句
	_, err = sqlDB.Exec(createSQL)
	if err != nil {
		c.logger.Error("创建数据库失败",
			clog.ErrorValue(err),
			clog.String("database", dbName),
			clog.String("sql", createSQL),
		)
		return fmt.Errorf("failed to create database: %w", err)
	}

	c.logger.Info("数据库创建成功", clog.String("database", dbName))
	return nil
}

// NewDB 根据提供的配置创建一个新的 DB 实例。
// 这是核心工厂函数，按配置组装所有组件。
func NewDB(cfg Config) (DB, error) {
	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	moduleLogger.Info("创建数据库实例",
		clog.String("driver", cfg.Driver),
		clog.String("dsn", maskDSN(cfg.DSN)),
		clog.Int("maxOpenConns", cfg.MaxOpenConns),
		clog.Int("maxIdleConns", cfg.MaxIdleConns),
		clog.Duration("connMaxLifetime", cfg.ConnMaxLifetime),
		clog.Duration("connMaxIdleTime", cfg.ConnMaxIdleTime),
		clog.String("logLevel", cfg.LogLevel),
		clog.Duration("slowThreshold", cfg.SlowThreshold),
	)

	// 创建 GORM 配置
	gormConfig := &gorm.Config{
		Logger: NewClogLogger(moduleLogger, cfg),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: cfg.TablePrefix,
		},
		DisableForeignKeyConstraintWhenMigrating: cfg.DisableForeignKeyConstraintWhenMigrating,
	}

	// 根据驱动类型创建数据库连接
	var db *gorm.DB
	var err error

	switch cfg.Driver {
	case "mysql":
		db, err = gorm.Open(mysql.Open(cfg.DSN), gormConfig)
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DSN), gormConfig)
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DSN), gormConfig)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	if err != nil {
		moduleLogger.Error("数据库连接失败",
			clog.ErrorValue(err),
			clog.String("driver", cfg.Driver),
		)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	if err := configureConnectionPool(db, cfg); err != nil {
		moduleLogger.Error("配置连接池失败", clog.ErrorValue(err))
		return nil, fmt.Errorf("failed to configure connection pool: %w", err)
	}

	// 配置分库分表（如果启用）
	if cfg.Sharding != nil {
		if err := configureSharding(db, cfg.Sharding); err != nil {
			moduleLogger.Error("配置分库分表失败", clog.ErrorValue(err))
			return nil, fmt.Errorf("failed to configure sharding: %w", err)
		}
	}

	moduleLogger.Info("数据库实例创建成功")

	// 创建客户端实例
	return newClient(db, cfg, moduleLogger), nil
}

// CreateDatabaseIfNotExistsWithConfig 使用指定配置创建数据库（如果不存在）
// 这个函数不依赖于全局数据库实例，专门用于创建数据库
func CreateDatabaseIfNotExistsWithConfig(cfg Config, dbName string) error {
	moduleLogger.Info("使用指定配置创建数据库", clog.String("database", dbName))

	// 修改配置，连接到 mysql 系统数据库而不是目标数据库
	tempCfg := cfg
	switch cfg.Driver {
	case "mysql":
		// 连接到 mysql 系统数据库
		tempCfg.DSN = "root:mysql@tcp(localhost:3306)/mysql?charset=utf8mb4&parseTime=True&loc=Local"
	case "postgres":
		// 连接到 postgres 系统数据库
		tempCfg.DSN = "host=localhost user=root password=mysql dbname=postgres sslmode=disable"
	case "sqlite":
		// SQLite 不需要创建数据库
		moduleLogger.Info("SQLite 数据库文件会自动创建", clog.String("database", dbName))
		return nil
	default:
		return fmt.Errorf("unsupported database driver for database creation: %s", cfg.Driver)
	}

	// 创建临时数据库连接
	tempDB, err := NewDB(tempCfg)
	if err != nil {
		return fmt.Errorf("failed to create temporary database connection: %w", err)
	}
	defer tempDB.Close()

	// 使用临时连接创建目标数据库
	return tempDB.CreateDatabaseIfNotExists(dbName)
}

// configureConnectionPool 配置数据库连接池
func configureConnectionPool(db *gorm.DB, cfg Config) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	moduleLogger.Info("数据库连接池配置完成",
		clog.Int("maxOpenConns", cfg.MaxOpenConns),
		clog.Int("maxIdleConns", cfg.MaxIdleConns),
		clog.Duration("connMaxLifetime", cfg.ConnMaxLifetime),
		clog.Duration("connMaxIdleTime", cfg.ConnMaxIdleTime),
	)

	return nil
}

// maskDSN 遮蔽 DSN 中的敏感信息用于日志记录
func maskDSN(dsn string) string {
	// 简单的遮蔽实现，实际项目中可能需要更复杂的逻辑
	if len(dsn) > 20 {
		return dsn[:10] + "***" + dsn[len(dsn)-7:]
	}
	return "***"
}

// newClient 创建一个新的数据库客户端实例
func newClient(db *gorm.DB, config Config, logger clog.Logger) DB {
	return &client{
		db:     db,
		config: config,
		logger: logger,
	}
}
