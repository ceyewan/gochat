package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// configCenter 是 ConfigCenter 接口的内部实现
type configCenter struct {
	client    *clientv3.Client
	config    ConfigCenterConfig
	logger    clog.Logger
	watchers  map[string]context.CancelFunc // 监听器映射
	watcherMu sync.RWMutex
	closed    bool
	closeMu   sync.RWMutex
}

// newConfigCenter 创建新的配置中心实例
func newConfigCenter(client *clientv3.Client, config ConfigCenterConfig, logger clog.Logger) ConfigCenter {
	return &configCenter{
		client:   client,
		config:   config,
		logger:   logger,
		watchers: make(map[string]context.CancelFunc),
	}
}

// Get 获取配置值
func (cc *configCenter) Get(ctx context.Context, key string) (*ConfigValue, error) {
	cc.closeMu.RLock()
	defer cc.closeMu.RUnlock()

	if cc.closed {
		return nil, fmt.Errorf("config center is closed")
	}

	configKey := cc.buildConfigKey(key)

	resp, err := cc.client.Get(ctx, configKey)
	if err != nil {
		cc.logger.Error("获取配置失败",
			clog.Err(err),
			clog.String("key", key),
		)
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("config not found: %s", key)
	}

	var configValue ConfigValue
	if err := json.Unmarshal(resp.Kvs[0].Value, &configValue); err != nil {
		cc.logger.Error("解析配置值失败",
			clog.Err(err),
			clog.String("key", key),
		)
		return nil, fmt.Errorf("failed to unmarshal config value: %w", err)
	}

	cc.logger.Debug("获取配置成功",
		clog.String("key", key),
		clog.Int64("version", configValue.Version),
	)

	return &configValue, nil
}

// Set 设置配置值，支持版本控制
func (cc *configCenter) Set(ctx context.Context, key string, value interface{}, version int64) error {
	cc.closeMu.RLock()
	defer cc.closeMu.RUnlock()

	if cc.closed {
		return fmt.Errorf("config center is closed")
	}

	// 验证配置值
	if cc.config.EnableValidation {
		if err := cc.validateConfigValue(key, value); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}
	}

	configKey := cc.buildConfigKey(key)

	// 序列化值
	valueStr, err := cc.serializeValue(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// 构建配置值对象
	now := time.Now()
	configValue := ConfigValue{
		Key:        key,
		Value:      valueStr,
		Version:    version,
		CreateTime: now,
		UpdateTime: now,
		Metadata:   make(map[string]string),
	}

	// 如果启用版本控制，检查版本冲突
	if cc.config.EnableVersioning && version > 0 {
		currentConfig, err := cc.Get(ctx, key)
		if err == nil && currentConfig.Version >= version {
			return fmt.Errorf("version conflict: current version %d, provided version %d",
				currentConfig.Version, version)
		}
	}

	// 如果版本为0，自动生成版本号
	if version == 0 {
		configValue.Version = time.Now().UnixNano()
	}

	// 序列化配置值
	configData, err := json.Marshal(configValue)
	if err != nil {
		return fmt.Errorf("failed to marshal config value: %w", err)
	}

	// 保存配置
	_, err = cc.client.Put(ctx, configKey, string(configData))
	if err != nil {
		cc.logger.Error("设置配置失败",
			clog.Err(err),
			clog.String("key", key),
			clog.Int64("version", configValue.Version),
		)
		return fmt.Errorf("failed to set config: %w", err)
	}

	// 如果启用版本控制，保存历史版本
	if cc.config.EnableVersioning {
		if err := cc.saveVersionHistory(ctx, configValue); err != nil {
			cc.logger.Warn("保存版本历史失败",
				clog.Err(err),
				clog.String("key", key),
				clog.Int64("version", configValue.Version),
			)
		}
	}

	cc.logger.Info("设置配置成功",
		clog.String("key", key),
		clog.Int64("version", configValue.Version),
		clog.String("value_preview", cc.getValuePreview(valueStr)),
	)

	return nil
}

