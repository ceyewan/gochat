# DB 模块使用示例

这个目录包含了 `im-infra/db` 模块的完整使用示例，展示了数据库操作的各个方面，包括基础 CRUD、数据库迁移、事务处理、分片机制和性能优化。

## 📁 示例目录结构

```
examples/
├── README.md              # 本文档
├── basic/                  # 基础 CRUD 操作示例
│   └── main.go
├── migration/             # 数据库迁移示例
│   └── main.go
├── transaction/           # 事务操作示例
│   └── main.go
├── sharding/              # 分片功能示例
│   └── main.go
└── performance/           # 性能测试示例
    └── main.go
```

## 🚀 快速开始

### 前置条件

1. **MySQL 数据库**: 确保 MySQL 服务正在运行
2. **数据库配置**: 修改示例中的数据库连接字符串以匹配你的环境
3. **Go 环境**: Go 1.19 或更高版本

### 运行示例

```bash
# 进入 examples 目录
cd im-infra/db/examples

# 运行基础示例
cd basic && go run main.go

# 运行迁移示例
cd ../migration && go run main.go

# 运行事务示例
cd ../transaction && go run main.go

# 运行分片示例
cd ../sharding && go run main.go

# 运行性能测试示例
cd ../performance && go run main.go
```

## 📚 示例详解

### 1. 基础操作示例 (`basic/main.go`)

**核心功能**:
- ✅ 数据库连接和配置
- ✅ Logger 依赖注入
- ✅ 基础 CRUD 操作 (Create, Read, Update, Delete)
- ✅ 连接池状态监控
- ✅ 错误处理最佳实践

**学习要点**:
```go
// 创建数据库实例，注入 Logger
database, err := db.New(ctx, cfg, db.WithLogger(logger), db.WithComponentName("basic-example"))

// 基础 CRUD 操作
gormDB := database.GetDB()
gormDB.Create(&user)          // 创建
gormDB.First(&user, 1)        // 读取
gormDB.Save(&user)            // 更新
gormDB.Delete(&user)          // 删除
```

**适用场景**: 新手入门，了解基本的数据库操作流程

### 2. 数据库迁移示例 (`migration/main.go`)

**核心功能**:
- ✅ 渐进式模式演进 (V1 → V2 → V3)
- ✅ 字段添加和修改
- ✅ 外键关系处理
- ✅ 数据回填 (Data Backfill)
- ✅ 兼容性处理

**学习要点**:
```go
// V1: 基础模型
type UserV1 struct {
    ID   uint   `gorm:"primaryKey"`
    Name string `gorm:"size:100;not null"`
    Email string `gorm:"uniqueIndex;size:100"`
}

// V2: 添加新字段
type UserV2 struct {
    UserV1
    Age       int       `gorm:"default:0"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

