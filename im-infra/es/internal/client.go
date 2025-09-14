package internal

import (
	"fmt"
	"net/http"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/elastic/go-elasticsearch/v8"
)

// Client 是官方 Elasticsearch 客户端的包装器
type Client struct {
	*elasticsearch.Client
	logger clog.Logger
}

// NewClient 创建一个新的 Elasticsearch 客户端
func NewClient(cfg *Config, logger clog.Logger) (*Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		CloudID:   cfg.CloudID,
		APIKey:    cfg.APIKey,
		// 自定义传输层，添加日志记录
		Transport: &loggingTransport{
			transport: http.DefaultTransport,
			logger:    logger.With(clog.String("sub_component", "es_transport")),
		},
	}

	esClient, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		logger.Error("创建 elasticsearch 客户端失败", clog.Err(err))
		return nil, fmt.Errorf("创建 elasticsearch 客户端失败: %w", err)
	}

	// Ping 服务器以验证连接
	res, err := esClient.Ping()
	if err != nil {
		logger.Error("ping elasticsearch 失败", clog.Err(err))
		return nil, fmt.Errorf("ping elasticsearch 失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		logger.Error("elasticsearch ping 失败", clog.String("status", res.Status()))
		return nil, fmt.Errorf("elasticsearch ping 失败: %s", res.Status())
	}

	logger.Info("elasticsearch 客户端创建并连接成功")

	return &Client{
		Client: esClient,
		logger: logger,
	}, nil
}

// loggingTransport 是一个自定义的 http.RoundTripper，记录请求和响应
type loggingTransport struct {
	transport http.RoundTripper
	logger    clog.Logger
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.logger.Debug("向 elasticsearch 发送请求",
		clog.String("method", req.Method),
		clog.String("url", req.URL.String()),
	)

	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		t.logger.Error("elasticsearch 请求失败", clog.Err(err))
		return nil, err
	}

	t.logger.Debug("收到 elasticsearch 响应",
		clog.String("status", resp.Status),
	)
	return resp, nil
}

// Close 是一个占位符，因为底层客户端没有 Close 方法
// 传输层由 http 包管理
func (c *Client) Close() error {
	c.logger.Info("elasticsearch 客户端不需要显式关闭")
	return nil
}
