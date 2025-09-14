# ES 包使用指南

## 概述

本文档记录了 GoChat ES 包的实际使用经验和最佳实践，基于真实环境测试和问题解决过程。

## 核心功能

ES 包提供了两个核心搜索功能：

1. **全局搜索** (`SearchGlobal`)：在所有文档中搜索关键词
2. **会话搜索** (`SearchInSession`)：在特定会话中搜索关键词

## 快速开始

### 1. 基本配置

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/ceyewan/gochat/im-infra/clog"
    "github.com/ceyewan/gochat/im-infra/es"
)

// 定义可索引的数据结构
type Message struct {
    ID        string    `json:"id"`
    SessionID string    `json:"session_id"`  // 关键：用于会话过滤
    Content   string    `json:"content"`      // 关键：用于全文搜索
    Timestamp time.Time `json:"timestamp"`
}

// 实现 Indexable 接口
func (m Message) GetID() string {
    return m.ID
}

func main() {
    ctx := context.Background()
    logger := clog.Namespace("my-app")
    
    // 创建 ES provider
    cfg := es.GetDefaultConfig("development")
    cfg.Addresses = []string{"http://localhost:9200"}
    
    provider, err := es.New[Message](ctx, cfg, es.WithLogger(logger))
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()
    
    // 使用 provider...
}
```

### 2. 批量索引数据

```go
// 创建消息
messages := []Message{
    {
        ID:        "msg-1",
        SessionID: "session-123",
        Content:   "Hello, this is a test message",
        Timestamp: time.Now(),
    },
    {
        ID:        "msg-2", 
        SessionID: "session-456",
        Content:   "Another message in different session",
        Timestamp: time.Now(),
    },
}

// 批量索引
indexName := "my-messages"
err = provider.BulkIndex(ctx, indexName, messages)
if err != nil {
    log.Fatal(err)
}

// 重要：等待索引完成
time.Sleep(5 * time.Second)
```

### 3. 全局搜索

```go
// 在所有文档中搜索关键词
result, err := provider.SearchGlobal(ctx, indexName, "hello", 1, 10)
if err != nil {
    log.Fatal(err)
}

log.Printf("找到 %d 条消息：", result.Total)
for _, msg := range result.Items {
    log.Printf("- %s: %s", (*msg).ID, (*msg).Content)
}
```

### 4. 会话搜索

```go
// 在特定会话中搜索关键词
sessionResult, err := provider.SearchInSession(ctx, indexName, "session-123", "test", 1, 5)
if err != nil {
    log.Fatal(err)
}

log.Printf("在会话 session-123 中找到 %d 条消息：", sessionResult.Total)
for _, msg := range sessionResult.Items {
    log.Printf("- %s: %s", (*msg).ID, (*msg).Content)
}
```

## ⚠️ 重要注意事项

### 1. 索引延迟问题

**现象**：数据索引后立即搜索可能返回 404 错误或空结果
**原因**：Elasticsearch 需要时间创建索引和刷新数据
**解决方案**：

```go
// 索引数据后等待足够时间
err = provider.BulkIndex(ctx, indexName, messages)
if err != nil {
    log.Fatal(err)
}

// 重要：等待索引完成
time.Sleep(5 * time.Second)  // 开发环境建议 5-10 秒
```

### 2. 分页行为

**现象**：搜索返回总数量大于实际显示的结果数量
**原因**：搜索方法的 `page` 和 `size` 参数控制分页
**示例**：

```go
// 这将只返回前 5 条结果，即使总共有 10 条匹配
result, err := provider.SearchGlobal(ctx, indexName, "keyword", 1, 5)
// page=1, size=5 → 返回第 1 页，每页 5 条

