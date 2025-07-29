package clog

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ConfigManager 配置管理器 - 简洁优雅的配置中心集成方案
//
// 设计原则：
// 1. 解决循环依赖：通过接口抽象避免直接依赖 coord
// 2. 两阶段初始化：支持降级启动和配置重载
// 3. 动态配置：支持配置热更新
// 4. 简洁API：提供简单易用的配置管理接口
type ConfigManager struct {
	// 配置源（可选，用于从配置中心获取配置）
	source ConfigSource

	// 配置参数
	env       string
	service   string
	component string

	// 当前配置（原子操作）
	currentConfig atomic.Value // *Config

	// 配置监听器
	watcher ConfigWatcher

	// 控制
	mu       sync.RWMutex
	stopCh   chan struct{}
	watching bool
}

// ConfigSource 配置源接口 - 抽象配置获取逻辑
type ConfigSource interface {
	// GetConfig 获取配置
	GetConfig(ctx context.Context, env, service, component string) (*Config, error)

	// Watch 监听配置变化（可选实现）
	Watch(ctx context.Context, env, service, component string) (ConfigWatcher, error)
}

// ConfigWatcher 配置监听器接口
type ConfigWatcher interface {
	Chan() <-chan *Config
	Close()
}

// 全局配置管理器
var globalConfigManager = &ConfigManager{
	env:       "dev",      // 默认环境
	service:   "im-infra", // 默认服务
	component: "clog",     // 组件名
	stopCh:    make(chan struct{}),
}

func init() {
	// 设置默认配置
	defaultConfig := DefaultConfig()
	globalConfigManager.currentConfig.Store(&defaultConfig)
}

// NewConfigManager 创建配置管理器
func NewConfigManager(env, service, component string) *ConfigManager {
	cm := &ConfigManager{
		env:       env,
		service:   service,
		component: component,
		stopCh:    make(chan struct{}),
	}
	// 设置默认配置
	defaultConfig := DefaultConfig()
	cm.currentConfig.Store(&defaultConfig)
	return cm
}

// SetupConfigCenter 设置配置中心 - 简化的API
// 这是推荐的配置中心集成方式
func SetupConfigCenter(source ConfigSource, env, service, component string) {
	globalConfigManager.Setup(source, env, service, component)
}

// Setup 设置配置管理器
func (cm *ConfigManager) Setup(source ConfigSource, env, service, component string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 停止之前的监听
	cm.stopWatching()

	// 更新配置参数
	cm.env = env
	cm.service = service
	cm.component = component
	cm.source = source

	// 如果有配置源，尝试获取配置并启动监听
	if source != nil {
		cm.loadConfigFromSource()
		cm.startWatching()
	}
}

// GetCurrentConfig 获取当前配置
func (cm *ConfigManager) GetCurrentConfig() *Config {
	if config := cm.currentConfig.Load(); config != nil {
		return config.(*Config)
	}
	defaultConfig := DefaultConfig()
	return &defaultConfig
}

// GetCurrentConfig 获取全局当前配置
func GetCurrentConfig() *Config {
	return globalConfigManager.GetCurrentConfig()
}

// loadConfigFromSource 从配置源加载配置
func (cm *ConfigManager) loadConfigFromSource() {
	if cm.source == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := cm.source.GetConfig(ctx, cm.env, cm.service, cm.component)
	if err != nil {
		// 记录错误但不阻断，继续使用当前配置
		logger := Module("clog.config")
		logger.Warn("failed to load config from source, using current config",
			Err(err),
			String("env", cm.env),
			String("service", cm.service),
			String("component", cm.component))
		return
	}

	// 更新配置
	cm.currentConfig.Store(config)

	logger := Module("clog.config")
	logger.Info("config loaded from source",
		String("env", cm.env),
		String("service", cm.service),
		String("component", cm.component),
		String("level", config.Level),
		String("format", config.Format))
}

// startWatching 启动配置监听
func (cm *ConfigManager) startWatching() {
	if cm.source == nil || cm.watching {
		return
	}

	ctx := context.Background()
	watcher, err := cm.source.Watch(ctx, cm.env, cm.service, cm.component)
	if err != nil {
		logger := Module("clog.config")
		logger.Warn("failed to start config watcher",
			Err(err),
			String("env", cm.env),
			String("service", cm.service),
			String("component", cm.component))
		return
	}

	cm.watcher = watcher
	cm.watching = true

	// 启动监听协程
	go cm.watchLoop()

	logger := Module("clog.config")
	logger.Info("config watcher started",
		String("env", cm.env),
		String("service", cm.service),
		String("component", cm.component))
}

// stopWatching 停止配置监听
func (cm *ConfigManager) stopWatching() {
	if !cm.watching {
		return
	}

	cm.watching = false

	if cm.watcher != nil {
		cm.watcher.Close()
		cm.watcher = nil
	}

	// 通知停止
	select {
	case cm.stopCh <- struct{}{}:
	default:
	}
}

// watchLoop 配置监听循环
func (cm *ConfigManager) watchLoop() {
	defer func() {
		if r := recover(); r != nil {
			logger := Module("clog.config")
			logger.Error("config watch loop panic", Any("recover", r))
		}
	}()

	for {
		select {
		case config := <-cm.watcher.Chan():
			if config != nil {
				// 更新配置
				cm.currentConfig.Store(config)

				// 触发全局 logger 重新初始化
				cm.triggerGlobalLoggerUpdate(config)

				logger := Module("clog.config")
				logger.Info("config updated from watcher",
					String("level", config.Level),
					String("format", config.Format))
			}
		case <-cm.stopCh:
			return
		}
	}
}

// triggerGlobalLoggerUpdate 触发全局 logger 更新
func (cm *ConfigManager) triggerGlobalLoggerUpdate(config *Config) {
	// 重新初始化全局 logger
	err := Init(*config)
	if err != nil {
		logger := Module("clog.config")
		logger.Error("failed to update global logger", Err(err))
	}
}

// getConfigFromManager 从配置管理器获取配置
// 这是内部函数，用于 Init 和 New 函数
func getConfigFromManager() *Config {
	return globalConfigManager.GetCurrentConfig()
}

// ReloadConfig 重新加载配置（用于两阶段初始化的第二阶段）
func ReloadConfig() {
	globalConfigManager.mu.RLock()
	defer globalConfigManager.mu.RUnlock()

	if globalConfigManager.source != nil {
		globalConfigManager.loadConfigFromSource()
	}
}

// Close 关闭配置管理器
func (cm *ConfigManager) Close() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stopWatching()
}

// Close 关闭全局配置管理器
func CloseConfigManager() {
	globalConfigManager.Close()
}
