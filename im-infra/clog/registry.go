package clog

import (
	"sync"
)

// MemoryLoggerRegistry 是 LoggerRegistry 接口的内存实现
type MemoryLoggerRegistry struct {
	mu            sync.RWMutex
	loggers       map[string]Logger
	defaultLogger Logger
}

// NewMemoryLoggerRegistry 创建一个新的内存日志器注册表
func NewMemoryLoggerRegistry() LoggerRegistry {
	return &MemoryLoggerRegistry{
		loggers: make(map[string]Logger),
	}
}

// Register 注册一个日志器
func (r *MemoryLoggerRegistry) Register(name string, logger Logger) error {
	if logger == nil {
		return NewLoggerError("Register", name, ErrInvalidConfig)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.loggers[name]; exists {
		return NewLoggerError("Register", name, ErrLoggerExists)
	}

	r.loggers[name] = logger

	// 如果是默认日志器且当前没有默认日志器，则设置为默认
	if name == "default" && r.defaultLogger == nil {
		r.defaultLogger = logger
	}

	return nil
}

// Get 获取指定名称的日志器
func (r *MemoryLoggerRegistry) Get(name string) (Logger, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logger, exists := r.loggers[name]
	return logger, exists
}

// SetDefault 设置默认日志器
func (r *MemoryLoggerRegistry) SetDefault(logger Logger) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultLogger = logger
}

// GetDefault 获取默认日志器
func (r *MemoryLoggerRegistry) GetDefault() Logger {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.defaultLogger
}

// List 列出所有已注册的日志器名称
func (r *MemoryLoggerRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.loggers))
	for name := range r.loggers {
		names = append(names, name)
	}
	return names
}

// Clear 清空所有日志器
func (r *MemoryLoggerRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.loggers = make(map[string]Logger)
	r.defaultLogger = nil
}
