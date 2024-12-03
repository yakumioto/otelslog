# otelslog

[![Go Report Card](https://goreportcard.com/badge/github.com/yakumioto/otelslog)](https://goreportcard.com/report/github.com/yakumioto/otelslog)
[![codecov](https://codecov.io/github/yakumioto/otelslog/graph/badge.svg?token=6ODsohX0G6)](https://codecov.io/github/yakumioto/otelslog)
[![codebeat badge](https://codebeat.co/badges/dd9f3cd1-265a-4de0-be8a-0d6fcf690220)](https://codebeat.co/projects/github-com-yakumioto-otelslog-main)
[![GoDoc](https://pkg.go.dev/badge/github.com/yakumioto/otelslog)](https://pkg.go.dev/github.com/yakumioto/otelslog)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

otelslog is a Go package that seamlessly integrates structured logging (slog) with OpenTelemetry tracing. By bridging these two essential observability tools, otelslog makes it easier to correlate logs with distributed traces, providing deeper insights into your application's behavior and performance.

[中文版](README_zhCN.md)

## Features

otelslog enhances your application's observability by providing:

- Automatic correlation between logs and traces through trace and span ID injection
- Flexible span creation based on log levels
- Support for mandatory spans that are always created regardless of log level
- Rich support for nested attribute groups in both logs and traces
- Customizable configuration through functional options
- Thread-safe operation for concurrent use

## Installation

To add otelslog to your project, use Go modules:

```bash
go get github.com/yakumioto/otelslog
```

## Quick Start

Here's how to get started with otelslog:

```go
package main

import (
    "log/slog"
    "os"
    
    "github.com/yakumioto/otelslog"
)

func main() {
    // Initialize your tracer
    
    // Set up the default logger with otelslog handler
    slog.SetDefault(slog.New(
        otelslog.NewHandler(slog.NewJSONHandler(os.Stdout, nil)),
    ))

    // Create a span and log with it
    span := otelslog.NewSpan("process-request")
    slog.Info("handling request", 
        "operation", span,
        "user_id", "123",
    )
    defer span.End()
}
```

## Advanced Usage

### Custom Configuration

You can customize the handler's behavior using functional options:

```go
handler := otelslog.NewHandler(
    slog.NewJSONHandler(os.Stdout, nil),
    otelslog.WithTraceIDKey("trace_id"),
    otelslog.WithSpanIDKey("span_id"),
    otelslog.WithTraceLevel(slog.LevelDebug),
    otelslog.WithNoSpanEvents(),
)
```

### Working with Groups

otelslog fully supports slog's grouping functionality:

```go
span := otelslog.NewSpan("process-request")
slog.WithGroup("request").Info("handling request",
    "operation", span,
    slog.Group("user",
        slog.String("id", userID),
        slog.String("role", userRole),
    ),
)
defer span.End()
```

### Mandatory Spans

For critical operations that should always be traced:

```go
span := otelslog.NewMustSpan("critical-operation")
slog.Info("processing important request",
    "operation", span,
    "user_id", userID,
)
defer span.End()
```

### Nested Spans

otelslog supports creating nested spans for tracking sub-operations:

```go
span1 := otelslog.NewSpan("parent-operation")
slog.Info("starting parent operation", "operation", span1)

span2 := otelslog.NewSpan("child-operation")
slog.InfoContext(span1.Context(), "performing sub-operation", "operation", span2)

defer span2.End()
defer span1.End()
```

## Best Practices

To get the most out of otelslog:

1. Always use meaningful span names that clearly describe the operation
2. Use `defer span.End()` immediately after span creation to ensure proper cleanup
3. Configure appropriate trace levels to balance visibility with performance
4. Use groups to organize related attributes in both logs and traces
5. Consider using `NewMustSpan` for critical operations that should always be traced
6. Leverage span context propagation for tracking request flows across service boundaries

## Acknowledgements

This project was inspired by [slog-otel](https://github.com/remychantenay/slog-otel). We would like to thank its creators and contributors for their innovative work in combining structured logging with OpenTelemetry.

## Related Projects

If you're interested in structured logging and OpenTelemetry integration, you might also find these projects useful:

- [slog-otel](https://github.com/remychantenay/slog-otel) - A handler bringing OpenTelemetry to slog
- [slog](https://pkg.go.dev/log/slog@go1.23.3) - The official structured logging package for Go
- [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go) - The official OpenTelemetry SDK for Go