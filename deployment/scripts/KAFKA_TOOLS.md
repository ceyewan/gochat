# Kafka 管理工具

本目录包含 GoChat 项目的 Kafka 管理脚本。

## 脚本说明

### kafka-admin.sh
完整的 Kafka 管理脚本，提供以下功能：
- 创建/删除 Topics
- 查看 Topics 和 Consumer Groups 状态
- 重置 Consumer Group 偏移量
- 实时监控 Topics

### init-kafka.sh
快速初始化脚本，用于在部署时自动创建所有必要的 Topics。

## 使用方法

### 1. 初始化环境
```bash
# 在项目启动前执行，创建所有必要的 Topics
./init-kafka.sh
```

### 2. 日常管理
```bash
# 查看所有 Topics
./kafka-admin.sh list

# 查看特定 Topic 详情
./kafka-admin.sh describe gochat.messages.upstream

# 查看 Consumer Groups
./kafka-admin.sh list-groups

# 实时监控
./kafka-admin.sh monitor
```

### 3. 故障排除
```bash
# 检查 Kafka 连接
./kafka-admin.sh check

# 重置 Consumer Group 偏移量
./kafka-admin.sh reset-offset logic.upstream.group gochat.messages.upstream latest

# 查看 Consumer Group 详情
./kafka-admin.sh describe-group logic.upstream.group
```

## 环境变量

- `KAFKA_BROKER`: Kafka broker 地址 (默认: localhost:9092)
- `REPLICATION_FACTOR`: 副本因子 (默认: 1)
- `PARTITIONS`: 默认分区数 (默认: 3)

## GoChat Topics 结构

### 核心消息流
- `gochat.messages.upstream` - 上行消息 (gateway → logic)
- `gochat.messages.downstream.{instanceID}` - 下行消息 (logic → gateway)
- `gochat.messages.persist` - 持久化消息 (logic → task)
- `gochat.tasks.fanout` - 大群扇出任务 (logic → task)

### 领域事件
- `gochat.user-events` - 用户事件 (上线、下线、资料更新)
- `gochat.message-events` - 消息事件 (已读、撤回)
- `gochat.notifications` - 系统通知

### Consumer Groups
- `logic.upstream.group` - im-logic 消费上行消息
- `gateway.downstream.group.{instanceID}` - im-gateway 消费下行消息
- `task.fanout.group` - im-task 处理扇出任务
- `task.persist.group` - im-task 持久化消息
