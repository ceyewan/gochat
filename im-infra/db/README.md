# db

一个现代化、高性能的 Go 数据库基础设施库，基于 GORM v2 构建。db 提供简洁、可组合的接口，支持连接池管理、日志集成、分库分表等高级特性。

## 功能特色

- 🚀 **基于 GORM v2**：充分利用最新的 GORM ORM 框架，性能与兼容性俱佳
- 🎯 **接口驱动**：抽象清晰，封装合理，用户通过 `GetDB()` 获取原生 GORM 实例
- 🌟 **全局数据库方法**：支持 `db.GetDB()` 等全局数据库方法，无需显式创建数据库实例
- 📦 **自定义数据库实例**：`db.New(config)` 创建自定义配置的数据库实例
- 🔧 **数据库管理**：自动创建数据库、表结构迁移等便捷功能
- 🔄 **连接池管理**：内置连接池和错误恢复机制
- 🏷️ **日志集成**：与 clog 日志库深度集成，提供详细的操作日志和慢查询监控
- ⚡ **高性能**：优化的连接管理和查询性能
- 🎨 **配置灵活**：丰富的配置选项和预设配置
- 🔧 **零额外依赖**：仅依赖 GORM 和 clog
- 📊 **分库分表支持**：基于 gorm.io/sharding 的可选分库分表功能

## 安装

```bash
go get github.com/ceyewan/gochat/im-infra/db
```

## 快速开始

### 基本用法

#### 全局数据库方法（推荐）

```go
package main

import (
    "context"
    "github.com/ceyewan/gochat/im-infra/db"
)

type User struct {
    ID       uint   `gorm:"primaryKey"`
    Username string `gorm:"uniqueIndex"`
    Email    string
}

func main() {
    ctx := context.Background()
    
    // 使用全局数据库实例
    gormDB := db.GetDB()
    
    // 自动迁移
    gormDB.AutoMigrate(&User{})
    
    // 创建用户
    user := &User{Username: "alice", Email: "alice@example.com"}
    gormDB.WithContext(ctx).Create(user)
    
    // 查询用户
    var users []User
    gormDB.WithContext(ctx).Find(&users)
}
```

#### 自定义数据库实例

```go
package main

import (
    "context"
    "github.com/ceyewan/gochat/im-infra/db"
)

func main() {
    // 创建自定义配置
    cfg := db.Config{
        DSN:             "root:mysql@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local",
        Driver:          "mysql",
        MaxOpenConns:    20,
        MaxIdleConns:    10,
        LogLevel:        "info",
        TablePrefix:     "app_",
    }
    
    // 创建数据库实例
    database, err := db.New(cfg)
    if err != nil {
        panic(err)
    }
    defer database.Close()
    
    // 使用数据库实例
    gormDB := database.GetDB()
    // ... 使用 gormDB 进行数据库操作
}
```

#### 数据库管理功能

```go
package main

import (
    "context"
    "github.com/ceyewan/gochat/im-infra/db"
)

type User struct {
    ID       uint   `gorm:"primaryKey"`
    Username string `gorm:"uniqueIndex"`
    Email    string
}

func main() {
    ctx := context.Background()

    // 创建数据库（如果不存在）
    cfg := db.DefaultConfig()
    err := db.CreateDatabaseIfNotExistsWithConfig(cfg, "myapp")
    if err != nil {
        panic(err)
    }

    // 自动迁移表结构
    err = db.AutoMigrate(&User{})
    if err != nil {
        panic(err)
    }

    // 使用数据库
    gormDB := db.GetDB()
    gormDB.WithContext(ctx).Create(&User{Username: "alice", Email: "alice@example.com"})
}
```

### 配置选项

#### 配置示例

```go
// MySQL 配置
cfg := db.Config{
    DSN:             "root:mysql@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local",
    Driver:          "mysql",
    MaxOpenConns:    50,
    MaxIdleConns:    25,
    LogLevel:        "warn",
    SlowThreshold:   200 * time.Millisecond,
    TablePrefix:     "myapp_",
    EnableMetrics:   true,
    EnableTracing:   true,
}

// PostgreSQL 配置
cfg := db.Config{
    DSN:             "host=localhost user=user password=pass dbname=db sslmode=disable",
    Driver:          "postgres",
    MaxOpenConns:    25,
    MaxIdleConns:    10,
}

// SQLite 配置
cfg := db.Config{
    DSN:             "./database.db",
    Driver:          "sqlite",
    MaxOpenConns:    1,  // SQLite 建议使用单连接
    MaxIdleConns:    1,
}
```

