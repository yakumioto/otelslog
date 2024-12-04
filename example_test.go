// /*
//  * Copyright (c) 2024 yakumioto <yaku.mioto@gmail.com>
//  * All rights reserved.
//  */

package otelslog_test

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/yakumioto/otelslog"
)

// initTracer initializes an OTLP exporter, and configures the corresponding trace provider.
func initTracer(ctx context.Context) (func(), error) {
	// Create OTLP exporter
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint("127.0.0.1:4317"), // Your collector endpoint
			otlptracegrpc.WithInsecure(),                 // For testing only
		),
	)
	if err != nil {
		return nil, err
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("your-service-name"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global TracerProvider
	otel.SetTracerProvider(tp)

	// Return a cleanup function
	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("Error shutting down tracer provider", "error", err)
		}
	}, nil
}

// ExampleNewHandler shows how to use the default logger.
func ExampleNewHandler() {
	// Set the default logger to use the OTEL SLOG handler with JSON output to standard output.
	slog.SetDefault(slog.New(otelslog.NewHandler(slog.NewJSONHandler(os.Stdout, nil))))
	slog.Info("hello, world")

	// Set the default logger to use the OTEL SLOG handler with JSON output to standard output.
	// And set the trace ID key to "trace_id", span ID key to "span_id", and the trace level to debug.
	slog.SetDefault(slog.New(
		otelslog.NewHandler(
			slog.NewJSONHandler(os.Stdout, nil),
			otelslog.WithTraceIDKey("trace_id"),
			otelslog.WithSpanIDKey("span_id"),
			otelslog.WithTraceLevel(slog.LevelDebug),
		),
	))

	// Initialize the tracer.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cleanup, err := initTracer(ctx)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// no trace log
	slog.Info("hello, world")

	// trace with slog attributes
	span1 := otelslog.NewSpanContextWithContext(ctx, "", "span1")
	slog.Info("processing request1",
		"trace1", span1,
		"key", "1",
		slog.Group("group1",
			slog.String("key1", "value1"),
			slog.String("key2", "value2"),
		),
	)
	defer span1.End()

	// trace with slog.XXXContext
	span2Ctx := otelslog.NewSpanContextWithContext(span1, "trace2", "span2")
	slog.InfoContext(span2Ctx, "processing request2",
		"key", "1",
		slog.Group("group2",
			slog.String("key1", "value1"),
			slog.String("key2", "value2"),
		),
	)
	defer span2Ctx.Done()

	// trace with slog.With
	span3Ctx := otelslog.NewSpanContextWithContext(span2Ctx, "trace3", "span3")
	slog.Default().WithGroup("group3").With("trace3", span3Ctx).Error("processing request3",
		"key", "1",
		slog.Group("group4",
			slog.String("key1", "value1"),
			slog.String("key2", "value2"),
		),
	)
	defer span3Ctx.End()
}
