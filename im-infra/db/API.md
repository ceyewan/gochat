# DB API 文档

## 核心接口

### DB 接口

```go
type DB interface {
    // GetDB 返回原生的 GORM 数据库实例
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
}
```

## 全局函数

### 数据库实例管理

```go
// New 根据配置创建新的数据库实例
func New(cfg Config) (DB, error)

// GetDB 获取全局默认数据库的 GORM 实例
func GetDB() *gorm.DB

// Module 创建带模块标识的数据库实例
func Module(name string) DB
```

### 全局数据库操作

```go
// Ping 检查全局数据库连接
func Ping(ctx context.Context) error

// Close 关闭全局数据库连接
func Close() error

// Stats 获取全局数据库连接池统计信息
func Stats() sql.DBStats

// WithContext 获取带上下文的全局数据库实例
func WithContext(ctx context.Context) *gorm.DB

// Transaction 使用全局数据库执行事务
func Transaction(fn func(tx *gorm.DB) error) error
```

## 配置

### Config 结构体

```go
type Config struct {
    // 基础连接配置
    DSN    string // 数据库连接字符串
    Driver string // 数据库驱动 ("mysql", "postgres", "sqlite")
    
    // 连接池配置
    MaxOpenConns    int           // 最大打开连接数
    MaxIdleConns    int           // 最大空闲连接数
    ConnMaxLifetime time.Duration // 连接最大生存时间
    ConnMaxIdleTime time.Duration // 连接最大空闲时间
    
    // 日志配置
    LogLevel      string        // 日志级别 ("silent", "error", "warn", "info")
    SlowThreshold time.Duration // 慢查询阈值
    
    // 性能配置
    EnableMetrics bool // 启用指标收集
    EnableTracing bool // 启用链路追踪
    
    // 其他配置
    TablePrefix                              string // 表名前缀
    DisableForeignKeyConstraintWhenMigrating bool   // 迁移时禁用外键约束
    
    // 分库分表配置
    Sharding *ShardingConfig // 可选的分库分表配置
}
```

### 预设配置函数

```go
// DefaultConfig 返回默认配置
func DefaultConfig() Config

// DevelopmentConfig 返回开发环境配置
func DevelopmentConfig() Config

// ProductionConfig 返回生产环境配置
func ProductionConfig() Config

// TestConfig 返回测试环境配置
func TestConfig() Config

// HighPerformanceConfig 返回高性能配置
func HighPerformanceConfig() Config
```

### 配置构建器

```go
type ConfigBuilder struct {
    config Config
}

// NewConfigBuilder 创建配置构建器
func NewConfigBuilder() *ConfigBuilder

// 配置方法（链式调用）
func (b *ConfigBuilder) DSN(dsn string) *ConfigBuilder
func (b *ConfigBuilder) Driver(driver string) *ConfigBuilder
func (b *ConfigBuilder) MaxOpenConns(max int) *ConfigBuilder
func (b *ConfigBuilder) MaxIdleConns(max int) *ConfigBuilder
func (b *ConfigBuilder) ConnLifetime(lifetime, idleTime time.Duration) *ConfigBuilder
func (b *ConfigBuilder) LogLevel(level string) *ConfigBuilder
func (b *ConfigBuilder) SlowThreshold(threshold time.Duration) *ConfigBuilder
func (b *ConfigBuilder) TablePrefix(prefix string) *ConfigBuilder
func (b *ConfigBuilder) EnableMetrics() *ConfigBuilder
func (b *ConfigBuilder) EnableTracing() *ConfigBuilder
func (b *ConfigBuilder) DisableForeignKeyConstraints() *ConfigBuilder
func (b *ConfigBuilder) Sharding(cfg *ShardingConfig) *ConfigBuilder
func (b *ConfigBuilder) Build() Config
```

### 便捷配置函数

```go
// MySQLConfig 创建 MySQL 配置构建器
func MySQLConfig(dsn string) *ConfigBuilder

// PostgreSQLConfig 创建 PostgreSQL 配置构建器
func PostgreSQLConfig(dsn string) *ConfigBuilder

// SQLiteConfig 创建 SQLite 配置构建器
func SQLiteConfig(dsn string) *ConfigBuilder
```

