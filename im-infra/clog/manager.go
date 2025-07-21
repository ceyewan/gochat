package clog

// 全局日志服务实例
var globalLoggerService LoggerService

func init() {
	globalLoggerService = NewDefaultLoggerService()
}

// registerLogger 将一个新的日志记录器注册到全局管理器中。
// 保持此函数以维持向后兼容性
func registerLogger(logger Logger) {
	if registry := getGlobalRegistry(); registry != nil {
		_ = registry.Register(logger.GetConfig().Name, logger)
		if logger.GetConfig().Name == "default" {
			registry.SetDefault(logger)
		}
	}
}

// GetLogger 获取一个已命名的日志记录器。
// 如果指定名称的记录器不存在，它将返回默认记录器。
// 如果默认记录器也不存在，它将返回 nil。
func GetLogger(name string) Logger {
	logger, exists := globalLoggerService.GetLogger(name)
	if !exists {
		// 回退到默认日志器
		if registry := getGlobalRegistry(); registry != nil {
			return registry.GetDefault()
		}
	}
	return logger
}

// SyncAll 刷新所有已注册的日志记录器中的缓冲日志条目。
func SyncAll() {
	_ = globalLoggerService.SyncAll()
}

// getGlobalRegistry 获取全局注册表（内部使用）
func getGlobalRegistry() LoggerRegistry {
	if service, ok := globalLoggerService.(*DefaultLoggerService); ok {
		return service.GetRegistry()
	}
	return nil
}

// getGlobalDefaultLogger 获取全局默认日志器（内部使用）
func getGlobalDefaultLogger() Logger {
	if registry := getGlobalRegistry(); registry != nil {
		return registry.GetDefault()
	}
	return nil
}

// SetGlobalLoggerService 设置全局日志服务（主要用于测试）
func SetGlobalLoggerService(service LoggerService) {
	globalLoggerService = service
}
