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

	// slog attributes and group keys
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

// Handle processes the slog.Record and adds OpenTelemetry attributes and events.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	ctx, record = h.handleTrace(ctx, record)

	if err := h.handleSpan(ctx, &record); err != nil {
		return err
	}

	return h.nextHandle(ctx, record)
}

// WithAttrs returns a new slog.Handler that includes the given slog.Attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		attrs:        attrs,
		groupKeys:    h.groupKeys,
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
		attrs:        h.attrs,
		groupKeys:    append(h.groupKeys, name),
		traceIDKey:   h.traceIDKey,
		spanIDKey:    h.spanIDKey,
		spanEventKey: h.spanEventKey,
		spanEvent:    h.spanEvent,
		traceLevel:   h.traceLevel,
		Next:         h.Next.WithGroup(name),
	}
}

// handleTrace handles the trace context for the slog record.
// If the context is a SpanContext, it updates the context and record.
// Otherwise, it retrieves the trace span from the slog attributes and updates the context and record.
// It returns the updated context and record.
func (h *Handler) handleTrace(ctx context.Context, record slog.Record) (context.Context, slog.Record) {
	traceSpan, attrs := h.getTraceSpan(h.collectAttributes(record))
	if traceSpan != nil {
		if traceSpan.Context != nil {
			ctx = h.traceStart(traceSpan.Context, record.Level, traceSpan)
		} else {
			ctx = h.traceStart(ctx, record.Level, traceSpan)
		}
		newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
		newRecord.AddAttrs(attrs...)
		return ctx, newRecord
	}

	if spanCtx, ok := ctx.(*SpanContext); ok {
		if spanCtx.Context == nil {
			spanCtx.Context = context.Background()
		}
		ctx = h.traceStart(spanCtx.Context, record.Level, spanCtx)
		return ctx, record
	}

	return ctx, record
}

// traceStart starts the span and returns the updated context.
// If the span is nil, it returns the context unchanged.
// If the level is greater than or equal to the trace level, it starts the span.
// If the span must be created, it ensures the span is created.
func (h *Handler) traceStart(ctx context.Context, level slog.Level, span *SpanContext) context.Context {
	if span == nil {
		return ctx
	}

	if level >= h.traceLevel || span.must {
		span.Context, span.Span = otel.Tracer(span.traceName).Start(ctx, span.spanName)
		return span
	}

	return ctx
}

// collectAttributes collects slog attributes from the record and the handler's attributes.
// It returns the collected attributes.
func (h *Handler) collectAttributes(record slog.Record) []slog.Attr {
	attrs := make([]slog.Attr, 0, record.NumAttrs()+len(h.attrs))
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	return attrs
}

// getTraceSpan retrieves the TraceSpan from the attributes and returns it along with the remaining attributes.
// It returns nil and the original attributes if no TraceSpan is found.
func (h *Handler) getTraceSpan(attrs []slog.Attr) (*SpanContext, []slog.Attr) {
	for i, attr := range attrs {
		if span, ok := attr.Value.Resolve().Any().(*SpanContext); ok {
			span.traceName = attr.Key
			return span, slices.Delete(attrs, i, i+1)
		}
	}
	return nil, attrs
}

// nextHandle calls the next slog.Handler in the chain if it exists and is enabled for the given slog.Level.
// It returns nil if the next handler does not exist or is not enabled.
func (h *Handler) nextHandle(ctx context.Context, record slog.Record) error {
	if h.Next != nil && h.Next.Enabled(ctx, record.Level) {
		return h.Next.Handle(ctx, record)
	}

	return nil
}

// handleSpan handles the span for the slog record.
// It returns nil if the span is not recording.
// Otherwise, it adds span events and trace IDs to the span.
func (h *Handler) handleSpan(ctx context.Context, record *slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return nil
	}

	if h.spanEvent {
		h.addSpanEvents(span, record)
	}

	h.addTraceIDs(span, record)
	h.setSpanStatus(span, record)

	return nil
}

// addSpanEvents adds span events to the span.
// It collects the event attributes from the record and adds them to the span as an event.
func (h *Handler) addSpanEvents(span trace.Span, record *slog.Record) {
	eventAttrs := h.collectEventAttributes(record)
	span.AddEvent(h.spanEventKey, trace.WithAttributes(eventAttrs...))
}