// 要获取所有结果，需要遍历所有页面
for page := 1; ; page++ {
    pageResult, err := provider.SearchGlobal(ctx, indexName, "keyword", page, 100)
    if err != nil {
        break
    }
    if len(pageResult.Items) == 0 {
        break
    }
    // 处理当前页的结果...
}
```

### 3. 数据结构要求

**关键字段**：

```go
type YourStruct struct {
    ID        string    `json:"id"`          // 必需：唯一标识符
    SessionID string    `json:"session_id"`  // 必需：会话过滤（使用 session_id.keyword）
    Content   string    `json:"content"`     // 必需：全文搜索内容
    Timestamp time.Time `json:"timestamp"`   // 可选：时间排序
}
```

## 🐛 常见问题解决

### 问题 1：会话搜索返回空结果

**现象**：全局搜索正常，但会话搜索返回 0 条结果
**原因**：Elasticsearch 字段映射问题，`session_id` 字段被分析
**解决方案**：已在代码中修复，使用 `session_id.keyword` 子字段

### 问题 2：搜索返回 404 错误

**现象**：搜索请求返回 "404 Not Found"
**原因**：索引尚未创建完成
**解决方案**：增加等待时间或实现重试机制

```go
// 重试机制示例
func searchWithRetry[T es.Indexable](provider es.Provider[T], ctx context.Context, 
    index, keyword string, page, size int, maxRetries int) (*es.SearchResult[T], error) {
    
    for i := 0; i < maxRetries; i++ {
        result, err := provider.SearchGlobal(ctx, index, keyword, page, size)
        if err == nil {
            return result, nil
        }
        
        // 如果是 404 错误，等待后重试
        if strings.Contains(err.Error(), "404") {
            time.Sleep(time.Duration(i+1) * time.Second)
            continue
        }
        
        // 其他错误直接返回
        return nil, err
    }
    
    return nil, fmt.Errorf("搜索失败，已重试 %d 次", maxRetries)
}
```

### 问题 3：批量索引数据丢失

**现象**：调用 `BulkIndex` 后数据无法搜索到
**原因**：批量索引器尚未刷新
**解决方案**：

```go
// 方法 1：等待足够时间
time.Sleep(5 * time.Second)

// 方法 2：手动触发刷新（如果 ES 客户端支持）
// provider.BulkIndexer.Flush()  // 如果可用

// 方法 3：关闭 provider（会自动刷新）
// defer provider.Close()
```

## 📊 性能优化建议

### 1. 批量索引配置

```go
cfg := es.GetDefaultConfig("development")
cfg.BulkIndexer.Workers = 4              // 增加工作线程
cfg.BulkIndexer.FlushBytes = 5 * 1024 * 1024  // 5MB 刷新阈值
cfg.BulkIndexer.FlushInterval = 10 * time.Second // 10秒刷新间隔
```

### 2. 搜索优化

```go
// 使用合适的页面大小
pageSize := 50  // 不要太大，避免内存问题

// 限制搜索结果
maxResults := 1000
if result.Total > int64(maxResults) {
    log.Printf("警告：搜索结果过多 (%d)，考虑添加更多过滤条件", result.Total)
}
```

### 3. 索引管理

```go
// 使用有意义的索引名称
indexName := fmt.Sprintf("messages-%s", time.Now().Format("2006-01"))
// 或者按应用/环境命名
indexName := "app-prod-messages"
```

## 🔍 调试技巧

### 1. 启用调试日志

```go
// 创建 logger 时启用调试
logger := clog.Namespace("debug-es")
```

### 2. 检查索引状态

```go
// 简单的健康检查
healthResult, err := provider.SearchGlobal(ctx, indexName, "", 1, 1)
if err != nil {
    log.Printf("索引 %s 可能未就绪：%v", indexName, err)
}
```

### 3. 验证数据结构

```go
// 确保 JSON 标签正确
type TestStruct struct {
    SessionID string `json:"session_id"`  // 必须与搜索字段匹配
    Content   string `json:"content"`
}

// 测试数据
testData := TestStruct{
    SessionID: "test-session",
    Content:   "test content",
}
```

## 📝 总结

ES 包提供了强大的全文搜索和会话过滤功能，正确使用时可以很好地支持即时通讯系统的搜索需求。关键要点：

1. **字段映射**：确保 `session_id` 字段正确映射为 `session_id.keyword`
2. **索引延迟**：为索引操作预留足够时间
3. **分页处理**：正确理解和使用分页参数
4. **错误处理**：实现适当的重试和容错机制

遵循本指南中的建议，可以避免常见问题并充分发挥 ES 包的功能。