# 规范: `es` - 分布式索引组件

## 1. 哲学

`es` 组件是 GoChat 的核心搜索基础设施，它提供了一个统一、高性能的接口，用于将业务数据索引到 Elasticsearch 并执行复杂的文本搜索。它的设计遵循 `im-infra` 的核心原则：**封装复杂性，提供极简且通用的接口**。

为了实现业务无关性，`es` 组件采用**泛型**设计。它不关心被索引数据的具体结构，只要求数据模型实现一个简单的 `Indexable` 接口。这使得该组件可以被任何服务用于索引任何类型的数据（如消息、用户资料、文章等），而不仅仅是聊天消息。

## 2. 接口契约 (Provider)

### 2.1 `Indexable` 接口

任何希望被 `es` 组件索引的结构体，都必须实现 `Indexable` 接口。

```go
package es

// Indexable 定义了可被索引对象必须满足的契约。
// 任何实现了此接口的结构体都可以被 es.Provider 处理。
type Indexable interface {
    // GetID 返回该对象在 Elasticsearch 中的唯一文档 ID。
    // 通常是业务主键，如 MessageID, UserID 等。
    GetID() string
}
```

### 2.2 `Provider` 接口 (泛型)

`Provider` 接口利用 Go 泛型来提供类型安全和业务解耦。

```go
package es

import "context"

// SearchResult 代表搜索返回的泛型结果
type SearchResult[T Indexable] struct {
    Total    int64
    Messages []*T
}

// Provider 是 es 组件暴露的核心接口
type Provider interface {
    // BulkIndex 异步批量索引实现了 Indexable 接口的任何类型的文档。
    BulkIndex[T Indexable](ctx context.Context, items []T) error

    // SearchGlobal 在所有文档中进行全局文本搜索。
    SearchGlobal[T Indexable](ctx context.Context, operatorID string, keyword string, page, size int) (*SearchResult[T], error)

    // SearchInSession 在特定会话中进行文本搜索。
    // 注意：此方法虽然通用，但隐含了数据模型中存在 "session_id" 字段的约定。
    SearchInSession[T Indexable](ctx context.Context, operatorID, sessionID, keyword string, page, size int) (*SearchResult[T], error)

    // Close 关闭客户端连接，释放资源。
    Close() error
}
```

### 2.3 构造函数与选项

```go
// New 构造函数遵循标准签名
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)

// Option 定义了函数式选项
type Option func(*options)

// WithLogger 注入日志记录器
func WithLogger(logger clog.Logger) Option
```

## 3. 配置契约

配置保持不变，因为它与数据模型无关。

```go
package es

// Config 定义了 Elasticsearch 组件的配置
type Config struct {
    Addresses     []string `json:"addresses"`
    Username      string   `json:"username"`
    Password      string   `json:"password"`
    IndexName     string   `json:"index_name"`
    BulkIndexer   struct {
        Workers       int `json:"workers"`
        FlushBytes    int `json:"flush_bytes"`
        FlushInterval int `json:"flush_interval_ms"`
    } `json:"bulk_indexer"`
}
```

## 4. 核心用法

### 4.1 定义业务模型并实现 `Indexable`

在业务代码中（例如 `im-task` 服务），定义你的数据结构。

```go
// a. 定义你的消息结构体
type MyMessage struct {
    MessageID   string `json:"message_id"`
    SessionID   string `json:"session_id"`
    SenderID    string `json:"sender_id"`
    Content     string `json:"content"`
    Timestamp   int64  `json:"timestamp"`
}

// b. 实现 Indexable 接口
func (m MyMessage) GetID() string {
    return m.MessageID
}
```

### 4.2 初始化 Provider

初始化过程不变。

```go
// 在 main.go 或 server.go 中
esProvider, err := es.New(
    context.Background(),
    esConfig,
    es.WithLogger(logger),
)
if err != nil {
    panic(err)
}
```

### 4.3 写入数据 (泛型调用)

```go
func handleMessages(ctx context.Context, msgs []MyMessage) {
    // 直接将 MyMessage 切片传入，无需转换
    if err := esProvider.BulkIndex(ctx, msgs); err != nil {
        logger.Error("Failed to bulk index messages", clog.Err(err))
    }
}
```

### 4.4 读取数据 (泛型调用)

```go
func (s *Service) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
    // 调用时需指定泛型类型为你的业务模型
    searchResult, err := s.esProvider.SearchInSession[MyMessage](
        ctx,
        req.GetOperatorId(),
        req.GetSessionId(),
        req.GetKeyword(),
        int(req.GetPage()),
        int(req.GetSize()),
    )
    if err != nil {
        return nil, err
    }

    // searchResult.Messages 是 []*MyMessage 类型，可以直接使用
    for _, msg := range searchResult.Messages {
        // msg.MessageID, msg.Content ...
    }
    
    // ... 转换为 gRPC 响应
    return resp, nil
}
```

这个设计将 `es` 组件的通用性提升到了一个新的水平，完全符合 `im-infra` 的设计哲学。我将基于这个新设计来更新所有相关文档。