// V3: 扩展字段和关系
type UserV3 struct {
    UserV2
    Phone     string `gorm:"size:20"`
    Status    string `gorm:"size:20;default:active"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
    Profile   ProfileV3 `gorm:"foreignKey:UserID"`
}
```

**适用场景**: 生产环境数据库 schema 演进，向前兼容性设计

### 3. 事务操作示例 (`transaction/main.go`)

**核心功能**:
- ✅ 复杂业务事务处理
- ✅ 金融转账场景
- ✅ 错误回滚机制
- ✅ 嵌套事务处理
- ✅ 重试逻辑

**学习要点**:
```go
// 使用事务进行转账操作
err := database.Transaction(func(tx *gorm.DB) error {
    // 1. 查询账户并加锁
    var fromAccount, toAccount Account
    if err := tx.Set("gorm:query_option", "FOR UPDATE").
        Where("account_no = ?", fromAccountNo).First(&fromAccount).Error; err != nil {
        return err
    }
    
    // 2. 验证余额
    if fromAccount.Balance < amount {
        return errors.New("余额不足")
    }
    
    // 3. 更新余额
    fromAccount.Balance -= amount
    toAccount.Balance += amount
    
    // 4. 保存更改
    if err := tx.Save(&fromAccount).Error; err != nil {
        return err
    }
    return tx.Save(&toAccount).Error
})
```

**适用场景**: 金融系统、库存管理、复杂业务流程

### 4. 分片功能示例 (`sharding/main.go`)

**核心功能**:
- ✅ 基于 user_id 的 Hash 分片
- ✅ 多表分片配置
- ✅ 跨分片查询
- ✅ 分片数据的 CRUD 操作
- ✅ 批量操作优化

**学习要点**:
```go
// 配置分片规则
shardingConfig := db.NewShardingConfig("user_id", 4) // 分成 4 个分片
shardingConfig.Tables = map[string]*db.TableShardingConfig{
    "orders": {
        ShardingKey:       "user_id",
        NumberOfShards:    4,
        ShardingAlgorithm: "hash",
    },
    "messages": {
        ShardingKey:       "user_id", 
        NumberOfShards:    4,
        ShardingAlgorithm: "hash",
    },
}

// 查询时必须包含分片键
gormDB.Where("user_id = ?", userID).Find(&userOrders)
```

**适用场景**: 高并发应用、大数据量场景、水平扩展需求

### 5. 性能测试示例 (`performance/main.go`)

**核心功能**:
- ✅ 批量插入性能测试
- ✅ 并发操作性能测试
- ✅ 查询性能基准测试
- ✅ 事务性能评估
- ✅ 连接池监控
- ✅ 性能报告生成

**学习要点**:
```go
// 批量插入性能测试
batchSizes := []int{100, 500, 1000, 2000}
for _, batchSize := range batchSizes {
    products := generateTestProducts(batchSize)
    start := time.Now()
    
    if err := gormDB.CreateInBatches(products, 100).Error; err != nil {
        // 处理错误
    }
    
    duration := time.Since(start)
    throughput := float64(batchSize) / duration.Seconds()
    // 记录性能指标
}

// 并发插入测试
var wg sync.WaitGroup
for i := 0; i < concurrency; i++ {
    wg.Add(1)
    go func(workerID int) {
        defer wg.Done()
        // 并发执行数据库操作
    }(i)
}
wg.Wait()
```

**适用场景**: 性能调优、容量规划、系统压测

## 🛠️ 配置说明

### 数据库连接配置

每个示例都使用类似的配置模式:

```go
// 基础配置
cfg := db.MySQLConfig("root:mysql@tcp(localhost:3306)/database_name?charset=utf8mb4&parseTime=True&loc=Local")

// 性能优化配置 (适用于性能测试)
cfg.MaxOpenConns = 50                  // 最大连接数
cfg.MaxIdleConns = 25                  // 最大空闲连接数  
cfg.ConnMaxLifetime = time.Hour        // 连接最大生存时间
cfg.ConnMaxIdleTime = 30 * time.Minute // 连接最大空闲时间
```

### Logger 集成

所有示例都展示了如何正确集成日志:

```go
// 创建模块化的 Logger
logger := clog.Module("db-example-name")

// 注入到数据库实例
database, err := db.New(ctx, cfg, 
    db.WithLogger(logger),
    db.WithComponentName("example-component"))
```

## 📊 性能基准

基于性能测试示例的典型结果:

| 操作类型 | 记录数 | 平均耗时 | 吞吐量(ops/s) | 平均延迟(ms) |
|---------|--------|----------|---------------|--------------|
| 批量插入_1000条 | 1000 | ~200ms | ~5000 | ~0.2 |
| 并发插入_10协程 | 1000 | ~150ms | ~6666 | ~0.15 |
| 简单ID查询 | 1000次 | ~100ms | ~10000 | ~0.1 |
| 索引字段查询 | 1000次 | ~150ms | ~6666 | ~0.15 |
| 关联查询 | 1000次 | ~300ms | ~3333 | ~0.3 |
| 事务操作 | 100次 | ~500ms | ~200 | ~5 |

*注：实际性能会根据硬件配置、网络延迟和数据库配置而有所不同*

## 🔧 故障排查

### 常见问题

1. **连接失败**:
   ```
   Error: failed to connect to database
   ```
   **解决**: 检查 MySQL 服务是否启动，数据库连接字符串是否正确

2. **权限错误**:
   ```
   Error: Access denied for user
   ```
   **解决**: 确保数据库用户有足够的权限创建数据库和表

3. **分片表创建失败**:
   ```
   Error: table doesn't exist
   ```
   **解决**: 确保已安装 gorm.io/sharding 插件，并正确配置分片规则

4. **性能测试超时**:
   ```
   Error: context deadline exceeded
   ```
   **解决**: 增加上下文超时时间，或减少测试数据量

### 调试技巧

1. **启用 SQL 日志**:
   ```go
   cfg.LogLevel = "info" // 显示所有 SQL 语句
   ```

2. **监控连接池**:
   ```go
   stats := database.Stats()
   logger.Info("连接池状态", clog.Int("openConnections", stats.OpenConnections))
   ```

3. **使用事务调试**:
   ```go
   err := database.Transaction(func(tx *gorm.DB) error {
       // 在事务中添加详细日志
       logger.Info("执行事务步骤", clog.String("step", "1"))
       return nil
   })
   ```

## 📖 扩展阅读

- [GORM 官方文档](https://gorm.io/docs/)
- [MySQL 性能优化指南](https://dev.mysql.com/doc/refman/8.0/en/optimization.html)
- [数据库分片最佳实践](https://github.com/go-gorm/sharding)
- [im-infra/db 设计文档](../DESIGN.md)
- [im-infra/db API 文档](../API.md)

## 🤝 贡献指南

如果你想添加新的示例或改进现有示例:

1. 创建新的示例目录
2. 遵循现有的代码结构和注释风格
3. 确保包含完整的错误处理
4. 添加充分的日志记录
5. 更新本 README 文档

## 📄 许可证

这些示例代码遵循项目的 MIT 许可证。
