# GoChat 配置管理

本目录包含 GoChat 项目的所有配置文件和配置管理工具。

## 📁 目录结构

```
config/
├── dev/                    # 开发环境配置
│   ├── im-repo/           # im-repo 服务配置
│   │   ├── cache.json     # 缓存配置
│   │   ├── db.json        # 数据库配置
│   │   ├── clog.json      # 日志配置
│   │   ├── coord.json     # 协调服务配置
│   │   ├── mq.json        # 消息队列配置
│   │   └── metrics.json   # 指标配置
│   ├── im-logic/          # im-logic 服务配置
│   ├── im-gateway/        # im-gateway 服务配置
│   └── im-task/           # im-task 服务配置
├── config-cli/            # 配置管理工具
└── README.md              # 本文件
```

## 🎯 配置规范

### 配置路径规范

所有配置遵循统一的路径规范：
```
/config/{env}/{service}/{component}
```

- `env`: 环境名称 (dev/test/prod)
- `service`: 服务名称 (im-repo/im-logic/im-gateway/im-task)
- `component`: 组件名称 (cache/db/clog/coord/mq/metrics)

### 配置文件格式

- 所有配置文件使用 JSON 格式
- 配置文件名与组件名保持一致
- 配置内容包含该组件的完整配置参数

## 🔧 配置管理工具

### config-cli 工具

位于 `config/config-cli/` 目录下的简化配置管理工具，专注于核心功能：

- **sync**: 将 JSON 配置文件原子地写入 etcd（唯一功能）

### 使用示例

```bash
# 进入配置工具目录
cd config/config-cli

# 同步所有开发环境配置
./config-cli sync dev

# 同步特定服务配置
./config-cli sync dev im-repo

# 同步特定组件配置
./config-cli sync dev im-repo cache

# 预览操作（干运行）
./config-cli sync dev --dry-run
```

## 📋 配置组件说明

### cache.json - 缓存配置
```json
{
  "addr": "redis:6379",
  "password": "",
  "db": 0,
  "poolSize": 10,
  "enableTracing": true,
  "enableMetrics": true,
  "keyPrefix": "service-name"
}
```

### db.json - 数据库配置
```json
{
  "dsn": "user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
  "driver": "mysql",
  "maxOpenConns": 25,
  "maxIdleConns": 10,
  "enableMetrics": true,
  "enableTracing": true
}
```

### clog.json - 日志配置
```json
{
  "level": "info",
  "format": "json",
  "output": "file",
  "filename": "/app/logs/app.log",
  "maxSize": 100,
  "maxBackups": 3,
  "maxAge": 7
}
```

### coord.json - 协调服务配置
```json
{
  "endpoints": ["etcd1:2379", "etcd2:2379", "etcd3:2379"],
  "timeout": "5s",
  "serviceName": "service-name",
  "servicePort": 8080
}
```

### mq.json - 消息队列配置
```json
{
  "brokers": ["kafka1:9092", "kafka2:9092", "kafka3:9092"],
  "producer": {
    "clientId": "service-producer",
    "acks": "all"
  },
  "consumer": {
    "groupId": "service-group",
    "autoOffsetReset": "earliest"
  }
}
```

### metrics.json - 指标配置
```json
{
  "serviceName": "service-name",
  "exporterType": "prometheus",
  "prometheusListenAddr": ":9091",
  "jaegerEndpoint": "http://jaeger:14268/api/traces"
}
```

## 🚀 配置加载流程

### 两阶段初始化

1. **阶段一：引导启动**
   - 使用代码内置的默认配置
   - 初始化基础日志和协调服务

2. **阶段二：功能完备**
   - 从 etcd 配置中心加载完整配置
   - 如果配置中心不可用，优雅降级到默认配置

### 配置优先级

1. **etcd 配置中心** (最高优先级)
2. **本地配置文件** (中等优先级)
3. **代码默认配置** (最低优先级，保底)

## 🔄 配置更新流程

### 开发环境配置更新

1. 修改本地配置文件
2. 使用 config-cli 工具同步到 etcd
3. 应用服务自动检测配置变化并重新加载

```bash
# 修改配置文件后，同步到 etcd
./config-cli sync dev im-repo cache

# 或批量同步整个服务
./config-cli sync dev im-repo

# 强制同步（跳过确认）
./config-cli sync dev --force
```

## 🛠️ 故障排除

### 常见问题

1. **配置加载失败**
   - 检查 etcd 连接状态
   - 验证配置文件格式
   - 查看应用日志中的配置加载信息

2. **配置不生效**
   - 确认配置已正确写入 etcd
   - 检查应用是否支持动态配置重载
   - 重启应用服务

3. **config-cli 工具问题**
   - 检查 etcd 连接配置
   - 验证配置文件路径
   - 查看工具输出的错误信息

### 调试命令

```bash
# 检查 etcd 中的配置
./config-cli list --prefix /config/dev

# 获取特定配置的详细信息
./config-cli get --key /config/dev/im-repo/cache --format json

# 监听配置变化（调试用）
./config-cli watch --key /config/dev --verbose
```

## 📚 最佳实践

1. **配置文件管理**
   - 配置文件使用版本控制管理
   - 敏感信息使用环境变量或密钥管理
   - 定期备份 etcd 配置数据

2. **配置更新**
   - 配置更新前先验证格式
   - 重要配置更新前先备份
   - 分批更新，避免影响所有服务

3. **监控和告警**
   - 监控配置加载失败的情况
   - 设置配置中心不可用的告警
   - 记录配置变更的审计日志

## 🔐 安全考虑

1. **访问控制**
   - 限制对 etcd 的访问权限
   - 使用 RBAC 控制配置的读写权限

2. **敏感信息**
   - 数据库密码等敏感信息不直接存储在配置文件中
   - 使用环境变量或密钥管理系统

3. **配置加密**
   - 生产环境考虑对敏感配置进行加密
   - 使用 TLS 保护 etcd 通信