# GoChat 日志系统设置指南

## 🎯 日志架构概览

我们采用现代化的日志收集和分析方案：

```
应用服务 → Vector → Loki → Grafana
```

### 核心组件

1. **Vector**: 高性能日志收集器，负责收集和处理日志
2. **Loki**: 轻量级日志存储，类似 Prometheus 的设计理念
3. **Grafana**: 统一可视化平台，支持日志查询和仪表板
4. **RedisInsight**: 现代化的 Redis 管理界面

## 🔧 配置说明

### 应用日志配置

应用服务需要配置以下环境变量：

```yaml
environment:
  - LOG_LEVEL=info
  - LOG_FORMAT=json
  - LOG_OUTPUT=file
  - LOG_FILE=/app/logs/app.log
```

### im-infra/clog 配置示例

```json
{
  "level": "info",
  "format": "json",
  "output": "file",
  "filename": "/app/logs/app.log",
  "maxSize": 100,
  "maxBackups": 3,
  "maxAge": 7,
  "compress": true,
  "initialFields": {
    "service": "im-repo",
    "environment": "dev",
    "version": "1.0.0"
  }
}
```

## 📊 服务访问地址

### 管理界面

| 服务 | 地址 | 用户名/密码 | 用途 |
|------|------|-------------|------|
| Grafana | http://localhost:3000 | admin/gochat_grafana_2024 | 日志查询和可视化 |
| RedisInsight | http://localhost:8001 | - | Redis 可视化管理 |
| Kafka UI | http://localhost:8080 | - | Kafka 集群管理 |
| etcd Manager | http://localhost:8081 | - | etcd 集群管理 |
| phpMyAdmin | http://localhost:8083 | - | MySQL 管理 |

### API 端点

| 服务 | 地址 | 用途 |
|------|------|------|
| Loki API | http://localhost:3100 | 日志数据查询 |
| Vector API | http://localhost:8686 | 日志收集器状态 |
| Prometheus | http://localhost:9090 | 指标数据查询 |
| Jaeger | http://localhost:16686 | 分布式链路追踪 |

## 🚀 使用指南

### 1. 启动日志系统

```bash
# 启动基础设施（包含日志组件）
./scripts/start-infra.sh

# 检查日志组件状态
./scripts/health-check.sh --component monitoring
```

### 2. 在 Grafana 中查看日志

1. 访问 http://localhost:3000
2. 使用 admin/gochat_grafana_2024 登录
3. 导航到 "Explore" 页面
4. 选择 "Loki" 数据源
5. 使用 LogQL 查询语法查询日志

### 3. 常用 LogQL 查询示例

```logql
# 查看所有应用日志
{environment="dev"}

# 查看特定服务的日志
{environment="dev", service="im-repo"}

# 查看错误级别日志
{environment="dev", level="ERROR"}

# 搜索包含特定关键词的日志
{environment="dev"} |= "error"

# 查看最近5分钟的日志统计
sum by (service) (count_over_time({environment="dev"}[5m]))
```

### 4. 日志仪表板

系统预置了以下仪表板：

- **GoChat 日志概览**: 显示日志量分布、级别分布和实时日志流
- **服务日志详情**: 按服务分类的详细日志分析
- **错误日志监控**: 专门监控错误和异常日志

## 🔍 日志格式规范

### 标准日志字段

所有应用日志应包含以下标准字段：

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "service": "im-repo",
  "module": "user-service",
  "message": "User created successfully",
  "trace_id": "abc123def456",
  "user_id": "12345",
  "request_id": "req-789",
  "environment": "dev",
  "version": "1.0.0"
}
```

### 日志级别使用指南

- **DEBUG**: 详细的调试信息，仅在开发环境使用
- **INFO**: 一般信息，记录正常的业务流程
- **WARN**: 警告信息，可能的问题但不影响功能
- **ERROR**: 错误信息，需要关注和处理的问题
- **FATAL**: 致命错误，导致服务无法继续运行

## 🛠️ 故障排除

### 常见问题

1. **日志没有出现在 Grafana 中**
   - 检查 Vector 是否正常运行：`docker logs gochat-vector`
   - 检查 Loki 是否正常运行：`docker logs gochat-loki`
   - 验证日志文件是否正确挂载：`docker exec gochat-vector ls -la /var/log/apps`

2. **Vector 无法读取日志文件**
   - 检查文件权限：确保 Vector 容器有读取权限
   - 检查文件路径：确保日志文件路径正确
   - 查看 Vector 配置：`docker exec gochat-vector cat /etc/vector/vector.toml`

3. **Grafana 无法连接 Loki**
   - 检查网络连接：`docker exec gochat-grafana ping loki`
   - 检查 Loki 健康状态：`curl http://localhost:3100/ready`
   - 检查 Grafana 数据源配置

### 调试命令

```bash
# 查看 Vector 日志
docker logs gochat-vector -f

# 查看 Loki 日志
docker logs gochat-loki -f

# 测试 Loki API
curl http://localhost:3100/loki/api/v1/labels

# 查看 Vector 配置
docker exec gochat-vector cat /etc/vector/vector.toml

# 检查日志文件
docker exec gochat-vector ls -la /var/log/apps/
```

## 📈 性能优化

### Vector 优化

- 调整批量大小：`batch.max_bytes = 1048576`
- 配置缓冲区：`buffer.max_events = 10000`
- 启用压缩：`compression = "gzip"`

### Loki 优化

- 配置保留期：`retention_period = 168h`
- 启用压缩：`compactor.working_directory`
- 调整索引配置：`schema_config.configs`

### 存储优化

- 定期清理旧日志
- 监控磁盘使用情况
- 配置日志轮转策略

## 🔐 安全考虑

1. **敏感信息过滤**: 确保不记录密码、令牌等敏感信息
2. **访问控制**: 生产环境启用 Grafana 的用户认证
3. **网络安全**: 限制日志组件的网络访问
4. **数据加密**: 生产环境考虑启用传输加密

## 📚 参考资料

- [Vector 官方文档](https://vector.dev/docs/)
- [Loki 官方文档](https://grafana.com/docs/loki/)
- [LogQL 查询语法](https://grafana.com/docs/loki/latest/logql/)
- [Grafana 日志面板配置](https://grafana.com/docs/grafana/latest/panels/visualizations/logs/)