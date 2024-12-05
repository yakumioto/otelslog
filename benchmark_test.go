/*
 * Copyright (c) 2024 yakumioto <yaku.mioto@gmail.com>
 * All rights reserved.
 */

package otelslog

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setUpBenchmarkTracer() {
	exporter := tracetest.NewInMemoryExporter()
	traceProvider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(traceProvider)
}

func BenchmarkTextSlog(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	slog.SetDefault(slog.New(
		slog.NewTextHandler(buf, nil),
	),
	)
	setUpBenchmarkTracer()
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tracer := otel.Tracer("benchmark")
		_, span := tracer.Start(ctx, "benchmark")
		defer span.End()
		slog.InfoContext(ctx, "hello, world")
	}
}

func BenchmarkJSONSlog(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	slog.SetDefault(slog.New(
		slog.NewJSONHandler(buf, nil),
	),
	)

	setUpBenchmarkTracer()
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tracer := otel.Tracer("benchmark")
		_, span := tracer.Start(ctx, "benchmark")
		defer span.End()
		slog.InfoContext(ctx, "hello, world")
	}
}

func BenchmarkJSONOtelSlogWithAttr(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	slog.SetDefault(slog.New(
		NewHandler(
			slog.NewJSONHandler(buf, nil),
		),
	))
	setUpBenchmarkTracer()
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		span := NewSpanContext("span")
		defer span.End()
		slog.InfoContext(ctx, "hello, world", "trace", span)
	}
}
func BenchmarkTextOtelSlogWithAttr(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	slog.SetDefault(slog.New(
		NewHandler(
			slog.NewTextHandler(buf, nil),
		),
	))
	setUpBenchmarkTracer()
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		span := NewSpanContext("span")
		defer span.End()
		slog.InfoContext(ctx, "hello, world", "trace", span)
	}
}
func BenchmarkJSONOtelSlogWithContext(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	slog.SetDefault(slog.New(
		NewHandler(
			slog.NewJSONHandler(buf, nil),
		),
	))
	setUpBenchmarkTracer()
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		spanCtx := NewMustSpanContextWithContext(ctx, "span")
		defer spanCtx.Done()
		slog.InfoContext(spanCtx, "hello, world")
	}
}
func BenchmarkTextOtelSlogWithContext(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	slog.SetDefault(slog.New(
		NewHandler(
			slog.NewTextHandler(buf, nil),
		),
	))
	setUpBenchmarkTracer()
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		spanCtx := NewMustSpanContextWithContext(ctx, "span")
		defer spanCtx.Done()
		slog.InfoContext(spanCtx, "hello, world")
	}
}
