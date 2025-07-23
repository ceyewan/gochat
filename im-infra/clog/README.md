# `im-infra/clog` - GoChat 高性能结构化日志库

`clog` 是一个现代化、高性能的 Go 结构化日志库，基于 Go 1.21+ 标准库 `log/slog` 构建。它为 GoChat 微服务生态提供了统一、可扩展且易于使用的日志记录解决方案。

## 1. 为什么选择结构化日志 (`slog`)？

在复杂的分布式系统中，日志不仅仅是打印信息，它更是系统“可观测性”的基石。传统的非结构化日志（如 `fmt.Println` 或 `log.Print`）在开发时简单直观，但在生产环境中进行故障排查和数据分析时，却显得力不从心。

**结构化日志**通过将日志信息以键值对（Key-Value）的形式进行记录，解决了这个问题。每一条日志都是一条机器可读的数据记录，这带来了革命性的优势：

-   **高效查询与分析**：你可以像查询数据库一样过滤日志，例如 `level=error` 且 `module=database`。
-   **数据关联**：可以轻松地将 `trace_id`、`user_id` 等上下文信息注入到日志中，将分散的日志点串联成完整的事件流。
-   **自动化与告警**：基于结构化的日志数据，可以轻松地对接日志分析平台（如 ELK、Loki），实现自动化的监控和告警。

Go 1.21 版本正式引入的 `log/slog` 库，标志着结构化日志正式成为 Go 的官方标准。`clog` 正是构建在这一坚实基础之上，充分利用其高性能和高扩展性。

## 2. 功能特色

-   🚀 **基于 `slog`**：完全兼容 Go 标准库，享受原生的高性能和零依赖优势。
-   🌟 **全局与模块化**：提供 `clog.Info()` 等全局方法用于快速开发，同时通过 `clog.Module("name")` 支持模块化日志，实现大型项目中的日志隔离。
-   🔄 **多目标输出（Teeing）**：可将日志同时输出到多个目标，例如，在控制台输出人类可读的 `text` 格式，同时向文件和远端服务写入 `json` 格式。
-   ⚡ **动态日志级别**：可在服务运行时通过 `logger.SetLevel("debug")` 动态调整日志输出级别，无需重启服务。
-   🏷️ **TraceID 自动注入**：能自动从 `context.Context` 中提取 TraceID 并添加到每条日志中，无缝对接分布式链路追踪。
-   📁 **内置文件滚动**：集成了可靠的 `lumberjack` 库，提供开箱即用的日志文件切分和压缩功能。
-   🔧 **接口驱动**：所有核心功能均通过 `Logger` 接口暴露，便于测试和自定义扩展。

## 3. 快速上手

### 推荐方式：全局方法与模块日志

在你的 Go 文件中，可以直接使用全局方法记录日志。对于不同的业务模块，使用 `clog.Module()` 获取带模块标识的日志器。

```go
package main

import (
	"context"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// 在包级别获取并缓存模块日志器，性能最佳
var dbLogger = clog.Module("database")

func main() {
    // 1. 使用全局方法进行通用日志记录
	clog.Info("服务启动", clog.String("version", "1.0.2"))
	clog.Warn("这是一个警告", clog.String("component", "main"))

    // 2. 使用模块日志器记录特定模块的日志
    // 注意：会自动带上 "module=database" 字段
	dbLogger.Info("数据库连接成功", clog.String("host", "localhost"))

    // 3. 记录错误
    if err != nil {
        clog.Error("发生了一个错误", clog.Err(err))
    }

    // 4. 传递 context 以自动注入 TraceID
    ctx := context.WithValue(context.Background(), "trace_id", "req-xyz-789")
    clog.InfoContext(ctx, "这是一条带 TraceID 的日志")
}
```

### 自定义配置

如果你需要更高级的控制，例如将日志输出到文件，可以通过 `clog.New()` 创建一个独立的日志器实例。

```go
package main

import (
	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	cfg := clog.Config{
		Level: "debug", // 设置为 Debug 级别
		Outputs: []clog.OutputConfig{
			// 输出到控制台
			{
				Format: "text",
				Writer: "stdout",
			},
			// 输出到文件，并启用滚动
			{
				Format: "json",
				Writer: "file",
				FileRotation: &clog.FileRotationConfig{
					Filename:   "logs/app.log",
					MaxSize:    100, // 100 MB
					MaxAge:     30,  // 30 天
					MaxBackups: 10,
					Compress:   true,
				},
			},
		},
		AddSource: true, // 在日志中添加源码文件和行号
	}

	// 使用自定义配置创建新的日志器
	logger, err := clog.New(cfg)
	if err != nil {
		panic(err)
	}

	logger.Info("这条日志会同时输出到控制台和文件")
	logger.Debug("这条 Debug 级别的日志现在也会显示")
}
```

## 4. 核心设计与技术点

### a. `Tee` 模式：多 Handler 聚合

`clog` 的多目标输出能力是通过 `Tee`（三通管）模式实现的。当你配置多个 `Outputs` 时，`clog` 会为每个输出创建一个独立的 `slog.Handler`。然后，通过一个自定义的 `TeeHandler` 将这些独立的 Handlers 聚合起来。当一条日志产生时，`TeeHandler` 会将该日志记录分发给所有下游的 Handlers，从而实现了“一次写入，多处生效”的效果。

### b. `sync.Once` 与全局单例

全局方法（如 `clog.Info`）依赖一个全局的默认日志器实例。为了保证高性能和线程安全，这个实例通过 `sync.Once` 实现懒加载和单例模式。这意味着只有在第一次调用全局日志方法时，实例才会被创建，并且后续的所有调用都将无锁地复用这个已经创建好的实例。

### c. `Module` 方法与日志器继承

与 `slog` 的 `WithGroup` 不同，`clog.Module()` 提供了更符合模块化开发的日志划分方式。`someLogger.Module("my_module")` 本质上是 `someLogger.With(clog.String("module", "my_module"))` 的语法糖。它创建了一个新的子日志器，该子日志器继承了父日志器的所有配置和已有关联字段，并在此基础上附加了一个 `module` 字段，实现了配置的继承和上下文的扩展。

## 5. API 与配置详解

要了解所有可用的 API、配置选项和最佳实践，请参考 [`API.md`](./API.md) 文档。
