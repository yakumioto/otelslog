package otelslog

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type Options func(*Handler)

func WithTraceIDKey(key string) Options {
	return func(h *Handler) {
		h.TraceIDKey = key
	}
}

func WithSpanIDKey(key string) Options {
	return func(h *Handler) {
		h.SpanIDKey = key
	}
}

type Handler struct {
	TraceIDKey string
	SpanIDKey  string
	Next       slog.Handler
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Next.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanContextFromContext(ctx)
	if span.HasTraceID() {
		record.AddAttrs(slog.String(h.TraceIDKey, span.SpanID().String()))
	}
	if span.HasSpanID() {
		record.AddAttrs(slog.String(h.SpanIDKey, span.SpanID().String()))
	}

	return h.Next.Handle(ctx, record)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	panic("TODO: Implement")
}

func (h *Handler) WithGroup(name string) slog.Handler {
	panic("TODO: Implement")
}