// Delete 删除配置，支持版本控制
func (cc *configCenter) Delete(ctx context.Context, key string, version int64) error {
	cc.closeMu.RLock()
	defer cc.closeMu.RUnlock()

	if cc.closed {
		return fmt.Errorf("config center is closed")
	}

	// 如果启用版本控制，检查版本
	if cc.config.EnableVersioning && version > 0 {
		currentConfig, err := cc.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to get current config for version check: %w", err)
		}
		if currentConfig.Version != version {
			return fmt.Errorf("version mismatch: current version %d, provided version %d",
				currentConfig.Version, version)
		}
	}

	configKey := cc.buildConfigKey(key)

	_, err := cc.client.Delete(ctx, configKey)
	if err != nil {
		cc.logger.Error("删除配置失败",
			clog.Err(err),
			clog.String("key", key),
		)
		return fmt.Errorf("failed to delete config: %w", err)
	}

	// 清理版本历史
	if cc.config.EnableVersioning {
		if err := cc.cleanVersionHistory(ctx, key); err != nil {
			cc.logger.Warn("清理版本历史失败",
				clog.Err(err),
				clog.String("key", key),
			)
		}
	}

	cc.logger.Info("删除配置成功", clog.String("key", key))
	return nil
}

// GetVersion 获取配置的当前版本号
func (cc *configCenter) GetVersion(ctx context.Context, key string) (int64, error) {
	cc.closeMu.RLock()
	defer cc.closeMu.RUnlock()

	if cc.closed {
		return 0, fmt.Errorf("config center is closed")
	}

	configValue, err := cc.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	return configValue.Version, nil
}

// GetHistory 获取配置的历史版本
func (cc *configCenter) GetHistory(ctx context.Context, key string, limit int) ([]ConfigVersion, error) {
	cc.closeMu.RLock()
	defer cc.closeMu.RUnlock()

	if cc.closed {
		return nil, fmt.Errorf("config center is closed")
	}

	if !cc.config.EnableVersioning {
		return nil, fmt.Errorf("versioning is not enabled")
	}

	historyPrefix := cc.buildHistoryPrefix(key)

	resp, err := cc.client.Get(ctx, historyPrefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		cc.logger.Error("获取配置历史失败",
			clog.Err(err),
			clog.String("key", key),
		)
		return nil, fmt.Errorf("failed to get config history: %w", err)
	}

	var versions []ConfigVersion
	count := 0
	for _, kv := range resp.Kvs {
		if limit > 0 && count >= limit {
			break
		}

		var version ConfigVersion
		if err := json.Unmarshal(kv.Value, &version); err != nil {
			cc.logger.Warn("解析历史版本失败",
				clog.Err(err),
				clog.String("key", string(kv.Key)),
			)
			continue
		}

		versions = append(versions, version)
		count++
	}

	cc.logger.Debug("获取配置历史成功",
		clog.String("key", key),
		clog.Int("count", len(versions)),
	)

	return versions, nil
}

