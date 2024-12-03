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

func WithTraceLevel(level slog.Level) Options {
	return func(h *Handler) {
		h.traceLevel = level
	}
}

// NewHandler creates a new slog.Handler with the given options.
func NewHandler(handler slog.Handler, opts ...Options) *Handler {
	h := &Handler{
		traceIDKey:   "trace_id",
		spanIDKey:    "span_id",
		spanEventKey: "log",
		spanEvent:    true,
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

	attrs     []slog.Attr
	groupKeys []string

	// Key used to record slog attributes as span events
	spanEventKey string

	// Controls whether slog attributes should be recorded as span events
	spanEvent bool

	// Controls the level of slog records to be traced
	traceLevel slog.Level

	// Next slog.Handler in the chain
	Next slog.Handler
}

// Enabled checks if the handler is enabled for the given slog.Level.
func (h *Handler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *Handler) traceStart(ctx context.Context, level slog.Level, span *Span) context.Context {
	if span == nil {
		return ctx
	}

	if level >= h.traceLevel || span.must {
		span.ctx, span.Span = otel.Tracer(span.traceName).Start(ctx, span.spanName)
		return span.ctx
	}

	return ctx
}

// getTraceSpan retrieves the TraceSpan from the attributes and returns it along with the remaining attributes.
func (h *Handler) getTraceSpan(attrs []slog.Attr) (*Span, []slog.Attr) {
	for i, attr := range attrs {
		if span, ok := attr.Value.Resolve().Any().(*Span); ok {
			span.traceName = attr.Key
			return span, slices.Delete(attrs, i, i+1)
		}
	}
	return nil, attrs
}

func (h *Handler) nextHandle(ctx context.Context, record slog.Record) error {
	// TODO: check if the next handler is nil
	if h.Next != nil && h.Next.Enabled(ctx, record.Level) {
		return h.Next.Handle(ctx, record)
	}

	return nil
}

// Handle processes the slog.Record and adds OpenTelemetry attributes and events.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, record.NumAttrs()+len(h.attrs))
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	traceSpan, attrs := h.getTraceSpan(attrs)
	ctx = h.traceStart(ctx, record.Level, traceSpan)

	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	newRecord.AddAttrs(attrs...)

	// Get the current span from the context
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return h.nextHandle(ctx, newRecord)
	}

	// Add slog attributes as span events
	if h.spanEvent {
		eventAttrs := make([]attribute.KeyValue, 0, newRecord.NumAttrs())
		newRecord.Attrs(func(attr slog.Attr) bool {
			convertAttrs(attr, func(kv attribute.KeyValue) {
				if kv != (attribute.KeyValue{}) {
					eventAttrs = append(eventAttrs, kv)
				}
			}, append(h.groupKeys, h.spanEventKey)...)
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

	return h.nextHandle(ctx, newRecord)
}

// WithAttrs returns a new slog.Handler that includes the given slog.Attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		attrs:        attrs,
		traceIDKey:   h.traceIDKey,
		spanIDKey:    h.spanIDKey,
		spanEventKey: h.spanEventKey,
		spanEvent:    h.spanEvent,
		traceLevel:   h.traceLevel,
		Next:         h.Next,
	}
}

// WithGroup returns a new slog.Handler that includes the given slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		groupKeys:    append(h.groupKeys, name),
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

type Span struct {
	trace.Span
	ctx       context.Context
	traceName string
	spanName  string
	must      bool
}

// NewSpan creates a new Span with the given span name.
func NewSpan(spanName string) *Span {
	return &Span{
		spanName: spanName,
	}
}

// NewMustSpan creates a new Span with the given span name and ensures it is always created.
func NewMustSpan(spanName string) *Span {
	return &Span{
		spanName: spanName,
		must:     true,
	}
}

// End ends the span.
func (s *Span) End() {
	if s.Span != nil {
		s.Span.End()
	}
}

// Context returns the context associated with the span.
func (s *Span) Context() context.Context {
	return s.ctx
}
