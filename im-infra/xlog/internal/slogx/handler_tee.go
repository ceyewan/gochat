package slogx

import (
	"context"
	"log/slog"
)

// TeeHandler is a slog.Handler that distributes log records to multiple downstream handlers.
// It implements the fan-out pattern, allowing a single log record to be processed by
// multiple handlers simultaneously (e.g., one for JSON output, another for text output).
type TeeHandler struct {
	handlers []slog.Handler
}

// NewTeeHandler creates a new TeeHandler that distributes records to the given handlers.
// If no handlers are provided, it returns a no-op handler.
func NewTeeHandler(handlers ...slog.Handler) *TeeHandler {
	// Filter out nil handlers
	validHandlers := make([]slog.Handler, 0, len(handlers))
	for _, h := range handlers {
		if h != nil {
			validHandlers = append(validHandlers, h)
		}
	}

	return &TeeHandler{
		handlers: validHandlers,
	}
}

// Enabled reports whether the handler handles records at the given level.
// The TeeHandler is enabled if at least one of its downstream handlers is enabled.
func (h *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle processes the record by distributing it to all downstream handlers.
// If any handler returns an error, Handle returns that error immediately.
// This means that all handlers before the failing one will have processed the record,
// but handlers after the failing one will not.
func (h *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WithAttrs returns a new TeeHandler whose downstream handlers have been
// updated with the given attributes.
func (h *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}

	return &TeeHandler{
		handlers: newHandlers,
	}
}

// WithGroup returns a new TeeHandler whose downstream handlers have been
// updated with the given group name.
func (h *TeeHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}

	return &TeeHandler{
		handlers: newHandlers,
	}
}

// Handlers returns a copy of the downstream handlers.
// This is useful for testing and debugging.
func (h *TeeHandler) Handlers() []slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	copy(handlers, h.handlers)
	return handlers
}
