package config

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Logger 日志接口，避免循环依赖
type Logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
}

// Validator 配置验证器接口
type Validator[T any] interface {
	Validate(config *T) error
}

// ConfigUpdater 配置更新器接口，用于在配置更新时执行自定义逻辑
type ConfigUpdater[T any] interface {
	OnConfigUpdate(oldConfig, newConfig *T) error
}

// Manager 通用配置管理器 - 泛型实现，支持任意配置类型
//
// 设计原则：
// 1. 类型安全：使用泛型确保配置类型安全
// 2. 降级策略：配置中心不可用时自动使用默认配置
// 3. 热更新：支持配置热更新和监听
// 4. 可扩展：支持自定义验证器和更新器
// 5. 无循环依赖：通过接口抽象避免依赖具体实现
type Manager[T any] struct {
	// 配置中心
	configCenter ConfigCenter

	// 配置参数
	env       string
	service   string
	component string

	// 当前配置（原子操作）
	currentConfig atomic.Value // *T

	// 默认配置
	defaultConfig T

	// 可选组件
	validator Validator[T]
	updater   ConfigUpdater[T]
	logger    Logger

	// 配置监听器
	watcher Watcher[any]

	// 控制
	mu       sync.RWMutex
	stopCh   chan struct{}
	watching bool
}

// ManagerOption 配置管理器选项
type ManagerOption[T any] func(*Manager[T])

// WithValidator 设置配置验证器
func WithValidator[T any](validator Validator[T]) ManagerOption[T] {
	return func(m *Manager[T]) {
		m.validator = validator
	}
}

// WithUpdater 设置配置更新器
func WithUpdater[T any](updater ConfigUpdater[T]) ManagerOption[T] {
	return func(m *Manager[T]) {
		m.updater = updater
	}
}

// WithLogger 设置日志器
func WithLogger[T any](logger Logger) ManagerOption[T] {
	return func(m *Manager[T]) {
		m.logger = logger
	}
}

// NewManager 创建配置管理器
func NewManager[T any](
	configCenter ConfigCenter,
	env, service, component string,
	defaultConfig T,
	opts ...ManagerOption[T],
) *Manager[T] {
	m := &Manager[T]{
		configCenter:  configCenter,
		env:           env,
		service:       service,
		component:     component,
		defaultConfig: defaultConfig,
		stopCh:        make(chan struct{}),
	}

	// 应用选项
	for _, opt := range opts {
		opt(m)
	}

	// 设置默认配置
	m.currentConfig.Store(&defaultConfig)

	// 如果有配置中心，尝试加载配置
	if configCenter != nil {
		m.loadConfigFromCenter()
		m.startWatching()
	}

	return m
}

// GetCurrentConfig 获取当前配置
func (m *Manager[T]) GetCurrentConfig() *T {
	if config := m.currentConfig.Load(); config != nil {
		return config.(*T)
	}
	// 返回默认配置的副本
	defaultCopy := m.defaultConfig
	return &defaultCopy
}

// ReloadConfig 重新加载配置
func (m *Manager[T]) ReloadConfig() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.configCenter != nil {
		m.loadConfigFromCenter()
	}
}

// Close 关闭配置管理器
func (m *Manager[T]) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopWatching()
}

// loadConfigFromCenter 从配置中心加载配置
func (m *Manager[T]) loadConfigFromCenter() {
	if m.configCenter == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := m.buildConfigKey()
	var config T
	err := m.configCenter.Get(ctx, key, &config)
	if err != nil {
		// 记录错误但不阻断，继续使用当前配置
		if m.logger != nil {
			m.logger.Warn("failed to load config from center, using current config",
				"error", err,
				"key", key,
				"env", m.env,
				"service", m.service,
				"component", m.component)
		}
		return
	}

	// 验证配置
	if m.validator != nil {
		if err := m.validator.Validate(&config); err != nil {
			if m.logger != nil {
				m.logger.Warn("invalid config from center, using current config",
					"error", err,
					"key", key)
			}
			return
		}
	}

	// 安全更新配置
	if err := m.safeUpdateConfig(&config); err != nil {
		if m.logger != nil {
			m.logger.Error("failed to update config safely",
				"error", err,
				"key", key)
		}
		return
	}

	if m.logger != nil {
		m.logger.Info("config loaded from center",
			"key", key,
			"env", m.env,
			"service", m.service,
			"component", m.component)
	}
}

