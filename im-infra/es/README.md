# ES - 分布式泛型索引组件

## 1. 概述

`es` 组件是 `im-infra` 层的一个核心基础设施，它为 GoChat 系统提供了强大的实时数据索引和搜索能力。该组件基于 Elasticsearch 构建，旨在提供一个高性能、可扩展、**业务无关**的倒排索引解决方案。

**核心设计**:

*   **泛型接口**: 使用 Go 泛型，可以索引和搜索任何实现了 `es.Indexable` 接口的数据结构。
*   **封装客户端**: 封装与 Elasticsearch 交互的底层细节，包括连接、批量写入和查询构建。
*   **提供统一接口**: 暴露简洁的 `Provider` 接口，用于批量索引和搜索。

该组件是**纯粹的基础设施**，不包含任何业务逻辑。它不知道被索引数据的具体内容，只负责执行上层业务代码下达的索引和搜索指令。

## 2. 接口契约 (Provider)

### 2.1 `Indexable` 接口

任何希望被索引的业务模型都必须实现此接口。

```go
package es

// Indexable 定义了可被索引对象必须满足的契约。
type Indexable interface {
    // GetID 返回该对象在 Elasticsearch 中的唯一文档 ID。
    GetID() string
}
```

### 2.2 `Provider` 接口 (泛型)

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
    SearchInSession[T Indexable](ctx context.Context, operatorID, sessionID, keyword string, page, size int) (*SearchResult[T], error)

    // Close 关闭客户端连接，释放资源。
    Close() error
}
```

### 2.3 构造函数

```go
// New 构造函数遵循标准签名
New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

## 3. 配置契约

配置结构保持不变，因为它与具体的数据模型解耦。

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

## 4. 实现计划

1.  **[ ] 创建 `im-infra/es` 目录结构**: 包含 `es.go`, `config.go`, `options.go`, `README.md`, `internal/`, `examples/`。
2.  **[ ] 实现 `Provider` 和 `Indexable` 接口**: 在 `es.go` 中定义泛型接口。
3.  **[ ] 实现 `internal/client.go`**: 封装 `go-elasticsearch` 客户端的初始化、批量索引和搜索逻辑，需要处理泛型。
4.  **[ ] 实现 `New` 构造函数**: 组装 `es` 组件实例。
5.  **[ ] 创建 `im-infra/es/examples/`**: 提供一个完整的使用示例，演示如何定义 `Indexable` 模型并使用泛型接口。
6.  **[ ] 更新 `docs/08_infra` 相关文档**: (已完成)
7.  **[ ] 更新 `im-task` 服务**:
    *   在 `im-task` 中添加对 `es.Provider` 的依赖。
    *   定义 `Message` 结构体并实现 `Indexable` 接口。
    *   修改 `im-task` 的主逻辑，调用 `es.Provider.BulkIndex[Message]`。