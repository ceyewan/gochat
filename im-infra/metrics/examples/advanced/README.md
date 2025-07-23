# 高级 Metrics 示例

这个示例展示了如何在复杂的微服务环境中使用 `im-infra/metrics` 库，包括：

- 完整的 gRPC 和 HTTP 服务集成
- 自定义业务指标的创建和使用
- 高级配置选项
- 生产级别的日志记录和错误处理
- 优雅关闭机制

## 功能特性

### 1. 双协议支持
- **gRPC 服务器** (`:8081`): 支持 gRPC 调用的可观测性
- **HTTP 服务器** (`:8080`): 提供 REST API 和 Web 界面

### 2. 自动可观测性
- **链路追踪**: 自动为所有请求创建 span
- **指标收集**: 自动收集请求计数、延迟等指标
- **错误追踪**: 自动记录和追踪错误

### 3. 自定义业务指标
- **业务事件计数器**: 记录登录、消息发送等业务事件
- **响应时间直方图**: 分析业务操作的性能分布
- **消息大小直方图**: 监控消息大小分布
- **连接计数器**: 跟踪活跃连接数

### 4. 生产级配置
- **环境变量配置**: 支持通过环境变量调整配置
- **采样策略**: 可配置的 trace 采样率
- **多种导出器**: 支持 Jaeger、Zipkin、Prometheus 等

## 快速开始

### 1. 运行示例

```bash
# 直接运行
go run main.go

# 或者编译后运行
go build -o advanced-demo main.go
./advanced-demo
```

### 2. 环境变量配置

你可以通过环境变量自定义配置：

```bash
# 设置服务名称
export SERVICE_NAME="my-awesome-service"

# 设置 trace 导出器
export EXPORTER_TYPE="jaeger"
export EXPORTER_ENDPOINT="http://jaeger:14268/api/traces"

# 设置 Prometheus 地址
export PROMETHEUS_ADDR=":9091"

# 设置采样策略
export SAMPLER_TYPE="trace_id_ratio"
export SAMPLER_RATIO="0.1"

# 运行应用
go run main.go
```

### 3. 测试 API

启动应用后，你可以测试以下 API：

#### 用户登录
```bash
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username": "demo", "password": "password123"}'
```

#### 获取用户资料
```bash
curl http://localhost:8080/api/v1/users/12345/profile
```

#### 发送消息
```bash
curl -X POST http://localhost:8080/api/v1/messages/send \
  -H "Content-Type: application/json" \
  -d '{"user_id": 12345, "content": "Hello, World!", "to_user": 67890}'
```

#### 健康检查
```bash
curl http://localhost:8080/api/v1/health
```

## 可观测性数据

### 1. Prometheus 指标

访问 `http://localhost:9091/metrics` 查看所有指标，包括：

#### 自动收集的指标
- `rpc_server_requests_count`: gRPC 服务端请求计数
- `rpc_server_duration`: gRPC 服务端请求延迟
- `http_server_requests_count`: HTTP 服务端请求计数
- `http_server_duration`: HTTP 服务端请求延迟

#### 自定义业务指标
- `business_events_total`: 业务事件总数
- `business_response_duration_seconds`: 业务操作响应时间
- `message_size_bytes`: 消息大小分布
- `active_connections_total`: 活跃连接数

### 2. 链路追踪

如果配置了 Jaeger 或 Zipkin，你可以在对应的 UI 中查看完整的请求链路：

- **Jaeger UI**: 通常在 `http://localhost:16686`
- **Zipkin UI**: 通常在 `http://localhost:9411`

### 3. 日志记录

应用使用 `clog` 库进行结构化日志记录，包括：

- 服务启动和关闭日志
- 请求处理详情
- 业务操作记录
- 错误和异常信息

## 配置说明

### 默认配置

```go
cfg := metrics.DefaultConfig()
cfg.ServiceName = "advanced-metrics-demo"
cfg.ExporterType = "stdout"
cfg.PrometheusListenAddr = ":9091"
cfg.SamplerType = "trace_id_ratio"
cfg.SamplerRatio = 0.1
cfg.SlowRequestThreshold = 200 * time.Millisecond
```

### 配置项详解

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `ServiceName` | `advanced-metrics-demo` | 服务标识名称 |
| `ExporterType` | `stdout` | Trace 导出器类型 |
| `ExporterEndpoint` | `http://localhost:14268/api/traces` | 导出器端点地址 |
| `PrometheusListenAddr` | `:9091` | Prometheus 监听地址 |
| `SamplerType` | `trace_id_ratio` | 采样策略类型 |
| `SamplerRatio` | `0.1` | 采样比例 (10%) |
| `SlowRequestThreshold` | `200ms` | 慢请求阈值 |

## 生产环境建议

### 1. 性能优化
- 根据服务流量调整采样比例
- 合理设置慢请求阈值
- 使用合适的导出器批次大小

### 2. 监控告警
- 基于业务指标设置告警规则
- 监控错误率和响应时间
- 设置服务健康状态检查

### 3. 日志管理
- 配置合适的日志级别
- 使用日志聚合系统收集日志
- 定期轮转和清理日志文件

### 示例告警规则 (Prometheus)

```yaml
groups:
- name: advanced-demo-alerts
  rules:
  - alert: HighErrorRate
    expr: rate(business_events_total{status="failed"}[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      
  - alert: SlowRequests
    expr: histogram_quantile(0.95, rate(business_response_duration_seconds_bucket[5m])) > 1.0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "95th percentile response time is too high"
```

## 扩展示例

### 添加自定义指标

```go
// 创建新的计数器
customCounter, err := metrics.NewCounter(
    "custom_operations_total",
    "Total number of custom operations",
)

// 使用计数器
customCounter.Inc(ctx, 
    attribute.String("operation", "custom_op"),
    attribute.String("status", "success"))
```

### 添加中间件

```go
// 自定义 HTTP 中间件
func customMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        // 记录自定义指标
        customMetric.Record(c.Request.Context(), 
            time.Since(start).Seconds())
    }
}
```

## 故障排查

### 常见问题

1. **指标端点无法访问**
   - 检查 `PrometheusListenAddr` 配置
   - 确认端口未被占用

2. **Traces 未显示**
   - 检查导出器配置和网络连接
   - 验证采样配置

3. **日志级别过高**
   - 调整 clog 日志级别
   - 检查日志配置

### 调试模式

```bash
# 启用详细日志
export LOG_LEVEL=debug
go run main.go
```

## 相关文档

- [基础示例](../basic/README.md)
- [API 文档](../../API.md)
- [配置指南](../../README.md)
- [OpenTelemetry 文档](https://opentelemetry.io/docs/)