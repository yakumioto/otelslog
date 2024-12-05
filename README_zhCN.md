# otelslog

[![CI Status](https://github.com/yakumioto/otelslog/actions/workflows/main.yaml/badge.svg)](https://github.com/yakumioto/otelslog/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/yakumioto/otelslog)](https://goreportcard.com/report/github.com/yakumioto/otelslog)
[![codecov](https://codecov.io/github/yakumioto/otelslog/graph/badge.svg?token=6ODsohX0G6)](https://codecov.io/github/yakumioto/otelslog)
[![codebeat badge](https://codebeat.co/badges/dd9f3cd1-265a-4de0-be8a-0d6fcf690220)](https://codebeat.co/projects/github-com-yakumioto-otelslog-main)
[![GoDoc](https://pkg.go.dev/badge/github.com/yakumioto/otelslog)](https://pkg.go.dev/github.com/yakumioto/otelslog)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Release](https://img.shields.io/github/v/release/yakumioto/otelslog.svg)](https://github.com/yakumioto/otelslog/releases)

otelslog 是一个无缝集成结构化日志（slog）和 OpenTelemetry 分布式追踪的 Go 语言工具包。它通过自动关联日志与分布式追踪数据来丰富您的可观测性技术栈，提供应用行为的深度洞察。该工具包在保持 slog 简洁性的同时，增添了强大的追踪能力，帮助您理解分布式系统中的操作流程。

[English Version](README.md)

## 核心特性

otelslog 通过以下强大特性增强应用的可观测性：

* 自动追踪上下文集成
  * 无缝注入追踪 ID 和 Span ID 到日志记录
  * 内置跨服务边界的上下文传播
  * 支持具有完整父子关系的嵌套 Span

* 灵活配置
  * 可自定义追踪 ID 和 Span ID 的字段名
  * 可配置创建追踪的最低日志级别
  * 可选的 Span 事件记录
  * 支持绕过日志级别过滤的强制 Span

* 丰富的结构化数据支持
  * 完整支持日志和追踪中的嵌套属性分组
  * slog 和 OpenTelemetry 格式之间的类型感知属性转换
  * 在分布式追踪中保留属性层次结构

* 卓越的运维特性
  * 线程安全设计，支持并发使用
  * 内存高效的属性处理

## 安装

使用 Go modules 添加 otelslog 到您的项目：

```bash
go get github.com/yakumioto/otelslog
```

## 快速入门

以下是使用 otelslog 的最小示例：

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
    // 初始化基础追踪器用于演示
    tp := trace.NewTracerProvider()
    otel.SetTracerProvider(tp)
    defer tp.Shutdown(context.Background())
    
    // 使用 otelslog handler 设置默认日志记录器
    slog.SetDefault(slog.New(
        otelslog.NewHandler(slog.NewJSONHandler(os.Stdout, nil)),
    ))

    // 创建 span 并包含在日志中
    span := otelslog.NewSpanContext("service", "process-request")
    slog.Info("处理请求", 
        "operation", span,
        slog.Group("user",
            slog.String("id", "123"),
            slog.String("role", "admin"),
        ),
    )
    defer span.End()
}
```

## 高级用法

### Handler 配置

使用函数选项自定义 handler 行为：

```go
slog.SetDefault(slog.New(
    otelslog.NewHandler(
        slog.NewJSONHandler(os.Stdout, nil),
        otelslog.WithTraceIDKey("trace_id"),     // 自定义追踪 ID 字段
        otelslog.WithSpanIDKey("span_id"),       // 自定义 Span ID 字段
        otelslog.WithTraceLevel(slog.LevelDebug), // 设置最低追踪级别
        otelslog.WithNoSpanEvents(),             // 禁用 Span 事件
    ),
))
```

### 上下文传播与嵌套 Span

通过proper上下文传播跟踪应用程序中的操作：

```go
// 创建根 Span
span1 := otelslog.NewSpanContext("service", "parent-operation")
slog.Info("启动父操作", 
    "operation", span1,
    "request_id", "req-123",
)

// 使用上下文创建子 Span
span2Ctx := otelslog.NewSpanContextWithContext(span1, "service", "child-operation")
slog.InfoContext(span2Ctx, "处理子操作",
    slog.Group("metrics",
        slog.Int("items_processed", 42),
        slog.Duration("processing_time", time.Second),
    ),
)

defer span2Ctx.Done()
defer span1.End()
```

### 强制 Span 与关键操作

确保关键操作始终被追踪，无视日志级别：

```go
span := otelslog.NewMustSpanContext("service", "critical-operation")
slog.Info("处理关键请求",
    "operation", span,
    slog.Group("transaction",
        slog.String("id", "tx-789"),
        slog.Float64("amount", 1299.99),
    ),
)
defer span.End()
```

### 使用结构化数据

使用 slog 的强大分组功能组织日志数据：

```go
span := otelslog.NewSpanContext("service", "user-management")
slog.Default().WithGroup("request").Info("更新用户配置",
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

## OpenTelemetry 集成

设置完整的追踪管道与 OTLP 导出：

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

## 基准测试

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

## 最佳实践

为了在您的应用中最大化 otelslog 的效益：

* 设计 Span 以反映应用的逻辑操作。选择能清晰描述正在执行的操作的 Span 名称，使追踪更易理解和分析。

* 使用分组创建一致的属性层次结构。将相关属性组织在一起，以在日志和追踪中保持清晰性，使跨可观测性工具的信息关联更容易。

* 有效使用上下文传播。始终通过应用的调用链传递上下文，以维持 Span 之间适当的父子关系，确保准确的分布式追踪。

* 考虑性能影响。在高吞吐量场景中，配置适当的追踪级别并在不需要 Span 事件时使用 WithNoSpanEvents 来优化性能。

* 正确处理 Span 生命周期。在创建 Span 后立即使用 defer 调用 span.End()，以确保正确清理和准确的持续时间测量。

* 谨慎使用强制 Span。对于必须追踪的操作使用 NewMustSpanContext，但要注意额外开销。

## 致谢

本项目受 [slog-otel](https://github.com/remychantenay/slog-otel) 启发。我们感谢其创建者和贡献者在结合结构化日志与 OpenTelemetry 方面的开创性工作。

## 相关项目

探索这些相关项目以增强您的可观测性技术栈：

* [slog-otel](https://github.com/remychantenay/slog-otel) - 另一种将 OpenTelemetry 引入 slog 的方法
* [slog](https://pkg.go.dev/log/slog) - Go 的官方结构化日志包
* [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go) - OpenTelemetry 的官方 Go SDK

## 许可证

本项目采用 Apache License 2.0 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。