/*
 * Copyright (c) 2024 yakumioto <yaku.mioto@gmail.com>
 * All rights reserved.
 */

package otelslog

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

// NewHandler creates a new slog.Handler with the given options.
func NewHandler(handler slog.Handler, opts ...Options) *Handler {
	h := &Handler{
		traceIDKey:   "trace_id",
		spanIDKey:    "span_id",
		spanEventKey: "log",
		spanEvent:    true,
		traceLevel:   slog.LevelInfo,
		Next:         handler,
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

	attrs []slog.Attr

	// Key used to record slog attributes as span events
	spanEventKey string

	// Controls whether slog attributes should be recorded as span events
	spanEvent bool

	// Controls the level of slog records that will be traced
	traceLevel slog.Level

	// Next slog.Handler in the chain
	Next slog.Handler
}

// Enabled checks if the handler is enabled for the given slog.Level.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Next.Enabled(ctx, level)
}

func (h *Handler) getSpanWithAttrs(attrs []slog.Attr) *SpanAttr {
	var span *SpanAttr
	for i, attr := range attrs {
		if _, ok := attr.Value.Resolve().Any().(*SpanAttr); ok {
			span = attr.Value.Resolve().Any().(*SpanAttr)
			attrs = slices.Delete(attrs, i, 1)
		}
	}
	return span
}

func (h *Handler) traceWithSpan(ctx context.Context, spanAttr *SpanAttr) context.Context {
	if spanAttr == nil {
		return ctx
	}

	spanCtx, span := otel.Tracer(spanAttr.traceName).Start(ctx, spanAttr.spanName)
	spanAttr.ctx = spanCtx
	spanAttr.Span = span
	return spanCtx
}

// Handle processes the slog.Record and adds OpenTelemetry attributes and events.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, record.NumAttrs()+len(h.attrs))
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	ctx = h.traceWithSpan(ctx, h.getSpanWithAttrs(attrs))

	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	newRecord.AddAttrs(attrs...)

	// Get the current span from the context
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return h.Next.Handle(ctx, newRecord)
	}

	// Add slog attributes as span events
	if h.spanEvent {
		eventAttrs := make([]attribute.KeyValue, 0, newRecord.NumAttrs())
		newRecord.Attrs(func(attr slog.Attr) bool {
			convertAttrs(attr, func(kv attribute.KeyValue) {
				if kv != (attribute.KeyValue{}) {
					eventAttrs = append(eventAttrs, kv)
				}
			}, h.spanEventKey)
			return true
		})

		eventAttrs = append(eventAttrs, attribute.String(slog.MessageKey, newRecord.Message))
		eventAttrs = append(eventAttrs, attribute.String(slog.LevelKey, newRecord.Level.String()))
		eventAttrs = append(eventAttrs, attribute.String(slog.TimeKey, newRecord.Time.Format(time.RFC3339)))
		span.AddEvent(h.spanEventKey, trace.WithAttributes(eventAttrs...))
	}

	// Add trace and span IDs to the slog record
	spanCtx := span.SpanContext()
	if spanCtx.HasTraceID() {
		newRecord.AddAttrs(slog.String(h.traceIDKey, spanCtx.TraceID().String()))
	}
	if spanCtx.HasSpanID() {
		newRecord.AddAttrs(slog.String(h.spanIDKey, spanCtx.SpanID().String()))
	}

	// Set the span status based on the slog record level
	switch newRecord.Level {
	case slog.LevelError:
		span.SetStatus(codes.Error, newRecord.Message)
	}

	// Pass the record to the next handler in the chain
	return h.Next.Handle(ctx, newRecord)
}

// WithAttrs returns a new slog.Handler that includes the given slog.Attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		attrs:        attrs,
		traceIDKey:   h.traceIDKey,
		spanIDKey:    h.spanIDKey,
		spanEventKey: h.spanEventKey,
		spanEvent:    h.spanEvent,
		Next:         h.Next,
	}
}

// WithGroup returns a new slog.Handler that includes the given slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		traceIDKey:   h.traceIDKey,
		spanIDKey:    h.spanIDKey,
		spanEventKey: h.spanEventKey,
		spanEvent:    h.spanEvent,
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

type SpanAttr struct {
	trace.Span
	ctx       context.Context
	traceName string
	spanName  string
	must      bool
}

func NewSpanAttr(traceName, spanName string, mustOpts ...bool) *SpanAttr {
	must := false
	if len(mustOpts) > 0 {
		must = mustOpts[0]
	}

	return &SpanAttr{
		traceName: traceName,
		spanName:  spanName,
		must:      must,
	}
}

func (sa *SpanAttr) End() {
	if sa.Span != nil {
		sa.Span.End()
	}
}

func (sa *SpanAttr) Must() bool {
	return sa.must
}

func (sa *SpanAttr) Attr() []any {
	return []any{fmt.Sprintf("%s--%s", sa.traceName, sa.spanName), sa}
}
