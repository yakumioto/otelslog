/*
Package otelslog provides a powerful integration between Go's structured logging (slog) and OpenTelemetry
tracing. This package enables developers to correlate logs and traces seamlessly, creating a comprehensive
observability solution that enhances debugging and monitoring capabilities in distributed systems.

# Core Concepts

The package centers around a custom slog.Handler implementation that automatically enriches log entries
with distributed tracing context while maintaining the flexibility and simplicity of slog's interface.
It preserves the hierarchical structure of logged attributes in both logs and trace spans, ensuring
consistency across your observability data.

# Key Features

Trace Context Integration:
  - Automatic injection of trace and span IDs into log records
  - Preservation of parent-child relationships between spans
  - Support for context propagation across service boundaries

Flexible Configuration:
  - Customizable trace and span ID field names
  - Configurable minimum log level for trace creation
  - Optional span event recording
  - Support for mandatory spans that bypass log level filtering

Structured Data Support:
  - Full support for slog's group functionality
  - Hierarchical attribute preservation in both logs and spans
  - Type-aware attribute conversion between slog and OpenTelemetry formats

# Basic Usage

1. Setting up the handler with default configuration:

	slog.SetDefault(slog.New(otelslog.NewHandler(slog.NewJSONHandler(os.Stdout, nil))))
	slog.Info("hello, world")

2. Configuring the handler with custom options:

	slog.SetDefault(slog.New(
	    otelslog.NewHandler(
	        slog.NewJSONHandler(os.Stdout, nil),
	        otelslog.WithTraceIDKey("trace_id"),
	        otelslog.WithSpanIDKey("span_id"),
	        otelslog.WithTraceLevel(slog.LevelDebug),
	    ),
	))

# Advanced Usage

1. Creating spans with context propagation:

	// Create a root span
	span1 := otelslog.NewSpanContext("trace", "span1")
	slog.Info("processing request",
	    "operation", span1,
	    "key1", "value1",
	)
	defer span1.End()

	// Create a child span
	span2Ctx := otelslog.NewSpanContextWithContext(span1, "trace", "span2")
	slog.InfoContext(span2Ctx, "nested operation")
	defer span2Ctx.Done()

2. Working with attribute groups:

	span := otelslog.NewSpanContext("trace", "span")
	slog.Default().WithGroup("request").Info("processing",
	    "operation", span,
	    slog.Group("user",
	        slog.String("id", "123"),
	        slog.String("role", "admin"),
	    ),
	)
	defer span.End()

3. Creating mandatory spans:

	span := otelslog.NewMustSpanContext("trace", "critical-operation")
	slog.Info("critical processing", "operation", span)
	defer span.End()

# Configuration Options

The handler supports several functional options for customization:

WithTraceIDKey(key string):

	Customizes the field name for trace IDs in log records

WithSpanIDKey(key string):

	Customizes the field name for span IDs in log records

WithSpanEventKey(key string):

	Customizes the field name used when recording log entries as span events

WithTraceLevel(level slog.Level):

	Sets the minimum log level at which spans are created

WithNoSpanEvents():

	Disables the recording of log entries as span events

# Best Practices

1. Span Management:
  - Use defer for span.End() calls to ensure proper cleanup
  - Create spans with meaningful names that describe the operation
  - Use NewMustSpanContext for critical operations that should always be traced

2. Context Handling:
  - Propagate context through your application using NewSpanContextWithContext
  - Use InfoContext/ErrorContext when you have an existing context
  - Maintain proper parent-child relationships between spans

3. Attribute Organization:
  - Use groups to logically organize related attributes
  - Maintain consistent attribute naming across your application
  - Consider the hierarchical structure when designing attribute groups

4. Performance Considerations:
  - Configure appropriate trace levels to control span creation
  - Use WithNoSpanEvents when span events are not needed
  - Consider the overhead of attribute conversion in high-throughput scenarios

# Thread Safety

The handler implementation is fully thread-safe and can be safely used concurrently
from multiple goroutines. All operations on spans and contexts are designed to be
thread-safe as well.

# Integration with OpenTelemetry

The package seamlessly integrates with OpenTelemetry's trace API and SDK. It supports:
  - Standard OpenTelemetry trace exporters
  - Custom trace samplers
  - Resource attributes for service identification
  - Context propagation across service boundaries

Example Setup with OTLP Exporter:

	func initTracer(ctx context.Context) (func(), error) {
	    exporter, err := otlptrace.New(
	        ctx,
	        otlptracegrpc.NewClient(
	            otlptracegrpc.WithEndpoint("localhost:4317"),
	            otlptracegrpc.WithInsecure(),
	        ),
	    )
	    if err != nil {
	        return nil, err
	    }

	    tp := sdktrace.NewTracerProvider(
	        sdktrace.WithBatcher(exporter),
	        sdktrace.WithResource(resource.NewWithAttributes(
	            semconv.ServiceName("your-service"),
	        )),
	    )
	    otel.SetTracerProvider(tp)

	    return func() { tp.Shutdown(ctx) }, nil
	}

The package is designed to be a comprehensive solution for correlating logs and traces
in Go applications, providing the flexibility and features needed for effective
observability in modern distributed systems.
*/
package otelslog
