gochat/im-infra/es/DESIGN.md
# ES 包设计文档

## 概述

ES 包是 GoChat 基础设施的核心组件，提供 Elasticsearch 的索引和搜索功能。该包采用模块化设计，封装 Elasticsearch 客户端的复杂性，为上层业务提供简洁的泛型 API。

## 当前设计思想

### 核心原则
- **职责单一**：专注于 Elasticsearch 封装，不涉及业务逻辑
- **泛型支持**：使用 Go 泛型支持任意可索引数据结构
- **简单直观**：用户只需导入一个包即可使用
- **依赖隔离**：避免循环依赖，保持清晰的包结构

### 架构设计
```
es/
├── es.go              # 主入口 + Provider 实现
├── provider.go        # Provider 接口定义
├── config.go          # 配置类型别名
├── options.go         # 选项模式
└── internal/          # 内部工具包
    ├── client.go      # ES 客户端封装
    └── config.go      # 内部配置
```

### 依赖关系
```
es 主包 → es/internal 工具包 (单向依赖，无循环)
```

## 方案对比

### 方案 1：接口实现同包（当前方案）✅

**优点**：
- 包结构简单，用户只需导入一个包
- 符合 Go 标准库设计模式（如 `encoding/json`）
- 快速上手，API 直观
- 适合功能相对固定的工具包

**缺点**：
- 接口和实现耦合，不易扩展
- 难以支持多种实现（如不同搜索引擎后端）
- 不适合需要插件化的复杂场景

**适用场景**：
- 功能相对稳定，不需要多种实现
- 主要用于封装第三方服务
- 追求简单性和易用性

### 方案 2：独立接口包（coord 包模式）

**优点**：
- 接口和实现完全解耦，支持插件化
- 便于测试和依赖注入
- 支持多种实现（如 ES, OpenSearch, Solr）
- 符合企业级架构设计模式

**缺点**：
- 包结构复杂，用户需要了解多个包
- 学习曲线较陡
- 可能过度设计

**适用场景**：
- 需要支持多种后端实现
- 复杂分布式系统
- 企业级框架开发

## 重构指南：从同包模式到独立接口包

### 步骤 1：创建接口包
```bash
# 创建独立的接口包
mkdir -p interfaces
touch interfaces/provider.go
```

### 步骤 2：移动接口定义
```go
// interfaces/provider.go
package interfaces

import "context"

type Indexable interface {
    GetID() string
}

type Provider[T Indexable] interface {
    BulkIndex(ctx context.Context, index string, items []T) error
    SearchGlobal(ctx context.Context, index, keyword string, page, size int) (*SearchResult[T], error)
    SearchInSession(ctx context.Context, index, sessionID, keyword string, page, size int) (*SearchResult[T], error)
    Close() error
}

type SearchResult[T Indexable] struct {
    Total int64
    Items []*T
}
```

### 步骤 3：重构主包
```go
// es.go
package es

import (
    "github.com/ceyewan/gochat/im-infra/es/interfaces"
    "github.com/ceyewan/gochat/im-infra/es/internal"
)

// 重新导出接口，便于用户使用
type Indexable = interfaces.Indexable
type Provider[T Indexable] = interfaces.Provider[T]
type SearchResult[T Indexable] = interfaces.SearchResult[T]

// 实现工厂函数
func New[T Indexable](ctx context.Context, cfg *Config, opts ...Option) (interfaces.Provider[T], error) {
    // 实现逻辑保持不变
    return &provider[T]{...}, nil
}
```

### 步骤 4：更新实现包
```go
// internal/provider.go (如果需要)
package internal

import "github.com/ceyewan/gochat/im-infra/es/interfaces"

type provider[T interfaces.Indexable] struct {
    // 实现细节
}
```

### 步骤 5：更新用户代码
```go
// 旧代码
import "github.com/ceyewan/gochat/im-infra/es"

// 新代码（接口重导出，无需修改）
import "github.com/ceyewan/gochat/im-infra/es"
```

## 决策建议

### 当前阶段：保持同包模式
- ES 包功能相对稳定，主要封装 Elasticsearch
- 用户体验简单，无需了解复杂包结构
- 符合当前项目规模和复杂度

### 未来重构时机
- 当需要支持多种搜索引擎后端时
- 当项目复杂度显著增加时
- 当需要插件化架构时

### 重构成本评估
- **低成本**：接口重导出保持向后兼容
- **渐进式**：可以逐步迁移，不影响现有代码
- **可逆转**：如果独立接口包过于复杂，可以回退

## 总结

当前的设计在简单性和易用性之间取得了良好平衡。未来如果需要支持多种搜索引擎后端，可以考虑重构为独立接口包模式，但这需要根据实际业务需求来决定。

**设计原则**：没有最好的设计，只有最适合当前场景的设计。