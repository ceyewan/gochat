// Package clog 提供一个灵活、可扩展的日志系统，基于 uber-go/zap。
package clog

// Init 初始化默认的全局日志记录器。
//
// 示例:
//
//	err := clog.Init(
//	    clog.WithLevel("debug"),
//	    clog.WithFilename("logs/my-app.log"),
//	)
func Init(opts ...Option) error {
	// 确保 name 选项存在，如果没有则设置为 "default"
	var nameSet bool
	for _, opt := range opts {
		tempCfg := &Config{}
		opt(tempCfg)
		if tempCfg.Name != "" {
			nameSet = true
			break
		}
	}
	if !nameSet {
		opts = append(opts, WithName("default"))
	}

	return globalLoggerService.Init(opts...)
}

// Module 创建或获取一个模块专用的日志记录器。
// 模块日志记录器继承默认记录器的设置，但可以有自己的名称和字段。
//
// 示例:
//
//	dbLogger := clog.Module("database")
//	dbLogger.Info("连接数据库成功")
func Module(moduleName string, opts ...Option) Logger {
	logger, err := globalLoggerService.GetOrCreateModule(moduleName, opts...)
	if err != nil {
		// 如果创建失败，返回默认日志器或 nil
		if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
			return defaultLogger
		}
		// 如果连默认日志器都没有，返回 nil（调用者需要检查）
		return nil
	}
	return logger
}

// SetDefaultLevel 设置默认日志器的级别。
func SetDefaultLevel(level string) error {
	return globalLoggerService.SetDefaultLevel(level)
}

// Debug 使用默认日志器记录 debug 级别消息。
func Debug(msg string, fields ...Field) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Debug(msg, fields...)
	}
}

// Info 使用默认日志器记录 info 级别消息。
func Info(msg string, fields ...Field) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Info(msg, fields...)
	}
}

// Warn 使用默认日志器记录 warn 级别消息。
func Warn(msg string, fields ...Field) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Warn(msg, fields...)
	}
}

// Error 使用默认日志器记录 error 级别消息。
func Error(msg string, fields ...Field) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Error(msg, fields...)
	}
}

// Fatal 使用默认日志器记录 fatal 级别消息然后退出。
func Fatal(msg string, fields ...Field) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Fatal(msg, fields...)
	}
}

// Debugf 使用默认日志器记录格式化的 debug 级别消息。
func Debugf(format string, args ...interface{}) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Debugf(format, args...)
	}
}

// Infof 使用默认日志器记录格式化的 info 级别消息。
func Infof(format string, args ...interface{}) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Infof(format, args...)
	}
}

// Warnf 使用默认日志器记录格式化的 warn 级别消息。
func Warnf(format string, args ...interface{}) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Warnf(format, args...)
	}
}

// Errorf 使用默认日志器记录格式化的 error 级别消息。
func Errorf(format string, args ...interface{}) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Errorf(format, args...)
	}
}

// Fatalf 使用默认日志器记录格式化的 fatal 级别消息然后退出。
func Fatalf(format string, args ...interface{}) {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		defaultLogger.Fatalf(format, args...)
	}
}

// Sync 刷新默认日志器中任何缓冲的日志条目。
func Sync() error {
	if defaultLogger := getGlobalDefaultLogger(); defaultLogger != nil {
		return defaultLogger.Sync()
	}
	return nil
}
