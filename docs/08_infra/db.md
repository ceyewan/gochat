# 基础设施: DB 数据库

## 1. 概述

`db` 是 `gochat` 项目的数据库基础设施模块，基于 `GORM v2` 构建。它是一个**专注于 MySQL** 的、以**分库分表**为核心设计的高性能数据库操作层。

`db` 组件的设计哲学是**“封装便利，但不隐藏能力”**。它封装了数据库连接、配置、事务和分片等复杂逻辑，同时通过 `GetDB()` 方法提供了对原生 `*gorm.DB` 的完全访问，让开发者可以利用 GORM 的全部功能。

## 2. 核心用法

### 2.1 初始化

`db.New` 是创建数据库实例的唯一入口。

```go
import "github.com/ceyewan/gochat/im-infra/db"

// 1. 创建一个基础的 MySQL 配置
// DSN (Data Source Name) 是标准的数据库连接字符串
cfg := db.MySQLConfig("root:mysql@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local")

// 2. （可选）配置连接池参数
cfg.MaxOpenConns = 50
cfg.MaxIdleConns = 25
cfg.ConnMaxLifetime = time.Hour

// 3. 创建数据库实例，并注入 logger
logger := clog.Module("db-example")
database, err := db.New(ctx, cfg, db.WithLogger(logger))
if err != nil {
    log.Fatalf("无法创建数据库实例: %v", err)
}
defer database.Close()
```

### 2.2 基本操作 (通过 GORM)

`db` 组件的核心是让你直接使用 GORM。

```go
// 1. 获取原生的 GORM DB 实例
gormDB := database.GetDB()

// 2. 定义 GORM 模型
type User struct {
    ID       uint   `gorm:"primaryKey"`
    Username string `gorm:"uniqueIndex"`
}

// 3. 自动迁移表结构
err = database.AutoMigrate(&User{})
if err != nil {
    // ...
}

// 4. 使用 GORM 进行 CRUD 操作
user := &User{Username: "alice"}
result := gormDB.WithContext(ctx).Create(user)
if result.Error != nil {
    // ...
}

var foundUser User
gormDB.WithContext(ctx).First(&foundUser, "username = ?", "alice")
```

### 2.3 事务操作

`db` 组件提供了便捷的事务封装。

```go
err := database.Transaction(func(tx *gorm.DB) error {
    // tx 是一个带事务的 *gorm.DB 实例
    if err := tx.Create(&User{Username: "bob"}).Error; err != nil {
        // 返回任意 error 都会导致事务回滚
        return err
    }
    if err := tx.Create(&User{Username: "charlie"}).Error; err != nil {
        return err
    }
    // 函数正常返回，事务会自动提交
    return nil
})
```

### 2.4 分库分表

这是 `db` 组件最核心的功能。

```go
// 1. 定义分片模型
type Message struct {
    ID       uint64 `gorm:"primaryKey"`
    UserID   uint64 `gorm:"index"` // 使用 UserID 作为分片键
    Content  string
}

// 2. 创建分片配置
shardingConfig := db.NewShardingConfig(
    "user_id", // 分片键字段名
    16,        // 分成 16 个表 (message_00 到 message_15)
)
// 注册需要分片的表
shardingConfig.Tables["messages"] = &db.TableShardingConfig{}

// 3. 将分片配置应用到主配置
cfg.Sharding = shardingConfig

// 4. 初始化带分片的数据库实例
shardedDB, _ := db.New(ctx, cfg)

// 5. GORM 操作会自动路由到正确的分片
// GORM 会自动将这条记录插入到 `messages_XX` 中的某一张表
err = shardedDB.GetDB().Create(&Message{UserID: 12345, Content: "hello"}).Error

// 查询时必须带上分片键
var messages []Message
err = shardedDB.GetDB().Where("user_id = ?", 12345).Find(&messages).Error
```

## 3. API 参考

```go
// DB 定义了数据库操作的核心接口。
type DB interface {
	// GetDB 获取原生的 GORM 实例，用于执行所有 CRUD 操作。
	GetDB() *gorm.DB
	// WithContext 返回一个带 context 的 GORM 实例。
	WithContext(ctx context.Context) *gorm.DB
	// Transaction 执行一个数据库事务。
	Transaction(fn func(tx *gorm.DB) error) error
	// AutoMigrate 自动迁移数据库表结构，支持分片表的创建。
	AutoMigrate(dst ...interface{}) error

	// Ping 检查数据库连接。
	Ping(ctx context.Context) error
	// Close 关闭数据库连接。
	Close() error
	// Stats 获取数据库连接池的统计信息。
	Stats() sql.DBStats
}

// New 是创建数据库实例的唯一入口。
func New(ctx context.Context, cfg Config, opts ...Option) (DB, error)

// Config 是 db 的主配置结构体。
type Config struct {
	DSN             string
	Driver          string // 仅支持 "mysql"
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	LogLevel        string // GORM 日志级别
	SlowThreshold   time.Duration
	Sharding        *ShardingConfig // 分片配置
}

// ShardingConfig 定义了分库分表配置。
type ShardingConfig struct {
	ShardingKey       string
	NumberOfShards    int
	Tables            map[string]*TableShardingConfig
}