package internal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
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
		c.logger.Error("获取底层数据库连接失败", clog.Err(err))
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	err = sqlDB.PingContext(ctx)
	if err != nil {
		c.logger.Error("数据库 Ping 失败", clog.Err(err))
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
		c.logger.Error("获取底层数据库连接失败", clog.Err(err))
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	err = sqlDB.Close()
	if err != nil {
		c.logger.Error("关闭数据库连接失败", clog.Err(err))
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	c.logger.Info("数据库连接已关闭")
	return nil
}

// Stats 返回数据库连接池统计信息
func (c *client) Stats() sql.DBStats {
	sqlDB, err := c.db.DB()
	if err != nil {
		c.logger.Error("获取底层数据库连接失败", clog.Err(err))
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
			clog.Err(err),
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
			clog.Err(err),
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

// CreateDatabaseIfNotExists 如果数据库不存在则创建（仅支持MySQL）
func (c *client) CreateDatabaseIfNotExists(dbName string) error {
	c.logger.Info("检查并创建MySQL数据库", clog.String("database", dbName))

	// 获取底层数据库连接
	sqlDB, err := c.db.DB()
	if err != nil {
		c.logger.Error("获取底层数据库连接失败", clog.Err(err))
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	// 只支持 MySQL
	if c.config.Driver != "mysql" {
		return fmt.Errorf("unsupported database driver for database creation: %s, only mysql is supported", c.config.Driver)
	}

	// MySQL 数据库创建语句
	createSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)

	// 执行创建数据库语句
	_, err = sqlDB.Exec(createSQL)
	if err != nil {
		c.logger.Error("创建MySQL数据库失败",
			clog.Err(err),
			clog.String("database", dbName),
			clog.String("sql", createSQL),
		)
		return fmt.Errorf("failed to create database: %w", err)
	}

	c.logger.Info("MySQL数据库创建成功", clog.String("database", dbName))
	return nil
}

// NewDB 根据提供的配置创建一个新的 DB 实例（仅支持MySQL）
func NewDB(cfg Config, logger clog.Logger) (DB, error) {
	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 只支持 MySQL
	if cfg.Driver != "mysql" {
		return nil, fmt.Errorf("unsupported database driver: %s, only mysql is supported", cfg.Driver)
	}

	logger.Info("创建MySQL数据库实例",
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
		Logger: NewClogLogger(logger, cfg),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: cfg.TablePrefix,
		},
		DisableForeignKeyConstraintWhenMigrating: cfg.DisableForeignKeyConstraintWhenMigrating,
	}

	// 创建MySQL数据库连接
	db, err := gorm.Open(mysql.Open(cfg.DSN), gormConfig)

	// 如果连接失败且启用了自动创建数据库，尝试创建数据库
	if err != nil && cfg.AutoCreateDatabase && isMySQLDatabaseNotExistError(err) {
		logger.Info("检测到MySQL数据库不存在，尝试自动创建",
			clog.String("driver", cfg.Driver),
			clog.Err(err),
		)

		// 解析MySQL数据库名称
		dbName, parseErr := parseMySQLDatabaseName(cfg.DSN)
		if parseErr != nil {
			logger.Error("解析MySQL数据库名称失败", clog.Err(parseErr))
			return nil, fmt.Errorf("failed to parse database name: %w", parseErr)
		}

		if dbName != "" {
			// 创建系统数据库连接DSN
			systemDSN := createMySQLSystemDSN(cfg.DSN)

			// 创建临时配置连接到MySQL系统数据库
			tempCfg := cfg
			tempCfg.DSN = systemDSN
			tempCfg.AutoCreateDatabase = false // 避免递归

			logger.Info("连接到MySQL系统数据库以创建目标数据库",
				clog.String("systemDSN", maskDSN(systemDSN)),
				clog.String("targetDatabase", dbName),
			)

			// 创建临时数据库连接
			tempDB, tempErr := NewDB(tempCfg, logger)
			if tempErr != nil {
				logger.Error("连接MySQL系统数据库失败", clog.Err(tempErr))
				return nil, fmt.Errorf("failed to connect to system database: %w", tempErr)
			}
			defer tempDB.Close()

			// 创建目标数据库
			createErr := tempDB.CreateDatabaseIfNotExists(dbName)
			if createErr != nil {
				logger.Error("自动创建MySQL数据库失败",
					clog.Err(createErr),
					clog.String("database", dbName),
				)
				return nil, fmt.Errorf("failed to auto-create database '%s': %w", dbName, createErr)
			}

			logger.Info("MySQL数据库自动创建成功，重新尝试连接",
				clog.String("database", dbName),
			)

			// 重新尝试连接到目标数据库
			db, err = gorm.Open(mysql.Open(cfg.DSN), gormConfig)
		}
	}

	if err != nil {
		logger.Error("MySQL数据库连接失败",
			clog.Err(err),
			clog.String("driver", cfg.Driver),
		)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	if err := configureConnectionPool(db, cfg, logger); err != nil {
		logger.Error("配置连接池失败", clog.Err(err))
		return nil, fmt.Errorf("failed to configure connection pool: %w", err)
	}

	// 配置分库分表（如果启用）
	if cfg.Sharding != nil {
		if err := configureSharding(db, cfg.Sharding); err != nil {
			logger.Error("配置分库分表失败", clog.Err(err))
			return nil, fmt.Errorf("failed to configure sharding: %w", err)
		}
	}

	logger.Info("MySQL数据库实例创建成功")

	// 创建客户端实例
	return newClient(db, cfg, logger), nil
}

// CreateDatabaseIfNotExistsWithConfig 使用指定配置创建MySQL数据库（如果不存在）
func CreateDatabaseIfNotExistsWithConfig(cfg Config, dbName string) error {
	// 创建一个默认的logger
	logger := clog.Namespace("db")

	logger.Info("使用指定配置创建MySQL数据库", clog.String("database", dbName))

	// 只支持 MySQL
	if cfg.Driver != "mysql" {
		return fmt.Errorf("unsupported database driver for database creation: %s, only mysql is supported", cfg.Driver)
	}

	// 修改配置，连接到 mysql 系统数据库而不是目标数据库
	tempCfg := cfg
	tempCfg.DSN = createMySQLSystemDSN(cfg.DSN)

	// 创建临时数据库连接
	tempDB, err := NewDB(tempCfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create temporary database connection: %w", err)
	}
	defer tempDB.Close()

	// 使用临时连接创建目标数据库
	return tempDB.CreateDatabaseIfNotExists(dbName)
}

// configureConnectionPool 配置数据库连接池
func configureConnectionPool(db *gorm.DB, cfg Config, logger clog.Logger) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	logger.Info("数据库连接池配置完成",
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

// parseMySQLDatabaseName 从MySQL DSN中解析数据库名称
func parseMySQLDatabaseName(dsn string) (string, error) {
	// MySQL DSN 格式: user:password@tcp(host:port)/dbname?params
	// 或者: user:password@/dbname?params
	parts := strings.Split(dsn, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid MySQL DSN format: %s", dsn)
	}

	dbPart := parts[len(parts)-1]
	// 移除查询参数
	if idx := strings.Index(dbPart, "?"); idx != -1 {
		dbPart = dbPart[:idx]
	}

	if dbPart == "" {
		return "", fmt.Errorf("database name not found in MySQL DSN: %s", dsn)
	}

	return dbPart, nil
}

// isMySQLDatabaseNotExistError 检查错误是否是"MySQL数据库不存在"错误
func isMySQLDatabaseNotExistError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// MySQL 错误码 1049: Unknown database
	return strings.Contains(errStr, "Unknown database") ||
		strings.Contains(errStr, "Error 1049")
}

// createMySQLSystemDSN 创建连接到MySQL系统数据库的DSN
func createMySQLSystemDSN(originalDSN string) string {
	// 将数据库名替换为 mysql 系统数据库
	parts := strings.Split(originalDSN, "/")
	if len(parts) < 2 {
		// 如果DSN格式不正确，返回一个默认的系统数据库DSN
		return "root:mysql@tcp(localhost:3306)/mysql?charset=utf8mb4&parseTime=True&loc=Local"
	}

	// 获取查询参数
	dbPart := parts[len(parts)-1]
	var params string
	if idx := strings.Index(dbPart, "?"); idx != -1 {
		params = dbPart[idx:]
	}

	// 重新构建DSN，使用mysql系统数据库
	parts[len(parts)-1] = "mysql" + params
	return strings.Join(parts, "/")
}

// newClient 创建一个新的数据库客户端实例
func newClient(db *gorm.DB, config Config, logger clog.Logger) DB {
	return &client{
		db:     db,
		config: config,
		logger: logger,
	}
}
