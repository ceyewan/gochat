package clog

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ceyewan/gochat/im-infra/coord/config"
)

// ConfigCenterAdapter 配置中心适配器
// 将 coord.ConfigCenter 适配为 clog.ConfigSource
type ConfigCenterAdapter struct {
	configCenter config.ConfigCenter
	logger       Logger
}

// NewConfigCenterAdapter 创建配置中心适配器
func NewConfigCenterAdapter(configCenter config.ConfigCenter) *ConfigCenterAdapter {
	return &ConfigCenterAdapter{
		configCenter: configCenter,
		logger:       Module("clog.config"),
	}
}

// GetConfig 实现 ConfigSource 接口
func (a *ConfigCenterAdapter) GetConfig(ctx context.Context, env, service, component string) (*Config, error) {
	key := fmt.Sprintf("/config/%s/%s/%s", env, service, component)

	var config Config
	err := a.configCenter.Get(ctx, key, &config)
	if err != nil {
		a.logger.Warn("failed to get config from config center",
			Err(err),
			String("key", key),
			String("env", env),
			String("service", service),
			String("component", component))
		return nil, err
	}

	a.logger.Debug("successfully got config from config center",
		String("key", key),
		String("level", config.Level),
		String("format", config.Format),
		String("output", config.Output))

	return &config, nil
}

// Watch 实现 ConfigSource 接口
func (a *ConfigCenterAdapter) Watch(ctx context.Context, env, service, component string) (ConfigWatcher, error) {
	key := fmt.Sprintf("/config/%s/%s/%s", env, service, component)

	// 创建一个空的 Config 实例用于类型推断
	var config Config
	watcher, err := a.configCenter.Watch(ctx, key, &config)
	if err != nil {
		a.logger.Warn("failed to create config watcher",
			Err(err),
			String("key", key),
			String("env", env),
			String("service", service),
			String("component", component))
		return nil, err
	}

	a.logger.Info("successfully created config watcher",
		String("key", key))

	return &configCenterWatcher{
		watcher: watcher,
		logger:  a.logger,
		key:     key,
		stopCh:  make(chan struct{}),
	}, nil
}

// configCenterWatcher 配置中心监听器适配器
type configCenterWatcher struct {
	watcher config.Watcher[any]
	logger  Logger
	key     string
	ch      chan *Config
	once    sync.Once
	closed  bool
	mu      sync.RWMutex
	stopCh  chan struct{} // 添加停止信号通道
}

// Chan 实现 ConfigWatcher 接口
func (w *configCenterWatcher) Chan() <-chan *Config {
	w.once.Do(func() {
		w.ch = make(chan *Config, 1)
		go w.watchLoop()
	})
	return w.ch
}

// Close 实现 ConfigWatcher 接口
func (w *configCenterWatcher) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return
	}

	w.closed = true

	// 发送停止信号
	close(w.stopCh)

	if w.watcher != nil {
		w.watcher.Close()
	}
	if w.ch != nil {
		close(w.ch)
	}

	w.logger.Debug("config watcher closed", String("key", w.key))
}

// watchLoop 监听配置变化
func (w *configCenterWatcher) watchLoop() {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("config watcher panic", Any("recover", r), String("key", w.key))
		}
	}()

	for {
		select {
		case <-w.stopCh:
			w.logger.Debug("config watcher stopped", String("key", w.key))
			return
		case event, ok := <-w.watcher.Chan():
			if !ok {
				w.logger.Debug("config watcher channel closed", String("key", w.key))
				return
			}

			if event.Type == config.EventTypePut {
				// 尝试将事件值转换为 Config
				if configData, err := w.parseConfig(event.Value); err == nil {
					w.logger.Info("config changed",
						String("key", w.key),
						String("level", configData.Level),
						String("format", configData.Format))

					select {
					case w.ch <- configData:
					case <-w.stopCh:
						return
					default:
						// 如果通道满了，丢弃旧的配置更新
						w.logger.Warn("config channel full, dropping old update", String("key", w.key))
					}
				} else {
					w.logger.Error("failed to parse config from event",
						Err(err),
						String("key", w.key),
						Any("value", event.Value))
				}
			}
		}
	}
}

// parseConfig 解析配置
func (w *configCenterWatcher) parseConfig(value any) (*Config, error) {
	// 如果已经是 Config 类型，直接返回
	if config, ok := value.(*Config); ok {
		return config, nil
	}

	// 尝试通过 JSON 序列化/反序列化转换
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config value: %w", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// SetupConfigCenterFromCoord 设置配置中心
// 这是一个便利函数，用于快速设置配置中心作为 clog 的配置源
func SetupConfigCenterFromCoord(configCenter config.ConfigCenter, env, service, component string) {
	adapter := NewConfigCenterAdapter(configCenter)
	SetupConfigCenter(adapter, env, service, component)
}