// collectEventAttributes collects the event attributes from the record.
// It collects the slog attributes from the record and the handler's group keys.
// It returns the collected attributes.
func (h *Handler) collectEventAttributes(record *slog.Record) []attribute.KeyValue {
	eventAttrs := make([]attribute.KeyValue, 0, record.NumAttrs()+3) // +3 for message, level, time

	record.Attrs(func(attr slog.Attr) bool {
		convertAttrs(attr, func(kv attribute.KeyValue) {
			if kv != (attribute.KeyValue{}) {
				eventAttrs = append(eventAttrs, kv)
			}
		}, h.groupKeys...)
		return true
	})

	// 添加基础属性
	eventAttrs = append(eventAttrs,
		attribute.String(slog.MessageKey, record.Message),
		attribute.String(slog.LevelKey, record.Level.String()),
		attribute.String(slog.TimeKey, record.Time.Format(time.RFC3339)))

	return eventAttrs
}

// addTraceIDs adds the trace IDs to the record.
// It adds the trace ID and span ID to the record as slog attributes.
func (h *Handler) addTraceIDs(span trace.Span, record *slog.Record) {
	spanCtx := span.SpanContext()
	if spanCtx.HasTraceID() {
		record.AddAttrs(slog.String(h.traceIDKey, spanCtx.TraceID().String()))
	}
	if spanCtx.HasSpanID() {
		record.AddAttrs(slog.String(h.spanIDKey, spanCtx.SpanID().String()))
	}
}

// setSpanStatus sets the span status based on the record level.
// It sets the span status to error if the record level is error.
func (h *Handler) setSpanStatus(span trace.Span, record *slog.Record) {
	if record.Level == slog.LevelError {
		span.SetStatus(codes.Error, record.Message)
	}
}

// convertAttrs converts slog.Attrs to OpenTelemetry attributes.
// It handles group keys by prefixing the attribute key with the group keys.
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
// It handles different types of values and returns the appropriate attribute.KeyValue.
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

// SpanContext is a wrapper around trace.Span that provides a context.Context.
// It contains the span, context, trace name, span name, and a flag to ensure the span is created.
type SpanContext struct {
	trace.Span
	context.Context
	traceName string
	spanName  string
	must      bool
}

// NewSpanContext creates a new SpanContext with the given span name.
func NewSpanContext(spanName string, traceNameOpt ...string) *SpanContext {
	traceName := ""
	if len(traceNameOpt) > 0 {
		traceName = traceNameOpt[0]
	}

	return &SpanContext{
		traceName: traceName,
		spanName:  spanName,
	}
}

// NewMustSpanContext creates a new SpanContext with the given span name and ensures it is always created.
func NewMustSpanContext(spanName string, traceNameOpt ...string) *SpanContext {
	traceName := ""
	if len(traceNameOpt) > 0 {
		traceName = traceNameOpt[0]
	}

	return &SpanContext{
		traceName: traceName,
		spanName:  spanName,
		must:      true,
	}
}

// NewSpanContextWithContext creates a new SpanContext with the given context.
func NewSpanContextWithContext(ctx context.Context, spanName string, traceNameOpt ...string) *SpanContext {
	traceName := ""
	if len(traceNameOpt) > 0 {
		traceName = traceNameOpt[0]
	}

	return &SpanContext{
		Context:   ctx,
		traceName: traceName,
		spanName:  spanName,
	}
}

// NewMustSpanContextWithContext creates a new SpanContext with the given context and ensures it is always created.
func NewMustSpanContextWithContext(ctx context.Context, spanName string, traceNameOpt ...string) *SpanContext {
	traceName := ""
	if len(traceNameOpt) > 0 {
		traceName = traceNameOpt[0]
	}

	return &SpanContext{
		Context:   ctx,
		traceName: traceName,
		spanName:  spanName,
		must:      true,
	}
}

// End ends the span.
func (s *SpanContext) End() {
	if s.Span != nil {
		s.Span.End()
	}
}

// Done ends the span and returns the context's done channel.
func (s *SpanContext) Done() <-chan struct{} {
	s.End()
	return s.Context.Done()
}
