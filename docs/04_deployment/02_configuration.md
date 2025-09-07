# 配置管理

本文档解释如何管理 GoChat 微服务的配置。

## 1. 概述

GoChat 使用由 `etcd` 驱动的集中式配置管理系统。配置文件以 JSON 格式编写并存储在 `/config/dev` 目录中。提供了一个命令行工具 `config-cli`，用于将这些本地文件同步到 `etcd` 服务器。

## 2. 配置结构

-   **本地文件**: 所有配置文件都位于 `config/dev/` 中。每个服务都有自己的子目录，该服务中的每个组件都有自己的 JSON 文件。
    -   示例：`config/dev/im-repo/db.json`
-   **etcd 路径**: `etcd` 中的配置路径遵循严格的模式：
    -   `/config/{environment}/{service}/{component}`
    -   示例：`/config/dev/im-repo/db`

## 3. `config-cli` 工具

`config-cli` 工具用于将本地 JSON 配置文件与 `etcd` 服务器同步。

-   **位置**: `config/config-cli/`
-   **用法**:
    ```bash
    # 导航到工具的目录
    cd config/config-cli

    # 同步 'dev' 环境的所有配置
    ./config-cli sync dev

    # 同步特定服务的所有配置
    ./config-cli sync dev im-repo

    # 同步单个组件的配置
    ./config-cli sync dev im-repo db
    ```

## 4. 服务中的配置加载

微服务以两阶段过程加载其配置以确保弹性。

1.  **引导阶段**: 启动时，服务加载一个最小的、硬编码的默认配置。这足以初始化日志记录和 `coord`（etcd 客户端）组件。
2.  **完整配置加载**: 然后服务使用 `coord` 组件连接到 `etcd` 并获取其完整配置。
    -   如果与 `etcd` 的连接失败，服务将继续使用默认配置运行，确保即使配置服务暂时不可用也能启动。
    -   `coord` 组件还监视 `etcd` 中的更改，允许在不重新启动服务的情况下进行动态热重载配置。

## 5. 配置模式示例

### `db.json`（数据库）

```json
{
  "dsn": "user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
  "driver": "mysql",
  "maxOpenConns": 25,
  "maxIdleConns": 10
}
```

### `cache.json`（Redis 缓存）

```json
{
  "addr": "redis:6379",
  "password": "",
  "db": 0
}
```

### `coord.json`（etcd）

```json
{
  "endpoints": ["etcd1:2379", "etcd2:2379", "etcd3:2379"],
  "timeout": "5s"
}
```

### `clog.json`（日志记录）

```json
{
  "level": "info",
  "format": "json",
  "output": "file",
  "filename": "/app/logs/app.log"
}