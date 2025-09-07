# 基础设施: DB 数据库

## 1. 设计理念

`db` 是 `gochat` 项目的数据库基础设施组件，基于 `GORM v2` 构建。它是一个**专注于 MySQL** 的、以**分库分表**为核心设计的高性能数据库操作层。

`db` 组件的设计哲学是 **“封装便利，但不隐藏能力”**。它封装了数据库连接、配置、事务和分片等复杂逻辑，同时通过 `DB()` 方法提供了对原生 `*gorm.DB` 的完全访问，让开发者可以利用 GORM 的全部功能，保证了灵活性。

## 2. 核心 API 契约

### 2.1 构造函数

```go
// Config 是 db 组件的主配置结构体。
type Config struct {
	DSN             string          `json:"dsn"`
	Driver          string          `json:"driver"` // 仅支持 "mysql"
	MaxOpenConns    int             `json:"maxOpenConns"`
	MaxIdleConns    int             `json:"maxIdleConns"`
	ConnMaxLifetime time.Duration   `json:"connMaxLifetime"`
	LogLevel        string          `json:"logLevel"` // GORM 日志级别
	SlowThreshold   time.Duration   `json:"slowThreshold"`
	Sharding        *ShardingConfig `json:"sharding"` // 分片配置
}

// ShardingConfig 定义了分库分表配置。
type ShardingConfig struct {
	ShardingKey    string                     `json:"shardingKey"`
	NumberOfShards int                        `json:"numberOfShards"`
	Tables         map[string]*TableShardingConfig `json:"tables"`
}

// New 是创建数据库 Provider 实例的唯一入口。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Provider 接口

`Provider` 接口定义了数据库操作的核心能力。

```go
// Provider 提供了访问数据库的能力。
type Provider interface {
	// DB 获取一个 gorm.DB 实例用于执行查询。
	// 【重要】: 在执行操作前，应调用 .WithContext(ctx) 传入当前请求的上下文。
	DB() *gorm.DB

	// Transaction 执行一个数据库事务。
	// 回调函数中的 tx 已包含事务上下文，可以直接使用。
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error

	// AutoMigrate 自动迁移数据库表结构，能正确处理分片表的创建。
	AutoMigrate(ctx context.Context, dst ...interface{}) error

	// Ping 检查数据库连接。
	Ping(ctx context.Context) error
	// Close 关闭数据库连接池。
	Close() error
}
```

## 3. 标准用法

### 场景 1: 基本 CRUD 操作

```go
// 在服务初始化时注入 dbProvider
type UserService struct {
    db db.Provider
}

// 在业务方法中使用
func (s *UserService) CreateUser(ctx context.Context, username string) (*User, error) {
    user := &User{Username: username}
    
    // 从 Provider 获取 gorm.DB 实例，并传入当前请求的 context
    result := s.db.DB().WithContext(ctx).Create(user)
    if result.Error != nil {
        return nil, result.Error
    }
    
    return user, nil
}
```

### 场景 2: 执行事务

`Transaction` 方法封装了事务的提交和回滚逻辑，是执行事务的首选方式。

```go
func (s *AccountService) Transfer(ctx context.Context, fromUserID, toUserID string, amount int64) error {
    return s.db.Transaction(ctx, func(tx *gorm.DB) error {
        // tx 已经是带事务的 *gorm.DB 实例，可以直接使用

        // 1. 扣款
        if err := tx.Model(&Account{}).Where("user_id = ?", fromUserID).Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
            // 返回任意 error 都会导致事务回滚
            return err
        }

        // 2. 加款
        if err := tx.Model(&Account{}).Where("user_id = ?", toUserID).Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
            return err
        }

        // 函数正常返回，事务会自动提交
        return nil
    })
}
```

### 场景 3: 使用分库分表

`db` 组件对分片操作是透明的，开发者只需正确配置并使用 GORM 即可。

```go
// 1. 在配置中心定义分片规则
/*
{
  "sharding": {
    "shardingKey": "user_id",
    "numberOfShards": 16,
    "tables": {
      "messages": {}
    }
  }
}
*/

// 2. 定义 GORM 模型
type Message struct {
    ID       uint64 `gorm:"primaryKey"`
    UserID   uint64 `gorm:"index"` // UserID 是分片键
    Content  string
}

// 3. GORM 操作会自动路由到正确的分片
func (s *MessageRepo) CreateMessage(ctx context.Context, msg *Message) error {
    // GORM 会根据 msg.UserID 的值，自动计算出哈希，
    // 并将这条记录插入到对应的 `messages_XX` 表中。
    return s.db.DB().WithContext(ctx).Create(msg).Error
}

func (s *MessageRepo) GetMessages(ctx context.Context, userID uint64) ([]*Message, error) {
    var messages []*Message
    // 查询时必须带上分片键 `user_id`，以便 GORM 能定位到正确的表。
    err := s.db.DB().WithContext(ctx).Where("user_id = ?", userID).Find(&messages).Error
    return messages, err
}