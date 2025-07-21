package clog

import (
	"errors"
	"fmt"
)

// 预定义的错误类型
var (
	ErrLoggerNotFound     = errors.New("logger not found")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrInitFailed         = errors.New("logger initialization failed")
	ErrLoggerExists       = errors.New("logger already exists")
	ErrInvalidLevel       = errors.New("invalid log level")
	ErrInvalidFormat      = errors.New("invalid log format")
	ErrFileRotationConfig = errors.New("invalid file rotation configuration")
)

// LoggerError 包装日志操作错误，提供更多上下文信息
type LoggerError struct {
	Op   string // 操作名称
	Name string // 日志器名称
	Err  error  // 底层错误
}

func (e *LoggerError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("logger %s[%s]: %v", e.Op, e.Name, e.Err)
	}
	return fmt.Sprintf("logger %s: %v", e.Op, e.Err)
}

func (e *LoggerError) Unwrap() error {
	return e.Err
}

// NewLoggerError 创建一个新的日志器错误
func NewLoggerError(op, name string, err error) *LoggerError {
	return &LoggerError{
		Op:   op,
		Name: name,
		Err:  err,
	}
}

// 错误检查辅助函数
func IsLoggerNotFound(err error) bool {
	return errors.Is(err, ErrLoggerNotFound)
}

func IsInvalidConfig(err error) bool {
	return errors.Is(err, ErrInvalidConfig)
}

func IsInitFailed(err error) bool {
	return errors.Is(err, ErrInitFailed)
}
