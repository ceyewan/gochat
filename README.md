# GoChat 微服务即时通讯系统

GoChat 是一个基于 Go 语言构建的现代化、分布式的即时通讯（IM）系统。采用微服务架构，实现高性能、高可用、可扩展的后端服务。

## 🏗️ 系统架构

### 微服务组件

- **im-gateway**: 网关服务，处理客户端连接和协议转换
- **im-logic**: 业务逻辑服务，处理核心业务逻辑和消息分发
- **im-repo**: 数据仓储服务，统一的数据访问层
- **im-task**: 异步任务服务，处理重负载和非实时任务

### 基础设施

- **MySQL**: 主数据库，存储用户、消息、群组等核心数据
- **Redis**: 缓存和会话存储
- **Kafka**: 消息队列，用于服务间异步通信
- **etcd**: 服务发现和配置中心
- **Elasticsearch**: 全文搜索引擎（可选）
- **MinIO**: 对象存储服务（可选）

## 📁 项目结构

```
gochat/
├── api/                          # API 定义
│   ├── proto/                    # protobuf 接口定义
│   │   ├── im_logic/v1/         # im-logic 服务接口
│   │   └── im_repo/v1/          # im-repo 服务接口
│   ├── kafka/                   # Kafka 消息协议
│   └── gen/                     # 生成的代码（自动生成）
├── im-gateway/                  # 网关服务
│   ├── cmd/                     # 服务入口
│   ├── internal/                # 内部实现
│   │   ├── config/             # 配置管理
│   │   ├── server/             # 服务器实现
│   │   ├── handler/            # HTTP/WebSocket 处理器
│   │   ├── middleware/         # 中间件
│   │   └── client/             # gRPC 客户端
│   └── Dockerfile              # Docker 构建文件
├── im-logic/                   # 业务逻辑服务
│   ├── cmd/                    # 服务入口
│   ├── internal/               # 内部实现
│   │   ├── config/            # 配置管理
│   │   ├── server/            # gRPC 服务器
│   │   ├── service/           # 业务服务实现
│   │   ├── consumer/          # Kafka 消费者
│   │   └── client/            # 下游服务客户端
│   └── Dockerfile             # Docker 构建文件
├── im-repo/                   # 数据仓储服务
│   ├── cmd/                   # 服务入口
│   ├── internal/              # 内部实现
│   │   ├── config/           # 配置管理
│   │   ├── server/           # gRPC 服务器
│   │   ├── service/          # 数据服务实现
│   │   ├── model/            # 数据模型
│   │   ├── repository/       # 数据访问层
│   │   └── cache/            # 缓存管理
│   └── Dockerfile            # Docker 构建文件
├── im-task/                  # 异步任务服务
│   ├── cmd/                  # 服务入口
│   ├── internal/             # 内部实现
│   │   ├── config/          # 配置管理
│   │   ├── server/          # 任务处理服务器
│   │   ├── processor/       # 任务处理器
│   │   └── dispatcher/      # 任务分发器
│   └── Dockerfile           # Docker 构建文件
├── im-infra/                # 基础设施库（已存在）
├── docs/                    # 项目文档
├── scripts/                 # 脚本文件
├── configs/                 # 配置文件
├── docker-compose.dev.yml   # 开发环境配置
├── Makefile                 # 构建脚本
└── go.mod                   # Go 模块定义
```

## 🚀 快速开始

### 环境要求

- Go 1.21+
- Docker & Docker Compose
- Protocol Buffers 编译器

### 1. 克隆项目

```bash
git clone https://github.com/ceyewan/gochat.git
cd gochat
```

### 2. 安装开发工具

```bash
make install-tools
```

### 3. 启动开发环境

```bash
# 启动基础设施服务（MySQL, Redis, Kafka, etcd 等）
make dev

# 等待服务启动完成（约 30 秒）
```

### 4. 生成 protobuf 代码

```bash
make proto
```

### 5. 构建服务

```bash
# 构建所有服务
make build

# 或构建单个服务
make build-service SERVICE=im-gateway
```

### 6. 运行服务

```bash
# 运行网关服务
./bin/im-gateway

# 运行业务逻辑服务
./bin/im-logic

# 运行数据仓储服务
./bin/im-repo

# 运行异步任务服务
./bin/im-task
```

## 🔧 开发指南

### 生成 protobuf 代码

```bash
make proto
```

### 运行测试

```bash
# 运行所有测试
make test

# 运行指定服务的测试
make test-service SERVICE=im-gateway
```

### 代码检查和格式化

```bash
# 代码检查
make lint

# 格式化代码
make fmt
```

### Docker 构建

```bash
# 构建 Docker 镜像
make docker-build

# 推送 Docker 镜像
make docker-push
```

## 📖 核心功能

### 用户认证
- 用户注册和登录
- JWT 令牌认证
- 令牌刷新机制

### 实时通讯
- 单聊消息
- 群聊消息
- 世界聊天室
- WebSocket 长连接

### 会话管理
- 会话列表
- 历史消息查询
- 消息已读状态
- 未读消息计数

### 群组管理
- 创建群组
- 加入/退出群组
- 群组成员管理
- 群组权限控制

## 🔌 接口文档

### HTTP API
- 用户认证: `/api/v1/auth/*`
- 会话管理: `/api/v1/conversations/*`
- 群组管理: `/api/v1/groups/*`

### WebSocket
- 连接地址: `/ws`
- 支持消息类型: `send-message`, `new-message`, `ping/pong`

### gRPC 服务
- im-logic: 业务逻辑接口
- im-repo: 数据访问接口

## 🛠️ 配置说明

### 环境变量

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=3306
DB_USER=gochat
DB_PASSWORD=gochat123
DB_NAME=gochat

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=redis123

# Kafka 配置
KAFKA_BROKERS=localhost:9092

# etcd 配置
ETCD_ENDPOINTS=localhost:2379
```

### 配置文件

每个服务都有独立的配置文件，支持 YAML 格式：

```yaml
# im-gateway 配置示例
server:
  http_addr: ":8080"
  ws_path: "/ws"
  
jwt:
  secret: "your-secret-key"
  access_token_expire: "24h"
  
grpc:
  logic:
    service_name: "im-logic"
    direct_addr: "localhost:9001"
```

## 📊 监控和日志

### 健康检查
- HTTP 端点: `/health`
- 返回服务状态和依赖检查结果

### 日志格式
- 结构化 JSON 日志
- 包含 TraceID 用于链路追踪
- 支持不同日志级别

### 监控指标
- 服务性能指标
- 业务指标（消息量、在线用户数等）
- 基础设施指标

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🆘 支持

如果您遇到问题或有疑问，请：

1. 查看 [文档](docs/)
2. 搜索 [Issues](https://github.com/ceyewan/gochat/issues)
3. 创建新的 Issue

## 🗺️ 路线图

- [x] 基础架构搭建
- [x] 用户认证系统
- [x] 实时消息通讯
- [x] 群组管理
- [ ] 文件上传下载
- [ ] 消息推送
- [ ] 全文搜索
- [ ] 性能优化
- [ ] 监控告警