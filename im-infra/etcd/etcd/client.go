package etcd

import (
	"fmt"
	"log"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client 封装etcd客户端及相关操作
type Client struct {
	client *clientv3.Client
	config *Config
	logger *log.Logger
}

var (
	defaultClient *Client
	clientOnce    sync.Once
)

// NewClient 创建一个新的etcd客户端
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: config.DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &Client{
		client: client,
		config: config,
		logger: log.Default(),
	}, nil
}

// InitDefaultClient 初始化默认客户端
func InitDefaultClient(config *Config) error {
	var err error
	clientOnce.Do(func() {
		defaultClient, err = NewClient(config)
	})
	return err
}

// GetDefaultClient 获取默认客户端
func GetDefaultClient() (*Client, error) {
	if defaultClient == nil {
		return nil, fmt.Errorf("default client not initialized")
	}
	return defaultClient, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// CloseDefaultClient 关闭默认客户端
func CloseDefaultClient() error {
	if defaultClient == nil {
		return nil
	}
	err := defaultClient.Close()
	if err == nil {
		defaultClient = nil
	}
	return err
}
