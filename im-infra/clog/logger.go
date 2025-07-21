package clog

import (
	"go.uber.org/zap"
)

// ZapLogger 实现 LoggerInterface 接口，封装了 zap 日志功能
type ZapLogger struct {
	zap         *zap.Logger        // 底层zap日志器
	sugar       *zap.SugaredLogger // 语法糖日志器
	config      *Config            // 日志配置
	atomicLevel zap.AtomicLevel    // 原子级别控制
}

// 确保 ZapLogger 实现了 LoggerInterface 接口
var _ LoggerInterface = (*ZapLogger)(nil)

// Field 代表一个日志字段，是 zap.Field 的别名
type Field = zap.Field

// 提供常用字段类型的创建函数，方便使用
var (
	String   = zap.String
	Uint64   = zap.Uint64
	Int      = zap.Int
	Int64    = zap.Int64
	Float64  = zap.Float64
	Bool     = zap.Bool
	Any      = zap.Any
	Err      = zap.Error
	Time     = zap.Time
	Duration = zap.Duration
)

// Logger 是 LoggerInterface 的类型别名，用于向后兼容
type Logger = LoggerInterface

// isValidLevel 检查日志级别是否有效
func isValidLevel(level string) bool {
	validLevels := []string{DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// SetLevel 动态更改日志级别
func (l *ZapLogger) SetLevel(level string) error {
	if !isValidLevel(level) {
		return NewLoggerError("SetLevel", l.config.Name, ErrInvalidLevel)
	}
	l.atomicLevel.SetLevel(parseLevel(level))
	return nil
}

// Debug 在 debug 级别记录消息
func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.zap.Debug(msg, fields...)
}

// Info 在 info 级别记录消息
func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.zap.Info(msg, fields...)
}

// Warn 在 warn 级别记录消息
func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.zap.Warn(msg, fields...)
}

// Error 在 error 级别记录消息
func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.zap.Error(msg, fields...)
}

// Fatal 在 fatal 级别记录消息然后调用 os.Exit(1)
func (l *ZapLogger) Fatal(msg string, fields ...Field) {
	l.zap.Fatal(msg, fields...)
}

// Debugf 记录格式化的 debug 级别消息
func (l *ZapLogger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

// Infof 记录格式化的 info 级别消息
func (l *ZapLogger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

// Warnf 记录格式化的 warn 级别消息
func (l *ZapLogger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

// Errorf 记录格式化的 error 级别消息
func (l *ZapLogger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

// Fatalf 记录格式化的 fatal 级别消息然后调用 os.Exit(1)
func (l *ZapLogger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}

// Sync 刷新任何缓冲的日志条目
func (l *ZapLogger) Sync() error {
	if l.zap != nil {
		return l.zap.Sync()
	}
	return nil
}

// Close 正确关闭日志器
func (l *ZapLogger) Close() error {
	return l.Sync()
}

// GetConfig 获取日志器配置
func (l *ZapLogger) GetConfig() *Config {
	return l.config
}
