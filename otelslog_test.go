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

func setupTracer() (*tracetest.SpanRecorder, *trace.TracerProvider) {
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tracerProvider)
	return spanRecorder, tracerProvider
}

func setupLogger() *bytes.Buffer {
	buf := &bytes.Buffer{}
	slog.SetDefault(slog.New(NewHandler(slog.NewJSONHandler(buf, nil))))
	return buf
}

func TestHandler(t *testing.T) {
	spanRecorder, _ := setupTracer()
	buf := setupLogger()

	span := NewSpanAttr("test-trace", "test-span", true)
	slog.Warn("with out span test")
	slog.With(span.Attr()...).Warn("test")
	span.End()

	spans := spanRecorder.Ended()
	t.Log(spans[0].Events()[0].Attributes)

	t.Log(buf.String())
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
			expected: []attribute.KeyValue{attribute.Int64("log.group.subgroup.key10", 10000000000)}, // 10 seconds in nanoseconds
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
