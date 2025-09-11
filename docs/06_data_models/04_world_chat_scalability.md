# 世界聊天室扩展性设计

## 1. 核心问题

世界聊天室（type=3）所有消息共享同一个 `conversation_id`，按会话ID分片会导致热点问题。

## 2. 分片策略

### 时间分片（推荐）
```sql
-- 按月分片
CREATE TABLE world_messages_202401 PARTITION OF world_messages
FOR VALUES FROM (202401) TO (202402);
```

### 混合分片（高并发场景）
```sql
-- 时间 + 哈希双重分片
CREATE TABLE world_messages_202401 PARTITION OF world_messages
FOR VALUES FROM (202401) TO (202402)
PARTITION BY HASH (shard_key);
```

## 3. 查询优化

### 热冷数据分层
- **热数据**：最近7天，SSD存储
- **冷数据**：历史数据，HDD/对象存储

### 缓存策略
```yaml
world_chat:recent_messages: 100条最新消息 (TTL: 5min)
world_chat:user_last_read:  用户已读位置 (TTL: 24h)
```

## 4. 消息扇出

### 批量处理
```go
type WorldChatFanoutProcessor struct {
    batchSize  int    // 1000
    bufferTime time.Duration // 100ms
}
```

### 智能扇出
- 5分钟内活跃用户：立即推送
- 长时间不活跃：延迟推送或跳过

## 5. 容量规划

```yaml
假设:
  daily_active_users: 1000000
  messages_per_user_per_day: 10
  avg_message_size: 200 bytes

结果:
  daily_storage: 2GB
  yearly_storage: 730GB
```

## 6. 监控指标

- `world_chat_message_rate`: 消息速率
- `world_chat_fanout_latency`: 扇出延迟
- `world_chat_active_users`: 在线用户数