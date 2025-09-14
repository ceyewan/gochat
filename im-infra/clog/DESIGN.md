# clog Design Document

## 🎯 Design Goals

clog is the official structured logging library for the GoChat project, built on uber-go/zap. It aims to provide a **concise, high-performance, context-aware** logging solution that fully complies with GoChat's development standards.

### Core Design Principles

1. **Standards Priority**: Strictly follows im-infra component design norms, using the standard Provider pattern.
2. **Context Awareness**: Automatically extracts `trace_id` from `context.Context` to support distributed tracing.
3. **Hierarchical Namespaces**: Unified namespace system with chainable calls for clear module boundaries.
4. **Type Safety**: Encapsulates context keys to avoid conflicts and provides compile-time type checks.
5. **Environment Awareness**: Offers environment-specific default configurations for development and production.
6. **High Performance**: Leverages zap's zero-allocation logging engine with minimal overhead.
7. **Strong Observability**: Complete namespace paths and structured fields enable precise filtering, analysis, and request chain visualization.

These principles ensure clog is simple to use, performant, and integrates seamlessly into microservices architectures like GoChat.

## 🏗️ Architecture Overview

The architecture is layered to separate concerns: public API, configuration, core logic, internal implementations, and the zap foundation.

### High-Level Architecture

```
Public API Layer
├── clog.Info/Warn/Error/Fatal (Global methods)
├── clog.Namespace() (Hierarchical namespaces)
├── clog.WithContext() / C() (Context-aware logger)
├── clog.WithTraceID() (TraceID injection)
└── clog.New/Init (Provider mode)

Configuration Layer
├── Config struct (Level, Format, Output, etc.)
├── GetDefaultConfig(env) (Environment-aware defaults)
├── Option pattern (e.g., WithNamespace)
└── ParseOptions() (Option resolution)

Core Layer
├── getDefaultLogger() (Singleton with atomic replacement)
├── TraceID management (Type-safe context keys)
└── Global atomic logger replacement

Internal Layer
├── internal.Logger interface
├── zapLogger implementation (zap wrapper)
└── Hierarchical namespace handling

Zap Foundation
├── zap.Logger
├── zapcore.Core
└── zapcore.Encoder (JSON/Console)
```

This layered design promotes modularity, testability, and extensibility while maintaining a clean public API.

### Key Components

#### 1. Provider Mode Implementation

**Purpose**: Enables dependency injection and follows im-infra norms for component initialization.

**Core Functions**:
- `New(ctx context.Context, config *Config, opts ...Option) (Logger, error)`: Creates an independent logger instance. The `ctx` controls initialization (not held by the logger). Options customize behavior (e.g., namespace).
- `Init(ctx context.Context, config *Config, opts ...Option) error`: Initializes the global default logger. Fails if already initialized (use replacement for hot-swaps).
- `GetDefaultConfig(env string) *Config`: Returns optimized defaults:
  - "development": Debug level, console format, colors enabled.
  - "production": Info level, JSON format, colors disabled.

**Implementation Highlights**:
- Parses options into a struct for namespace injection.
- Creates zap logger with config (encoder, output, level).
- Fallback to a no-op logger on errors for graceful degradation.
- Thread-safe singleton for global logger using `sync.Once` and `atomic.Value`.

**Rationale**: The Provider pattern ensures configurability without global state pollution. Environment defaults reduce boilerplate and prevent misconfigurations in dev/prod.

#### 2. Hierarchical Namespace System

**Purpose**: Provides a unified way to tag logs with service/module/component paths, replacing fragmented service/module APIs.

**Core Function**:
- `Namespace(name string) Logger`: Returns a child logger with the namespace appended (e.g., root "im-gateway" + "user" → "im-gateway.user"). Chainable for deep paths like "im-gateway.payment.processor.stripe".

**Implementation**:
- Namespaces are zap fields added once during logger creation (`zap.String("namespace", fullPath)`).
- Root set via `WithNamespace` option during init/New.
- Chain creates new loggers wrapping the parent, avoiding repeated string operations.

**Example Output**:
```json
{"namespace": "im-gateway.user.auth", "trace_id": "abc123", "msg": "Password validation"}
```

**Rationale**:
- **Unified Concept**: Eliminates confusion between "service" and "module"; everything is a namespace layer.
- **Flexibility**: Arbitrary depth vs. fixed two-level structures.
- **Observability**: Full paths enable queries like "filter logs from payment.processor.*".
- **Consistency**: Single `Namespace()` method for all levels, reducing API surface.

Compared to traditional systems:

| Aspect          | Traditional (Service + Module) | Hierarchical Namespaces |
|-----------------|--------------------------------|--------------------------|
| API Count      | 2 (WithService + Module)      | 1 (WithNamespace + Namespace) |
| Concept Complexity | High (blurry boundaries)     | Low (unified)           |
| Extensibility  | Poor (fixed layers)           | Strong (arbitrary depth)|
| Readability    | Medium                        | High (path-like)        |

#### 3. Type-Safe TraceID Management

**Purpose**: Enables distributed tracing by linking logs across services without manual propagation.

