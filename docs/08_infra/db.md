# 基础设施: DB 数据库

## 1. 设计理念

`db` 是 `gochat` 项目的数据库基础设施组件，基于 `GORM v2` 构建。它是一个**专注于 MySQL** 的、以**分库分表**为核心设计的高性能数据库操作层。

`db` 组件的设计哲学是 **“封装便利，但不隐藏能力”**。它封装了数据库连接、配置、事务和分片等复杂逻辑，同时通过 `DB()` 方法提供了对原生 `*gorm.DB` 的完全访问，让开发者可以利用 GORM 的全部功能，保证了灵活性。

## 2. 核心 API 契约

### 2.1 构造函数与 Options

```go
// Config 是 db 组件的主配置结构体。
type Config struct {
	DSN             string          `json:"dsn"`             // 数据库连接字符串
	Driver          string          `json:"driver"`          // 数据库驱动，仅支持 "mysql"
	MaxOpenConns    int             `json:"maxOpenConns"`    // 最大打开连接数
	MaxIdleConns    int             `json:"maxIdleConns"`    // 最大空闲连接数
	ConnMaxLifetime time.Duration   `json:"connMaxLifetime"` // 连接最大生命周期
	LogLevel        string          `json:"logLevel"`        // GORM 日志级别: "silent", "info", "warn", "error"
	SlowThreshold   time.Duration   `json:"slowThreshold"`   // 慢查询阈值
	Sharding        *ShardingConfig `json:"sharding"`        // 分片配置，可选
}

// GetDefaultConfig 返回默认的数据库配置。
// 开发环境：较少连接数，较详细的日志级别，较短的超时时间
// 生产环境：较多连接数，较少的日志输出，较长的连接生命周期
func GetDefaultConfig(env string) *Config

// ShardingConfig 定义了分库分表配置。
type ShardingConfig struct {
	ShardingKey    string                     `json:"shardingKey"`    // 分片键字段名，如 "user_id"
	NumberOfShards int                        `json:"numberOfShards"` // 分片总数，如 16
	Tables         map[string]*TableShardingConfig `json:"tables"`     // 表名到分片配置的映射
}

// TableShardingConfig 定义了单个表的分片配置。
type TableShardingConfig struct {
	ShardingKey    string `json:"shardingKey,omitempty"`    // 可选：覆盖全局分片键
	NumberOfShards int    `json:"numberOfShards,omitempty"` // 可选：覆盖全局分片数
}

// Option 定义了用于定制 db Provider 的函数。
type Option func(*provider)

// WithLogger 将一个 clog.Logger 实例注入 GORM，用于结构化记录 SQL 日志。
// 这是与 clog 组件联动的推荐做法。
func WithLogger(logger clog.Logger) Option

// New 是创建数据库 Provider 实例的唯一入口。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Provider 接口

`Provider` 接口定义了数据库操作的核心能力。

```go
// Provider 提供了访问数据库的能力。
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
```

## 3. 标准用法

### 场景 1: 基本 CRUD 操作

```go
// 在服务初始化时注入 dbProvider
type UserService struct {
    db db.Provider
}

func initializeDB() (db.Provider, error) {
    // 使用默认配置（推荐），或从配置中心加载
    config := db.GetDefaultConfig("development") // "development" or "production"
    
    // 根据需要覆盖特定配置
    config.DSN = "user:password@tcp(127.0.0.1:3306)/gochat?charset=utf8mb4&parseTime=True&loc=Local"
    
    return db.New(context.Background(), config, db.WithLogger(clog.Module("gorm")))
}

