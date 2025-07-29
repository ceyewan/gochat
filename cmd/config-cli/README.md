# Config CLI - 统一配置管理命令行工具

一个功能完整的配置管理 CLI 工具，整合了配置初始化、查看、更新、监听等所有功能。

## 功能特性

- 📦 **配置初始化**：从本地 JSON 文件批量初始化配置到配置中心
- 🔍 **配置查看**：多种格式查看配置（树形、表格、JSON）
- 🔒 **原子更新**：使用 Compare-And-Set (CAS) 机制确保并发安全
- 🔄 **深度合并**：智能合并配置，只更新指定字段，保留其他字段
- 🗑️ **字段删除**：支持使用点号路径删除嵌套字段
- 👀 **实时监听**：监听配置变化，实时显示更新
- 📋 **列表查看**：列出指定前缀下的所有配置键
- 🛡️ **错误重试**：自动重试版本冲突，确保操作成功

## 安装

```bash
# 编译
cd cmd/config-cli
go build -o config-cli

# 或者直接运行
go run . --help
```

## 使用方法

### 全局选项

```bash
--endpoints strings    etcd endpoints (default [localhost:2379])
--username string      etcd username
--password string      etcd password
--timeout duration     operation timeout (default 10s)
```

### 配置初始化

从本地 JSON 文件批量初始化配置到配置中心。配置文件应按 `{env}/{service}/{component}.json` 结构组织。

```bash
# 初始化所有配置
config-cli init

# 初始化指定环境的所有配置
config-cli init dev

# 初始化指定环境和服务的所有配置
config-cli init dev im-infra

# 初始化指定的单个配置
config-cli init dev im-infra clog

# 指定配置文件目录
config-cli init --config-path ./config

# 干运行模式（只显示将要执行的操作）
config-cli init --dry-run

# 强制执行（不询问确认）
config-cli init --force
```

### 查看配置列表

```bash
# 列出所有配置（树形格式）
config-cli list

# 列出指定环境的配置
config-cli list dev

# 列出指定环境和服务的配置
config-cli list dev im-infra

# 表格格式显示
config-cli list --format table

# JSON 格式显示
config-cli list --format json

# 显示详细信息（包含配置内容）
config-cli list --detailed
```

### 查看具体配置

```bash
# 查看完整配置
config-cli get /config/dev/gochat/clog

# 使用自定义 etcd 地址
config-cli --endpoints etcd1:2379,etcd2:2379 get /config/prod/app/db
```

### 更新配置（深度合并）

```bash
# 只更新日志级别，保留其他配置
config-cli set /config/dev/gochat/clog '{"level":"debug"}'

# 更新嵌套配置
config-cli set /config/dev/gochat/db '{"connection":{"maxIdleConns":50}}'

# 批量更新多个字段
config-cli set /config/dev/gochat/clog '{"level":"info","format":"json","output":"stdout"}'
```

### 删除字段

```bash
# 删除顶级字段
config-cli delete /config/dev/gochat/clog enableColor

# 删除嵌套字段（使用点号路径）
config-cli delete /config/dev/gochat/db connection.maxIdleConns
config-cli delete /config/dev/gochat/clog rotation.maxSize
```

### 完全替换配置

```bash
# 完全替换整个配置
config-cli replace /config/dev/gochat/clog '{
  "level": "info",
  "format": "json",
  "output": "stdout",
  "enableColor": false
}'
```

### 实时监听配置变化

```bash
# 监听配置变化
config-cli watch /config/dev/gochat/clog

# 输出示例：
# 👀 Watching configuration changes for: /config/dev/gochat/clog
# Press Ctrl+C to stop...
# 
# 🔄 Configuration changed [PUT]: /config/dev/gochat/clog
# {
#   "level": "debug",
#   "format": "json"
# }
```

### 列出配置键

```bash
# 列出所有开发环境配置
config-cli list /config/dev

# 列出特定服务的配置
config-cli list /config/dev/gochat
```

## 完整工作流程

### 1. 项目初始化

```bash
# 1. 准备配置文件（按 env/service/component.json 结构）
mkdir -p config/dev/im-infra
echo '{"level":"info","format":"console"}' > config/dev/im-infra/clog.json

# 2. 批量初始化到配置中心
config-cli init --config-path ./config

# 3. 验证配置已写入
config-cli list dev im-infra
```

