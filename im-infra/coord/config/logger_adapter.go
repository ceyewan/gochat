package config

// LoggerAdapter 适配器，将具体的 Logger 实现适配为 config.Logger 接口
// 这个适配器可以被所有需要适配日志器的模块使用，避免代码重复
type LoggerAdapter struct {
	logger interface {
		Debug(msg string, fields ...any)
		Info(msg string, fields ...any)
		Warn(msg string, fields ...any)
		Error(msg string, fields ...any)
	}
}

// NewLoggerAdapter 创建新的日志适配器
func NewLoggerAdapter(logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
}) *LoggerAdapter {
	return &LoggerAdapter{logger: logger}
}

func (a *LoggerAdapter) Debug(msg string, fields ...any) {
	a.logger.Debug(msg, fields...)
}

func (a *LoggerAdapter) Info(msg string, fields ...any) {
	a.logger.Info(msg, fields...)
}

func (a *LoggerAdapter) Warn(msg string, fields ...any) {
	a.logger.Warn(msg, fields...)
}

func (a *LoggerAdapter) Error(msg string, fields ...any) {
	a.logger.Error(msg, fields...)
}

// ClogLoggerAdapter 专门用于适配 clog.Logger 的适配器
// 这个适配器处理 clog 特有的字段转换逻辑
type ClogLoggerAdapter struct {
	logger interface {
		Debug(msg string, fields ...interface{}) // clog 使用 interface{} 而不是 any
		Info(msg string, fields ...interface{})
		Warn(msg string, fields ...interface{})
		Error(msg string, fields ...interface{})
	}
	fieldConverter func(fields ...any) []interface{}
}

// NewClogLoggerAdapter 创建新的 clog 日志适配器
func NewClogLoggerAdapter(
	logger interface {
		Debug(msg string, fields ...interface{})
		Info(msg string, fields ...interface{})
		Warn(msg string, fields ...interface{})
		Error(msg string, fields ...interface{})
	},
	fieldConverter func(fields ...any) []interface{},
) *ClogLoggerAdapter {
	return &ClogLoggerAdapter{
		logger:         logger,
		fieldConverter: fieldConverter,
	}
}

func (a *ClogLoggerAdapter) Debug(msg string, fields ...any) {
	if a.fieldConverter != nil {
		a.logger.Debug(msg, a.fieldConverter(fields...)...)
	} else {
		// 简单转换
		convertedFields := make([]interface{}, len(fields))
		for i, field := range fields {
			convertedFields[i] = field
		}
		a.logger.Debug(msg, convertedFields...)
	}
}

func (a *ClogLoggerAdapter) Info(msg string, fields ...any) {
	if a.fieldConverter != nil {
		a.logger.Info(msg, a.fieldConverter(fields...)...)
	} else {
		convertedFields := make([]interface{}, len(fields))
		for i, field := range fields {
			convertedFields[i] = field
		}
		a.logger.Info(msg, convertedFields...)
	}
}

func (a *ClogLoggerAdapter) Warn(msg string, fields ...any) {
	if a.fieldConverter != nil {
		a.logger.Warn(msg, a.fieldConverter(fields...)...)
	} else {
		convertedFields := make([]interface{}, len(fields))
		for i, field := range fields {
			convertedFields[i] = field
		}
		a.logger.Warn(msg, convertedFields...)
	}
}

func (a *ClogLoggerAdapter) Error(msg string, fields ...any) {
	if a.fieldConverter != nil {
		a.logger.Error(msg, a.fieldConverter(fields...)...)
	} else {
		convertedFields := make([]interface{}, len(fields))
		for i, field := range fields {
			convertedFields[i] = field
		}
		a.logger.Error(msg, convertedFields...)
	}
}
