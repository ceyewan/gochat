# GoChat 配置管理命令行工具 (`config-cli`)

本工具用于将本地的 JSON 配置文件同步到 etcd 配置中心，是开发环境中管理配置的核心工具。

## 🎯 核心功能

`config-cli` 简化了配置同步流程，支持以下核心操作：

- **同步所有配置**: 一次性将所有环境 (`dev`, `prod` 等) 的配置推送到 etcd。
- **同步指定环境**: 只推送特定环境（如 `dev`）的配置。

## 🚀 使用示例

所有命令都需要在 `config/config-cli/` 目录下执行。

### 1. 同步所有环境的配置

此命令会扫描 `config/` 目录下的所有环境，并将它们的配置同步到 etcd。

```bash
# 预览将要执行的操作
./config-cli sync --dry-run

# 确认并执行同步
./config-cli sync
```

### 2. 同步开发环境 (`dev`) 的配置

这是最常用的命令，用于将 `config/dev/` 目录下的所有配置同步到 etcd。

```bash
# 预览将要执行的操作
./config-cli sync dev --dry-run

# 确认并执行同步
./config-cli sync dev
```

## ⚙️ 全局选项

- `--endpoints`: 指定 etcd 的地址 (默认为 `localhost:2379`)。
- `--username`: etcd 的用户名。
- `--password`: etcd 的密码。
- `--timeout`: 操作超时时间 (默认为 `10s`)。
- `--config-path, -c`: 配置文件根目录的路径 (默认为 `..`)。
- `--dry-run`: 干运行模式，只显示将要执行的操作，不实际写入。
- `--force`: 强制执行，跳过交互式确认环节。

### 强制执行示例

在 CI/CD 或自动化脚本中，可以使用 `--force` 标志来避免交互式提示。

```bash
# 强制同步 dev 环境配置
./config-cli sync dev --force
