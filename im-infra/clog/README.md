# clog 内部日志库

## 1. 概述

`clog` 是一个基于 `uber-go/zap` 的高性能、结构化日志库，专为内部项目设计。它提供了简洁的 API、灵活的配置和模块化的日志管理功能。

**核心特性**:
- **高性能**: 基于 `zap` 的零内存分配设计。
- **结构化日志**: 支持 JSON 和 Console 两种输出格式。
- **模块化**: 为不同业务模块创建独立的日志器。
- **灵活配置**: 支持文件轮转、动态级别调整、TraceID 等。
- **高可测试性**: 基于接口设计，易于在单元测试中 Mock。

## 2. 文件结构与功能

```
.
├── clog.go              # 公开的全局 API 函数 (Init, Module, Info, etc.)
├── clog_test.go         # 单元测试和基准测试
├── config.go            # Config 和 FileRotationConfig 结构体定义及验证
├── errors.go            # 自定义错误类型和错误辅助函数
├── factory.go           # 底层 zap.Logger 的创建工厂
├── interfaces.go        # 核心组件的抽象接口 (Logger, LoggerRegistry, etc.)
├── logger.go            # Logger 接口的具体实现 (ZapLogger)
├── manager.go           # 全局服务实例的管理和访问
├── merger.go            # 配置合并逻辑的实现
├── registry.go          # 日志器实例注册表的内存实现
├── service.go           # 核心业务逻辑的服务层实现
├── API.md               # 详细的 API 参考和代码示例
├── ARCHITECTURE.md      # 架构设计和扩展指南
└── README.md            # 项目入口和高级概述
```

## 3. 核心概念

### 3.1. 日志级别

- `Debug`: 用于开发调试。
- `Info`: 记录关键业务流程。
- `Warn`: 记录可预期的、非致命的异常。
- `Error`: 记录需要关注和修复的错误。
- `Fatal`: 记录导致应用退出的致命错误。

### 3.2. 结构化日志

始终使用结构化字段（如 `clog.String`, `clog.Int`）代替字符串拼接，这有利于日志的后续查询和分析。

### 3.3. 模块化

通过 `clog.Module("module-name")` 为不同业务模块创建专用的日志器，以区分日志来源，方便问题排查。

## 4. 使用流程

1.  **初始化**: 在应用启动时，调用 `clog.Init()` 配置全局日志器。
2.  **记录日志**: 在业务代码中，使用全局函数 `clog.Info()`、`clog.Error()` 等记录日志。
3.  **模块化**: 对于独立的业务模块，通过 `clog.Module()` 获取模块专用的日志器实例进行记录。
4.  **刷新日志**: 在应用退出前，调用 `clog.Sync()` 或 `clog.SyncAll()` 确保所有缓冲的日志都已写入目标。

## 5. 更多信息

- **代码示例与 API**: 查看 `API.md` 获取完整的函数参考和使用示例。
- **架构与扩展**: 查看 `ARCHITECTURE.md` 了解内部实现和二次开发指南。