package internal

import (
	"fmt"
	"net/http"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/elastic/go-elasticsearch/v8"
)

// Client is a wrapper around the official Elasticsearch client.
type Client struct {
	*elasticsearch.Client
	logger clog.Logger
}

// NewClient creates a new Elasticsearch client.
func NewClient(cfg *Config, logger clog.Logger) (*Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		CloudID:   cfg.CloudID,
		APIKey:    cfg.APIKey,
		// Custom transport to add logging
		Transport: &loggingTransport{
			transport: http.DefaultTransport,
			logger:    logger.With(clog.String("sub_component", "es_transport")),
		},
	}

	esClient, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		logger.Error("failed to create elasticsearch client", clog.Err(err))
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Ping the server to verify the connection
	res, err := esClient.Ping()
	if err != nil {
		logger.Error("failed to ping elasticsearch", clog.Err(err))
		return nil, fmt.Errorf("failed to ping elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		logger.Error("elasticsearch ping failed", clog.String("status", res.Status()))
		return nil, fmt.Errorf("elasticsearch ping failed: %s", res.Status())
	}

	logger.Info("elasticsearch client created and connected successfully")

	return &Client{
		Client: esClient,
		logger: logger,
	}, nil
}

// loggingTransport is a custom http.RoundTripper that logs requests and responses.
type loggingTransport struct {
	transport http.RoundTripper
	logger    clog.Logger
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.logger.Debug("sending request to elasticsearch",
		clog.String("method", req.Method),
		clog.String("url", req.URL.String()),
	)

	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		t.logger.Error("elasticsearch request failed", clog.Err(err))
		return nil, err
	}

	t.logger.Debug("received response from elasticsearch",
		clog.String("status", resp.Status),
	)
	return resp, nil
}

// Close is a placeholder as the underlying client does not have a Close method.
// The transport is managed by the http package.
func (c *Client) Close() error {
	c.logger.Info("elasticsearch client does not require explicit closing")
	return nil
}
