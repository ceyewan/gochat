package config

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

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
// 6. 明确生命周期：通过 Start/Stop 方法管理生命周期
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
	logger    clog.Logger

	// 配置监听器
	watcher Watcher[any]

	// 控制
	mu       sync.RWMutex
	stopCh   chan struct{}
	watching bool

	// 生命周期控制
	started bool
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
func WithLogger[T any](logger clog.Logger) ManagerOption[T] {
	return func(m *Manager[T]) {
		m.logger = logger
	}
}

// NewManager 创建配置管理器
// 注意：创建后需要调用 Start() 方法来启动配置监听
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

	// 不再自动启动，需要显式调用 Start() 方法
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

// Start 启动配置管理器和监听器
// 这个方法是幂等的，可以安全地多次调用
func (m *Manager[T]) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return
	}

	// 启动时加载一次配置
	if m.configCenter != nil {
		m.loadConfigFromCenter()
		m.startWatching()
	}

	m.started = true
}

// Stop 停止配置管理器和监听器
// 这个方法是幂等的，可以安全地多次调用
func (m *Manager[T]) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return
	}

	m.stopWatching()
	m.started = false
}

// ReloadConfig 重新加载配置
func (m *Manager[T]) ReloadConfig() {
	if m.configCenter != nil {
		m.loadConfigFromCenter()
	}
}

// Close 关闭配置管理器（保持向后兼容）
// 推荐使用 Stop() 方法
func (m *Manager[T]) Close() {
	m.Stop()
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
				clog.Err(err),
				clog.String("key", key),
				clog.String("env", m.env),
				clog.String("service", m.service),
				clog.String("component", m.component))
		}
		return
	}

	// 使用原子的验证和更新方法
	if err := m.safeUpdateAndApply(&config); err != nil {
		if m.logger != nil {
			m.logger.Error("failed to apply config from center",
				clog.Err(err),
				clog.String("key", key))
		}
		return
	}

	if m.logger != nil {
		m.logger.Info("config loaded from center",
			clog.String("key", key),
			clog.String("env", m.env),
			clog.String("service", m.service),
			clog.String("component", m.component))
	}
}

// safeUpdateAndApply 原子地验证、更新和应用配置
// 这个方法确保验证和更新是原子操作，避免系统状态不一致
func (m *Manager[T]) safeUpdateAndApply(newConfig *T) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 1. 验证配置
	if m.validator != nil {
		if err := m.validator.Validate(newConfig); err != nil {
			if m.logger != nil {
				m.logger.Warn("invalid config received, update rejected", clog.Err(err))
			}
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// 2. 调用更新器（两阶段提交）
	oldConfig := m.currentConfig.Load().(*T)
	if m.updater != nil {
		if err := m.updater.OnConfigUpdate(oldConfig, newConfig); err != nil {
			if m.logger != nil {
				m.logger.Error("config updater failed, update rejected", clog.Err(err))
			}
			return fmt.Errorf("updater failed: %w", err)
		}
	}

	// 3. 原子地更新配置指针
	m.currentConfig.Store(newConfig)

	if m.logger != nil {
		m.logger.Info("config updated and applied successfully", clog.String("key", m.buildConfigKey()))
	}
	return nil
}

// safeUpdateConfig 安全地更新配置（保持向后兼容）
// 推荐使用 safeUpdateAndApply 方法
func (m *Manager[T]) safeUpdateConfig(newConfig *T) error {
	return m.safeUpdateAndApply(newConfig)
}

// buildConfigKey 构建配置键
func (m *Manager[T]) buildConfigKey() string {
	return "/config/" + m.env + "/" + m.service + "/" + m.component
}

// startWatching 启动配置监听
// 注意：此方法应该在 m.mu.Lock() 保护下调用
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
				clog.Err(err),
				clog.String("key", m.buildConfigKey()))
		}
		return
	}

	m.watcher = watcher
	m.watching = true

	// 启动监听协程
	go m.watchLoop()

	if m.logger != nil {
		m.logger.Info("config watcher started",
			clog.String("key", m.buildConfigKey()))
	}
}

// stopWatching 停止配置监听
// 注意：此方法应该在 m.mu.Lock() 保护下调用
func (m *Manager[T]) stopWatching() {
	if !m.watching {
		return
	}

	m.watching = false

	if m.watcher != nil {
		m.watcher.Close()
		m.watcher = nil
	}

	// 安全地关闭 channel，通知 watchLoop 退出
	// 使用 select + default 避免重复关闭已关闭的 channel
	select {
	case <-m.stopCh:
		// channel 已经关闭
	default:
		close(m.stopCh)
	}

	// 重新创建 stopCh 以便下次使用
	m.stopCh = make(chan struct{})
}

// watchLoop 配置监听循环
func (m *Manager[T]) watchLoop() {
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("config watch loop panic",
					clog.Any("recover", r),
					clog.String("key", m.buildConfigKey()))
			}
		}
	}()

	for {
		select {
		case event, ok := <-m.watcher.Chan():
			if !ok {
				if m.logger != nil {
					m.logger.Debug("config watcher channel closed",
						clog.String("key", m.buildConfigKey()))
				}
				return
			}

			if event.Type == EventTypePut {
				// 解析配置
				if config, err := m.parseConfig(event.Value); err == nil {
					// 使用原子的验证和更新方法
					if err := m.safeUpdateAndApply(config); err != nil {
						if m.logger != nil {
							m.logger.Error("failed to apply config from watcher",
								clog.Err(err),
								clog.String("key", m.buildConfigKey()))
						}
						continue
					}

					if m.logger != nil {
						m.logger.Info("config updated from watcher",
							clog.String("key", m.buildConfigKey()))
					}
				} else {
					if m.logger != nil {
						m.logger.Error("failed to parse config from event",
							clog.Err(err),
							clog.String("key", m.buildConfigKey()),
							clog.Any("value", event.Value))
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
// 为了保持向后兼容性，这个函数会自动启动管理器
func SimpleManager[T any](
	configCenter ConfigCenter,
	env, service, component string,
	defaultConfig T,
	logger clog.Logger,
) *Manager[T] {
	manager := NewManager(configCenter, env, service, component, defaultConfig,
		WithLogger[T](logger))
	manager.Start()
	return manager
}

// ValidatedManager 创建带验证器的配置管理器
// 为了保持向后兼容性，这个函数会自动启动管理器
func ValidatedManager[T any](
	configCenter ConfigCenter,
	env, service, component string,
	defaultConfig T,
	validator Validator[T],
	logger clog.Logger,
) *Manager[T] {
	manager := NewManager(configCenter, env, service, component, defaultConfig,
		WithValidator[T](validator),
		WithLogger[T](logger))
	manager.Start()
	return manager
}

// FullManager 创建功能完整的配置管理器
// 为了保持向后兼容性，这个函数会自动启动管理器
func FullManager[T any](
	configCenter ConfigCenter,
	env, service, component string,
	defaultConfig T,
	validator Validator[T],
	updater ConfigUpdater[T],
	logger clog.Logger,
) *Manager[T] {
	manager := NewManager(configCenter, env, service, component, defaultConfig,
		WithValidator[T](validator),
		WithUpdater[T](updater),
		WithLogger[T](logger))
	manager.Start()
	return manager
}
