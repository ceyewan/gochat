package etcd

import (
	"fmt"
	"sync"

	"github.com/ceyewan/gochat/im-infra/clog"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client 封装etcd客户端及相关操作
type Client struct {
	client *clientv3.Client
	config *Config
	logger clog.Logger
}

var (
	defaultClient *Client
	clientOnce    sync.Once
)

// NewClient 创建一个新的etcd客户端
func NewClient(config *Config) (*Client, error) {
	// 创建 etcd 客户端专用的日志器
	etcdLogger := clog.Default().With("module", "etcd")

	if config == nil {
		etcdLogger.Info("使用默认配置创建 etcd 客户端")
		config = DefaultConfig()
	}

	etcdLogger.Info("开始创建 etcd 客户端",
		clog.Any("endpoints", config.Endpoints),
		clog.Duration("dial_timeout", config.DialTimeout))

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: config.DialTimeout,
	})
	if err != nil {
		etcdLogger.Error("创建 etcd 客户端失败",
			clog.Err(err),
			clog.Any("endpoints", config.Endpoints),
			clog.Duration("dial_timeout", config.DialTimeout))
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	etcdLogger.Info("etcd 客户端创建成功",
		clog.Any("endpoints", config.Endpoints))

	return &Client{
		client: client,
		config: config,
		logger: etcdLogger,
	}, nil
}

// InitDefaultClient 初始化默认客户端
func InitDefaultClient(config *Config) error {
	etcdLogger := clog.Default().With("module", "etcd")
	etcdLogger.Info("开始初始化默认 etcd 客户端")

	var err error
	clientOnce.Do(func() {
		defaultClient, err = NewClient(config)
		if err != nil {
			etcdLogger.Error("初始化默认 etcd 客户端失败", clog.Err(err))
		} else {
			etcdLogger.Info("默认 etcd 客户端初始化成功")
		}
	})
	return err
}

// GetDefaultClient 获取默认客户端
func GetDefaultClient() (*Client, error) {
	etcdLogger := clog.Default().With("module", "etcd")

	if defaultClient == nil {
		etcdLogger.Error("默认 etcd 客户端未初始化")
		return nil, fmt.Errorf("default client not initialized")
	}

	etcdLogger.Debug("获取默认 etcd 客户端成功")
	return defaultClient, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	c.logger.Info("开始关闭 etcd 客户端连接")

	if c.client != nil {
		err := c.client.Close()
		if err != nil {
			c.logger.Error("关闭 etcd 客户端连接失败", clog.Err(err))
			return err
		}
		c.logger.Info("etcd 客户端连接已关闭")
		return nil
	}

	c.logger.Warn("etcd 客户端连接已经为空，无需关闭")
	return nil
}

// CloseDefaultClient 关闭默认客户端
func CloseDefaultClient() error {
	etcdLogger := clog.Default().With("module", "etcd")
	etcdLogger.Info("开始关闭默认 etcd 客户端")

	if defaultClient == nil {
		etcdLogger.Warn("默认 etcd 客户端为空，无需关闭")
		return nil
	}

	err := defaultClient.Close()
	if err != nil {
		etcdLogger.Error("关闭默认 etcd 客户端失败", clog.Err(err))
		return err
	}

	defaultClient = nil
	etcdLogger.Info("默认 etcd 客户端已关闭并清理")
	return nil
}
