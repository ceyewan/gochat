# im-repo 数据仓储服务

im-repo 是即时通讯系统的数据仓储服务，负责处理用户、消息、群组和在线状态等数据的存储和管理。

## 功能特性

- **用户管理**：用户注册、登录、信息管理
- **消息存储**：消息持久化、序列号生成、幂等性检查
- **群组管理**：群组创建、成员管理、权限控制
- **会话管理**：会话列表、已读位置、未读消息统计
- **在线状态**：用户在线状态管理、TTL 控制
- **缓存优化**：Redis 缓存加速、Cache-Aside 策略
- **健康检查**：服务健康状态监控

## 技术栈

- **语言**：Go 1.21+
- **框架**：gRPC、GORM
- **数据库**：MySQL 8.0
- **缓存**：Redis 7.0
- **容器化**：Docker、Docker Compose

## 快速开始

### 环境要求

- Go 1.21+
- MySQL 8.0+
- Redis 7.0+
- Docker & Docker Compose（可选）

### 本地开发

1. **克隆代码**
```bash
git clone <repository-url>
cd im-repo
```

2. **安装依赖**
```bash
make tidy
```

3. **启动依赖服务**
```bash
# 使用 Docker Compose 启动 MySQL 和 Redis
docker-compose up -d mysql redis
```

4. **配置环境变量**
```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=password
export DB_NAME=im_repo
export REDIS_HOST=localhost
export REDIS_PORT=6379
```

5. **运行服务**
```bash
make run
```

### Docker 部署

1. **构建镜像**
```bash
make docker-build
```

2. **启动完整环境**
```bash
docker-compose up -d
```

3. **查看服务状态**
```bash
docker-compose ps
```

## 配置说明

### 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `SERVER_PORT` | 服务端口 | `8080` |
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `3306` |
| `DB_USER` | 数据库用户名 | `root` |
| `DB_PASSWORD` | 数据库密码 | - |
| `DB_NAME` | 数据库名称 | `im_repo` |
| `REDIS_HOST` | Redis 主机 | `localhost` |
| `REDIS_PORT` | Redis 端口 | `6379` |
| `REDIS_PASSWORD` | Redis 密码 | - |
| `LOG_LEVEL` | 日志级别 | `info` |

### 配置文件

配置文件位于 `configs/` 目录下，支持 YAML、JSON、TOML 格式。

## API 接口

### gRPC 服务

- **UserService**：用户管理服务
- **MessageService**：消息管理服务
- **ConversationService**：会话管理服务
- **GroupService**：群组管理服务
- **OnlineStatusService**：在线状态服务

### 健康检查

```bash
# gRPC 健康检查
grpc_health_probe -addr=localhost:8080
```

## 开发指南

### 项目结构

```
im-repo/
├── cmd/server/          # 服务入口
├── internal/
│   ├── config/         # 配置管理
│   ├── model/          # 数据模型
│   ├── repository/     # 数据仓储层
│   ├── service/        # gRPC 服务层
│   └── server/         # 服务器管理
├── scripts/            # 脚本文件
├── Dockerfile          # Docker 构建文件
├── docker-compose.yml  # Docker Compose 配置
├── Makefile           # 构建脚本
└── README.md          # 项目说明
```

### 开发工具

```bash
# 安装开发工具
make install-tools

# 代码格式化
make fmt

# 代码检查
make lint

# 运行测试
make test

# 测试覆盖率
make test-coverage
```

### 数据库迁移

数据库迁移在服务启动时自动执行，也可以手动运行：

```bash
# 查看迁移状态
go run cmd/migrate/main.go status

# 执行迁移
go run cmd/migrate/main.go up

# 回滚迁移
go run cmd/migrate/main.go down
```

## 监控和运维

### 日志

服务使用结构化日志，支持多种输出格式：

```bash
# 查看服务日志
docker-compose logs -f im-repo
```

### 指标监控

服务暴露 Prometheus 指标：

```bash
curl http://localhost:8080/metrics
```

### 性能调优

1. **数据库优化**
   - 合理使用索引
   - 分页查询优化
   - 连接池配置

2. **缓存优化**
   - 热点数据缓存
   - 缓存过期策略
   - 缓存穿透防护

3. **并发优化**
   - gRPC 连接池
   - 数据库连接池
   - 异步处理

## 故障排查

### 常见问题

1. **数据库连接失败**
   - 检查数据库服务状态
   - 验证连接参数
   - 查看网络连通性

2. **Redis 连接失败**
   - 检查 Redis 服务状态
   - 验证连接参数
   - 查看内存使用情况

3. **服务启动失败**
   - 检查端口占用
   - 查看配置文件
   - 检查依赖服务

### 调试模式

```bash
# 启用调试日志
export LOG_LEVEL=debug
make run
```

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证，详见 [LICENSE](LICENSE) 文件。