### 2. 配置文件结构

```
config/
├── dev/
│   └── im-infra/
│       ├── clog.json    # 日志配置
│       ├── db.json      # 数据库配置
│       └── cache.json   # 缓存配置
├── test/
│   └── im-infra/
│       └── clog.json
└── prod/
    └── im-infra/
        ├── clog.json
        ├── db.json
        └── cache.json
```

## 使用场景

### 1. 开发环境配置调试

```bash
# 快速切换日志级别进行调试
config-cli set /config/dev/gochat/clog '{"level":"debug"}'

# 调试完成后恢复
config-cli set /config/dev/gochat/clog '{"level":"info"}'
```

### 2. 生产环境配置更新

```bash
# 安全地更新数据库连接池大小
config-cli --endpoints prod-etcd:2379 set /config/prod/gochat/db '{
  "connection": {
    "maxOpenConns": 100,
    "maxIdleConns": 50
  }
}'
```

### 3. 配置迁移和备份

```bash
# 查看当前配置
config-cli get /config/dev/gochat/clog > clog-backup.json

# 应用到新环境
config-cli replace /config/test/gochat/clog "$(cat clog-backup.json)"
```

### 4. 监控配置变化

```bash
# 在一个终端监听配置变化
config-cli watch /config/prod/gochat/clog

# 在另一个终端进行配置更新
config-cli set /config/prod/gochat/clog '{"level":"warn"}'
```

## 安全特性

### 原子更新

使用 etcd 的 Compare-And-Set 机制，确保配置更新的原子性：

1. 获取当前配置和版本号
2. 在本地进行合并/修改
3. 只有当远程版本号未变时才允许更新
4. 如果版本冲突，自动重试

### 深度合并

智能合并策略，避免意外删除配置：

```bash
# 现有配置
{
  "level": "info",
  "format": "json",
  "output": "stdout",
  "rotation": {
    "maxSize": "100MB",
    "maxAge": 7
  }
}

# 执行更新
config-cli set /config/app/clog '{"level":"debug","rotation":{"maxSize":"200MB"}}'

# 结果（保留了其他字段）
{
  "level": "debug",        # 更新
  "format": "json",        # 保留
  "output": "stdout",      # 保留  
  "rotation": {
    "maxSize": "200MB",    # 更新
    "maxAge": 7            # 保留
  }
}
```

## 错误处理

工具具有完善的错误处理和重试机制：

- **版本冲突**：自动重试最多 5 次
- **网络错误**：显示详细错误信息
- **JSON 格式错误**：提供格式验证
- **字段路径错误**：验证删除路径的有效性

## 与旧工具的对比

| 功能 | 旧工具 (config/config.go + view.go) | 新工具 (config-cli) |
|------|-----------------------------------|---------------------|
| 配置初始化 | ✅ 支持批量初始化 | ✅ 支持批量初始化 + 干运行 |
| 配置查看 | ✅ 基本列表和查看 | ✅ 多格式显示 + 详细信息 |
| 配置更新 | ❌ 破坏性覆盖 | ✅ 智能深度合并 |
| 原子性 | ❌ 非原子操作 | ✅ CAS 原子更新 |
| 功能完整性 | ❌ 分散在多个工具 | ✅ 统一工具 |
| 部署 | ❌ 依赖源码环境 | ✅ 独立二进制 |
| 安全性 | ❌ 容易误删配置 | ✅ 并发安全 |
| 用户体验 | ❌ 多个命令行工具 | ✅ 统一直观界面 |

## 迁移指南

### 从旧工具迁移

```bash
# 旧方式：使用多个工具
cd config
go run config/config.go dev im-infra clog    # 初始化配置
go run view/view.go dev im-infra              # 查看配置
go run update/update.go dev im-infra clog '{"level":"debug"}'  # 更新配置

# 新方式：使用统一工具
config-cli init dev im-infra clog            # 初始化配置
config-cli list dev im-infra                 # 查看配置
config-cli set /config/dev/im-infra/clog '{"level":"debug"}'  # 更新配置
```

### 配置文件兼容性

新工具完全兼容现有的配置文件结构，无需修改任何配置文件。
