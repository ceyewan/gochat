package slogx

import (
	"context"
	"log/slog"
)

// TeeHandler 是一个 slog.Handler，可以将日志记录分发到多个下游 handler。
// 实现了扇出模式，允许单条日志同时被多个 handler 处理（如同时输出 JSON 和文本）。
type TeeHandler struct {
	handlers []slog.Handler
}

// NewTeeHandler 创建一个 TeeHandler，将日志分发到指定的 handlers。
// 如果没有提供 handler，则返回一个空操作 handler。
func NewTeeHandler(handlers ...slog.Handler) *TeeHandler {
	// 过滤掉 nil 的 handler
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

// Enabled 判断是否有下游 handler 能处理指定级别的日志。
// 只要有一个下游 handler 启用，则 TeeHandler 启用。
func (h *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle 将日志记录分发给所有下游 handler。
// 如果有 handler 返回错误，则立即返回该错误。
// 这意味着错误之前的 handler 已处理日志，之后的 handler 不再处理。
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

// WithAttrs 返回一个新的 TeeHandler，其下游 handler 都带有指定属性。
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

// WithGroup 返回一个新的 TeeHandler，其下游 handler 都带有指定分组名。
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

// Handlers 返回下游 handlers 的副本。
// 便于测试和调试。
func (h *TeeHandler) Handlers() []slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	copy(handlers, h.handlers)
	return handlers
}
