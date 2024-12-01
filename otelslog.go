/*
 * Copyright (c) 2024 yakumioto <yaku.mioto@gmail.com>
 * All rights reserved.
 */

package otelslog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Options is a functional option for the Handler.
type Options func(*Handler)

// WithTraceIDKey sets the key used to record the trace ID in slog records.
func WithTraceIDKey(key string) Options {
	return func(h *Handler) {
		h.traceIDKey = key
	}
}

// WithSpanIDKey sets the key used to record the span ID in slog records.
func WithSpanIDKey(key string) Options {
	return func(h *Handler) {
		h.spanIDKey = key
	}
}

// WithSpanEventKey sets the key used to record slog attributes as span events.
func WithSpanEventKey(key string) Options {
	return func(h *Handler) {
		h.spanEventKey = key
	}
}

// WithNoSpanEvents disables recording slog attributes as span events.
func WithNoSpanEvents() Options {
	return func(h *Handler) {
		h.spanEvent = false
	}
}

// WithNoSpanBaggageAttrs disables including span baggage attributes in slog records.
func WithNoSpanBaggageAttrs() Options {
	return func(h *Handler) {
		h.baggage = false
	}
}

// NewHandler creates a new slog.Handler with the given options.
func NewHandler(opts ...Options) *Handler {
	h := &Handler{
		traceIDKey:   "trace_id",
		spanIDKey:    "span_id",
		spanEventKey: "log",
		spanEvent:    true,
		baggage:      true,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Handler is responsible for managing OpenTelemetry trace context and handling slog attributes.
// It contains keys for trace and span IDs, controls for recording span events,
// and options for including baggage attributes in slog records.
type Handler struct {
	// OpenTelemetry trace context keys
	traceIDKey string
	spanIDKey  string

	// Key used to record slog attributes as span events
	spanEventKey string

	// Controls whether slog attributes should be recorded as span events
	spanEvent bool

	// Controls whether to record baggage attributes in slog records
	baggage bool

	// Next slog.Handler in the chain
	Next slog.Handler
}

// Enabled checks if the handler is enabled for the given slog.Level.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Next.Enabled(ctx, level)
}

// Handle processes the slog.Record and adds OpenTelemetry attributes and events.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	if ctx == nil {
		return h.Next.Handle(ctx, record)
	}

	// Add baggage attributes to the slog record
	if h.baggage {
		for _, m := range baggage.FromContext(ctx).Members() {
			record.AddAttrs(slog.String(m.Key(), m.Value()))
		}
	}

	// Get the current span from the context
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return h.Next.Handle(ctx, record)
	}

	// Add slog attributes as span events
	if h.spanEvent {
		eventAttrs := make([]attribute.KeyValue, 0, record.NumAttrs())
		record.Attrs(func(attr slog.Attr) bool {
			convertAttrs(attr, func(kv attribute.KeyValue) {
				eventAttrs = append(eventAttrs, kv)
			}, h.spanEventKey)
			return true
		})

		span.AddEvent(h.spanEventKey, trace.WithAttributes(eventAttrs...))
	}

	// Add trace and span IDs to the slog record
	spanCtx := span.SpanContext()
	if spanCtx.HasTraceID() {
		record.AddAttrs(slog.String(h.traceIDKey, spanCtx.TraceID().String()))
	}
	if spanCtx.HasSpanID() {
		record.AddAttrs(slog.String(h.spanIDKey, spanCtx.SpanID().String()))
	}

	// Set the span status based on the slog record level
	switch record.Level {
	case slog.LevelError:
		span.SetStatus(codes.Error, record.Message)
	}

	// Pass the record to the next handler in the chain
	return h.Next.Handle(ctx, record)
}

// WithAttrs returns a new slog.Handler that includes the given slog.Attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		traceIDKey:   h.traceIDKey,
		spanIDKey:    h.spanIDKey,
		spanEventKey: h.spanEventKey,
		spanEvent:    h.spanEvent,
		baggage:      h.baggage,
		Next:         h.Next.WithAttrs(attrs),
	}
}

// WithGroup returns a new slog.Handler that includes the given slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		traceIDKey:   h.traceIDKey,
		spanIDKey:    h.spanIDKey,
		spanEventKey: h.spanEventKey,
		spanEvent:    h.spanEvent,
		baggage:      h.baggage,
		Next:         h.Next.WithGroup(name),
	}
}

// convertAttrs converts slog.Attrs to OpenTelemetry attributes.
func convertAttrs(attr slog.Attr, handler func(attribute.KeyValue), groupKeys ...string) {
	key := attr.Key
	if len(groupKeys) > 0 {
		key = strings.Join(groupKeys, ".") + "." + attr.Key
	}

	val := attr.Value.Resolve()

	switch val.Kind() {
	case slog.KindBool:
		handler(attribute.Bool(key, val.Bool()))
	case slog.KindDuration:
		handler(attribute.Int64(key, int64(val.Duration())))
	case slog.KindFloat64:
		handler(attribute.Float64(key, val.Float64()))
	case slog.KindInt64:
		handler(attribute.Int64(key, val.Int64()))
	case slog.KindString:
		handler(attribute.String(key, val.String()))
	case slog.KindTime:
		handler(attribute.String(key, val.Time().Format(time.RFC3339)))
	// case slog.KindUint64: // attribute.KeyValue does not support Uint64
	// 	handler(attribute.Uint64(key, val.Uint64()))
	case slog.KindGroup:
		for _, groupAttr := range val.Group() {
			convertAttrs(groupAttr, handler, key)
		}
	case slog.KindAny:
		handler(convertAnyValue(key, val.Any()))
	default:
		handler(attribute.String(key, fmt.Sprintf("%+v", val.Any())))
	}
}

// convertAnyValue converts slog.Any to OpenTelemetry attributes.
func convertAnyValue(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case []string:
		return attribute.StringSlice(key, v)
	case []int:
		return attribute.IntSlice(key, v)
	case []int64:
		return attribute.Int64Slice(key, v)
	case []float64:
		return attribute.Float64Slice(key, v)
	case []bool:
		return attribute.BoolSlice(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%+v", v))
	}
}

func TraceError(traceName, spanName, msg string, args ...any) trace.Span {
	tracer := otel.Tracer(traceName)
	ctx := context.Background()
	_, span := tracer.Start(ctx, spanName)
	slog.ErrorContext(ctx, msg, args...)
	return span
}

func TraceErrorContext(ctx context.Context, traceName, spanName, msg string, args ...any) trace.Span {
	tracer := otel.Tracer(traceName)
	ctx, span := tracer.Start(ctx, spanName)
	slog.ErrorContext(ctx, msg, args...)
	return span
}
