package db

import (
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/config"
	"github.com/ceyewan/gochat/im-infra/db/internal"
)

// configValidator 实现 db 配置验证器
type configValidator struct{}

// Validate 验证 db 配置的有效性
func (v *configValidator) Validate(cfg *Config) error {
	return cfg.Validate()
}

// dbLoggerAdapter 专门用于适配 clog.Logger 的适配器
type dbLoggerAdapter struct {
	logger clog.Logger
}

func (a *dbLoggerAdapter) Debug(msg string, fields ...any) {
	a.logger.Debug(msg, convertFields(fields...)...)
}

func (a *dbLoggerAdapter) Info(msg string, fields ...any) {
	a.logger.Info(msg, convertFields(fields...)...)
}

func (a *dbLoggerAdapter) Warn(msg string, fields ...any) {
	a.logger.Warn(msg, convertFields(fields...)...)
}

func (a *dbLoggerAdapter) Error(msg string, fields ...any) {
	a.logger.Error(msg, convertFields(fields...)...)
}

// convertFields 将 any 类型的字段转换为 clog 字段
func convertFields(fields ...any) []clog.Field {
	if len(fields) == 0 {
		return nil
	}

	result := make([]clog.Field, 0, len(fields)/2)
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			value := fields[i+1]
			result = append(result, clog.Any(key, value))
		}
	}
	return result
}

// newConfigManager 创建新的配置管理器（使用通用实现）
func newConfigManager(
	configCenter config.ConfigCenter,
	env, service, component string,
	defaultConfig Config,
) *config.Manager[Config] {
	validator := &configValidator{}
	logger := &dbLoggerAdapter{logger: clog.Module("db.config")}

	return config.ValidatedManager(
		configCenter,
		env, service, component,
		defaultConfig,
		validator,
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
	defaultConfig := internal.DefaultConfig()
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

// SetupConfigCenter 设置配置中心 - 简化的API
// 这是推荐的配置中心集成方式
// 注意：这个函数使用全局状态，推荐使用 NewConfigManager 进行依赖注入
func SetupConfigCenter(configCenter config.ConfigCenter, env, service, component string) {
	// 替换全局配置管理器为通用实现
	defaultConfig := internal.DefaultConfig()
	globalConfigManager = newConfigManager(configCenter, env, service, component, defaultConfig)
}

// SetupConfigCenterFromCoord 设置配置中心
// 这是一个便利函数，用于快速设置配置中心作为 db 的配置源
// 注意：这个函数使用全局状态，推荐使用 NewConfigManager 进行依赖注入
func SetupConfigCenterFromCoord(configCenter config.ConfigCenter, env, service, component string) {
	SetupConfigCenter(configCenter, env, service, component)
}

// 全局配置管理器（使用通用实现）
var globalConfigManager *config.Manager[Config]

func init() {
	// 初始化全局配置管理器（无配置中心）
	defaultConfig := internal.DefaultConfig()
	globalConfigManager = newConfigManager(nil, "dev", "im-infra", "db", defaultConfig)
}

// GetCurrentConfig 获取全局当前配置
func GetCurrentConfig() *Config {
	return globalConfigManager.GetCurrentConfig()
}

// ReloadConfig 重新加载配置
func ReloadConfig() {
	globalConfigManager.ReloadConfig()
}

// getConfigFromManager 从配置管理器获取配置
// 这是内部函数，用于 getDefaultDB 函数
func getConfigFromManager() *Config {
	return globalConfigManager.GetCurrentConfig()
}

// NewConfigManagerLocal 创建本地配置管理器（无配置中心）
// 这个函数用于向后兼容，推荐使用 NewConfigManager
func NewConfigManagerLocal(env, service, component string) *config.Manager[Config] {
	defaultConfig := internal.DefaultConfig()
	return newConfigManager(nil, env, service, component, defaultConfig)
}