**Core Functions**:
- `WithTraceID(ctx context.Context, traceID string) context.Context`: Injects traceID into ctx using a private struct key (avoids string key collisions).
- `WithContext(ctx context.Context) Logger`: Extracts traceID if present and returns a logger with it as a field (`zap.String("trace_id", id)`). Falls back to default if absent.
- `C(ctx context.Context) Logger`: Alias for `WithContext` for brevity.

**Implementation**:
- Private key: `var traceIDKey struct{}` ensures type safety.
- Extraction uses direct type assertion (no reflection) for performance.
- New ctx created (immutable, concurrent-safe).

**Workflow**:
1. Middleware/Interceptor: `ctx = WithTraceID(originalCtx, traceID)`.
2. Business code: `logger = WithContext(ctx)` (auto-adds trace_id).

**Rationale**:
- **Encapsulation**: Hides key details; users don't manage context values manually.
- **Type Safety**: Compile-time checks prevent errors like wrong types.
- **API Symmetry**: Inject (WithTraceID) + Extract (WithContext) form a complete, intuitive pair.
- **Isolation**: Per-request ctx ensures no cross-request leakage.
- **Performance**: Zero runtime overhead beyond zap field addition.

This follows Go best practices for context (e.g., no globals, immutable propagation).

#### 4. Configuration System

**Purpose**: Centralizes all config logic for maintainability.

**Core Struct**:
```go
type Config struct {
    Level       string           `json:"level"`      // Log level
    Format      string           `json:"format"`     // "json" or "console"
    Output      string           `json:"output"`     // Target (stdout/file)
    AddSource   bool             `json:"add_source"` // Include file:line
    EnableColor bool             `json:"enable_color"`
    RootPath    string           `json:"root_path"`  // For relative paths
    Rotation    *RotationConfig  `json:"rotation"`   // File rotation
}

type RotationConfig struct {
    MaxSize    int  `json:"max_size"`    // MB
    MaxBackups int  `json:"max_backups"`
    MaxAge     int  `json:"max_age"`     // Days
    Compress   bool `json:"compress"`
}
```

**Implementation**:
- Options parsed via functional pattern: `type Option func(*options)`.
- Validation on load (e.g., invalid level → error).
- Rotation uses lumberjack for file management (if Output is file).

**Rationale**: Concentrating config in `config.go` separates concerns, eases testing, and supports future extensions like hot-reloading.

## 🔧 Key Technical Decisions

### 1. Hierarchical Namespaces over Module System
- **Why?** Reduces API duplication, unifies concepts, and supports flexible depths for microservices. Traditional two-layer limits scalability in complex apps like GoChat.

### 2. Type-Safe Context Keys
- **Why?** Prevents runtime panics from type mismatches or key collisions. Encapsulates internals, aligning with Go's emphasis on safety and simplicity.

### 3. Centralized Config
- **Why?** Avoids scattered config code, improves maintainability, and enables unified validation/parsing.

### 4. Zap as Foundation
- **Why?** Proven zero-allocation performance, rich ecosystem (JSON/console encoders), and structured fields. Minimal wrapping preserves speed.

## 🎨 Design Patterns Applied

1. **Provider Pattern**: For initialization (New/Init), ensuring testability and DI.
2. **Functional Options**: Extensible config without breaking changes (e.g., add WithEncoder later).
3. **Singleton**: Global logger with atomic replacement for thread-safety and hot-updates.
4. **Decorator**: Namespace() wraps loggers, adding fields without altering core behavior.
5. **Adapter**: Wraps zap.Logger to enforce clog's interface and add features like traceID.

## 🚀 Performance Strategies

1. **Zero-Allocation Logging**: Direct zap.Field usage; no intermediate structs.
2. **Lazy Initialization**: Singletons load on first use.
3. **Efficient Fields**: TraceID/namespace added once per logger, not per log.
4. **No Reflection**: Type assertions for context extraction.
5. **Benchmarked**: Targets <1% overhead in hot paths (e.g., Info calls).

## 📊 Backward Compatibility & Migration

### Breaking Changes
- `Module()` → `Namespace()`: Unified API.
- Init/New signatures: Added ctx/opts for Provider compliance.
- TraceID: `context.WithValue(..., "traceID", ...)` → `WithTraceID()` for safety.
- Removed hooks like `SetTraceIDHook()`: Simplified to context-based.

### Migration Guide
1. **Namespaces**: Replace `Module("user")` with `Namespace("user")`.
2. **Init**: Add `context.Background()` and `&config`; use `WithNamespace("service")`.
3. **TraceID**: Use `WithTraceID(ctx, id)` in middleware; `WithContext(ctx)` in handlers.
4. **Globals**: Existing code works if no breaking APIs used; update for new features.

No runtime breaks for non-updated code; gradual migration supported.

## 🔮 Future Extensions

1. **Config Center**: etcd integration for dynamic levels/formats.
2. **Advanced Options**: Custom encoders, outputs, hooks.
3. **Monitoring**: Metrics (log rate, errors), auto-alerts.
4. **Tracing**: OpenTelemetry spans, auto-propagation.
5. **Sampling**: Rate-limiting for high-volume logs.

This design balances simplicity, power, and performance, making clog ideal for GoChat's observable, distributed systems.

For API reference and examples, see [README.md](README.md).
