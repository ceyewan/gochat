package internal

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
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

	// 如果连接失败且启用了自动创建数据库，尝试创建数据库
	if err != nil && cfg.AutoCreateDatabase && isDatabaseNotExistError(err, cfg.Driver) {
		moduleLogger.Info("检测到数据库不存在，尝试自动创建",
			clog.String("driver", cfg.Driver),
			clog.ErrorValue(err),
		)

		// 解析数据库名称
		dbName, parseErr := parseDatabaseName(cfg.DSN, cfg.Driver)
		if parseErr != nil {
			moduleLogger.Error("解析数据库名称失败", clog.ErrorValue(parseErr))
			return nil, fmt.Errorf("failed to parse database name: %w", parseErr)
		}

		if dbName != "" { // SQLite 返回空字符串，不需要创建
			// 创建系统数据库连接DSN
			systemDSN, systemErr := createSystemDSN(cfg.DSN, cfg.Driver)
			if systemErr != nil {
				moduleLogger.Error("创建系统数据库DSN失败", clog.ErrorValue(systemErr))
				return nil, fmt.Errorf("failed to create system DSN: %w", systemErr)
			}

			// 创建临时配置连接到系统数据库
			tempCfg := cfg
			tempCfg.DSN = systemDSN
			tempCfg.AutoCreateDatabase = false // 避免递归

			moduleLogger.Info("连接到系统数据库以创建目标数据库",
				clog.String("systemDSN", maskDSN(systemDSN)),
				clog.String("targetDatabase", dbName),
			)

			// 创建临时数据库连接
			tempDB, tempErr := NewDB(tempCfg)
			if tempErr != nil {
				moduleLogger.Error("连接系统数据库失败", clog.ErrorValue(tempErr))
				return nil, fmt.Errorf("failed to connect to system database: %w", tempErr)
			}
			defer tempDB.Close()

			// 创建目标数据库
			createErr := tempDB.CreateDatabaseIfNotExists(dbName)
			if createErr != nil {
				moduleLogger.Error("自动创建数据库失败",
					clog.ErrorValue(createErr),
					clog.String("database", dbName),
				)
				return nil, fmt.Errorf("failed to auto-create database '%s': %w", dbName, createErr)
			}

			moduleLogger.Info("数据库自动创建成功，重新尝试连接",
				clog.String("database", dbName),
			)

			// 重新尝试连接到目标数据库
			switch cfg.Driver {
			case "mysql":
				db, err = gorm.Open(mysql.Open(cfg.DSN), gormConfig)
			case "postgres":
				db, err = gorm.Open(postgres.Open(cfg.DSN), gormConfig)
			case "sqlite":
				db, err = gorm.Open(sqlite.Open(cfg.DSN), gormConfig)
			}
		}
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

// parseDatabaseName 从DSN中解析数据库名称
func parseDatabaseName(dsn, driver string) (string, error) {
	switch driver {
	case "mysql":
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

	case "postgres":
		// PostgreSQL DSN 格式: host=localhost user=user password=pass dbname=dbname sslmode=disable
		// 或者: postgres://user:pass@host:port/dbname?sslmode=disable
		if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
			// URL 格式
			u, err := url.Parse(dsn)
			if err != nil {
				return "", fmt.Errorf("invalid PostgreSQL URL format: %w", err)
			}
			dbName := strings.TrimPrefix(u.Path, "/")
			if dbName == "" {
				return "", fmt.Errorf("database name not found in PostgreSQL URL: %s", dsn)
			}
			return dbName, nil
		} else {
			// 键值对格式
			re := regexp.MustCompile(`dbname=([^\s]+)`)
			matches := re.FindStringSubmatch(dsn)
			if len(matches) < 2 {
				return "", fmt.Errorf("database name not found in PostgreSQL DSN: %s", dsn)
			}
			return matches[1], nil
		}

	case "sqlite":
		// SQLite DSN 就是文件路径，不需要创建数据库
		return "", nil

	default:
		return "", fmt.Errorf("unsupported driver for database name parsing: %s", driver)
	}
}

// isDatabaseNotExistError 检查错误是否是"数据库不存在"错误
func isDatabaseNotExistError(err error, driver string) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	switch driver {
	case "mysql":
		// MySQL 错误码 1049: Unknown database
		return strings.Contains(errStr, "Unknown database") ||
			strings.Contains(errStr, "Error 1049")

	case "postgres":
		// PostgreSQL 错误码 3D000: database does not exist
		return strings.Contains(errStr, "database") &&
			strings.Contains(errStr, "does not exist")

	case "sqlite":
		// SQLite 文件会自动创建，不会有数据库不存在的错误
		return false

	default:
		return false
	}
}

// createSystemDSN 创建连接到系统数据库的DSN
func createSystemDSN(originalDSN, driver string) (string, error) {
	switch driver {
	case "mysql":
		// 将数据库名替换为 mysql 系统数据库
		parts := strings.Split(originalDSN, "/")
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid MySQL DSN format: %s", originalDSN)
		}

		// 获取查询参数
		dbPart := parts[len(parts)-1]
		var params string
		if idx := strings.Index(dbPart, "?"); idx != -1 {
			params = dbPart[idx:]
		}

		// 重新构建DSN，使用mysql系统数据库
		parts[len(parts)-1] = "mysql" + params
		return strings.Join(parts, "/"), nil

	case "postgres":
		if strings.HasPrefix(originalDSN, "postgres://") || strings.HasPrefix(originalDSN, "postgresql://") {
			// URL 格式
			u, err := url.Parse(originalDSN)
			if err != nil {
				return "", fmt.Errorf("invalid PostgreSQL URL format: %w", err)
			}
			u.Path = "/postgres"
			return u.String(), nil
		} else {
			// 键值对格式，替换dbname
			re := regexp.MustCompile(`dbname=([^\s]+)`)
			return re.ReplaceAllString(originalDSN, "dbname=postgres"), nil
		}

	case "sqlite":
		// SQLite 不需要系统数据库
		return originalDSN, nil

	default:
		return "", fmt.Errorf("unsupported driver for system DSN creation: %s", driver)
	}
}

// newClient 创建一个新的数据库客户端实例
func newClient(db *gorm.DB, config Config, logger clog.Logger) DB {
	return &client{
		db:     db,
		config: config,
		logger: logger,
	}
}
