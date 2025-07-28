package configimpl

import (
	"context"
	"encoding/json"
	"path"
	"reflect"
	"strings"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/config"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdConfigCenter implements the config.ConfigCenter interface using etcd.
type EtcdConfigCenter struct {
	client *client.EtcdClient
	prefix string
	logger clog.Logger
}

// NewEtcdConfigCenter creates a new etcd-based config center.
func NewEtcdConfigCenter(c *client.EtcdClient, prefix string, logger clog.Logger) *EtcdConfigCenter {
	if prefix == "" {
		prefix = "/config"
	}
	if logger == nil {
		logger = clog.Module("coordination.config")
	}
	return &EtcdConfigCenter{
		client: c,
		prefix: prefix,
		logger: logger,
	}
}

// Get retrieves a configuration value and unmarshals it into the provided type `v`.
func (c *EtcdConfigCenter) Get(ctx context.Context, key string, v interface{}) error {
	if key == "" {
		return client.NewError(client.ErrCodeValidation, "config key cannot be empty", nil)
	}
	// Ensure `v` is a non-nil pointer.
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return client.NewError(client.ErrCodeValidation, "target value must be a non-nil pointer", nil)
	}

	configKey := path.Join(c.prefix, key)
	resp, err := c.client.Get(ctx, configKey)
	if err != nil {
		return err // The client already wraps this error.
	}

	if len(resp.Kvs) == 0 {
		return client.NewError(client.ErrCodeNotFound, "config key not found", nil)
	}

	return unmarshalValue(resp.Kvs[0].Value, v)
}

// Set serializes and stores a configuration value.
func (c *EtcdConfigCenter) Set(ctx context.Context, key string, value interface{}) error {
	if key == "" {
		return client.NewError(client.ErrCodeValidation, "config key cannot be empty", nil)
	}

	valueBytes, err := marshalValue(value)
	if err != nil {
		return client.NewError(client.ErrCodeValidation, "failed to serialize config value", err)
	}

	configKey := path.Join(c.prefix, key)
	_, err = c.client.Put(ctx, configKey, string(valueBytes))
	return err // The client already wraps this error.
}

// Delete removes a configuration key.
func (c *EtcdConfigCenter) Delete(ctx context.Context, key string) error {
	if key == "" {
		return client.NewError(client.ErrCodeValidation, "config key cannot be empty", nil)
	}

	configKey := path.Join(c.prefix, key)
	resp, err := c.client.Delete(ctx, configKey)
	if err != nil {
		return err
	}
	if resp.Deleted == 0 {
		return client.NewError(client.ErrCodeNotFound, "config key not found for deletion", nil)
	}
	return nil
}

// Watch watches for changes on a single key.
func (c *EtcdConfigCenter) Watch(ctx context.Context, key string, v interface{}) (config.Watcher[any], error) {
	if key == "" {
		return nil, client.NewError(client.ErrCodeValidation, "config key cannot be empty", nil)
	}
	configKey := path.Join(c.prefix, key)
	return c.watch(ctx, configKey, v, false)
}

// WatchPrefix watches for changes on all keys under a given prefix.
func (c *EtcdConfigCenter) WatchPrefix(ctx context.Context, prefix string, v interface{}) (config.Watcher[any], error) {
	if prefix == "" {
		return nil, client.NewError(client.ErrCodeValidation, "config prefix cannot be empty", nil)
	}
	configPrefix := path.Join(c.prefix, prefix)
	return c.watch(ctx, configPrefix, v, true)
}

// List lists all keys under a given prefix.
func (c *EtcdConfigCenter) List(ctx context.Context, prefix string) ([]string, error) {
	searchPrefix := path.Join(c.prefix, prefix)
	if !strings.HasSuffix(searchPrefix, "/") {
		searchPrefix += "/"
	}

	resp, err := c.client.Get(ctx, searchPrefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		keys[i] = strings.TrimPrefix(string(kv.Key), c.prefix+"/")
	}
	return keys, nil
}

