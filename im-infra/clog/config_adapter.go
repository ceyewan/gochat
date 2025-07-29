package clog

import (
	"fmt"
	"strings"

	"github.com/ceyewan/gochat/im-infra/coord/config"
)

// configValidator 实现 clog 配置验证器
type configValidator struct{}

// Validate 验证 clog 配置的有效性
func (v *configValidator) Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// 验证日志级别
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[strings.ToLower(cfg.Level)] {
		return fmt.Errorf("invalid log level: %s", cfg.Level)
	}

	// 验证格式
	validFormats := map[string]bool{
		"json": true, "console": true,
	}
	if !validFormats[cfg.Format] {
		return fmt.Errorf("invalid log format: %s", cfg.Format)
	}

	// 验证输出
	if cfg.Output == "" {
		return fmt.Errorf("output cannot be empty")
	}

	return nil
}

// configUpdater 实现 clog 配置更新器
type configUpdater struct{}

// OnConfigUpdate 在配置更新时执行自定义逻辑
func (u *configUpdater) OnConfigUpdate(oldConfig, newConfig *Config) error {
	// 尝试创建新的 logger 来验证配置是否可用
	testLogger, err := New(*newConfig)
	if err != nil {
		return fmt.Errorf("failed to create test logger with new config: %w", err)
	}

	// 测试 logger 可以正常工作
	testLogger.Debug("config validation test")

	// 尝试更新全局 logger
	if err := Init(*newConfig); err != nil {
		return fmt.Errorf("failed to update global logger: %w", err)
	}

	return nil
}

// convertFields 将 any 类型的字段转换为 clog 字段
func convertFields(fields ...any) []Field {
	if len(fields) == 0 {
		return nil
	}

	result := make([]Field, 0, len(fields)/2)
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			value := fields[i+1]
			result = append(result, Any(key, value))
		}
	}
	return result
}

// clogLoggerAdapter 专门用于适配 clog.Logger 的适配器
type clogLoggerAdapter struct {
	logger Logger
}

func (a *clogLoggerAdapter) Debug(msg string, fields ...any) {
	a.logger.Debug(msg, convertFields(fields...)...)
}

func (a *clogLoggerAdapter) Info(msg string, fields ...any) {
	a.logger.Info(msg, convertFields(fields...)...)
}

func (a *clogLoggerAdapter) Warn(msg string, fields ...any) {
	a.logger.Warn(msg, convertFields(fields...)...)
}

func (a *clogLoggerAdapter) Error(msg string, fields ...any) {
	a.logger.Error(msg, convertFields(fields...)...)
}

// newConfigManager 创建新的配置管理器（使用通用实现）
func newConfigManager(
	configCenter config.ConfigCenter,
	env, service, component string,
	defaultConfig Config,
) *config.Manager[Config] {
	validator := &configValidator{}
	updater := &configUpdater{}

	// 使用专门的 clog 日志适配器
	clogLogger := Module("clog.config")
	logger := &clogLoggerAdapter{logger: clogLogger}

	return config.FullManager(
		configCenter,
		env, service, component,
		defaultConfig,
		validator,
		updater,
		logger,
	)
}

// ===== 新的依赖注入 API =====

// NewConfigManager 创建新的配置管理器实例（推荐使用）
// 这个函数返回一个未启动的配置管理器，需要手动调用 Start() 方法
func NewConfigManager(
	configCenter config.ConfigCenter,
	env, service, component string,
) *config.Manager[Config] {
	defaultConfig := DefaultConfig()
	return newConfigManager(configCenter, env, service, component, defaultConfig)
}

// NewConfigManagerWithDefaults 创建带自定义默认配置的配置管理器
func NewConfigManagerWithDefaults(
	configCenter config.ConfigCenter,
	env, service, component string,
	defaultConfig Config,
) *config.Manager[Config] {
	return newConfigManager(configCenter, env, service, component, defaultConfig)
}

// ===== 向后兼容的全局 API =====

// SetupConfigCenterFromCoord 设置配置中心
// 这是一个便利函数，用于快速设置配置中心作为 clog 的配置源
// 注意：这个函数使用全局状态，推荐使用 NewConfigManager 进行依赖注入
func SetupConfigCenterFromCoord(configCenter config.ConfigCenter, env, service, component string) {
	// 替换全局配置管理器为通用实现
	defaultConfig := DefaultConfig()
	globalConfigManager = newConfigManager(configCenter, env, service, component, defaultConfig)
}

// 全局配置管理器（使用通用实现）
var globalConfigManager *config.Manager[Config]

func init() {
	// 初始化全局配置管理器（无配置中心）
	defaultConfig := DefaultConfig()
	globalConfigManager = newConfigManager(nil, "dev", "im-infra", "clog", defaultConfig)
}

// GetCurrentConfig 获取全局当前配置
func GetCurrentConfig() *Config {
	return globalConfigManager.GetCurrentConfig()
}

// ReloadConfig 重新加载配置（用于两阶段初始化的第二阶段）
func ReloadConfig() {
	globalConfigManager.ReloadConfig()
}

// CloseConfigManager 关闭全局配置管理器
func CloseConfigManager() {
	globalConfigManager.Close()
}
