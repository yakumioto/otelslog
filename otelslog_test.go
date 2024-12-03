/*
 * Copyright (c) 2024 yakumioto <yaku.mioto@gmail.com>
 * All rights reserved.
 */

package otelslog

import (
	"bytes"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestHandler tests the Handler implementation.
func TestHandler(t *testing.T) {
	setupTracer := func() *tracetest.SpanRecorder {
		spanRecorder := tracetest.NewSpanRecorder()
		tracerProvider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
		otel.SetTracerProvider(tracerProvider)
		return spanRecorder
	}

	setupLogger := func(opts ...Options) *bytes.Buffer {
		buf := bytes.NewBuffer(nil)
		slog.SetDefault(slog.New(
			NewHandler(
				slog.NewJSONHandler(buf, nil),
				opts...,
			)))
		return buf
	}

	t.Run("with slog.HandlerOptions", func(t *testing.T) {
		h := NewHandler(slog.NewJSONHandler(bytes.NewBuffer(nil), nil),
			WithTraceIDKey("test_trace_id"),
			WithSpanIDKey("test_span_id"),
			WithSpanEventKey("test_span_event"),
			WithTraceLevel(slog.LevelDebug),
			WithNoSpanEvents())

		assert.Equal(t, "test_trace_id", h.traceIDKey)
		assert.Equal(t, "test_span_id", h.spanIDKey)
		assert.Equal(t, "test_span_event", h.spanEventKey)
		assert.Equal(t, slog.LevelDebug, h.traceLevel)
		assert.False(t, h.spanEvent)
	})

	t.Run("with out span events", func(t *testing.T) {
		buf := setupLogger()
		slog.Warn("with out span test", "key1", "value1")
		assert.Contains(t, buf.String(), `"level":"WARN"`)
		assert.Contains(t, buf.String(), `"msg":"with out span test"`)
		assert.Contains(t, buf.String(), `"key1":"value1"`)
	})

	t.Run("with span events", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger()

		span := NewSpan("span")
		slog.Warn("with span test", "operation", span, "key1", "value1")
		assert.Contains(t, buf.String(), `"level":"WARN"`)
		assert.Contains(t, buf.String(), `"msg":"with span test"`)
		assert.Contains(t, buf.String(), `"key1":"value1"`)
		span.End()

		spans := spanRecorder.Ended()

		assert.Equal(t, 1, len(spans))
		assert.Equal(t, "span", spans[0].Name())
		assert.Contains(t, spans[0].Events()[0].Attributes, attribute.String("log.key1", "value1"))
	})

	t.Run("with span no events", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger(WithNoSpanEvents())

		span := NewSpan("span")
		slog.Info("with span no events test", "operation", span)
		span.End()

		assert.Contains(t, buf.String(), `"level":"INFO"`)
		assert.Contains(t, buf.String(), `"msg":"with span no events test"`)

		spans := spanRecorder.Ended()

		assert.Equal(t, 1, len(spans))
		assert.Equal(t, "span", spans[0].Name())
		assert.Empty(t, spans[0].Events())
	})

	t.Run("with span on slog.With", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger()

		span := NewSpan("span")
		slog.With("operation", span).Info("with span on slog.With", slog.String("key1", "value1"))
		span.End()

		assert.Contains(t, buf.String(), `"level":"INFO"`)
		assert.Contains(t, buf.String(), `"msg":"with span on slog.With"`)
		assert.Contains(t, buf.String(), `"key1":"value1"`)

		spans := spanRecorder.Ended()

		assert.Equal(t, 1, len(spans))
		assert.Equal(t, "span", spans[0].Name())
		assert.Contains(t, spans[0].Events()[0].Attributes, attribute.String("log.key1", "value1"))
	})

	t.Run("with span on slog.WithGroup", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger()

		span := NewSpan("span")
		slog.Default().WithGroup("group").Info("with span on slog.WithGroup", "operation", span, "key1", "value1")
		span.End()

		assert.Contains(t, buf.String(), `"level":"INFO"`)
		assert.Contains(t, buf.String(), `"msg":"with span on slog.WithGroup"`)
		assert.Contains(t, buf.String(), `"group":{"key1":"value1",`)

		spans := spanRecorder.Ended()

		assert.Equal(t, 1, len(spans))
		assert.Equal(t, "span", spans[0].Name())
		assert.Contains(t, spans[0].Events()[0].Attributes, attribute.String("group.log.key1", "value1"))
	})

	t.Run("with span on slog.WithGroup nested", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger()

		span := NewSpan("span")
		slog.Default().WithGroup("group1").WithGroup("group2").Info("with span on slog.WithGroup nested", "operation", span, "key1", "value1")
		span.End()

		assert.Contains(t, buf.String(), `"level":"INFO"`)
		assert.Contains(t, buf.String(), `"msg":"with span on slog.WithGroup nested"`)
		assert.Contains(t, buf.String(), `"group1":{"group2":{"key1":"value1",`)

		spans := spanRecorder.Ended()

		assert.Equal(t, 1, len(spans))
		assert.Equal(t, "span", spans[0].Name())
		assert.Contains(t, spans[0].Events()[0].Attributes, attribute.String("group1.group2.log.key1", "value1"))
	})

	t.Run("with span on slog.Group", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger()

		span := NewSpan("span")
		slog.Default().Info("with span on slog.Group", "operation", span, slog.Group("group", slog.String("key1", "value1")))
		span.End()

		assert.Contains(t, buf.String(), `"level":"INFO"`)
		assert.Contains(t, buf.String(), `"msg":"with span on slog.Group"`)
		assert.Contains(t, buf.String(), `"group":{"key1":"value1"}`)

		spans := spanRecorder.Ended()

		assert.Equal(t, 1, len(spans))
		assert.Equal(t, "span", spans[0].Name())
		assert.Contains(t, spans[0].Events()[0].Attributes, attribute.String("log.group.key1", "value1"))
	})

	t.Run("with span on slog.Group nested", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger()

		span := NewSpan("span")
		slog.Default().Info("with span on slog.Group nested", "operation", span, slog.Group("group1", slog.Group("group2", slog.String("key1", "value1"))))
		span.End()

		assert.Contains(t, buf.String(), `"level":"INFO"`)
		assert.Contains(t, buf.String(), `"msg":"with span on slog.Group nested"`)
		assert.Contains(t, buf.String(), `"group1":{"group2":{"key1":"value1"}`)

		spans := spanRecorder.Ended()

		assert.Equal(t, 1, len(spans))
		assert.Equal(t, "span", spans[0].Name())
		assert.Contains(t, spans[0].Events()[0].Attributes, attribute.String("log.group1.group2.key1", "value1"))
	})

	t.Run("with span nested", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger()

		span1 := NewSpan("span1")
		slog.Info("with span nested", "operation1", span1, "key1", "value1")

		span2 := NewSpan("span2")
		slog.ErrorContext(span1.Context(), "with span nested", "operation2", span2, slog.String("key2", "value2"))

		span2.End()
		span1.End()

		assert.Contains(t, buf.String(), `"level":"INFO"`)
		assert.Contains(t, buf.String(), `"msg":"with span nested"`)
		assert.Contains(t, buf.String(), `"key1":"value1"`)
		assert.Contains(t, buf.String(), `"key2":"value2"`)

		spans := spanRecorder.Ended()

		assert.Equal(t, 2, len(spans))
		assert.Equal(t, spans[0].SpanContext().TraceID(), spans[1].SpanContext().TraceID())
	})

	t.Run("with no span on slog.Info", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger(WithTraceLevel(slog.LevelWarn))

		span := NewSpan("span")
		slog.Info("with no span on slog.Info", "operation", span, "key1", "value1")
		span.End()

		assert.Contains(t, buf.String(), `"level":"INFO"`)
		assert.Contains(t, buf.String(), `"msg":"with no span on slog.Info"`)
		assert.Contains(t, buf.String(), `"key1":"value1"`)

		spans := spanRecorder.Ended()

		assert.Equal(t, 0, len(spans))
	})

	t.Run("with must span", func(t *testing.T) {
		spanRecorder := setupTracer()
		buf := setupLogger(WithTraceLevel(slog.LevelWarn))

		span := NewMustSpan("span")
		slog.Info("with must span", "operation", span, "key1", "value1")
		span.End()

		assert.Contains(t, buf.String(), `"level":"INFO"`)
		assert.Contains(t, buf.String(), `"msg":"with must span"`)
		assert.Contains(t, buf.String(), `"key1":"value1"`)

		spans := spanRecorder.Ended()

		assert.Equal(t, 1, len(spans))
		assert.Equal(t, "span", spans[0].Name())
	})

	t.Run("with nil next handler", func(t *testing.T) {
		spanRecorder := setupTracer()
		slog.SetDefault(slog.New(NewHandler(nil)))

		span := NewMustSpan("span")
		slog.Info("with nil next handler", "operation", span, "key1", "value1")
		span.End()

		spans := spanRecorder.Ended()

		assert.Equal(t, 1, len(spans))
		assert.Equal(t, "span", spans[0].Name())
	})
}

