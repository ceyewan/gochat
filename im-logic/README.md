# IM-Logic Service

im-logic 是 GoChat 分布式即时通讯系统的逻辑处理服务，负责处理用户认证、会话管理、消息处理、群组管理等核心业务逻辑。

## 功能特性

- 🔐 用户认证和授权
- 💬 实时消息处理
- 👥 会话管理
- 🏠 群组管理
- 📨 消息路由和分发
- 🔄 异步任务处理
- 📊 监控和健康检查

## 架构设计

### 组件架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   im-gateway    │    │    im-logic     │    │    im-repo      │
│                 │    │                 │    │                 │
│  WebSocket API  │◄──►│   gRPC Server   │◄──►│   Data Store    │
│   HTTP API      │    │   Kafka Client  │    │   Cache Layer   │
│                 │    │   gRPC Client   │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │      Kafka      │
                    │                 │
                    │   Upstream      │
                    │   Downstream    │
                    │   Task Queue    │
                    └─────────────────┘
```

### 消息流程

1. **上行消息**：im-gateway → Kafka → im-logic → im-repo → Kafka → im-gateway
2. **下行消息**：im-logic → Kafka → im-gateway → Client
3. **异步任务**：im-logic → Kafka → im-task → 处理任务

## 快速开始

### 环境要求

- Go 1.21+
- Docker & Docker Compose
- MySQL 8.0+
- Redis 7.0+
- Kafka 3.4+
- etcd 3.5+

### 本地开发

1. **启动依赖服务**
   ```bash
   make dev
   ```

2. **构建并运行**
   ```bash
   make build
   ./bin/im-logic
   ```

3. **或直接运行**
   ```bash
   make run
   ```

### Docker 部署

1. **构建镜像**
   ```bash
   make docker-build
   ```

2. **启动服务**
   ```bash
   make docker-up
   ```

3. **查看日志**
   ```bash
   make docker-logs
   ```

## 配置说明

主要配置文件：`configs/config.yaml`

### 关键配置项

```yaml
# 服务器配置
server:
  grpc:
    host: "0.0.0.0"
    port: 9001
  http:
    port: 9002

# 数据库配置
database:
  host: "localhost"
  port: 3306
  name: "gochat"

# Redis 配置
redis:
  addr: "localhost:6379"

# Kafka 配置
kafka:
  brokers:
    - "localhost:9092"
  upstream_topic: "im-upstream-topic"
  downstream_topic_prefix: "im-downstream-topic-"

# JWT 配置
jwt:
  secret: "your-secret-key-here"
  access_token_expire: 24
```

## API 文档

### gRPC 接口

#### AuthService
- `Login` - 用户登录
- `Register` - 用户注册
- `RefreshToken` - 刷新令牌
- `Logout` - 用户登出
- `ValidateToken` - 验证令牌

#### ConversationService
- `GetConversations` - 获取会话列表
- `GetConversation` - 获取会话详情
- `GetMessages` - 获取历史消息
- `MarkAsRead` - 标记已读
- `GetUnreadCount` - 获取未读数

#### GroupService
- `CreateGroup` - 创建群组
- `GetGroup` - 获取群组信息
- `JoinGroup` - 加入群组
- `LeaveGroup` - 离开群组
- `GetGroupMembers` - 获取成员列表

### HTTP 健康检查

- `GET /health` - 健康检查
- `GET /ready` - 就绪检查
- `GET /live` - 存活检查
- `GET /metrics` - 监控指标

## 监控和日志

### 监控指标

服务集成了 Prometheus 监控指标，可通过以下端点访问：

- `http://localhost:9003/metrics`

### 日志配置

支持 JSON 和文本格式，可配置文件输出：

```yaml
logging:
  level: "info"
  format: "json"
  file_path: "/var/log/im-logic.log"
```

### 健康检查

内置健康检查机制，监控以下组件：
- gRPC 客户端连接
- Kafka 生产者/消费者
- 数据库连接
- Redis 连接

## 开发指南

### 项目结构

```
im-logic/
├── cmd/server/          # 主程序入口
├── internal/
│   ├── config/          # 配置管理
│   ├── server/          # 服务器组件
│   │   ├── grpc/        # gRPC 服务器
│   │   └── kafka/       # Kafka 客户端
│   └── service/         # 业务服务
├── configs/             # 配置文件
├── monitoring/          # 监控配置
├── Dockerfile           # Docker 镜像
├── docker-compose.yml   # Docker 编排
└── Makefile            # 构建脚本
```

### 开发流程

1. **安装依赖**
   ```bash
   make deps
   make install-tools
   ```

2. **代码检查**
   ```bash
   make fmt
   make lint
   ```

3. **运行测试**
   ```bash
   make test
   make test-coverage
   ```

4. **构建部署**
   ```bash
   make build
   make docker-build
   ```

### 调试技巧

1. **启用调试日志**
   ```bash
   LOG_LEVEL=debug ./bin/im-logic
   ```

2. **查看 Kafka 消息**
   ```bash
   docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic im-upstream-topic --from-beginning
   ```

3. **数据库连接测试**
   ```bash
   mysql -h localhost -u root -p gochat
   ```

## 性能优化

### 配置优化

1. **连接池配置**
   ```yaml
   database:
     max_conn: 100
     max_idle_conn: 20
   
   redis:
     pool_size: 20
     min_idle_conns: 5
   ```

2. **Kafka 批处理**
   ```yaml
   kafka:
     batch_size: 100
     batch_timeout: 100
   ```

### 缓存策略

- 用户信息缓存：Redis TTL 30分钟
- 会话信息缓存：Redis TTL 1小时
- 消息序列号：Redis 持久化存储

## 故障排除

### 常见问题

1. **服务启动失败**
   - 检查依赖服务是否正常
   - 确认配置文件路径正确
   - 查看日志文件排查错误

2. **Kafka 连接失败**
   - 检查 Kafka 服务状态
   - 确认 Topic 是否存在
   - 验证网络连接

3. **数据库连接失败**
   - 检查数据库服务状态
   - 确认连接参数正确
   - 验证用户权限

### 日志分析

```bash
# 查看错误日志
grep "ERROR" /var/log/im-logic.log

# 实时查看日志
tail -f /var/log/im-logic.log

# 过滤特定服务日志
grep "grpc-server" /var/log/im-logic.log
```

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交代码变更
4. 创建 Pull Request

## 许可证

MIT License

## 联系方式

- 项目主页：https://github.com/ceyewan/gochat
- 问题反馈：https://github.com/ceyewan/gochat/issues