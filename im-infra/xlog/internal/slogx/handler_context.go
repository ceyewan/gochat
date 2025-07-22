package slogx

import (
	"context"
	"log/slog"
)

// ContextHandler is a middleware slog.Handler that extracts values from context.Context
// and adds them as attributes to log records.
// It wraps another handler and delegates all operations to it after processing the context.
type ContextHandler struct {
	next       slog.Handler
	traceIDKey any
}

// NewContextHandler creates a new ContextHandler that extracts TraceID from context
// using the specified key and adds it to log records.
// The traceIDKey parameter specifies the context key to look for the TraceID.
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

// Enabled reports whether the handler handles records at the given level.
// It delegates to the next handler.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle processes the record by first extracting TraceID from context (if present)
// and adding it as an attribute, then delegating to the next handler.
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

// WithAttrs returns a new ContextHandler with the next handler updated with the given attributes.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		next:       h.next.WithAttrs(attrs),
		traceIDKey: h.traceIDKey,
	}
}

// WithGroup returns a new ContextHandler with the next handler updated with the given group.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		next:       h.next.WithGroup(name),
		traceIDKey: h.traceIDKey,
	}
}

// extractTraceID attempts to extract a TraceID from the context.
// It returns an empty string if no TraceID is found or if the value is not a string.
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

// Next returns the next handler in the chain.
// This is useful for testing and debugging.
func (h *ContextHandler) Next() slog.Handler {
	return h.next
}

// TraceIDKey returns the context key used for TraceID extraction.
// This is useful for testing and debugging.
func (h *ContextHandler) TraceIDKey() any {
	return h.traceIDKey
}
