package slogx

import (
	"context"
	"log/slog"
)

// ContextHandler 是一个中间件 slog.Handler，能从 context.Context 提取值并作为属性添加到日志记录。
// 它包装另一个 handler，处理 context 后将所有操作委托给下一个 handler。
type ContextHandler struct {
	next       slog.Handler
	traceIDKey any
}

// NewContextHandler 创建一个 ContextHandler，从 context 提取 TraceID 并添加到日志记录。
// traceIDKey 参数指定 context 中查找 TraceID 的 key。
func NewContextHandler(next slog.Handler, traceIDKey any) *ContextHandler {
	if next == nil {
		panic("ContextHandler: next handler cannot be nil")
	}

	if traceIDKey == nil {
		traceIDKey = "traceID" // Default key
	}

	return &ContextHandler{
		next:       next,
		traceIDKey: traceIDKey,
	}
}

// Enabled 判断 handler 是否处理指定级别的日志。
// 委托给下一个 handler。
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle 处理日志记录，优先从 context 提取 TraceID 并作为属性添加，然后委托给下一个 handler。
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Extract TraceID from context if present
	if traceID := h.extractTraceID(ctx); traceID != "" {
		// Create a new record with the TraceID attribute
		newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

		// Copy all existing attributes
		r.Attrs(func(a slog.Attr) bool {
			newRecord.AddAttrs(a)
			return true
		})

		// Add the TraceID attribute
		newRecord.AddAttrs(slog.String("trace_id", traceID))

		return h.next.Handle(ctx, newRecord)
	}

	// No TraceID found, delegate directly
	return h.next.Handle(ctx, r)
}

// WithAttrs 返回一个新的 ContextHandler，其下一个 handler 带有指定属性。
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		next:       h.next.WithAttrs(attrs),
		traceIDKey: h.traceIDKey,
	}
}

// WithGroup 返回一个新的 ContextHandler，其下一个 handler 带有指定分组名。
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		next:       h.next.WithGroup(name),
		traceIDKey: h.traceIDKey,
	}
}

// extractTraceID 尝试从 context 提取 TraceID。
// 未找到或类型不是 string 时返回空字符串。
func (h *ContextHandler) extractTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	value := ctx.Value(h.traceIDKey)
	if value == nil {
		return ""
	}

	// Try to convert to string
	if traceID, ok := value.(string); ok {
		return traceID
	}

	// If it's not a string, we could try other conversions,
	// but for now we'll just return empty string
	return ""
}

// Next 返回链中的下一个 handler。
// 便于测试和调试。
func (h *ContextHandler) Next() slog.Handler {
	return h.next
}

// TraceIDKey 返回用于 TraceID 提取的 context key。
// 便于测试和调试。
func (h *ContextHandler) TraceIDKey() any {
	return h.traceIDKey
}
