# 消息队列规范

## 概述

GoChat 系统使用 Apache Kafka 作为消息队列中间件，用于服务间的异步通信和解耦。本文档详细描述了 Kafka 的使用规范和消息路由策略。

## 系统架构

### 消息流向

```
Client → im-gateway → im-upstream-topic → im-logic → im-downstream-topic-{gateway_id} → im-gateway → Client
                                        ↓
                                        im-task-topic → im-task → 持久化/推送
```

### Topic 设计

| Topic | 用途 | 分区数 | 副本数 | 消费者组 |
|-------|------|--------|--------|----------|
| im-upstream-topic | 上游消息 (客户端发送) | 10 | 3 | im-logic-group |
| im-downstream-topic-{gateway_id} | 下游消息 (发送到客户端) | 3 | 2 | gateway-{id}-group |
| im-task-topic | 任务消息 (异步处理) | 5 | 3 | im-task-group |

## Topic 详细设计

### 1. im-upstream-topic

**用途**: 接收客户端发送的所有消息

**配置**:
- 分区数: 10 (根据并发量调整)
- 副本数: 3 (保证高可用)
- 保留时间: 7天
- 压缩类型: lz4

**消息格式**:

```json
{
  "message_id": "msg_123",
  "from_user_id": "user_123",
  "to_user_id": "user_456",
  "conversation_id": "conv_123",
  "content": "Hello World",
  "message_type": "text",
  "metadata": {},
  "gateway_id": "gateway_001",
  "timestamp": 1640995200000,
  "version": "1.0"
}
```

**路由策略**:
- 按 `from_user_id` hash 到分区
- 保证同一用户的消息顺序性

### 2. im-downstream-topic-{gateway_id}

**用途**: 发送消息到特定网关实例

**配置**:
- 分区数: 3 (每个网关实例独立 topic)
- 副本数: 2 (保证可用性)
- 保留时间: 1天
- 压缩类型: lz4

**消息格式**:

```json
{
  "message_id": "msg_123",
  "from_user_id": "user_123",
  "to_user_id": "user_456",
  "conversation_id": "conv_123",
  "content": "Hello World",
  "message_type": "text",
  "metadata": {},
  "timestamp": 1640995200000,
  "version": "1.0"
}
```

**路由策略**:
- 按 `to_user_id` hash 到分区
- 保证目标用户的消息顺序性

### 3. im-task-topic

**用途**: 异步任务处理，包括消息持久化、离线推送等

**配置**:
- 分区数: 5 (根据任务量调整)
- 副本数: 3 (保证高可用)
- 保留时间: 3天
- 压缩类型: lz4

**消息格式**:

```json
{
  "task_type": "persist_message",
  "task_id": "task_123",
  "payload": {
    "message_id": "msg_123",
    "conversation_id": "conv_123",
    "from_user_id": "user_123",
    "to_user_id": "user_456",
    "content": "Hello World",
    "message_type": "text",
    "metadata": {}
  },
  "timestamp": 1640995200000,
  "retry_count": 0,
  "version": "1.0"
}
```

**路由策略**:
- 按 `task_type` hash 到分区
- 保证同类型任务的顺序性

## 消息类型定义

### 1. 用户消息

#### 文本消息

```json
{
  "message_type": "text",
  "content": "Hello World",
  "metadata": {
    "format": "plain",
    "language": "zh-CN"
  }
}
```

#### 图片消息

```json
{
  "message_type": "image",
  "content": "https://example.com/image.jpg",
  "metadata": {
    "width": 800,
    "height": 600,
    "size": 1024000,
    "format": "jpg"
  }
}
```

#### 文件消息

```json
{
  "message_type": "file",
  "content": "https://example.com/file.pdf",
  "metadata": {
    "filename": "document.pdf",
    "size": 2048000,
    "mime_type": "application/pdf"
  }
}
```

#### 位置消息

```json
{
  "message_type": "location",
  "content": "北京市朝阳区",
  "metadata": {
    "latitude": 39.9042,
    "longitude": 116.4074,
    "accuracy": 10
  }
}
```

### 2. 系统消息

#### 会话消息

```json
{
  "message_type": "system_conversation",
  "content": "John 创建了会话",
  "metadata": {
    "action": "create",
    "conversation_id": "conv_123",
    "creator_id": "user_123"
  }
}
```

#### 群组消息

```json
{
  "message_type": "system_group",
  "content": "John 邀请 Alice 加入群组",
  "metadata": {
    "action": "invite",
    "group_id": "group_123",
    "inviter_id": "user_123",
    "invitee_id": "user_456"
  }
}
```

### 3. 任务消息

#### 消息持久化任务

```json
{
  "task_type": "persist_message",
  "payload": {
    "message_id": "msg_123",
    "conversation_id": "conv_123",
    "from_user_id": "user_123",
    "to_user_id": "user_456",
    "content": "Hello World",
    "message_type": "text",
    "metadata": {}
  }
}
```

#### 离线推送任务

```json
{
  "task_type": "offline_push",
  "payload": {
    "user_id": "user_456",
    "message_id": "msg_123",
    "title": "新消息",
    "content": "John: Hello World",
    "push_type": "message"
  }
}
```

#### 大群扇出任务

```json
{
  "task_type": "group_fanout",
  "payload": {
    "group_id": "group_123",
    "message_id": "msg_123",
    "exclude_user_ids": ["user_123"],
    "batch_size": 100
  }
}
```

## 消息路由策略

### 1. 单聊消息路由

```
Client A → Gateway A → Upstream Topic → Logic → Downstream Topic (Gateway B) → Gateway B → Client B
```

