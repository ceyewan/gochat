# GoChat 配置管理

这个目录包含了 GoChat 项目的配置文件和配置管理工具，支持统一的配置中心管理。

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
├── config.go            # 配置初始化工具
├── update.go            # 配置更新工具
├── view.go              # 配置查看工具
└── README.md            # 说明文档
```

### 配置文件示例

#### clog.json - 日志组件配置

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

#### cache.json - 缓存组件配置

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

#### db.json - 数据库组件配置

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

## 配置管理工具

### config.go - 配置初始化工具

统一的配置初始化工具，支持批量操作。

#### 使用方法

```bash
# 进入 config 目录
cd config

# 初始化所有配置
go run config.go

# 初始化指定环境的所有配置
go run config.go dev

# 初始化指定环境和服务的所有配置
go run config.go prod im-infra

# 初始化指定的单个配置
go run config.go dev im-infra clog
```

#### 功能

1. 自动扫描配置文件目录
2. 批量读取 JSON 配置文件
3. 连接到配置中心（etcd）
4. 批量写入配置到指定的键路径
5. 验证配置是否写入成功

### update.go - 配置更新工具

动态更新配置中心的配置。

#### 使用方法

```bash
# 更新配置
go run update.go <env> <service> <component> '<json_config>'

# 示例：更新日志级别和格式
go run update.go dev im-infra clog '{"level":"debug","format":"json"}'

# 示例：更新缓存连接池大小
go run update.go prod im-infra cache '{"poolSize":100,"maxIdleConns":50}'
```

#### 功能

1. 获取现有配置
2. 合并新的配置项
3. 更新到配置中心
4. 验证更新结果

### view.go - 配置查看工具

查看配置中心的配置内容。

#### 使用方法

```bash
# 列出所有配置
go run view.go

# 列出指定环境的所有配置
go run view.go dev

# 列出指定环境和服务的所有配置
go run view.go prod im-infra

# 查看指定配置的详细内容
go run view.go dev im-infra clog
```

#### 功能

1. 列出配置键
2. 查看配置详细内容
3. 格式化 JSON 输出

## 组件配置中心集成

### clog 配置中心集成

#### 基本使用

```go
package main

import (
    "github.com/ceyewan/gochat/im-infra/clog"
    "github.com/ceyewan/gochat/im-infra/coord"
)

func main() {
    // 1. 创建协调器
    coordinator, err := coord.New()
    if err != nil {
        panic(err)
    }
    defer coordinator.Close()

    // 2. 设置配置中心作为 clog 的配置源
    clog.SetupConfigCenter(coordinator.Config(), "prod", "im-infra", "clog")

    // 3. 初始化 clog（会从配置中心读取配置）
    err = clog.Init()
    if err != nil {
        // 如果配置中心不可用，会使用默认配置
        panic(err)
    }

    // 4. 使用日志
    clog.Info("Hello from config center!")
}
```

#### 高级使用

```go
// 手动设置配置源
adapter := clog.NewConfigCenterAdapter(coordinator.Config())
params := clog.ConfigParams{
    Env:       "prod",
    Service:   "im-infra",
    Component: "clog",
}
clog.SetConfigSource(adapter, params)

// 创建新的 logger（会从配置中心读取配置）
logger, err := clog.New()
```

#### 降级策略

clog 包实现了完善的降级策略：

1. **优先级**：配置中心 > 传入的配置 > 默认配置
2. **自动降级**：如果配置中心不可用，自动使用默认配置
3. **错误处理**：配置读取失败不会阻断程序运行

#### 动态配置更新

clog 支持动态配置更新：

1. 当配置中心的配置发生变化时，clog 会自动监听到变化
2. 自动重新初始化全局 logger 和模块 logger
3. 无需重启应用即可应用新配置

### 其他组件集成

其他组件（如 cache、db）也可以按照类似的方式集成配置中心：

1. 实现 ConfigSource 接口
2. 创建配置中心适配器
3. 在组件初始化时从配置中心读取配置
4. 支持动态配置更新

## 快速开始

### 1. 初始化配置

```bash
# 进入配置目录
cd config

# 初始化所有配置到配置中心
go run config.go

# 或者只初始化开发环境配置
go run config.go dev
```

### 2. 查看配置

```bash
# 查看所有配置
go run view.go

# 查看具体配置内容
go run view.go dev im-infra clog
```

### 3. 更新配置

```bash
# 动态更新配置
go run update.go dev im-infra clog '{"level":"debug","format":"json"}'
```

### 4. 在应用中使用

```go
// 设置配置中心
coordinator, _ := coord.New()
clog.SetupConfigCenter(coordinator.Config(), "dev", "im-infra", "clog")

// 初始化组件（自动从配置中心读取）
clog.Init()
```

## 测试和验证

### 运行演示程序

```bash
# 运行配置中心集成演示（不需要 etcd）
cd im-infra/clog/examples/config_fallback
go run main.go

# 运行动态配置更新演示（需要 etcd）
cd im-infra/clog/examples/dynamic_config
go run main.go
```

### 运行单元测试

```bash
cd im-infra/clog
go test -v -run "TestSetConfigSource|TestGetConfigFromSourceFallback|TestNewWithConfigSource|TestInitWithConfigSource"
```

## 功能验证

通过演示程序可以验证以下功能：

1. **配置初始化**：批量初始化多个组件配置
2. **配置查看**：列出和查看配置内容
3. **配置更新**：动态更新配置并立即生效
4. **降级策略**：配置中心不可用时自动降级
5. **动态更新**：配置变化时自动更新组件

## 扩展新组件

要为新组件添加配置中心支持：

### 1. 添加配置文件

```bash
# 创建配置文件
mkdir -p config/dev/im-infra
echo '{"key":"value"}' > config/dev/im-infra/newcomponent.json
```

### 2. 实现配置源接口

```go
// 在组件中实现 ConfigSource 接口
type NewComponentConfigSource struct {
    configCenter config.ConfigCenter
}

func (s *NewComponentConfigSource) GetConfig(ctx context.Context, env, service, component string) (*NewComponentConfig, error) {
    // 实现配置获取逻辑
}
```

### 3. 集成配置中心

```go
// 在组件初始化时集成配置中心
func SetupConfigCenter(configCenter config.ConfigCenter, env, service, component string) {
    adapter := NewConfigCenterAdapter(configCenter)
    params := ConfigParams{Env: env, Service: service, Component: component}
    SetConfigSource(adapter, params)
}
```

### 4. 初始化配置

```bash
# 使用统一工具初始化新组件配置
go run config.go dev im-infra newcomponent
```

## 最佳实践

1. **环境隔离**：不同环境使用不同的配置文件
2. **配置验证**：使用工具验证配置格式正确性
3. **版本控制**：配置文件纳入版本控制
4. **监控配置**：监控配置中心的可用性和配置变更
5. **降级测试**：定期测试配置中心不可用时的降级行为
6. **渐进部署**：先在开发环境验证配置，再推广到生产环境

## 故障排除

### 配置中心连接失败

**现象**：组件启动时提示配置中心连接失败

**解决方法**：
1. 检查 etcd 服务是否正常运行
2. 验证网络连接
3. 查看 coord 组件的错误日志
4. 组件会自动降级使用默认配置

### 配置格式错误

**现象**：配置初始化失败或解析错误

**解决方法**：
```bash
# 验证 JSON 格式
cd config
go run config.go dev im-infra clog
```

### 动态更新失败

**现象**：配置更新后组件没有应用新配置

**解决方法**：
1. 检查配置中心连接状态
2. 查看组件的错误日志
3. 手动重启组件或调用重新加载接口