### 分库分表

```go
// 创建分片配置
shardingConfig := &db.ShardingConfig{
    ShardingKey:       "user_id",
    NumberOfShards:    16,
    ShardingAlgorithm: "hash",
    Tables: map[string]*db.TableShardingConfig{
        "orders":   {},
        "payments": {},
    },
}

// 创建带分片的数据库配置
cfg := db.Config{
    DSN:      "root:mysql@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local",
    Driver:   "mysql",
    Sharding: shardingConfig,
}

database, err := db.New(cfg)
if err != nil {
    panic(err)
}

// 使用分片数据库（需要在查询中包含分片键）
gormDB := database.GetDB()
gormDB.Create(&Order{UserID: 123, Amount: 99.99}) // 会自动路由到正确的分片表
```

### 事务操作

```go
err := database.Transaction(func(tx *gorm.DB) error {
    // 在事务中执行多个操作
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    
    if err := tx.Create(&profile).Error; err != nil {
        return err
    }
    
    return nil
})
```

## 最佳实践

### 1. 连接池配置

```go
// ✅ 根据应用负载合理配置连接池
cfg := db.Config{
    DSN:             "root:mysql@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local",
    Driver:          "mysql",
    MaxOpenConns:    25,        // 最大连接数
    MaxIdleConns:    10,        // 最大空闲连接数
    ConnMaxLifetime: time.Hour, // 连接最大生存时间
}
```

### 2. 日志配置

```go
// ✅ 生产环境使用适当的日志级别
cfg := db.DefaultConfig()
cfg.LogLevel = "warn"
cfg.SlowThreshold = 200 * time.Millisecond
```

### 3. 模块化使用

```go
// ✅ 为不同业务模块创建专用数据库实例
type UserService struct {
    db db.DB
}

func NewUserService(cfg db.Config) *UserService {
    database, err := db.New(cfg)
    if err != nil {
        panic(err)
    }
    return &UserService{
        db: database,
    }
}

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    return s.db.GetDB().WithContext(ctx).Create(user).Error
}
```

### 4. 上下文使用

```go
// ✅ 使用带超时的上下文
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := database.GetDB().WithContext(ctx).Create(&user).Error
```

## 日志集成

db 与 clog 深度集成，自动记录：

- SQL 执行日志
- 慢查询警告（可配置阈值）
- 连接池状态变化
- 事务操作日志
- 详细的性能指标

```go
// 日志输出示例
// level=INFO msg="SQL 执行" elapsed=2ms sql="SELECT * FROM users WHERE id = ?" rows=1
// level=WARN msg="检测到慢查询" elapsed=250ms sql="SELECT * FROM orders" threshold=200ms
// level=ERROR msg="SQL 执行错误" elapsed=5ms sql="INSERT INTO users..." error="Duplicate entry"
```

## 监控和指标

启用指标收集：

```go
cfg := db.Config{
    DSN:           "root:mysql@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local",
    Driver:        "mysql",
    EnableMetrics: true,
    EnableTracing: true,
}
```

## 常见问题

### Q: 全局方法和自定义数据库实例的区别？
A: 全局方法适用于简单场景，自定义数据库实例适用于需要不同配置或命名空间隔离的场景。

### Q: 如何处理数据库连接错误？
A: db 包提供了 `Ping()` 方法来检查连接状态，建议在应用启动时进行连接检查。

### Q: 分库分表如何使用？
A: 配置分片规则后，在查询时必须包含分片键，GORM 会自动路由到正确的分片表。

### Q: 如何自定义日志格式？
A: db 包使用 clog 进行日志记录，可以通过配置 clog 来自定义日志格式。

## 示例

查看 [examples](./examples/) 目录获取更多使用示例：

- [基础功能演示](./examples/basic/main.go)
- [用户注册登录](./examples/user_auth/main.go)

## 许可证

MIT License