// watch is the internal implementation for watching keys or prefixes.
func (c *EtcdConfigCenter) watch(ctx context.Context, keyOrPrefix string, v interface{}, isPrefix bool) (config.Watcher[any], error) {
	// Ensure `v` is a non-nil pointer to get its type.
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil, client.NewError(client.ErrCodeValidation, "target value type must be a non-nil pointer", nil)
	}
	valueType := rv.Type().Elem()

	var opts []clientv3.OpOption
	if isPrefix {
		opts = append(opts, clientv3.WithPrefix())
	}

	watchCtx, cancel := context.WithCancel(ctx)
	etcdWatchCh := c.client.Watch(watchCtx, keyOrPrefix, opts...)
	eventCh := make(chan config.ConfigEvent[any], 10)

	w := &etcdWatcher{
		ch:     eventCh,
		cancel: cancel,
	}

	go func() {
		defer close(eventCh)
		for resp := range etcdWatchCh {
			if err := resp.Err(); err != nil {
				c.logger.Error("Watcher error", clog.String("key", keyOrPrefix), clog.Err(err))
				return
			}
			for _, event := range resp.Events {
				configEvent := c.convertEvent(event, valueType)
				if configEvent != nil {
					select {
					case eventCh <- *configEvent:
					case <-watchCtx.Done():
						return
					}
				}
			}
		}
	}()

	return w, nil
}

func (c *EtcdConfigCenter) convertEvent(event *clientv3.Event, valueType reflect.Type) *config.ConfigEvent[any] {
	relativeKey := strings.TrimPrefix(string(event.Kv.Key), c.prefix+"/")
	var eventType config.EventType
	var value interface{}

	switch event.Type {
	case clientv3.EventTypePut:
		eventType = config.EventTypePut
		value = c.parseEventValue(event.Kv.Value, valueType, relativeKey)
	case clientv3.EventTypeDelete:
		eventType = config.EventTypeDelete
		// Value is nil for delete events.
	default:
		return nil
	}

	return &config.ConfigEvent[any]{
		Type:  eventType,
		Key:   relativeKey,
		Value: value,
	}
}

// etcdWatcher implements the config.Watcher interface.
type etcdWatcher struct {
	ch     chan config.ConfigEvent[any]
	cancel context.CancelFunc
}

func (w *etcdWatcher) Chan() <-chan config.ConfigEvent[any] {
	return w.ch
}

func (w *etcdWatcher) Close() {
	w.cancel()
}

// marshalValue serializes a value. It prioritizes string and []byte, falling back to JSON.
func marshalValue(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		return json.Marshal(value)
	}
}

// unmarshalValue deserializes a value. It attempts JSON first, then falls back to a direct string conversion if the target is a string.
func unmarshalValue(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err == nil {
		return nil
	}

	// If JSON unmarshal fails, check if the target is a *string.
	if strPtr, ok := v.(*string); ok {
		*strPtr = string(data)
		return nil
	}

	// If it's not a *string and JSON failed, it's an error.
	return client.NewError(client.ErrCodeValidation, "value is not valid JSON for the target type", nil)
}

// parseEventValue 智能解析事件值，支持多种类型处理策略
func (c *EtcdConfigCenter) parseEventValue(data []byte, valueType reflect.Type, key string) interface{} {
	// 如果目标类型是 interface{}，尝试自动推断类型
	if valueType.Kind() == reflect.Interface && valueType.NumMethod() == 0 {
		return c.parseAsInterface(data, key)
	}

	// 尝试解析为目标类型
	newValue := reflect.New(valueType).Interface()
	if err := unmarshalValue(data, newValue); err != nil {
		// 类型转换失败时，记录警告但不丢弃事件
		c.logger.Warn("Failed to unmarshal event value, returning raw string",
			clog.String("key", key),
			clog.String("target_type", valueType.String()),
			clog.Err(err))

		// 返回原始字符串值作为降级处理
		return string(data)
	}

	return reflect.ValueOf(newValue).Elem().Interface()
}

// parseAsInterface 当目标类型是 interface{} 时，自动推断最合适的类型
func (c *EtcdConfigCenter) parseAsInterface(data []byte, key string) interface{} {
	// 首先尝试解析为 JSON
	var jsonValue interface{}
	if err := json.Unmarshal(data, &jsonValue); err == nil {
		return jsonValue
	}

	// JSON 解析失败，返回字符串
	return string(data)
}
