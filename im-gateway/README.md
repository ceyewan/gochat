# im-gateway 网关服务

im-gateway 是 GoChat 即时通讯系统的网关服务，负责处理客户端连接、协议转换和消息路由。

## 功能特性

- **HTTP API 服务**: 提供用户认证、会话管理、群组操作等 RESTful API
- **WebSocket 连接管理**: 维护客户端长连接，处理实时消息传输
- **协议转换**: HTTP/WebSocket ↔ gRPC/Kafka 协议转换
- **用户认证**: JWT Token 验证和用户身份管理
- **消息路由**: 上行消息投递到 Kafka，下行消息推送给客户端
- **服务发现**: 基于 etcd 的服务注册和发现
- **高可用**: 无状态设计，支持水平扩展和故障转移

## 架构设计

```
客户端 → WebSocket/HTTP → im-gateway → Kafka (upstream) → im-logic
                                      ↓
im-gateway ← Kafka (downstream-{gateway_id}) ← im-logic
```

## 技术栈

- **HTTP 框架**: Gin
- **WebSocket**: gorilla/websocket
- **gRPC 客户端**: google.golang.org/grpc
- **消息队列**: Apache Kafka (franz-go)
- **服务发现**: etcd
- **日志**: im-infra/clog
- **配置管理**: im-infra/coord

## 快速开始

### 环境要求

- Go 1.21+
- Docker & Docker Compose
- etcd
- Apache Kafka
- Redis

### 本地开发

1. **启动依赖服务**
   ```bash
   make dev
   ```

2. **构建和运行**
   ```bash
   make build
   make run
   ```

3. **或者使用 Docker**
   ```bash
   make docker-up
   ```

### 服务地址

- **HTTP API**: http://localhost:8080
- **WebSocket**: ws://localhost:8080/ws
- **健康检查**: http://localhost:8080/health
- **监控指标**: http://localhost:9090/metrics

## API 接口

### 认证接口

```bash
# 用户登录
POST /api/v1/auth/login
{
  "username": "testuser",
  "password": "password123"
}

# 用户注册
POST /api/v1/auth/register
{
  "username": "testuser",
  "password": "password123",
  "nickname": "测试用户"
}

# 刷新令牌
POST /api/v1/auth/refresh
{
  "refresh_token": "refresh_token_here"
}
```

### WebSocket 连接

```javascript
// 建立 WebSocket 连接
const ws = new WebSocket('ws://localhost:8080/ws?token=your_jwt_token');

// 发送消息
ws.send(JSON.stringify({
  type: 'send-message',
  message_id: 'msg123',
  data: {
    conversation_id: 'conv123',
    message_type: 1,
    content: 'Hello World!',
    client_msg_id: 'client123'
  },
  timestamp: Date.now()
}));

// 接收消息
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('收到消息:', message);
};
```

## 消息协议

### 上行消息 (客户端 → 服务端)

```json
{
  "trace_id": "trace-123",
  "user_id": "user123",
  "gateway_id": "gateway-001",
  "conversation_id": "conv123",
  "message_type": 1,
  "content": "Hello World!",
  "client_msg_id": "client123",
  "timestamp": 1634567890,
  "headers": {}
}
```

### 下行消息 (服务端 → 客户端)

```json
{
  "trace_id": "trace-123",
  "target_user_id": "user456",
  "message_id": "msg123",
  "conversation_id": "conv123",
  "sender_id": "user123",
  "message_type": 1,
  "content": "Hello World!",
  "seq_id": 1001,
  "timestamp": 1634567890,
  "headers": {}
}
```

## 配置说明

### 主配置文件 (configs/config.yaml)

```yaml
server:
  http_addr: ":8080"
  ws_path: "/ws"
  read_timeout: 30s
  write_timeout: 30s
  
jwt:
  secret: "your-secret-key"
  access_token_expire: 24h
  refresh_token_expire: 168h

grpc:
  logic:
    service_name: "im-logic"
    direct_addr: "localhost:9001"

kafka:
  brokers:
    - "localhost:9092"

coordinator:
  endpoints:
    - "localhost:2379"
```

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `APP_ENV` | 运行环境 | `development` |
| `GATEWAY_ID` | 网关实例ID | 自动生成 |
| `LOG_LEVEL` | 日志级别 | `info` |

## 开发指南

### 项目结构

```
im-gateway/
├── cmd/main.go              # 应用入口
├── internal/
│   ├── config/              # 配置管理
│   └── server/              # 服务器实现
│       ├── http.go          # HTTP 服务器
│       ├── websocket.go     # WebSocket 处理器
│       ├── grpc.go          # gRPC 客户端
│       ├── kafka.go         # Kafka 消费者
│       └── server.go        # 主服务器
├── configs/                 # 配置文件
├── Dockerfile               # Docker 构建文件
├── docker-compose.yml       # Docker Compose 配置
├── Makefile                # 构建脚本
└── README.md               # 项目说明
```

### 常用命令

```bash
# 构建
make build

# 运行
make run

# 测试
make test

# 格式化代码
make fmt

# 检查代码
make lint

# 启动开发环境
make dev

# 停止开发环境
make dev-stop

# 查看日志
make logs

# 构建 Docker 镜像
make docker-build
```

### 添加新的 API 接口

1. 在 `internal/server/http.go` 中添加路由处理函数
2. 实现 HTTP 处理逻辑
3. 调用相应的 gRPC 接口
4. 添加适当的错误处理和日志记录

### 添加新的 WebSocket 消息类型

1. 在 `internal/server/websocket.go` 中定义新的消息类型
2. 在 `handleMessage` 函数中添加处理逻辑
3. 实现消息的序列化和反序列化

## 监控和日志

### 监控指标

- HTTP 请求计数和延迟
- WebSocket 连接数
- Kafka 消息生产/消费计数
- gRPC 调用计数和延迟

### 日志级别

- `debug`: 详细调试信息
- `info`: 一般信息
- `warn`: 警告信息
- `error`: 错误信息

### 结构化日志

```json
{
  "level": "info",
  "module": "im-gateway",
  "message": "用户连接建立",
  "user_id": "user123",
  "gateway_id": "gateway-001",
  "timestamp": "2023-10-20T10:30:00Z"
}
```

## 部署

### Docker 部署

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run
```

### Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: im-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: im-gateway
  template:
    metadata:
      labels:
        app: im-gateway
    spec:
      containers:
      - name: im-gateway
        image: im-gateway:latest
        ports:
        - containerPort: 8080
        - containerPort: 9090
        env:
        - name: APP_ENV
          value: "production"
```

## 故障排除

### 常见问题

1. **WebSocket 连接失败**
   - 检查 JWT Token 是否有效
   - 确认网络连接正常
   - 查看服务日志

2. **Kafka 消息发送失败**
   - 检查 Kafka 服务是否正常运行
   - 确认 Topic 是否存在
   - 检查网络连接

3. **gRPC 调用失败**
   - 检查 im-logic 服务是否可用
   - 确认服务发现配置正确
   - 检查网络连接

### 日志查看

```bash
# 查看 Docker 容器日志
make docker-logs

# 查看特定服务日志
docker logs im-gateway

# 实时查看日志
docker logs -f im-gateway
```

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License