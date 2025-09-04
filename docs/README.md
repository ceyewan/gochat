# GoChat 项目文档中心

欢迎来到 GoChat 项目的统一文档中心。本文档旨在为所有开发者、测试人员和运维人员提供一个清晰、全面、一致的知识库。

## 1. 文档结构

本中心的所有文档都组织在 `new_docs` 目录下，结构如下：

-   **/architecture**: 包含系统的高层设计文档。
    -   [**`overview.md`**](./architecture/overview.md): 宏观架构设计，包括系统组件、技术选型和设计目标。
    -   [**`dataflow.md`**](./architecture/dataflow.md): 核心业务场景（如登录、消息发送）的数据流转图。
-   **/services**: 包含对每个微服务的详细设计说明。
    -   [**`im-infra.md`**](./services/im-infra.md): `im-infra` 基础库的设计理念和开发规范。
    -   [**`im-gateway.md`**](./services/im-gateway.md): `im-gateway` 网关层的职责、架构和核心流程。
    -   [**`im-logic.md`**](./services/im-logic.md): `im-logic` 业务逻辑层的职责、架构和核心流程。
    -   [**`im-task.md`**](./services/im-task.md): `im-task` 异步任务层的职责、架构和核心流程。
    -   [**`im-repo.md`**](./services/im-repo.md): `im-repo` 数据仓储层的职责、数据模型和缓存策略。
-   [**`api.md`**](./api.md): 统一的 API 参考，包括客户端的 RESTful/WebSocket 接口和内部的 gRPC/Kafka 协议。

## 2. 快速开始

-   **了解系统全貌**: 请从阅读 [**架构总览 (`overview.md`)**](./architecture/overview.md) 开始。
-   **理解核心流程**: 接着阅读 [**核心场景数据流 (`dataflow.md`)**](./architecture/dataflow.md)。
-   **深入特定服务**: 如果您需要开发或维护某个特定的微服务，请阅读 `/services` 目录下对应的文档。
-   **接口开发**: 如果您是前端开发者或需要与其他服务集成，请参考 [**API 统一规范 (`api.md`)**](./api.md)。

## 3. 项目简介

GoChat 是一个基于 Go 语言构建的现代化、分布式的即时通讯（IM）系统。它采用微服务架构，旨在实现一个功能完整、性能卓越且易于扩展的后端服务，为前端应用（Web/App）提供稳定可靠的实时通信能力。

### 核心功能
- **用户体系**: 用户注册、登录、认证与会话管理。
- **实时通讯**: 实现一对一单聊、多对多群聊、世界聊天室。
- **核心体验**: 会话列表管理、历史消息拉取、在线状态同步、消息已读回执。
- **多媒体消息**: 支持图片、文件等对象的上传、下载与发送。
- **智能服务 (V3.0 新增)**:
    - **AI 助手**: 提供上下文感知的会话式 AI 代理和群聊智能摘要。
    - **全局搜索**: 基于 Elasticsearch 实现聊天记录、用户、文件的全文检索。
    - **内容推荐**: 提供“可能认识的人”和“可能感兴趣的群”等推荐功能。
