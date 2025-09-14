package es

import "context"

// Indexable 定义了可被索引对象必须满足的契约
type Indexable interface {
	// GetID 返回该对象在 Elasticsearch 中的唯一文档 ID
	GetID() string
}

// SearchResult 代表搜索返回的泛型结果
type SearchResult[T Indexable] struct {
	Total int64 // 搜索结果总数
	Items []*T  // 搜索结果项
}

// Provider 是 es 组件暴露的核心接口
// 提供 Elasticsearch 的索引和搜索功能
type Provider[T Indexable] interface {
	// BulkIndex 异步批量索引实现了 Indexable 接口的任何类型的文档
	// index: Elasticsearch 索引名称
	// items: 要索引的文档列表
	BulkIndex(ctx context.Context, index string, items []T) error

	// SearchGlobal 在所有文档中进行全局文本搜索
	// index: 要搜索的索引名称
	// keyword: 搜索关键词
	// page: 页码（从1开始）
	// size: 每页大小
	SearchGlobal(ctx context.Context, index, keyword string, page, size int) (*SearchResult[T], error)

	// SearchInSession 在特定会话中进行文本搜索
	// index: 要搜索的索引名称
	// sessionID: 会话ID
	// keyword: 搜索关键词
	// page: 页码（从1开始）
	// size: 每页大小
	SearchInSession(ctx context.Context, index, sessionID, keyword string, page, size int) (*SearchResult[T], error)

	// Close 关闭客户端连接，释放资源
	Close() error
}
