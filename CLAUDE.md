# CLAUDE.md

本文件旨在为 Claude Code (claude.ai/code) 在使用本仓库代码时提供指导。

## 项目概述

GoChat 是一个使用 Go 语言构建的分布式即时通讯系统，采用微服务架构。系统由四个核心服务组成：im-gateway（API/WebSocket）、im-logic（业务逻辑）、im-repo（数据层）和 im-task（异步处理）。

## 基本命令

### 开发环境设置
```bash
# 启动开发环境 (MySQL, Redis, Kafka, etcd)
make infra-up

# 安装开发工具 (golangci-lint, protoc 插件)
make install-tools

# 下载依赖
make tidy
```

### 日常开发
```bash
# 格式化代码
make fmt

# 运行静态分析
make lint

# 修改 .proto 文件后生成 protobuf 代码
make proto

# 运行所有测试，启用竞争检测并生成覆盖率报告
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
make infra-down

# 清理构建产物
make clean
```

## 架构概述

### 服务通信流程
```
客户端 → WebSocket → im-gateway → Kafka (上游) → im-logic → gRPC → im-repo
                                           ↓
im-logic → Kafka (下游) → im-gateway → WebSocket → 客户端
                                           ↓
im-logic → Kafka (任务) → im-task (异步处理)
```

### 服务职责
- **im-gateway**: 客户端 WebSocket/HTTP 连接，消息路由（端口 8080）
- **im-logic**: 业务逻辑、身份验证、消息处理（gRPC 端口 9001）
- **im-repo**: 数据持久化，用户/消息存储（gRPC 端口 9002）
- **im-task**: 后台任务，大群扇出（Kafka 消费者）

### 关键基础设施
- **MySQL**: 主要数据存储（用户、消息、会话、群组）
- **Redis**: 缓存和会话管理
- **Kafka**: 消息队列（主题：im-upstream-topic, im-downstream-topic-{gateway_id}, im-task-topic）
- **etcd**: 服务发现和配置中心

### 通信模式
- **gRPC**: 同步服务间调用 (gateway→logic, logic→repo)
- **Kafka**: 异步服务间消息传递
- **WebSocket**: 实时客户端连接

### 消息流
1. 客户端通过 WebSocket 将消息发送到 im-gateway
2. im-gateway 发布到 Kafka 上游主题
3. im-logic 消费、验证、处理业务逻辑
4. im-logic 通过 gRPC 调用 im-repo 进行数据持久化
5. im-logic 发布到 Kafka 下游主题进行投递
6. im-gateway 通过 WebSocket 将消息投递给目标客户端

## 开发指南

### 文档结构
项目包含完整的文档体系，开发前建议按以下顺序阅读：

1. **必读文档**（开发前必须了解）：
   - `docs/README.md` - 项目文档中心导航
   - `docs/01_architecture/01_overview.md` - 系统架构概览
   - `docs/01_architecture/02_dataflow.md` - 核心业务流程数据流图
   - `docs/01_architecture/03_tech_stack.md` - 技术栈说明

2. **API定义**（服务接口和数据结构）：
   - `api/proto/im_logic/v1/auth.proto` - 认证服务定义
   - `api/proto/im_logic/v1/conversation.proto` - 会话服务定义
   - `api/proto/im_logic/v1/message.proto` - 消息服务定义
   - `api/proto/im_repo/v1/user.proto` - 用户数据访问定义
   - `api/proto/im_repo/v1/conversation.proto` - 会话数据访问定义
   - `api/proto/im_repo/v1/message.proto` - 消息数据访问定义
   - `docs/02_apis/00_openapi.yaml` - OpenAPI HTTP API规范
   - `docs/02_apis/01_http_ws_api.md` - HTTP和WebSocket API文档
   - `docs/02_apis/02_grpc_services.md` - 内部gRPC服务文档
   - `docs/02_apis/03_mq_topics.md` - Kafka Topic定义

3. **数据模型**（数据库设计）：
   - `docs/06_data_models/01_db_schema.md` - MySQL数据库架构详细说明
   - `docs/06_data_models/02_core_im_flows.md` - IM核心功能实现流程
   - `docs/06_data_models/03_auth_and_sync_flows.md` - 认证与同步流程
   - `docs/06_data_models/04_world_chat_scalability.md` - 世界聊天室扩展性设计

