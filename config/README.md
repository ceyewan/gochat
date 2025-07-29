# GoChat 配置管理

这个目录包含了 GoChat 项目的配置文件，配合 `config-cli` 工具进行统一的配置中心管理。

## 配置中心集成

GoChat 使用分布式配置中心来管理组件配置，支持动态配置更新和降级策略。

### 配置键格式

配置在配置中心中的键格式为：
```
/config/{env}/{service}/{component}
```

其中：
- `env`: 环境名称（如 dev, test, prod）
- `service`: 服务名称（如 im-infra）
- `component`: 组件名称（如 clog, cache, db）

例如：
- `/config/prod/im-infra/clog` - 生产环境下 im-infra 服务的 clog 组件配置
- `/config/dev/im-infra/cache` - 开发环境下 im-infra 服务的 cache 组件配置

### 目录结构

配置文件按照 `{env}/{service}/{component}.json` 的格式组织：

```
config/
├── dev/
│   └── im-infra/
│       ├── clog.json    # 开发环境日志配置
│       ├── cache.json   # 开发环境缓存配置
│       └── db.json      # 开发环境数据库配置
├── test/
│   └── im-infra/
│       └── clog.json    # 测试环境日志配置
├── prod/
│   └── im-infra/
│       ├── clog.json    # 生产环境日志配置
│       ├── cache.json   # 生产环境缓存配置
│       └── db.json      # 生产环境数据库配置
└── README.md            # 说明文档
```

## 配置管理工具

使用统一的 `config-cli` 工具进行配置管理：

### 安装工具

```bash
cd cmd/config-cli
go build -o config-cli
```

### 配置初始化

```bash
# 初始化所有配置
./config-cli init

# 初始化指定环境的所有配置
./config-cli init dev

# 初始化指定环境和服务的所有配置
./config-cli init prod im-infra

# 初始化指定的单个配置
./config-cli init dev im-infra clog

# 干运行模式（只显示将要执行的操作）
./config-cli init --dry-run
```

### 配置查看

```bash
# 列出所有配置
./config-cli list

# 列出指定环境的配置
./config-cli list dev

# 列出指定环境和服务的配置
./config-cli list dev im-infra

# 查看具体配置内容
./config-cli get /config/dev/im-infra/clog

# 表格格式显示
./config-cli list --format table

# 显示详细信息
./config-cli list --detailed
```

### 配置更新

```bash
# 深度合并更新（推荐）
./config-cli set /config/dev/im-infra/clog '{"level":"debug"}'

# 完全替换配置
./config-cli replace /config/dev/im-infra/clog '{"level":"info","format":"json"}'

# 删除特定字段
./config-cli delete /config/dev/im-infra/clog rotation.maxSize

# 实时监听配置变化
./config-cli watch /config/dev/im-infra/clog
```

## 配置文件示例

### clog.json - 日志组件配置

```json
{
  "level": "info",
  "format": "console",
  "output": "stdout",
  "addSource": true,
  "enableColor": true,
  "rootPath": "gochat",
  "rotation": {
    "maxSize": 100,
    "maxBackups": 10,
    "maxAge": 30,
    "compress": true
  }
}
```

### cache.json - 缓存组件配置

```json
{
  "addr": "localhost:6379",
  "password": "",
  "db": 0,
  "poolSize": 10,
  "minIdleConns": 5,
  "maxIdleConns": 10,
  "connMaxIdleTime": "10m",
  "connMaxLifetime": "30m",
  "dialTimeout": "5s",
  "readTimeout": "3s",
  "writeTimeout": "3s",
  "poolTimeout": "4s",
  "maxRetries": 3,
  "minRetryBackoff": "8ms",
  "maxRetryBackoff": "512ms",
  "enableTracing": true,
  "enableMetrics": true,
  "keyPrefix": "dev",
  "serializer": "json",
  "compression": false
}
```

### db.json - 数据库组件配置

```json
{
  "dsn": "root:mysql@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local",
  "driver": "mysql",
  "maxOpenConns": 25,
  "maxIdleConns": 10,
  "connMaxLifetime": "1h",
  "connMaxIdleTime": "30m",
  "logLevel": "info",
  "slowThreshold": "200ms",
  "enableMetrics": true,
  "enableTracing": true,
  "tablePrefix": "dev_"
}
```

## 组件配置中心集成

### 新的依赖注入方式（推荐）

```go
package main

import (
    "github.com/ceyewan/gochat/im-infra/clog"
    "github.com/ceyewan/gochat/im-infra/coord"
    "github.com/ceyewan/gochat/im-infra/db"
)

func main() {
    // 1. 创建协调器
    coordinator, err := coord.New()
    if err != nil {
        panic(err)
    }
    defer coordinator.Close()

    configCenter := coordinator.Config()

    // 2. 创建配置管理器（新方式）
    clogManager := clog.NewConfigManager(configCenter, "prod", "im-infra", "clog")
    clogManager.Start()
    defer clogManager.Stop()

    dbManager := db.NewConfigManager(configCenter, "prod", "im-infra", "db")
    dbManager.Start()
    defer dbManager.Stop()

    // 3. 使用组件
    logger := clog.Module("app")
    database := db.GetDB()

    // 应用逻辑...
}
```

### 向后兼容方式

```go
// 仍然支持全局方式
clog.SetupConfigCenterFromCoord(coordinator.Config(), "prod", "im-infra", "clog")
db.SetupConfigCenterFromCoord(coordinator.Config(), "prod", "im-infra", "db")

// 初始化组件（自动从配置中心读取）
clog.Init()
```

## 快速开始

### 1. 初始化配置

```bash
# 进入项目根目录
cd /path/to/gochat

# 编译配置管理工具
cd cmd/config-cli
go build -o config-cli

# 初始化所有配置到配置中心
./config-cli init --config-path ../../config

# 或者只初始化开发环境配置
./config-cli init dev --config-path ../../config
```

### 2. 查看配置

```bash
# 查看所有配置
./config-cli list

# 查看具体配置内容
./config-cli get /config/dev/im-infra/clog
```

### 3. 更新配置

```bash
# 动态更新配置
./config-cli set /config/dev/im-infra/clog '{"level":"debug","format":"json"}'
```

### 4. 在应用中使用

```go
// 设置配置中心
coordinator, _ := coord.New()
clogManager := clog.NewConfigManager(coordinator.Config(), "dev", "im-infra", "clog")
clogManager.Start()
defer clogManager.Stop()

// 初始化组件（自动从配置中心读取）
clog.Init()
```

## 最佳实践

1. **环境隔离**：不同环境使用不同的配置文件
2. **配置验证**：使用 `--dry-run` 验证配置格式正确性
3. **版本控制**：配置文件纳入版本控制
4. **监控配置**：监控配置中心的可用性和配置变更
5. **降级测试**：定期测试配置中心不可用时的降级行为
6. **渐进部署**：先在开发环境验证配置，再推广到生产环境