func TestConvertAttrs(t *testing.T) {
	tests := []struct {
		name     string
		attr     slog.Attr
		expected attribute.KeyValue
	}{
		{
			name:     "string",
			attr:     slog.String("key1", "value1"),
			expected: attribute.String("log.key1", "value1"),
		},
		{
			name:     "int64",
			attr:     slog.Int64("key2", 42),
			expected: attribute.Int64("log.key2", 42),
		},
		{
			name:     "bool",
			attr:     slog.Bool("key3", true),
			expected: attribute.Bool("log.key3", true),
		},
		{
			name:     "duration",
			attr:     slog.Duration("key4", 5*time.Second),
			expected: attribute.Int64("log.key4", 5000000000), // 5 seconds in nanoseconds
		},
		{
			name:     "float64",
			attr:     slog.Float64("key5", 3.14),
			expected: attribute.Float64("log.key5", 3.14),
		},
		{
			name:     "time",
			attr:     slog.Time("key6", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
			expected: attribute.String("log.key6", "2023-01-01T00:00:00Z"),
		},
		{
			name:     "any",
			attr:     slog.Any("key7", []string{"value1", "value2"}),
			expected: attribute.StringSlice("log.key7", []string{"value1", "value2"}),
		},
		{
			name:     "uint64",
			attr:     slog.Uint64("key8", 100),
			expected: attribute.String("log.key8", "100"), // attribute.KeyValue does not support Uint64, so we use String
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := make([]attribute.KeyValue, 0)
			convertAttrs(test.attr, func(kv attribute.KeyValue) {
				result = append(result, kv)
			}, "log")
			assert.Equal(t, test.expected, result[0])
		})
	}
}

