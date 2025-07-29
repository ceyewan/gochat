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

// EtcdConfigCenter 使用 etcd 实现 config.ConfigCenter 接口
type EtcdConfigCenter struct {
	client *client.EtcdClient // etcd 客户端
	prefix string             // 配置前缀
	logger clog.Logger        // 日志记录器
}

// NewEtcdConfigCenter 创建一个基于 etcd 的配置中心
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

// Get 获取配置值并反序列化到提供的类型 v
func (c *EtcdConfigCenter) Get(ctx context.Context, key string, v interface{}) error {
	if key == "" {
		return client.NewError(client.ErrCodeValidation, "config key cannot be empty", nil)
	}
	// 检查 v 是否为非 nil 指针
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return client.NewError(client.ErrCodeValidation, "target value must be a non-nil pointer", nil)
	}

	configKey := path.Join(c.prefix, key)
	resp, err := c.client.Get(ctx, configKey)
	if err != nil {
		return err // 客户端已包装错误
	}

	if len(resp.Kvs) == 0 {
		return client.NewError(client.ErrCodeNotFound, "config key not found", nil)
	}

	return unmarshalValue(resp.Kvs[0].Value, v)
}

// GetWithVersion 获取配置值和版本信息
func (c *EtcdConfigCenter) GetWithVersion(ctx context.Context, key string, v interface{}) (int64, error) {
	if key == "" {
		return 0, client.NewError(client.ErrCodeValidation, "config key cannot be empty", nil)
	}
	// 检查 v 是否为非 nil 指针
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return 0, client.NewError(client.ErrCodeValidation, "target value must be a non-nil pointer", nil)
	}

	configKey := path.Join(c.prefix, key)
	resp, err := c.client.Get(ctx, configKey)
	if err != nil {
		return 0, err // 客户端已包装错误
	}

	if len(resp.Kvs) == 0 {
		return 0, client.NewError(client.ErrCodeNotFound, "config key not found", nil)
	}

	kv := resp.Kvs[0]
	err = unmarshalValue(kv.Value, v)
	if err != nil {
		return 0, err
	}

	// 返回 etcd 的 ModRevision 作为版本号
	return kv.ModRevision, nil
}

// CompareAndSet 原子地比较并设置配置值
func (c *EtcdConfigCenter) CompareAndSet(ctx context.Context, key string, value interface{}, expectedVersion int64) error {
	if key == "" {
		return client.NewError(client.ErrCodeValidation, "config key cannot be empty", nil)
	}

	valueBytes, err := marshalValue(value)
	if err != nil {
		return client.NewError(client.ErrCodeValidation, "failed to serialize config value", err)
	}

	configKey := path.Join(c.prefix, key)

	// 使用 etcd 的事务来实现 CAS
	// 条件：ModRevision 等于期望版本
	// 成功：更新值
	// 失败：不执行任何操作
	txnResp, err := c.client.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(configKey), "=", expectedVersion)).
		Then(clientv3.OpPut(configKey, string(valueBytes))).
		Commit()

	if err != nil {
		return err // 客户端已包装错误
	}

	if !txnResp.Succeeded {
		return client.NewError(client.ErrCodeConflict, "config version mismatch, update rejected", nil)
	}

	return nil
}

// Set 序列化并存储配置值
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
	return err // 客户端已包装错误
}

// Delete 删除配置键
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

// Watch 监听单个配置键的变更
func (c *EtcdConfigCenter) Watch(ctx context.Context, key string, v interface{}) (config.Watcher[any], error) {
	if key == "" {
		return nil, client.NewError(client.ErrCodeValidation, "config key cannot be empty", nil)
	}
	configKey := path.Join(c.prefix, key)
	return c.watch(ctx, configKey, v, false)
}

// WatchPrefix 监听指定前缀下所有配置键的变更
func (c *EtcdConfigCenter) WatchPrefix(ctx context.Context, prefix string, v interface{}) (config.Watcher[any], error) {
	if prefix == "" {
		return nil, client.NewError(client.ErrCodeValidation, "config prefix cannot be empty", nil)
	}
	configPrefix := path.Join(c.prefix, prefix)
	return c.watch(ctx, configPrefix, v, true)
}

// List 列出指定前缀下的所有配置键
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

// watch 内部实现，监听单个键或前缀
func (c *EtcdConfigCenter) watch(ctx context.Context, keyOrPrefix string, v interface{}, isPrefix bool) (config.Watcher[any], error) {
	// 检查 v 是否为非 nil 指针以获取类型
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

// convertEvent 将 etcd 事件转换为配置事件
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
		// 删除事件不包含值
	default:
		return nil
	}

	return &config.ConfigEvent[any]{
		Type:  eventType,
		Key:   relativeKey,
		Value: value,
	}
}

// etcdWatcher 实现 config.Watcher 接口
type etcdWatcher struct {
	ch     chan config.ConfigEvent[any] // 事件通道
	cancel context.CancelFunc           // 取消函数
}

// Chan 返回事件通道
func (w *etcdWatcher) Chan() <-chan config.ConfigEvent[any] {
	return w.ch
}

// Close 停止监听
func (w *etcdWatcher) Close() {
	w.cancel()
}

// marshalValue 序列化值，优先处理 string 和 []byte，否则使用 JSON
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

// unmarshalValue 反序列化值，优先尝试 JSON，失败则尝试字符串
func unmarshalValue(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err == nil {
		return nil
	}

	// 如果目标是 *string，则直接赋值
	if strPtr, ok := v.(*string); ok {
		*strPtr = string(data)
		return nil
	}

	// 非 *string 且 JSON 失败，返回错误
	return client.NewError(client.ErrCodeValidation, "value is not valid JSON for the target type", nil)
}

// parseEventValue 智能解析事件值，支持多种类型处理策略
func (c *EtcdConfigCenter) parseEventValue(data []byte, valueType reflect.Type, key string) interface{} {
	// 如果目标类型是 interface{}，尝试自动推断类型
	if valueType.Kind() == reflect.Interface && valueType.NumMethod() == 0 {
		return c.parseAsInterface(data)
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
func (c *EtcdConfigCenter) parseAsInterface(data []byte) interface{} {
	// 首先尝试解析为 JSON
	var jsonValue interface{}
	if err := json.Unmarshal(data, &jsonValue); err == nil {
		return jsonValue
	}

	// JSON 解析失败，返回字符串
	return string(data)
}
