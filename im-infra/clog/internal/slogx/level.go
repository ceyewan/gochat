package slogx

import (
	"fmt"
	"log/slog"
	"strings"
)

// LevelManager 封装 slog.LevelVar，实现线程安全的日志级别动态调整。
// 支持运行时动态修改日志级别。
type LevelManager struct {
	levelVar *slog.LevelVar
}

// NewLevelManager 创建一个指定初始级别的 LevelManager。
// level 参数支持："debug"、"info"、"warn"、"error"。
func NewLevelManager(level string) (*LevelManager, error) {
	slogLevel, err := ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid level %q: %w", level, err)
	}

	lv := &slog.LevelVar{}
	lv.Set(slogLevel)

	return &LevelManager{levelVar: lv}, nil
}

// SetLevel 原子性地更新日志级别。
// level 参数支持："debug"、"info"、"warn"、"error"。
func (lm *LevelManager) SetLevel(level string) error {
	slogLevel, err := ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid level %q: %w", level, err)
	}

	lm.levelVar.Set(slogLevel)
	return nil
}

// Leveler 返回底层 slog.Leveler，可用于 Handler 创建。
func (lm *LevelManager) Leveler() slog.Leveler {
	return lm.levelVar
}

// Level 返回当前日志级别的字符串表示。
func (lm *LevelManager) Level() string {
	return LevelToString(lm.levelVar.Level())
}

// ParseLevel 将字符串日志级别转换为 slog.Level。
// 支持："debug"、"info"、"warn"、"error"（不区分大小写）。
func ParseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported level: %s", level)
	}
}

// LevelToString 将 slog.Level 转换为字符串表示。
func LevelToString(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "debug"
	case slog.LevelInfo:
		return "info"
	case slog.LevelWarn:
		return "warn"
	case slog.LevelError:
		return "error"
	default:
		// 对于自定义级别，使用数字表示
		return fmt.Sprintf("level(%d)", int(level))
	}
}
