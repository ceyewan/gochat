# config-cli - 简化的配置管理工具

`config-cli` 是一个轻量级的配置管理工具，专注于将 JSON 配置文件原子地写入 etcd 配置中心。

## 🎯 设计理念

- **简单专注**：只提供一个核心功能，避免功能膨胀。
- **原子操作**：配置更新即覆盖，确保数据一致性。
- **零学习成本**：命令简单直观，无需复杂参数。

## 🚀 快速开始

### 1. 构建工具

首先，进入 `config-cli` 目录并构建二进制文件。

```bash
cd config/config-cli
go build -o config-cli
```

### 2. 理解配置文件结构

工具期望的配置文件结构如下，其中 `config` 是根目录：

```
config/
└── dev/                    # 环境 (env)
    ├── im-repo/            # 服务 (service)
    │   ├── cache.json      # 组件 (component)
    │   ├── db.json
    │   └── clog.json
    └── im-logic/
        ├── clog.json
        └── mq.json
```

### 3. 基本用法

从 `config/config-cli` 目录执行命令。

```bash
# 同步所有环境的配置
# (--config-path 默认为 '..', 指向 config/ 目录)
./config-cli sync

# 同步开发环境 (dev) 的所有配置
./config-cli sync dev

# 同步特定服务的配置
./config-cli sync dev im-repo

# 同步特定组件的配置
./config-cli sync dev im-repo cache
```

## ⚠️ 注意事项

- **执行路径**：请务必在 `config/config-cli` 目录下执行 `./config-cli` 命令。
- **配置根目录**：`--config-path` 参数应指向包含**环境目录**（如 `dev`, `prod`）的根目录。默认值为 `..`，即 `config/` 目录。
- **覆盖模式**：此工具会**完全覆盖** etcd 中的现有配置，而不是合并。

## ⚙️ 命令行选项

### 全局选项

| 选项 | 默认值 | 说明 |
|--------------|--------------------|--------------------|
| `--endpoints`| `localhost:2379` | etcd 端点列表 |
| `--username` | "" | etcd 用户名 |
| `--password` | "" | etcd 密码 |
| `--timeout` | `10s` | 操作超时时间 |

### `sync` 命令选项

| 选项 | 默认值 | 说明 |
|---------------|---------|--------------------------------|
| `--config-path`| `..` | 配置文件根目录 (包含环境文件夹) |
| `--dry-run` | `false` | 预览模式，不实际写入 |
| `--force` | `false` | 强制执行，跳过确认 |

## 📋 使用示例

### 1. 预览变更

在不实际写入的情况下，预览将要同步的配置。

```bash
# 预览 dev 环境的所有变更
./config-cli sync dev --dry-run

# 输出示例:
# 📋 找到 5 个配置文件:
#
# 🌍 环境: dev
#   📦 服务: im-repo
#     ⚙️  cache -> /config/dev/im-repo/cache
#     ⚙️  db -> /config/dev/im-repo/db
#     ⚙️  clog -> /config/dev/im-repo/clog
#   📦 服务: im-logic
#     ⚙️  clog -> /config/dev/im-logic/clog
#     ⚙️  mq -> /config/dev/im-logic/mq
#
# 🔍 干运行模式：不会实际写入配置中心
```

### 2. 同步特定配置并强制执行

```bash
# 强制同步 im-repo 服务的 cache 配置，跳过确认
./config-cli sync dev im-repo cache --force
```

### 3. 连接远程 etcd

```bash
# 使用指定的 etcd 集群信息同步配置
./config-cli sync prod \
  --endpoints etcd1:2379,etcd2:2379 \
  --username admin \
  --password 'your-secret'
```

## 🔧 工作原理

1.  **扫描**：从 `--config-path` 指定的路径开始，递归查找所有匹配 `{env}/{service}/{component}.json` 格式的文件。
2.  **过滤**：根据 `sync` 命令提供的 `[env]`, `[service]`, `[component]` 参数筛选文件。
3.  **确认**：显示操作摘要，等待用户确认（除非使用 `--force`）。
4.  **写入**：将每个配置文件的**原始 JSON 内容**原子地写入 etcd，键路径为 `/config/{env}/{service}/{component}`。
