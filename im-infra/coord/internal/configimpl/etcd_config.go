package configimpl

import (
	"context"
	"encoding/json"
	"path"
	"strings"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	"github.com/ceyewan/gochat/im-infra/coord/internal/types"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdConfigCenter 基于 etcd 的配置中心实现
type EtcdConfigCenter struct {
	client *client.EtcdClient
	prefix string
	logger clog.Logger
}

// NewEtcdConfigCenter 创建新的配置中心实例
func NewEtcdConfigCenter(client *client.EtcdClient, prefix string) *EtcdConfigCenter {
	if prefix == "" {
		prefix = "/configimpl"
	}

	return &EtcdConfigCenter{
		client: client,
		prefix: prefix,
		logger: clog.Module("coordination.configimpl"),
	}
}

// Get 获取配置值
func (c *EtcdConfigCenter) Get(ctx context.Context, key string) (interface{}, error) {
	if key == "" {
		return nil, client.NewCoordinationError(
			client.ErrCodeValidation,
			"configimpl key cannot be empty",
			nil,
		)
	}

	configKey := path.Join(c.prefix, key)

	c.logger.Info("getting configimpl value",
		clog.String("key", configKey))

	resp, err := c.client.Get(ctx, configKey)
	if err != nil {
		c.logger.Error("failed to get configimpl value",
			clog.String("key", configKey),
			clog.Err(err))
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		c.logger.Debug("configimpl key not found",
			clog.String("key", configKey))
		return nil, client.NewCoordinationError(
			client.ErrCodeNotFound,
			"configimpl key not found",
			nil,
		)
	}

	value := resp.Kvs[0].Value

	// 尝试解析为 JSON，如果失败则返回原始字符串
	var result interface{}
	if err := json.Unmarshal(value, &result); err != nil {
		// 如果不是有效的 JSON，返回字符串
		result = string(value)
	}

	c.logger.Info("configimpl value retrieved successfully",
		clog.String("key", configKey),
		clog.String("value", string(value)))

	return result, nil
}

// Set 设置配置值（支持任意可序列化对象）
func (c *EtcdConfigCenter) Set(ctx context.Context, key string, value interface{}) error {
	if key == "" {
		return client.NewCoordinationError(
			client.ErrCodeValidation,
			"configimpl key cannot be empty",
			nil,
		)
	}

	configKey := path.Join(c.prefix, key)

	// 序列化值
	var valueBytes []byte
	var err error

	switch v := value.(type) {
	case string:
		valueBytes = []byte(v)
	case []byte:
		valueBytes = v
	default:
		// 对于其他类型，使用 JSON 序列化
		valueBytes, err = json.Marshal(value)
		if err != nil {
			c.logger.Error("failed to serialize configimpl value",
				clog.String("key", configKey),
				clog.Err(err))
			return client.NewCoordinationError(
				client.ErrCodeValidation,
				"failed to serialize configimpl value",
				err,
			)
		}
	}

	c.logger.Info("setting configimpl value",
		clog.String("key", configKey),
		clog.String("value", string(valueBytes)))

	_, err = c.client.Put(ctx, configKey, string(valueBytes))
	if err != nil {
		c.logger.Error("failed to set configimpl value",
			clog.String("key", configKey),
			clog.Err(err))
		return err
	}

	c.logger.Info("configimpl value set successfully",
		clog.String("key", configKey))

	return nil
}

// Delete 删除配置
func (c *EtcdConfigCenter) Delete(ctx context.Context, key string) error {
	if key == "" {
		return client.NewCoordinationError(
			client.ErrCodeValidation,
			"configimpl key cannot be empty",
			nil,
		)
	}

	configKey := path.Join(c.prefix, key)

	c.logger.Info("deleting configimpl value",
		clog.String("key", configKey))

	resp, err := c.client.Delete(ctx, configKey)
	if err != nil {
		c.logger.Error("failed to delete configimpl value",
			clog.String("key", configKey),
			clog.Err(err))
		return err
	}

	if resp.Deleted == 0 {
		c.logger.Debug("configimpl key not found for deletion",
			clog.String("key", configKey))
		return client.NewCoordinationError(
			client.ErrCodeNotFound,
			"configimpl key not found",
			nil,
		)
	}

	c.logger.Info("configimpl value deleted successfully",
		clog.String("key", configKey))

	return nil
}

// Watch 监听配置变化
func (c *EtcdConfigCenter) Watch(ctx context.Context, key string) (<-chan types.ConfigEvent, error) {
	if key == "" {
		return nil, client.NewCoordinationError(
			client.ErrCodeValidation,
			"configimpl key cannot be empty",
			nil,
		)
	}

	configKey := path.Join(c.prefix, key)

	c.logger.Info("starting to watch configimpl changes",
		clog.String("key", configKey))

	watchCh := c.client.Watch(ctx, configKey)
	eventCh := make(chan types.ConfigEvent, 10)

	go func() {
		defer close(eventCh)
		defer c.logger.Info("configimpl watch stopped",
			clog.String("key", configKey))

		for resp := range watchCh {
			if resp.Err() != nil {
				c.logger.Error("watch error occurred",
					clog.String("key", configKey),
					clog.Err(resp.Err()))
				continue
			}

			for _, event := range resp.Events {
				configEvent := c.convertEvent(event)
				if configEvent != nil {
					c.logger.Info("configimpl change detected",
						clog.String("key", configKey),
						clog.String("type", string(configEvent.Type)))

					select {
					case eventCh <- *configEvent:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return eventCh, nil
}

// List 列出所有配置键
func (c *EtcdConfigCenter) List(ctx context.Context, prefix string) ([]string, error) {
	var searchPrefix string
	if prefix == "" {
		searchPrefix = c.prefix + "/"
	} else {
		searchPrefix = path.Join(c.prefix, prefix)
	}

	c.logger.Info("listing configimpl keys",
		clog.String("prefix", searchPrefix))

	resp, err := c.client.Get(ctx, searchPrefix, clientv3.WithPrefix())
	if err != nil {
		c.logger.Error("failed to list configimpl keys",
			clog.String("prefix", searchPrefix),
			clog.Err(err))
		return nil, err
	}

	var keys []string
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		// 移除前缀，只返回相对键名
		if strings.HasPrefix(key, c.prefix+"/") {
			relativeKey := strings.TrimPrefix(key, c.prefix+"/")
			keys = append(keys, relativeKey)
		}
	}

	c.logger.Info("configimpl keys listed successfully",
		clog.String("prefix", searchPrefix),
		clog.Int("count", len(keys)))

	return keys, nil
}

// convertEvent 转换 etcd 事件为配置事件
func (c *EtcdConfigCenter) convertEvent(event *clientv3.Event) *types.ConfigEvent {
	key := string(event.Kv.Key)

	// 移除前缀，只返回相对键名
	if !strings.HasPrefix(key, c.prefix+"/") {
		return nil
	}
	relativeKey := strings.TrimPrefix(key, c.prefix+"/")

	var eventType types.EventType
	var value interface{}

	switch event.Type {
	case clientv3.EventTypePut:
		eventType = types.EventTypePut
		// 尝试解析为 JSON，如果失败则返回原始字符串
		valueBytes := event.Kv.Value
		if err := json.Unmarshal(valueBytes, &value); err != nil {
			value = string(valueBytes)
		}
	case clientv3.EventTypeDelete:
		eventType = types.EventTypeDelete
		value = nil
	default:
		return nil
	}

	return &types.ConfigEvent{
		Type:  eventType,
		Key:   relativeKey,
		Value: value,
	}
}
