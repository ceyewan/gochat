# 基础设施: UID 唯一ID生成

## 1. 概述

`uid` 是 `gochat` 项目中用于生成唯一标识符的组件集合。它提供了两种独立的、业界标准的 ID 生成方案，以满足不同场景的需求：

1.  **雪花算法 (Snowflake)**: 生成一个 `int64` 类型的、趋势递增的分布式唯一 ID。非常适合用作数据库主键、消息 ID 等需要排序和高性能的场景。
2.  **UUID v7**: 生成一个字符串类型的、时间有序的通用唯一标识符。非常适合用作对外暴露的资源 ID、请求 ID 等需要全局唯一且不希望暴露内部信息的场景。

`uid` 组件经过重构，将这两种方案完全解耦，提供了更清晰、更安全的 API。

## 2. 雪花算法 (Snowflake)

雪花算法是**有状态的**，它的创建依赖于一个全局唯一的 `instanceID`。

### 2.1 用法

```go
import "github.com/ceyewan/gochat/im-infra/uid/snowflake"

// 1. 从 coord 获取 instanceID (这是必须的前置步骤)
// "my-service" 是你的服务名, 1023 是ID上限
idAllocator, err := coordinator.InstanceIDAllocator("my-service", 1023)
if err != nil {
    // ... handle error
}
instanceID, err := idAllocator.AcquireID(ctx)
if err != nil {
    // ... handle error
}

// 2. 使用获取到的 instanceID 创建雪花算法生成器
snowGen, err := snowflake.New(int64(instanceID))
if err != nil {
    // ... handle error
}

// 3. 生成 ID
messageID := snowGen.Generate() // 返回 int64
```

### 2.2 接口定义

```go
package snowflake

// New 创建一个新的雪花算法生成器。
// instanceID 必须是通过 coord.InstanceIDAllocator 获取的唯一ID。
func New(instanceID int64) (Generator, error)

// Generator 定义了雪花算法生成器的接口。
type Generator interface {
	Generate() int64
}

// Parse 解析一个雪花ID，返回其组成部分。
// 这是一个独立的工具函数。
func Parse(id int64) (timestamp, instanceID, sequence int64)
```

## 3. UUID v7

UUID 的生成是**无状态的**，因此直接提供包级别函数，无需创建实例。

### 3.1 用法

```go
import "github.com/ceyewan/gochat/im-infra/uid/uuid"

// 直接调用函数生成 UUID v7 字符串
requestID := uuid.NewV7()

// 验证一个 UUID 字符串的格式
isValid := uuid.IsValid(requestID)
```

### 3.2 接口定义

```go
package uuid

// NewV7 生成一个符合 RFC 规范的、时间有序的 UUID v7 字符串。
func NewV7() string

// IsValid 检查一个字符串是否是合法的 UUID 格式。
func IsValid(s string) bool