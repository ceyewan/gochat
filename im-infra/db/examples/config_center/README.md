# DB 配置中心集成示例

本示例展示如何将 `im-infra/db` 与配置中心（coord）集成，实现动态配置获取和管理。

## 功能特性

- **动态配置获取**：从配置中心获取数据库配置
- **默认配置兜底**：配置中心不可用时自动使用默认配置
- **模块化实例**：支持为不同模块创建独立的数据库实例
- **配置热重载**：支持运行时重新加载配置
- **无侵入集成**：保持现有 API 完全兼容

## 使用方法

### 1. 基本集成

```go
// 初始化 coord 实例
coordInstance := coord.New(coord.Config{
    Endpoints: []string{"localhost:2379"},
    Timeout:   5 * time.Second,
})

// 设置配置中心
configCenter := coordInstance.ConfigCenter()
db.SetupConfigCenterFromCoord(configCenter, "dev", "gochat", "db")

// 使用数据库（会自动从配置中心获取配置）
database := db.GetDB()
```

### 2. 模块化实例

```go
// 为不同模块创建独立的数据库实例
userDB := db.Module("user")   // 配置路径: /config/dev/gochat/db-user
orderDB := db.Module("order") // 配置路径: /config/dev/gochat/db-order

// 每个模块可以有不同的数据库配置
```

### 3. 配置重载

```go
// 运行时重新加载配置
db.ReloadConfig()
```

## 配置格式

配置中心中的配置格式与 `db.Config` 结构体一致：

```json
{
  "dsn": "root:mysql@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local",
  "driver": "mysql",
  "maxOpenConns": 20,
  "maxIdleConns": 10,
  "connMaxLifetime": "1h",
  "connMaxIdleTime": "30m",
  "logLevel": "info",
  "slowThreshold": "500ms",
  "enableMetrics": true,
  "enableTracing": false,
  "tablePrefix": "dev_",
  "disableForeignKeyConstraintWhenMigrating": false,
  "autoCreateDatabase": true
}
```

## 配置路径规则

- **默认实例**：`/config/{env}/{service}/{component}`
- **模块实例**：`/config/{env}/{service}/{component}-{module}`

示例：
- 默认实例：`/config/dev/gochat/db`
- 用户模块：`/config/dev/gochat/db-user`
- 订单模块：`/config/dev/gochat/db-order`

## 运行示例

1. 启动 etcd：
```bash
etcd
```

2. 设置配置：
```bash
etcdctl put /config/dev/gochat/db '{"dsn":"root:mysql@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local","driver":"mysql","maxOpenConns":20,"maxIdleConns":10,"connMaxLifetime":"1h","connMaxIdleTime":"30m","logLevel":"info","slowThreshold":"500ms","enableMetrics":true,"enableTracing":false,"tablePrefix":"dev_","disableForeignKeyConstraintWhenMigrating":false,"autoCreateDatabase":true}'
```

3. 运行示例：
```bash
go run main.go
```

## 降级策略

当配置中心不可用时，db 模块会：

1. 记录警告日志
2. 使用默认配置继续工作
3. 不影响应用正常运行

这确保了系统的高可用性和容错能力。