// Watch 监听指定配置的变更
func (cc *configCenter) Watch(ctx context.Context, key string) (<-chan *ConfigChange, error) {
	cc.closeMu.RLock()
	defer cc.closeMu.RUnlock()

	if cc.closed {
		return nil, fmt.Errorf("config center is closed")
	}

	configKey := cc.buildConfigKey(key)
	ch := make(chan *ConfigChange, cc.config.WatchBufferSize)

	// 创建监听上下文
	watchCtx, cancel := context.WithCancel(ctx)

	// 保存取消函数
	cc.watcherMu.Lock()
	cc.watchers[key] = cancel
	cc.watcherMu.Unlock()

	go func() {
		defer close(ch)
		defer func() {
			cc.watcherMu.Lock()
			delete(cc.watchers, key)
			cc.watcherMu.Unlock()
		}()

		// 监听变化
		watchCh := cc.client.Watch(watchCtx, configKey)
		for {
			select {
			case <-watchCtx.Done():
				return
			case watchResp, ok := <-watchCh:
				if !ok {
					return
				}

				if watchResp.Err() != nil {
					cc.logger.Error("监听配置变化失败",
						clog.Err(watchResp.Err()),
						clog.String("key", key),
					)
					continue
				}

				for _, event := range watchResp.Events {
					change := cc.buildConfigChange(event)
					if change != nil {
						select {
						case ch <- change:
						case <-watchCtx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return ch, nil
}

// WatchPrefix 监听指定前缀下所有配置的变更
func (cc *configCenter) WatchPrefix(ctx context.Context, prefix string) (<-chan *ConfigChange, error) {
	cc.closeMu.RLock()
	defer cc.closeMu.RUnlock()

	if cc.closed {
		return nil, fmt.Errorf("config center is closed")
	}

	configPrefix := cc.buildConfigPrefix(prefix)
	ch := make(chan *ConfigChange, cc.config.WatchBufferSize)

	// 创建监听上下文
	watchCtx, cancel := context.WithCancel(ctx)

	// 保存取消函数
	watcherKey := fmt.Sprintf("prefix:%s", prefix)
	cc.watcherMu.Lock()
	cc.watchers[watcherKey] = cancel
	cc.watcherMu.Unlock()

	go func() {
		defer close(ch)
		defer func() {
			cc.watcherMu.Lock()
			delete(cc.watchers, watcherKey)
			cc.watcherMu.Unlock()
		}()

		// 监听变化
		watchCh := cc.client.Watch(watchCtx, configPrefix, clientv3.WithPrefix())
		for {
			select {
			case <-watchCtx.Done():
				return
			case watchResp, ok := <-watchCh:
				if !ok {
					return
				}

				if watchResp.Err() != nil {
					cc.logger.Error("监听配置前缀变化失败",
						clog.Err(watchResp.Err()),
						clog.String("prefix", prefix),
					)
					continue
				}

				for _, event := range watchResp.Events {
					change := cc.buildConfigChange(event)
					if change != nil {
						select {
						case ch <- change:
						case <-watchCtx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return ch, nil
}

// 辅助方法

// buildConfigKey 构建配置键名
func (cc *configCenter) buildConfigKey(key string) string {
	return fmt.Sprintf("%s/%s", cc.config.KeyPrefix, key)
}

// buildConfigPrefix 构建配置前缀
func (cc *configCenter) buildConfigPrefix(prefix string) string {
	return fmt.Sprintf("%s/%s", cc.config.KeyPrefix, prefix)
}

// buildHistoryKey 构建历史版本键名
func (cc *configCenter) buildHistoryKey(key string, version int64) string {
	return fmt.Sprintf("%s/history/%s/%d", cc.config.KeyPrefix, key, version)
}

// buildHistoryPrefix 构建历史版本前缀
func (cc *configCenter) buildHistoryPrefix(key string) string {
	return fmt.Sprintf("%s/history/%s/", cc.config.KeyPrefix, key)
}

// serializeValue 序列化配置值
func (cc *configCenter) serializeValue(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

// validateConfigValue 验证配置值
func (cc *configCenter) validateConfigValue(key string, value interface{}) error {
	// 这里可以实现具体的验证逻辑
	// 例如：检查值的类型、长度、格式等
	if value == nil {
		return fmt.Errorf("config value cannot be nil")
	}

	valueStr, err := cc.serializeValue(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value for validation: %w", err)
	}

	// 检查值的长度
	if len(valueStr) > 1024*1024 { // 1MB 限制
		return fmt.Errorf("config value too large: %d bytes", len(valueStr))
	}

	return nil
}

// saveVersionHistory 保存版本历史
func (cc *configCenter) saveVersionHistory(ctx context.Context, configValue ConfigValue) error {
	historyKey := cc.buildHistoryKey(configValue.Key, configValue.Version)

	version := ConfigVersion{
		Version:    configValue.Version,
		Value:      configValue.Value,
		CreateTime: configValue.UpdateTime,
		Author:     "system", // 可以从上下文中获取用户信息
		Comment:    "",
	}

	versionData, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal version history: %w", err)
	}

	_, err = cc.client.Put(ctx, historyKey, string(versionData))
	if err != nil {
		return fmt.Errorf("failed to save version history: %w", err)
	}

	// 清理过期的历史版本
	go cc.cleanOldVersionHistory(ctx, configValue.Key)

	return nil
}

// cleanVersionHistory 清理版本历史
func (cc *configCenter) cleanVersionHistory(ctx context.Context, key string) error {
	historyPrefix := cc.buildHistoryPrefix(key)

	_, err := cc.client.Delete(ctx, historyPrefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to clean version history: %w", err)
	}

	return nil
}

// cleanOldVersionHistory 清理过期的历史版本
func (cc *configCenter) cleanOldVersionHistory(ctx context.Context, key string) {
	if cc.config.MaxVersionHistory <= 0 {
		return
	}

	historyPrefix := cc.buildHistoryPrefix(key)

	resp, err := cc.client.Get(ctx, historyPrefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		cc.logger.Warn("获取历史版本列表失败",
			clog.Err(err),
			clog.String("key", key),
		)
		return
	}

	if len(resp.Kvs) <= cc.config.MaxVersionHistory {
		return
	}

	// 删除超出限制的历史版本
	for i := cc.config.MaxVersionHistory; i < len(resp.Kvs); i++ {
		_, err := cc.client.Delete(ctx, string(resp.Kvs[i].Key))
		if err != nil {
			cc.logger.Warn("删除过期历史版本失败",
				clog.Err(err),
				clog.String("history_key", string(resp.Kvs[i].Key)),
			)
		}
	}

	cc.logger.Debug("清理过期历史版本",
		clog.String("key", key),
		clog.Int("cleaned_count", len(resp.Kvs)-cc.config.MaxVersionHistory),
	)
}

// buildConfigChange 构建配置变更事件
func (cc *configCenter) buildConfigChange(event *clientv3.Event) *ConfigChange {
	change := &ConfigChange{
		Timestamp: time.Now(),
	}

	// 从键名中提取配置键
	keyParts := strings.Split(string(event.Kv.Key), "/")
	if len(keyParts) < 2 {
		return nil
	}
	change.Key = strings.Join(keyParts[2:], "/") // 去掉前缀部分

	switch event.Type {
	case clientv3.EventTypePut:
		if event.IsCreate() {
			change.Type = ConfigChangeCreate
		} else {
			change.Type = ConfigChangeUpdate
		}

		var configValue ConfigValue
		if err := json.Unmarshal(event.Kv.Value, &configValue); err == nil {
			change.NewValue = &configValue
		}

		if event.PrevKv != nil {
			var prevConfigValue ConfigValue
			if err := json.Unmarshal(event.PrevKv.Value, &prevConfigValue); err == nil {
				change.OldValue = &prevConfigValue
			}
		}

	case clientv3.EventTypeDelete:
		change.Type = ConfigChangeDelete

		if event.PrevKv != nil {
			var prevConfigValue ConfigValue
			if err := json.Unmarshal(event.PrevKv.Value, &prevConfigValue); err == nil {
				change.OldValue = &prevConfigValue
			}
		}
	}

	return change
}

// getValuePreview 获取值的预览（用于日志）
func (cc *configCenter) getValuePreview(value string) string {
	if len(value) <= 50 {
		return value
	}
	return value[:47] + "..."
}

// Close 关闭配置中心
func (cc *configCenter) Close() error {
	cc.closeMu.Lock()
	defer cc.closeMu.Unlock()

	if cc.closed {
		return nil
	}

	cc.closed = true

	// 取消所有监听器
	cc.watcherMu.Lock()
	for _, cancel := range cc.watchers {
		cancel()
	}
	cc.watchers = make(map[string]context.CancelFunc)
	cc.watcherMu.Unlock()

	cc.logger.Info("配置中心已关闭")
	return nil
}
