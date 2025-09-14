package es

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/es/internal"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

// provider 是 Provider 接口的内部实现
type provider[T Indexable] struct {
	client      *internal.Client
	bulkIndexer esutil.BulkIndexer
	logger      clog.Logger
}

// New 创建一个新的 es.Provider 实例
func New[T Indexable](ctx context.Context, cfg *Config, opts ...Option) (Provider[T], error) {
	options := &providerOptions{}
	for _, opt := range opts {
		opt(options)
	}

	var logger clog.Logger
	if options.logger != nil {
		logger = options.logger.With(clog.String("component", "es"))
	} else {
		logger = clog.Namespace("es")
	}

	// 创建 Elasticsearch 客户端
	client, err := internal.NewClient(cfg, logger)
	if err != nil {
		return nil, err
	}

	// 创建批量索引器
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        client.Client,
		NumWorkers:    cfg.BulkIndexer.Workers,
		FlushBytes:    cfg.BulkIndexer.FlushBytes,
		FlushInterval: cfg.BulkIndexer.FlushInterval,
		OnError: func(ctx context.Context, err error) {
			logger.Error("批量索引器错误", clog.Err(err))
		},
	})
	if err != nil {
		logger.Error("创建批量索引器失败", clog.Err(err))
		return nil, err
	}

	return &provider[T]{
		client:      client,
		bulkIndexer: bi,
		logger:      logger,
	}, nil
}

// BulkIndex 批量索引文档
func (p *provider[T]) BulkIndex(ctx context.Context, index string, items []T) error {
	for _, item := range items {
		payload, err := json.Marshal(item)
		if err != nil {
			p.logger.Error("批量索引时序列化文档失败",
				clog.Err(err),
				clog.String("item_id", item.GetID()))
			continue
		}

		err = p.bulkIndexer.Add(
			ctx,
			esutil.BulkIndexerItem{
				Index:      index,
				Action:     "index",
				DocumentID: item.GetID(),
				Body:       bytes.NewReader(payload),
			},
		)
		if err != nil {
			p.logger.Error("添加文档到批量索引器失败",
				clog.Err(err),
				clog.String("item_id", item.GetID()))
			return err
		}
	}
	return nil
}

// Close 关闭 es provider
func (p *provider[T]) Close() error {
	p.logger.Info("正在关闭 Elasticsearch provider")
	if err := p.bulkIndexer.Close(context.Background()); err != nil {
		p.logger.Error("关闭批量索引器失败", clog.Err(err))
		return err
	}
	p.logger.Info("Elasticsearch provider 已成功关闭")
	return nil
}

// SearchGlobal 在所有文档中进行全局搜索
func (p *provider[T]) SearchGlobal(ctx context.Context, index, keyword string, page, size int) (*SearchResult[T], error) {
	return p.search(ctx, index, keyword, page, size, nil)
}

// SearchInSession 在特定会话中进行搜索
func (p *provider[T]) SearchInSession(ctx context.Context, index, sessionID, keyword string, page, size int) (*SearchResult[T], error) {
	filter := map[string]interface{}{
		"term": map[string]interface{}{
			"session_id": sessionID,
		},
	}
	return p.search(ctx, index, keyword, page, size, filter)
}

// search 执行实际的搜索操作
func (p *provider[T]) search(ctx context.Context, index, keyword string, page, size int, filter map[string]interface{}) (*SearchResult[T], error) {
	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"multi_match": map[string]interface{}{
							"query":  keyword,
							"fields": []string{"content"},
						},
					},
				},
			},
		},
		"from": (page - 1) * size,
		"size": size,
		"sort": []map[string]interface{}{
			{
				"timestamp": map[string]string{
					"order": "desc",
				},
			},
		},
	}

	if filter != nil {
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = filter
	}

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		p.logger.Error("编码搜索查询失败", clog.Err(err))
		return nil, err
	}

	res, err := p.client.Search(
		p.client.Search.WithContext(ctx),
		p.client.Search.WithIndex(index),
		p.client.Search.WithBody(&buf),
		p.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		p.logger.Error("搜索请求失败", clog.Err(err))
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		p.logger.Error("搜索响应错误", clog.String("status", res.Status()))
		return nil, errors.New(res.Status())
	}

	var r struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source T `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		p.logger.Error("解码搜索响应失败", clog.Err(err))
		return nil, err
	}

	result := &SearchResult[T]{
		Total: r.Hits.Total.Value,
		Items: make([]*T, len(r.Hits.Hits)),
	}
	for i, hit := range r.Hits.Hits {
		item := hit.Source
		result.Items[i] = &item
	}

	return result, nil
}
