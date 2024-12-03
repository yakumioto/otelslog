# otelslog

[![Go Report Card](https://goreportcard.com/badge/github.com/yakumioto/otelslog)](https://goreportcard.com/report/github.com/yakumioto/otelslog)
[![codecov](https://codecov.io/github/yakumioto/otelslog/graph/badge.svg?token=6ODsohX0G6)](https://codecov.io/github/yakumioto/otelslog)
[![codebeat badge](https://codebeat.co/badges/dd9f3cd1-265a-4de0-be8a-0d6fcf690220)](https://codebeat.co/projects/github-com-yakumioto-otelslog-main)
[![GoDoc](https://pkg.go.dev/badge/github.com/yakumioto/otelslog)](https://pkg.go.dev/github.com/yakumioto/otelslog)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

otelslog 是一个将 Go 的结构化日志（slog）与 OpenTelemetry 分布式追踪无缝集成的 Go 包。通过桥接这两个重要的可观测性工具，otelslog 使得日志和分布式追踪的关联变得更加简单，从而能够更深入地洞察应用程序的行为和性能。

## 特性

otelslog 通过以下特性增强了应用程序的可观测性：

- 自动注入 trace ID 和 span ID，实现日志与追踪的关联
- 基于日志级别的灵活 span 创建机制
- 支持必要时强制创建 span，不受日志级别限制
- 完整支持日志和追踪中的嵌套属性分组
- 通过函数选项实现灵活配置
- 支持并发安全操作

## 安装

使用 Go modules 将 otelslog 添加到你的项目中：

```bash
go get github.com/yakumioto/otelslog
```

## 快速开始

以下是 otelslog 的基本使用示例：

```go
package main

import (
    "log/slog"
    "os"
    
    "github.com/yakumioto/otelslog"
)

func main() {
    // 初始化你的追踪器
    
    // 使用 otelslog handler 设置默认日志记录器
    slog.SetDefault(slog.New(
        otelslog.NewHandler(slog.NewJSONHandler(os.Stdout, nil)),
    ))

    // 创建 span 并记录日志
    span := otelslog.NewSpan("process-request")
    slog.Info("handling request", 
        "operation", span,
        "user_id", "123",
    )
    defer span.End()
}
```

## 进阶用法

### 自定义配置

你可以使用函数选项自定义 handler 的行为：

```go
handler := otelslog.NewHandler(
    slog.NewJSONHandler(os.Stdout, nil),
    otelslog.WithTraceIDKey("trace_id"),     // 自定义 trace ID 的键名
    otelslog.WithSpanIDKey("span_id"),       // 自定义 span ID 的键名
    otelslog.WithTraceLevel(slog.LevelDebug),// 设置追踪级别
    otelslog.WithNoSpanEvents(),             // 禁用 span 事件记录
)
```

### 使用分组功能

otelslog 完整支持 slog 的分组功能：

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

### 强制创建 Span

对于必须追踪的关键操作，可以使用强制创建的 span：

```go
span := otelslog.NewMustSpan("critical-operation")
slog.Info("processing important request",
    "operation", span,
    "user_id", userID,
)
defer span.End()
```

### 嵌套 Span

otelslog 支持创建嵌套 span 来追踪子操作：

```go
span1 := otelslog.NewSpan("parent-operation")
slog.Info("starting parent operation", "operation", span1)

span2 := otelslog.NewSpan("child-operation")
slog.InfoContext(span1.Context(), "performing sub-operation", "operation", span2)

defer span2.End()
defer span1.End()
```

## 最佳实践

为了更好地使用 otelslog，建议遵循以下最佳实践：

1. 使用有意义的 span 名称，清晰描述操作的目的
2. 在创建 span 后立即使用 `defer span.End()` 确保正确清理
3. 合理配置追踪级别，在可观测性和性能之间取得平衡
4. 使用分组功能组织日志和追踪中的相关属性
5. 对关键操作使用 `NewMustSpan` 确保始终被追踪
6. 在服务边界间正确传播 span 上下文

让我帮您将这些章节翻译成专业的中文版本，同时保持技术准确性和专业性：

## 致谢

本项目灵感源自 [slog-otel](https://github.com/remychantenay/slog-otel)。我们由衷感谢该项目的创作者和贡献者在结合结构化日志与 OpenTelemetry 追踪领域所做出的开创性工作。

## 相关项目

如果您对结构化日志记录和 OpenTelemetry 的集成感兴趣，以下项目可能对您有所帮助：

- [slog-otel](https://github.com/remychantenay/slog-otel) - 为 slog 引入 OpenTelemetry 能力的处理实现
- [slog](https://pkg.go.dev/log/slog@go1.23.3) - Go 语言官方提供的结构化日志记录包
- [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go) - OpenTelemetry 官方提供的 Go 语言 SDK 实现