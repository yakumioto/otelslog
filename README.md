# otelslog

[![CI Status](https://github.com/yakumioto/otelslog/actions/workflows/main.yaml/badge.svg)](https://github.com/yakumioto/otelslog/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/yakumioto/otelslog)](https://goreportcard.com/report/github.com/yakumioto/otelslog)
[![codecov](https://codecov.io/github/yakumioto/otelslog/graph/badge.svg?token=6ODsohX0G6)](https://codecov.io/github/yakumioto/otelslog)
[![codebeat badge](https://codebeat.co/badges/dd9f3cd1-265a-4de0-be8a-0d6fcf690220)](https://codebeat.co/projects/github-com-yakumioto-otelslog-main)
[![GoDoc](https://pkg.go.dev/badge/github.com/yakumioto/otelslog)](https://pkg.go.dev/github.com/yakumioto/otelslog)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Release](https://img.shields.io/github/v/release/yakumioto/otelslog.svg)](https://github.com/yakumioto/otelslog/releases)

otelslog is a Go package that seamlessly integrates structured logging (slog) with OpenTelemetry tracing. It enriches your observability stack by automatically correlating logs with distributed traces, providing deep insights into your application's behavior. The package maintains the simplicity of slog while adding powerful tracing capabilities that help you understand the flow of operations across your distributed system.

[中文版](README_zhCN.md)

## Key Features

otelslog enhances your application's observability through several powerful features:

* Automatic Trace Context Integration
  * Seamless injection of trace and span IDs into log records
  * Built-in context propagation across service boundaries
  * Support for nested spans with proper parent-child relationships

* Flexible Configuration
  * Customizable field names for trace and span IDs
  * Configurable minimum log level for trace creation
  * Optional span event recording
  * Support for mandatory spans that bypass log level filtering

* Rich Structured Data Support
  * Complete support for nested attribute groups in both logs and traces
  * Type-aware attribute conversion between slog and OpenTelemetry formats
  * Preservation of attribute hierarchies in distributed traces

* Operational Excellence
  * Thread-safe design for concurrent use
  * Memory-efficient attribute handling

## Installation

To add otelslog to your project, use Go modules:

```bash
go get github.com/yakumioto/otelslog
```

## Quick Start

Here's a minimal example to get you started with otelslog:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    
    "github.com/yakumioto/otelslog"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
    // Initialize a basic tracer for demonstration
    tp := trace.NewTracerProvider()
    otel.SetTracerProvider(tp)
    defer tp.Shutdown(context.Background())
    
    // Set up the default logger with otelslog handler
    slog.SetDefault(slog.New(
        otelslog.NewHandler(slog.NewJSONHandler(os.Stdout, nil)),
    ))

    // Create a span and include it in logging
    span := otelslog.NewSpanContext("service", "process-request")
    slog.Info("handling request", 
        "operation", span,
        slog.Group("user",
            slog.String("id", "123"),
            slog.String("role", "admin"),
        ),
    )
    defer span.End()
}
```

## Advanced Usage

### Configuring the Handler

Customize the handler's behavior using functional options:

```go
slog.SetDefault(slog.New(
    otelslog.NewHandler(
        slog.NewJSONHandler(os.Stdout, nil),
        otelslog.WithTraceIDKey("trace_id"),     // Custom trace ID field
        otelslog.WithSpanIDKey("span_id"),       // Custom span ID field
        otelslog.WithTraceLevel(slog.LevelDebug), // Set minimum trace level
        otelslog.WithNoSpanEvents(),             // Disable span events
    ),
))
```

### Context Propagation and Nested Spans

Track operations across your application with proper context propagation:

```go
// Create a root span
span1 := otelslog.NewSpanContext("service", "parent-operation")
slog.Info("starting parent operation", 
    "operation", span1,
    "request_id", "req-123",
)

// Create a child span with context
span2Ctx := otelslog.NewSpanContextWithContext(span1, "service", "child-operation")
slog.InfoContext(span2Ctx, "processing sub-operation",
    slog.Group("metrics",
        slog.Int("items_processed", 42),
        slog.Duration("processing_time", time.Second),
    ),
)

defer span2Ctx.Done()
defer span1.End()
```

### Mandatory Spans and Critical Operations

Ensure critical operations are always traced regardless of log level:

```go
span := otelslog.NewMustSpanContext("service", "critical-operation")
slog.Info("processing critical request",
    "operation", span,
    slog.Group("transaction",
        slog.String("id", "tx-789"),
        slog.Float64("amount", 1299.99),
    ),
)
defer span.End()
```

### Working with Structured Data

Organize your logging data using slog's powerful grouping features:

```go
span := otelslog.NewSpanContext("service", "user-management")
slog.Default().WithGroup("request").Info("updating user profile",
    "operation", span,
    slog.Group("user",
        slog.String("id", "user-456"),
        slog.Group("changes",
            slog.String("email", "new@example.com"),
            slog.String("role", "admin"),
        ),
    ),
)
defer span.End()
```

## Integration with OpenTelemetry

Set up a complete tracing pipeline with OTLP export:

```go
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
            semconv.ServiceVersion("1.0.0"),
        )),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
    )
    otel.SetTracerProvider(tp)

    return func() { tp.Shutdown(ctx) }, nil
}
```

## Benchmark

```
goos: darwin
goarch: amd64
pkg: github.com/yakumioto/otelslog
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkTextSlog-12                   	  255117	      4781 ns/op	    4260 B/op	       7 allocs/op
BenchmarkJSONSlog-12                   	  346534	      3367 ns/op	    4157 B/op	       7 allocs/op
BenchmarkJSONOtelSlogWithAttr-12       	  238906	      5670 ns/op	    5489 B/op	      20 allocs/op
BenchmarkTextOtelSlogWithAttr-12       	  245080	      4871 ns/op	    5342 B/op	      20 allocs/op
BenchmarkJSONOtelSlogWithContext-12    	  275746	      4189 ns/op	    5021 B/op	      18 allocs/op
BenchmarkTextOtelSlogWithContext-12    	  271627	      4346 ns/op	    5001 B/op	      19 allocs/op
```

## Best Practices

To maximize the benefits of otelslog in your application:

* Design spans to reflect your application's logical operations. Choose span names that clearly describe what operation is being performed, making traces easier to understand and analyze.

* Create a consistent attribute hierarchy using groups. Organize related attributes together to maintain clarity in both logs and traces, making it easier to correlate information across your observability tools.

* Use context propagation effectively. Always pass context through your application's call chain to maintain proper parent-child relationships between spans and ensure accurate distributed tracing.

* Consider performance implications. Configure appropriate trace levels and use WithNoSpanEvents when span events aren't needed to optimize performance in high-throughput scenarios.

* Handle span lifecycle properly. Always use defer for span.End() calls immediately after span creation to ensure proper cleanup and accurate duration measurements.

* Leverage mandatory spans judiciously. Use NewMustSpanContext for operations that must be traced regardless of log level, but be mindful of the additional overhead.

## Acknowledgements

This project was inspired by [slog-otel](https://github.com/remychantenay/slog-otel). We extend our gratitude to its creators and contributors for their pioneering work in combining structured logging with OpenTelemetry.

## Related Projects

Explore these related projects to enhance your observability stack:

* [slog-otel](https://github.com/remychantenay/slog-otel) - Another approach to bringing OpenTelemetry to slog
* [slog](https://pkg.go.dev/log/slog) - Go's official structured logging package
* [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go) - The official OpenTelemetry SDK for Go

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.