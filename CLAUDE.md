# CLAUDE.md

此文件为 Claude Code (claude.ai/code) 在处理此仓库代码时提供指导。

## 项目概述

GoChat 是一个使用 Go 构建的分布式即时消息系统，采用微服务架构。该系统由四个核心服务组成：im-gateway（API/WebSocket）、im-logic（业务逻辑）、im-repo（数据层）和 im-task（异步处理）。

## 基本命令

### 开发设置
```bash
# 启动开发环境（MySQL、Redis、Kafka、etcd）
make dev

# 安装开发工具（golangci-lint、protoc 插件）
make install-tools

# 下载依赖项
make deps
```

### 日常开发
```bash
# 格式化代码
make fmt

# 代码检查（如果需要安装 golangci-lint）
make lint

# 修改 .proto 文件后生成 protobuf 代码
make proto

# 使用竞态检测和覆盖率运行所有测试
make test

# 测试特定服务
make test-service SERVICE=im-gateway
```

### 构建
```bash
# 构建所有服务
make build

# 构建特定服务
make build-service SERVICE=im-logic

# 构建 Docker 镜像
make docker-build
```

### 清理
```bash
# 停止开发环境
make dev-down

# 清理构建产物
make clean
```

## 架构概述

### 服务通信流程
```
Client → WebSocket → im-gateway → Kafka (upstream) → im-logic → gRPC → im-repo
                                           ↓
im-logic → Kafka (downstream) → im-gateway → WebSocket → Client
                                           ↓
im-logic → Kafka (task) → im-task (async processing)
```

### 服务职责
- **im-gateway**：客户端 WebSocket/HTTP 连接、消息路由（端口 8080）
- **im-logic**：业务逻辑、认证、消息处理（gRPC 端口 9001）
- **im-repo**：数据持久化、用户/消息存储（gRPC 端口 9002）
- **im-task**：后台任务、大群组扇出（Kafka 消费者）

### 关键基础设施
- **MySQL**：主要数据存储（用户、消息、对话、群组）
- **Redis**：缓存和会话管理
- **Kafka**：消息队列（主题：im-upstream-topic、im-downstream-topic-{gateway_id}、im-task-topic）
- **etcd**：服务发现和配置中心

### 通信模式
- **gRPC**：同步服务间调用（gateway→logic、logic→repo）
- **Kafka**：异步服务间消息传递
- **WebSocket**：实时客户端连接

### 消息流程
1. 客户端通过 WebSocket 向 im-gateway 发送消息
2. im-gateway 发布到 Kafka 上游主题
3. im-logic 消费、验证、处理业务逻辑
4. im-logic 通过 gRPC 调用 im-repo 持久化数据
5. im-logic 发布到 Kafka 下游主题进行交付
6. im-gateway 通过 WebSocket 交付到目标客户端

## 开发指南

### 使用 Protobuf
- 所有服务 API 在 `/api/proto/` 中定义
- 修改 .proto 文件后运行 `make proto`
- 生成的代码位于 `/api/gen/`

### 测试
- 使用 Go 内置测试与 testify 断言
- 默认启用竞态检测（`-race` 标志）
- 覆盖率报告生成在 `coverage.html`
- 使用 `make test-service SERVICE=<service>` 测试特定服务代码

### 配置
- 基于 YAML 的配置，支持环境变量覆盖
- 每个服务的配置在 `/config/` 目录
- 通过 etcd 进行运行时配置更新

### 数据库
- MySQL 模式初始化在 `/scripts/mysql/init.sql`
- GORM 用于 ORM，支持 MySQL、PostgreSQL、SQLite
- 迁移脚本应添加到 `/scripts/mysql/`

### 错误处理
- 结构化错误响应与错误代码
- 熔断器模式用于服务保护
- 优雅关闭处理
- 所有服务的健康检查端点

### 日志记录
- 结构化 JSON 日志与上下文信息
- 日志级别：debug、info、warn、error
- 使用上下文日志记录请求 ID 进行跟踪

## 关键依赖项
- **Web 框架**：Gin 用于 HTTP，gorilla/websocket 用于 WebSocket
- **gRPC**：服务间通信
- **Kafka**：franz-go 客户端用于消息队列
- **数据库**：GORM 支持 MySQL/PostgreSQL/SQLite
- **Redis**：go-redis 用于缓存
- **认证**：JWT 与 golang.org/x/crypto
- **可观测性**：OpenTelemetry 用于跟踪，Prometheus 用于指标