## 分库分表

### ShardingConfig 结构体

```go
type ShardingConfig struct {
    ShardingKey       string                           // 分片键字段名
    NumberOfShards    int                              // 分片数量
    ShardingAlgorithm string                           // 分片算法 ("hash", "range", "mod")
    Tables            map[string]*TableShardingConfig  // 表级分片配置
}

type TableShardingConfig struct {
    ShardingKey       string // 表特定的分片键
    NumberOfShards    int    // 表特定的分片数量
    ShardingAlgorithm string // 表特定的分片算法
}
```

### 分片配置构建器

```go
type ShardingConfigBuilder struct {
    config ShardingConfig
}

// NewShardingConfigBuilder 创建分片配置构建器
func NewShardingConfigBuilder() *ShardingConfigBuilder

func (b *ShardingConfigBuilder) ShardingKey(key string) *ShardingConfigBuilder
func (b *ShardingConfigBuilder) NumberOfShards(num int) *ShardingConfigBuilder
func (b *ShardingConfigBuilder) Algorithm(algorithm string) *ShardingConfigBuilder
func (b *ShardingConfigBuilder) AddTable(tableName string, cfg *TableShardingConfig) *ShardingConfigBuilder
func (b *ShardingConfigBuilder) Build() *ShardingConfig
```

### 分片辅助工具

```go
type ShardingHelper struct {
    config *ShardingConfig
    logger clog.Logger
}

// NewShardingHelper 创建分片辅助工具
func NewShardingHelper(config *ShardingConfig) *ShardingHelper

// GetShardSuffix 根据分片键值获取分片后缀
func (h *ShardingHelper) GetShardSuffix(value interface{}) (string, error)
```

## 使用示例

### 基础使用

```go
// 1. 使用预设配置
cfg := db.DevelopmentConfig()
database, err := db.New(cfg)

// 2. 使用配置构建器
cfg := db.NewConfigBuilder().
    DSN("mysql://user:pass@localhost/db").
    MaxOpenConns(20).
    LogLevel("info").
    Build()
database, err := db.New(cfg)

// 3. 使用便捷函数
cfg := db.MySQLConfig("user:pass@tcp(localhost:3306)/db").
    MaxOpenConns(20).
    EnableMetrics().
    Build()
database, err := db.New(cfg)
```

### 事务操作

```go
err := database.Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    if err := tx.Create(&profile).Error; err != nil {
        return err
    }
    return nil
})
```

### 分库分表使用

```go
// 配置分片
shardingConfig := db.NewShardingConfigBuilder().
    ShardingKey("user_id").
    NumberOfShards(16).
    AddTable("orders", &db.TableShardingConfig{}).
    Build()

cfg := db.NewConfigBuilder().
    DSN("mysql://user:pass@localhost/db").
    Sharding(shardingConfig).
    Build()

database, err := db.New(cfg)

// 使用分片（查询必须包含分片键）
gormDB := database.GetDB()
gormDB.Create(&Order{UserID: 123, Amount: 99.99})
gormDB.Where("user_id = ?", 123).Find(&orders)
```

## 错误处理

所有函数都返回标准的 Go error，建议进行适当的错误处理：

```go
database, err := db.New(cfg)
if err != nil {
    log.Fatal("数据库连接失败:", err)
}

err = database.Ping(ctx)
if err != nil {
    log.Printf("数据库连接检查失败: %v", err)
}
```

## 性能监控

```go
// 获取连接池统计信息
stats := database.Stats()
fmt.Printf("打开连接数: %d\n", stats.OpenConnections)
fmt.Printf("使用中连接数: %d\n", stats.InUse)
fmt.Printf("空闲连接数: %d\n", stats.Idle)
```

## 日志集成

db 包与 clog 深度集成，自动记录：

- SQL 执行日志
- 慢查询警告
- 连接状态变化
- 事务操作
- 错误信息

日志级别可通过配置控制：

```go
cfg := db.NewConfigBuilder().
    LogLevel("info").           // 设置日志级别
    SlowThreshold(200*time.Millisecond). // 设置慢查询阈值
    Build()
```