// safeUpdateConfig 安全地更新配置
func (m *Manager[T]) safeUpdateConfig(newConfig *T) error {
	// 获取当前配置作为备份
	oldConfig := m.currentConfig.Load().(*T)

	// 如果有更新器，先调用更新器
	if m.updater != nil {
		if err := m.updater.OnConfigUpdate(oldConfig, newConfig); err != nil {
			return fmt.Errorf("config updater failed: %w", err)
		}
	}

	// 更新配置
	m.currentConfig.Store(newConfig)
	return nil
}

// buildConfigKey 构建配置键
func (m *Manager[T]) buildConfigKey() string {
	return "/config/" + m.env + "/" + m.service + "/" + m.component
}

// startWatching 启动配置监听
func (m *Manager[T]) startWatching() {
	if m.configCenter == nil || m.watching {
		return
	}

	ctx := context.Background()
	var config T
	watcher, err := m.configCenter.Watch(ctx, m.buildConfigKey(), &config)
	if err != nil {
		if m.logger != nil {
			m.logger.Warn("failed to start config watcher",
				"error", err,
				"key", m.buildConfigKey())
		}
		return
	}

	m.watcher = watcher
	m.watching = true

	// 启动监听协程
	go m.watchLoop()

	if m.logger != nil {
		m.logger.Info("config watcher started",
			"key", m.buildConfigKey())
	}
}

// stopWatching 停止配置监听
func (m *Manager[T]) stopWatching() {
	if !m.watching {
		return
	}

	m.watching = false

	if m.watcher != nil {
		m.watcher.Close()
		m.watcher = nil
	}

	// 通知停止
	select {
	case m.stopCh <- struct{}{}:
	default:
	}
}

// watchLoop 配置监听循环
func (m *Manager[T]) watchLoop() {
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("config watch loop panic",
					"recover", r,
					"key", m.buildConfigKey())
			}
		}
	}()

	for {
		select {
		case event, ok := <-m.watcher.Chan():
			if !ok {
				if m.logger != nil {
					m.logger.Debug("config watcher channel closed",
						"key", m.buildConfigKey())
				}
				return
			}

			if event.Type == EventTypePut {
				// 解析配置
				if config, err := m.parseConfig(event.Value); err == nil {
					// 安全更新配置
					if err := m.safeUpdateConfig(config); err != nil {
						if m.logger != nil {
							m.logger.Error("failed to update config from watcher",
								"error", err,
								"key", m.buildConfigKey())
						}
						continue
					}

					if m.logger != nil {
						m.logger.Info("config updated from watcher",
							"key", m.buildConfigKey())
					}
				} else {
					if m.logger != nil {
						m.logger.Error("failed to parse config from event",
							"error", err,
							"key", m.buildConfigKey(),
							"value", event.Value)
					}
				}
			}
		case <-m.stopCh:
			return
		}
	}
}

// parseConfig 解析配置
func (m *Manager[T]) parseConfig(value any) (*T, error) {
	// 如果已经是目标类型，直接返回
	if config, ok := value.(*T); ok {
		return config, nil
	}

	// 尝试通过 JSON 序列化/反序列化转换
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config value: %w", err)
	}

	var config T
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// ===== 便捷工厂函数 =====

// SimpleManager 创建简单的配置管理器（无验证器和更新器）
func SimpleManager[T any](
	configCenter ConfigCenter,
	env, service, component string,
	defaultConfig T,
	logger Logger,
) *Manager[T] {
	return NewManager(configCenter, env, service, component, defaultConfig,
		WithLogger[T](logger))
}

// ValidatedManager 创建带验证器的配置管理器
func ValidatedManager[T any](
	configCenter ConfigCenter,
	env, service, component string,
	defaultConfig T,
	validator Validator[T],
	logger Logger,
) *Manager[T] {
	return NewManager(configCenter, env, service, component, defaultConfig,
		WithValidator[T](validator),
		WithLogger[T](logger))
}

// FullManager 创建功能完整的配置管理器
func FullManager[T any](
	configCenter ConfigCenter,
	env, service, component string,
	defaultConfig T,
	validator Validator[T],
	updater ConfigUpdater[T],
	logger Logger,
) *Manager[T] {
	return NewManager(configCenter, env, service, component, defaultConfig,
		WithValidator[T](validator),
		WithUpdater[T](updater),
		WithLogger[T](logger))
}