**路由逻辑**:
1. 客户端 A 发送消息到网关 A
2. 网关 A 发送到上游 Topic
3. Logic 服务消费并处理消息
4. Logic 查询目标用户 B 的网关实例
5. 发送到网关 B 的下游 Topic
6. 网关 B 推送消息给客户端 B

### 2. 群聊消息路由

```
Client A → Gateway A → Upstream Topic → Logic → Task Topic → Task → 批量发送到多个网关
```

**路由逻辑**:
1. 客户端 A 发送群聊消息到网关 A
2. 网关 A 发送到上游 Topic
3. Logic 服务消费并处理消息
4. 对于小群，直接发送到各成员的网关
5. 对于大群，发送到 Task Topic 进行异步扇出

### 3. 离线消息路由

```
Logic → Task Topic → Task → 持久化到数据库 → 推送通知
```

**路由逻辑**:
1. Logic 检测到用户离线
2. 发送持久化任务到 Task Topic
3. Task 服务消费并持久化消息
4. 发送离线推送通知

## 消费者组配置

### 1. im-logic-group

**Topic**: im-upstream-topic
**配置**:
```yaml
group.id: im-logic-group
auto.offset.reset: earliest
enable.auto.commit: false
max.poll.records: 100
session.timeout.ms: 30000
heartbeat.interval.ms: 10000
```

### 2. gateway-{id}-group

**Topic**: im-downstream-topic-{gateway_id}
**配置**:
```yaml
group.id: gateway-{id}-group
auto.offset.reset: latest
enable.auto.commit: true
max.poll.records: 50
session.timeout.ms: 30000
heartbeat.interval.ms: 10000
```

### 3. im-task-group

**Topic**: im-task-topic
**配置**:
```yaml
group.id: im-task-group
auto.offset.reset: earliest
enable.auto.commit: false
max.poll.records: 200
session.timeout.ms: 30000
heartbeat.interval.ms: 10000
```

## 消息可靠性保证

### 1. 消息持久化

- **生产者**: 启用 `acks=all` 确保消息写入所有副本
- **消费者**: 手动提交 offset，确保处理完成后再提交
- **Broker**: 配置合理的保留策略和副本数

### 2. 消息顺序性

- **单用户消息**: 通过用户 ID hash 到同一分区
- **单会话消息**: 通过会话 ID hash 到同一分区
- **任务消息**: 通过任务类型 hash 到同一分区

### 3. 消息幂等性

- **消息 ID**: 每条消息有唯一 ID
- **去重处理**: 消费端进行消息去重
- **重试机制**: 失败消息重试但保证不重复处理

### 4. 死信队列

- **重试次数**: 最多重试 3 次
- **死信 Topic**: 失败消息发送到死信队列
- **告警机制**: 死信消息触发告警

## 性能优化

### 1. 批量处理

- **生产者**: 批量发送消息，提高吞吐量
- **消费者**: 批量拉取消息，减少网络开销
- **任务处理**: 批量处理任务，提高效率

### 2. 压缩配置

- **压缩算法**: 使用 lz4 压缩
- **压缩阈值**: 消息大小超过 1KB 时压缩
- **压缩比例**: 预期压缩比 50%

### 3. 缓存策略

- **元数据缓存**: 缓存用户、会话等元数据
- **网关缓存**: 缓存用户网关映射关系
- **消息缓存**: 缓存最近消息

## 监控和告警

### 1. 关键指标

- **消息积压**: 各 Topic 的 Lag 数量
- **吞吐量**: 每秒消息处理量
- **延迟**: 消息端到端延迟
- **错误率**: 消息处理错误率

### 2. 告警规则

- **消息积压**: Lag > 1000 持续 5 分钟
- **吞吐量异常**: QPS 下降 50% 持续 10 分钟
- **延迟异常**: 端到端延迟 > 5 秒
- **错误率**: 错误率 > 1% 持续 5 分钟

### 3. 日志记录

- **消息追踪**: 每条消息记录处理日志
- **错误日志**: 详细记录错误信息
- **性能日志**: 记录处理时间和吞吐量

## 扩展性设计

### 1. Topic 扩展

- **动态扩容**: 支持动态增加分区数
- **负载均衡**: 分区负载均衡算法
- **数据迁移**: 分区数据迁移策略

### 2. 消费者扩展

- **水平扩展**: 支持消费者实例动态扩展
- **负载均衡**: 消费者 rebalance 机制
- **故障转移**: 消费者故障自动转移

### 3. 存储扩展

- **存储分层**: 热数据、温数据、冷数据分层
- **数据归档**: 历史数据归档策略
- **容量规划**: 根据业务增长规划存储容量

## 故障处理

### 1. Broker 故障

- **副本切换**: 自动切换到其他副本
- ** leader 选举**: 自动进行 leader 选举
- **数据恢复**: 副本数据同步恢复

### 2. 生产者故障

- **重试机制**: 网络故障自动重试
- **故障转移**: 切换到其他 Broker
- **缓冲策略**: 本地缓冲未发送消息

### 3. 消费者故障

- **重平衡**: 消费者故障触发重平衡
- **offset 重置**: 支持从指定位置重置
- **状态恢复**: 消费者状态恢复机制

## 安全配置

### 1. 认证授权

- **SASL 认证**: 启用 SASL/PLAIN 认证
- **ACL 控制**: 配置 Topic 级别权限控制
- **网络隔离**: 生产环境网络隔离

### 2. 数据加密

- **传输加密**: 启用 SSL/TLS 加密
- **数据加密**: 敏感消息内容加密
- **密钥管理**: 密钥轮换和管理策略

### 3. 审计日志

- **操作审计**: 记录关键操作日志
- **访问审计**: 记录访问控制日志
- **安全审计**: 定期安全审计和检查