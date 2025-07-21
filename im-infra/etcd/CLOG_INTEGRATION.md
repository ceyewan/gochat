# etcd 包 clog 日志集成

本文档说明了 etcd 包如何集成 clog 日志库，提供详细的日志记录功能。

## 概述

etcd 包已经完全集成了 clog 日志库，使用 `module = "etcd"` 作为模块名。所有的 etcd 操作都会记录详细的日志，包括：

- 客户端创建和连接
- 服务注册和注销
- 服务发现和监听
- 租约管理
- 分布式锁操作
- 健康检查
- 错误处理

## 主要特性

### 1. 统一的日志模块
所有 etcd 相关的日志都使用 `etcd` 模块名：
```go
etcdLogger := clog.Module("etcd")
```

### 2. 详细的操作日志
每个重要操作都有对应的日志记录：
- **Info 级别**: 记录关键操作的开始、成功和结果
- **Debug 级别**: 记录详细的执行步骤和中间状态
- **Warn 级别**: 记录可恢复的错误和异常情况
- **Error 级别**: 记录严重错误和失败操作

### 3. 结构化日志字段
使用 clog 的结构化字段记录关键信息：
```go
etcdLogger.Info("服务注册成功",
    clog.String("service", serviceName),
    clog.String("instance", instanceID),
    clog.String("address", address),
    clog.Int64("lease_id", leaseID))
```

## 使用方法

### 1. 初始化 clog
在使用 etcd 包之前，需要先初始化 clog：

```go
err := clog.Init(
    clog.WithLevel("debug"),
    clog.WithConsoleOutput(true),
    clog.WithFilename("logs/app.log"),
    clog.WithFormat(clog.FormatJSON),
)
if err != nil {
    panic(err)
}
defer clog.Sync()
```

### 2. 使用基本客户端
```go
// 创建客户端（自动使用 clog）
client, err := etcd.NewClient(&etcd.Config{
    Endpoints:   []string{"localhost:2379"},
    DialTimeout: 5 * time.Second,
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### 3. 使用管理器
```go
// 创建管理器（自动使用 clog）
manager, err := etcd.NewEtcdManager(etcd.DefaultManagerOptions())
if err != nil {
    log.Fatal(err)
}
defer manager.Close()
```

### 4. 自定义日志器
如果需要自定义日志器，可以使用 ClogAdapter：
```go
etcdLogger := clog.Module("etcd")
options := &etcd.ManagerOptions{
    Endpoints: []string{"localhost:2379"},
    Logger:    etcd.NewClogAdapter(etcdLogger),
    // ... 其他选项
}
```

## 日志示例

### 客户端创建日志
```json
{
  "level": "info",
  "time": "2024-01-20T10:30:00Z",
  "module": "etcd",
  "msg": "开始创建 etcd 客户端",
  "endpoints": ["localhost:2379"],
  "dial_timeout": "5s"
}
```

### 服务注册日志
```json
{
  "level": "info",
  "time": "2024-01-20T10:30:01Z",
  "module": "etcd",
  "msg": "服务注册成功",
  "service": "user-service",
  "instance": "instance-1",
  "address": "localhost:8080",
  "lease_id": 123456789
}
```

### 服务发现日志
```json
{
  "level": "info",
  "time": "2024-01-20T10:30:02Z",
  "module": "etcd",
  "msg": "获取服务端点成功",
  "service": "user-service",
  "endpoint_count": 2,
  "endpoints": ["localhost:8080", "localhost:8081"]
}
```

### 错误日志
```json
{
  "level": "error",
  "time": "2024-01-20T10:30:03Z",
  "module": "etcd",
  "msg": "连接 etcd 失败",
  "error": "context deadline exceeded",
  "endpoints": ["localhost:2379"]
}
```

## 日志级别配置

建议的日志级别配置：

- **生产环境**: `info` 级别，记录关键操作和错误
- **测试环境**: `debug` 级别，记录详细的执行过程
- **开发环境**: `debug` 级别，便于调试和问题排查

```go
// 生产环境
clog.Init(clog.WithLevel("info"))

// 开发/测试环境
clog.Init(clog.WithLevel("debug"))
```

## 性能考虑

1. **结构化日志**: 使用 clog 的结构化字段而不是字符串拼接，提高性能
2. **日志级别**: 在生产环境使用适当的日志级别，避免过多的 debug 日志
3. **异步写入**: clog 支持异步写入，不会阻塞主要业务逻辑

## 故障排查

通过日志可以快速定位问题：

1. **连接问题**: 查看连接建立和健康检查日志
2. **服务注册问题**: 查看租约创建和服务写入日志
3. **服务发现问题**: 查看查询和解析日志
4. **性能问题**: 查看操作耗时和重试日志

## 示例程序

参考 `example_usage.go` 文件查看完整的使用示例。

运行示例：
```bash
go run im-infra/etcd/example_usage.go
```

## 注意事项

1. 确保在使用 etcd 包之前初始化 clog
2. 在程序结束时调用 `clog.Sync()` 确保日志写入
3. 根据环境选择合适的日志级别和输出格式
4. 定期清理日志文件，避免磁盘空间不足
