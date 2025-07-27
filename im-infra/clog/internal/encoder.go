package internal

import (
	"time"

	"go.uber.org/zap/zapcore"
)

// buildEncoderConfig 根据格式创建编码器配置
func buildEncoderConfig(format string, enableColor bool) zapcore.EncoderConfig {
	config := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Console 格式特殊处理
	if format == "console" {
		if enableColor {
			config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			config.EncodeLevel = zapcore.CapitalLevelEncoder
		}
	}

	return config
}

// customTimeEncoder 自定义时间编码格式
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// createEncoder 根据格式创建编码器
func createEncoder(format string, config zapcore.EncoderConfig) zapcore.Encoder {
	switch format {
	case "json":
		return zapcore.NewJSONEncoder(config)
	case "console":
		return zapcore.NewConsoleEncoder(config)
	default:
		return zapcore.NewJSONEncoder(config)
	}
}