4. **基础设施组件**（im-infra）：
   - `docs/08_infra/usage_guide.md` - 所有组件的生产级别初始化范例
   - `docs/08_infra/breaker.md` - 熔断器组件
   - `docs/08_infra/cache.md` - 缓存组件
   - `docs/08_infra/clog.md` - 日志组件
   - `docs/08_infra/coord.md` - 分布式协调组件
   - `docs/08_infra/db.md` - 数据库组件
   - `docs/08_infra/es.md` - 消息索引组件
   - `docs/08_infra/metrics.md` - 可观测性组件
   - `docs/08_infra/mq.md` - 消息队列组件
   - `docs/08_infra/once.md` - 幂等操作组件
   - `docs/08_infra/ratelimit.md` - 分布式限流组件
   - `docs/08_infra/uid.md` - 唯一ID生成组件

5. **开发指南**：
   - `docs/03_development/01_workflow.md` - 开发工作流程概述
   - `docs/03_development/02_style_guide.md` - 代码风格和约定标准
   - `docs/03_development/03_service_guide.md` - 微服务开发指南
   - `docs/03_development/04_testing_strategy.md` - 测试策略与规范

6. **部署指南**：
   - `docs/04_deployment/README.md` - 部署与配置指南

7. **服务文档**：
   - `docs/05_services/im-gateway.md` - 网关服务详细设计文档
   - `docs/05_services/im-logic.md` - 业务逻辑服务文档
   - `docs/05_services/im-repo.md` - 数据仓库服务文档
   - `docs/05_services/im-task.md` - 任务处理服务文档

8. **重构计划**：
   - `docs/07_todo_task/README.md` - 项目后续的核心重构和开发计划书

### 消息队列Topic设计
- **核心消息流**：`gochat.messages.upstream`、`gochat.messages.persist`、`gochat.messages.downstream.{gateway_id}`
- **领域事件**：`gochat.user-events`、`gochat.conversation-events`、`gochat.message-events`、`gochat.notifications`

### 核心数据表
- **`users`**: 用户基本信息
- **`conversations`**: 会话信息（单聊、群聊、世界聊天室统一抽象）
- **`conversation_members`**: 会话成员关系
- **`messages`**: 消息内容
- **`user_read_pointers`**: 用户已读状态指针

### 使用 Protobuf
- 所有服务 API 都定义在 `/api/proto/` 中
- 修改 .proto 文件后运行 `make proto`
- 生成的代码位于 `/api/gen/` 中

### 测试
- 使用 Go 内置测试和 testify 断言
- 默认启用竞争检测（`-race` 标志）
- 覆盖率报告生成在 `coverage.html`
- 使用 `make test-service SERVICE=<service>` 测试特定服务代码

### 配置
- 基于 YAML 的配置，支持环境变量覆盖
- 每个服务在 `/configs/` 目录中都有自己的配置
- 通过 etcd 进行运行时配置更新

### 数据库
- MySQL 模式初始化在 `/scripts/mysql/init.sql` 中
- 使用 GORM 作为 ORM，支持 MySQL、PostgreSQL、SQLite
- 迁移脚本应添加到 `/scripts/mysql/`

### 错误处理
- 带有错误码的结构化错误响应
- 服务保护的熔断器模式
- 优雅停机处理
- 所有服务的健康检查端点

### 日志
- 带有上下文信息的结构化 JSON 日志
- 日志级别：debug, info, warn, error
- 带请求 ID 的上下文日志用于追踪
- 参考 [](./docs/08_infra/clog.md)

## 关键依赖
- **Web 框架**: Gin 用于 HTTP，gorilla/websocket 用于 WebSocket
- **gRPC**: 服务间通信
- **Kafka**: franz-go 客户端用于消息队列
- **数据库**: GORM 支持 MySQL/PostgreSQL/SQLite
- **Redis**: go-redis 用于缓存
- **身份验证**: JWT 与 golang.org/x/crypto
- **可观测性**: OpenTelemetry 用于追踪，Prometheus 用于指标