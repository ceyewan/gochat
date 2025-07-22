package slogx

import (
	"fmt"
	"log/slog"
	"strings"
)

// LevelManager wraps a slog.LevelVar to provide thread-safe level updates.
// It allows dynamic adjustment of log levels at runtime.
type LevelManager struct {
	levelVar *slog.LevelVar
}

// NewLevelManager creates a new LevelManager with the specified initial level.
// The level parameter should be one of: "debug", "info", "warn", "error".
func NewLevelManager(level string) (*LevelManager, error) {
	slogLevel, err := ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid level %q: %w", level, err)
	}

	lv := &slog.LevelVar{}
	lv.Set(slogLevel)

	return &LevelManager{levelVar: lv}, nil
}

// SetLevel atomically updates the log level.
// The level parameter should be one of: "debug", "info", "warn", "error".
func (lm *LevelManager) SetLevel(level string) error {
	slogLevel, err := ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid level %q: %w", level, err)
	}

	lm.levelVar.Set(slogLevel)
	return nil
}

// Leveler returns the underlying slog.Leveler for use in Handler creation.
func (lm *LevelManager) Leveler() slog.Leveler {
	return lm.levelVar
}

// Level returns the current log level as a string.
func (lm *LevelManager) Level() string {
	return LevelToString(lm.levelVar.Level())
}

// ParseLevel converts a string level to slog.Level.
// Supported levels: "debug", "info", "warn", "error" (case-insensitive).
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

// LevelToString converts a slog.Level to its string representation.
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
		// For custom levels, use the numeric representation
		return fmt.Sprintf("level(%d)", int(level))
	}
}