func TestConvertAttrsWithGroup(t *testing.T) {
	tests := []struct {
		name     string
		attr     slog.Attr
		expected []attribute.KeyValue
	}{
		{
			name:     "string and int64",
			attr:     slog.Group("group", slog.String("key1", "value1"), slog.Int64("key2", 42)),
			expected: []attribute.KeyValue{attribute.String("log.group.key1", "value1"), attribute.Int64("log.group.key2", 42)},
		},
		{
			name:     "int64",
			attr:     slog.Group("group", slog.Int64("key2", 42)),
			expected: []attribute.KeyValue{attribute.Int64("log.group.key2", 42)},
		},
		{
			name:     "bool",
			attr:     slog.Group("group", slog.Bool("key3", true)),
			expected: []attribute.KeyValue{attribute.Bool("log.group.key3", true)},
		},
		{
			name:     "duration",
			attr:     slog.Group("group", slog.Duration("key4", 5*time.Second)),
			expected: []attribute.KeyValue{attribute.Int64("log.group.key4", 5000000000)}, // 5 seconds in nanoseconds
		},
		{
			name:     "float64",
			attr:     slog.Group("group", slog.Float64("key5", 3.14)),
			expected: []attribute.KeyValue{attribute.Float64("log.group.key5", 3.14)},
		},
		{
			name:     "time",
			attr:     slog.Group("group", slog.Time("key6", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))),
			expected: []attribute.KeyValue{attribute.String("log.group.key6", "2023-01-01T00:00:00Z")},
		},
		{
			name:     "nested string",
			attr:     slog.Group("group", slog.Group("subgroup", slog.String("key7", "value2"))),
			expected: []attribute.KeyValue{attribute.String("log.group.subgroup.key7", "value2")},
		},
		{
			name:     "nested int64",
			attr:     slog.Group("group", slog.Group("subgroup", slog.Int64("key8", 100))),
			expected: []attribute.KeyValue{attribute.Int64("log.group.subgroup.key8", 100)},
		},
		{
			name:     "nested bool",
			attr:     slog.Group("group", slog.Group("subgroup", slog.Bool("key9", false))),
			expected: []attribute.KeyValue{attribute.Bool("log.group.subgroup.key9", false)},
		},
		{
			name:     "nested duration",
			attr:     slog.Group("group", slog.Group("subgroup", slog.Duration("key10", 10*time.Second))),
			expected: []attribute.KeyValue{attribute.Int64("log.group.subgroup.key10", 10000000000)},
		},
		{
			name:     "nested float64",
			attr:     slog.Group("group", slog.Group("subgroup", slog.Float64("key11", 6.28))),
			expected: []attribute.KeyValue{attribute.Float64("log.group.subgroup.key11", 6.28)},
		},
		{
			name:     "nested time",
			attr:     slog.Group("group", slog.Group("subgroup", slog.Time("key12", time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)))),
			expected: []attribute.KeyValue{attribute.String("log.group.subgroup.key12", "2023-02-01T00:00:00Z")},
		},
		{
			name: "any",
			attr: slog.Group("group", slog.Any("key13",
				[]string{"value1", "value2"}), slog.Any("key14",
				[]int{1, 2}), slog.Any("key15", []int64{3, 4}),
				slog.Any("key16", []float64{5.0, 6.0}),
				slog.Any("key17", []bool{true, false}),
				slog.Any("key18", struct {
					Field1 string
					Field2 int
				}{
					Field1: "value1",
					Field2: 2,
				})),
			expected: []attribute.KeyValue{
				attribute.StringSlice("log.group.key13", []string{"value1", "value2"}),
				attribute.IntSlice("log.group.key14", []int{1, 2}),
				attribute.Int64Slice("log.group.key15", []int64{3, 4}),
				attribute.Float64Slice("log.group.key16", []float64{5.0, 6.0}),
				attribute.BoolSlice("log.group.key17", []bool{true, false}),
				attribute.String("log.group.key18", fmt.Sprintf("%+v", struct {
					Field1 string
					Field2 int
				}{
					Field1: "value1",
					Field2: 2,
				})),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := make([]attribute.KeyValue, 0)
			convertAttrs(test.attr, func(kv attribute.KeyValue) {
				result = append(result, kv)
			}, "log")
			assert.Equal(t, test.expected, result)
		})
	}
}
