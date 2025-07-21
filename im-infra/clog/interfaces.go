package clog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerInterface 定义日志记录器的抽象接口
type LoggerInterface interface {
	// 结构化日志方法
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	// 格式化日志方法
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})

	// 管理方法
	SetLevel(level string) error
	Sync() error
	Close() error

	// 获取配置信息
	GetConfig() *Config
}

// LoggerRegistry 定义日志器注册表的抽象接口
type LoggerRegistry interface {
	Register(name string, logger LoggerInterface) error
	Get(name string) (LoggerInterface, bool)
	SetDefault(logger LoggerInterface)
	GetDefault() LoggerInterface
	List() []string
	Clear()
}

// LoggerBuilder 定义日志器构建器的抽象接口
type LoggerBuilder interface {
	WithConfig(cfg *Config) LoggerBuilder
	WithCores(cores ...zapcore.Core) LoggerBuilder
	WithOptions(opts ...zap.Option) LoggerBuilder
	Build() (LoggerInterface, error)
}

// ConfigMerger 定义配置合并器的抽象接口
type ConfigMerger interface {
	Merge(base, override *Config) *Config
	MergeWithDefault(opts ...Option) *Config
}

// LoggerService 定义日志服务的抽象接口
type LoggerService interface {
	Init(opts ...Option) error
	GetOrCreateModule(name string, opts ...Option) (LoggerInterface, error)
	GetLogger(name string) (LoggerInterface, bool)
	SetDefaultLevel(level string) error
	SyncAll() error
}
