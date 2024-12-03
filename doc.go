/*
Package otelslog provides a slog.Handler implementation that integrates Go's structured logging (slog)
with OpenTelemetry tracing. This integration allows for seamless correlation between logs and traces,
making it easier to debug and monitor applications using both logging and distributed tracing.

The handler automatically enriches log entries with trace context information and can optionally
record log entries as span events, providing a comprehensive view of application behavior
across both logging and tracing systems.

Key Features:

  - Automatic injection of trace and span IDs into log records
  - Optional recording of log entries as span events
  - Support for nested attribute groups
  - Configurable trace level filtering
  - Support for mandatory spans regardless of log level
  - Flexible configuration through functional options

Basic Usage:

To create a new handler with default settings:

	slog.SetDefault(slog.New(otelslog.NewHandler(slog.NewJSONHandler(os.Stdout, nil))))
	slog.Info("hello, world")

With custom configuration:

	slog.SetDefault(slog.New(
	otelslog.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil),
		otelslog.WithTraceIDKey("trace_id"),
		otelslog.WithSpanIDKey("span_id"),
		otelslog.WithTraceLevel(slog.LevelDebug),
	)))
	slog.Info("hello, world")

Creating spans with logging:

	span := otelslog.NewSpan("span-name")
	slog.Info("processing request",
	    "operation-name", span,
	    "user_id", userID,
	)
	defer span.End()

For operations that must always create spans regardless of trace level:

	span := otelslog.NewMustSpan("critical-operation")
	slog.Info("processing important request",
	    "operation", span,
	    "user_id", userID,
	)
	defer span.End()

Configuration Options:

The handler supports several configuration options through functional options:

  - WithTraceIDKey: Customizes the key used for the trace ID in log records
  - WithSpanIDKey: Customizes the key used for the span ID in log records
  - WithSpanEventKey: Customizes the key used when recording log entries as span events
  - WithNoSpanEvents: Disables the recording of log entries as span events
  - WithTraceLevel: Sets the minimum log level at which spans are created

Group Support:

The handler fully supports slog's group functionality, maintaining the group hierarchy
in both logs and span events:

	span := otelslog.NewSpan("process-request")
	slog.WithGroup("request").Info("handling request",
	    "operation", span,
	    slog.Group("user",
	        slog.String("id", userID),
	        slog.String("role", userRole),
	    ),
	)

Log Level and Span Creation:

By default, spans are created for all log levels. This behavior can be customized
using WithTraceLevel to set a minimum level for span creation. However, spans created
with NewMustSpan will always be created regardless of the trace level setting.

Best Practices:

1. Use meaningful span names that describe the operation being performed
2. End spans using defer to ensure proper cleanup
3. Configure trace levels appropriately to control span creation overhead
4. Use groups to organize related attributes in both logs and traces
5. Consider using NewMustSpan for critical operations that should always be traced

Thread Safety:

The handler is safe for concurrent use by multiple goroutines.
*/
package otelslog