// 在业务方法中使用
func (s *UserService) CreateUser(ctx context.Context, username string) (*User, error) {
    user := &User{Username: username}
    
    // 从 Provider 获取 gorm.DB 实例，上下文已通过参数传入
    result := s.db.DB(ctx).Create(user)
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
        // tx 已经是带事务和上下文的 *gorm.DB 实例，可以直接使用

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

## 4. 设计注记

### 4.1 分片机制详解

本组件的分片功能基于 `gorm.io/sharding` 库实现，并注册了自定义的分片算法。

-   **分片配置层次**:
    -   **全局配置**: `ShardingConfig` 中的 `ShardingKey` 和 `NumberOfShards` 作为所有分片表的默认值。
    -   **表级配置**: `TableShardingConfig` 可以为特定表覆盖全局设置。如果 `ShardingKey` 或 `NumberOfShards` 为空/0，则使用全局配置。

-   **分片键 (Sharding Key) 类型**:
    -   **整数类型** (`int`, `int64`, `uint64` 等): 直接使用其数值。
    -   **字符串类型** (`string`):
        1.  **优先解析**: 尝试将字符串按十进制解析为整数。
        2.  **哈希备选**: 若无法解析，则使用 `hash = hash*31 + char_code` 算法计算哈希值。

-   **路由逻辑**:
    1.  获取分片键的数值（或哈希值）。
    2.  取该值的**绝对值**。
    3.  对分片总数 (`NumberOfShards`) 进行**取模**，得到分片索引。
    4.  最终表名后缀格式为 `_XX`，例如 `messages_01`, `messages_15`。

-   **重要行为**:
    -   **无分片键查询**: 如果一个查询没有在 `WHERE` 条件中包含分片键，`gorm.io/sharding` 的默认行为是**返回错误**，以防止危险的全部分片扫描。
    -   **AutoMigrate**: 调用 `AutoMigrate` 时，它会为每个分片都创建或更新表结构，例如，为 `messages` 表和 16 个分片创建 `messages_00` 到 `messages_15` 的所有表。

### 4.2 GORM 实例生命周期

从 `db.DB(ctx)` 获取的 `*gorm.DB` 实例是轻量级且无状态的。它只是一个包含了数据库连接池和上下文信息的会话对象。因此，最佳实践是：

-   **即用即取**: 在每个需要执行数据库操作的函数中，都通过调用 `db.DB(ctx)` 来获取一个新的会话实例。
-   **禁止持有**: 不要在 `struct` 中长期持有 `*gorm.DB` 实例，因为这会使其绑定的上下文失效，并可能导致并发问题。应始终持有 `db.Provider` 接口。

### 4.3 错误处理最佳实践

**分片查询错误**: 当查询分片表时未提供分片键，`gorm.io/sharding` 会返回错误。业务代码应该妥善处理这类错误，避免意外的全表扫描。

**事务错误**: 在 `Transaction` 方法的回调函数中，任何返回的 `error` 都会导致事务自动回滚。这是故意设计的安全机制。

### 4.4 GetDefaultConfig 默认值说明

`GetDefaultConfig` 根据环境返回优化的默认配置：

**开发环境 (development)**:
```go
&Config{
    Driver:          "mysql",
    MaxOpenConns:    25,          // 较少连接数，适合开发环境
    MaxIdleConns:    5,           // 较少空闲连接
    ConnMaxLifetime: 5*time.Minute,  // 较短生命周期，便于调试
    LogLevel:        "info",      // 详细日志，便于开发调试
    SlowThreshold:   100*time.Millisecond, // 较严格的慢查询阈值
    Sharding:        nil,         // 默认不开启分片
}
```

**生产环境 (production)**:
```go
&Config{
    Driver:          "mysql",
    MaxOpenConns:    100,         // 较多连接数，支持高并发
    MaxIdleConns:    10,          // 适当的空闲连接池
    ConnMaxLifetime: 1*time.Hour, // 较长生命周期，减少连接重建开销
    LogLevel:        "warn",      // 只记录警告和错误，减少日志量
    SlowThreshold:   500*time.Millisecond, // 较宽松的慢查询阈值
    Sharding:        nil,         // 默认不开启分片
}
```

用户仍需要根据实际情况设置 `DSN` 和可选的 `Sharding` 配置。