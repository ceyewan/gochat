# RateLimit 组件设计文档

## 概述

RateLimit 是一个基于 Redis 令牌桶算法的高性能分布式限流组件，专为 GoChat 系统设计。该组件支持动态配置、多维度限流、批量操作和实时监控，提供企业级的限流解决方案。

## 设计目标

- **高性能**: 基于 Redis Lua 脚本实现原子操作，支持高并发场景
- **分布式**: 天然支持分布式架构，适用于微服务集群
- **动态配置**: 与配置中心集成，支持实时调整限流规则
- **多维度**: 支持基于用户、IP、API 等多维度的限流策略
- **易扩展**: 模块化设计，支持自定义限流算法和存储后端
- **可观测**: 内置监控指标和统计功能，便于运维管理

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      RateLimit Module                       │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│ │ RateLimiter │ │ BatchOps    │ │ Metrics     │           │
│ │ Interface   │ │ Interface   │ │ Interface   │           │
│ └──────┬──────┘ └──────┬──────┘ └──────┬──────┘           │
├────────┼──────────────┼──────────────┼─────────────────────┤
│        │              │              │                     │
│ ┌──────┴──────┐ ┌──────┴──────┐ ┌──────┴──────┐           │
│ │Token Bucket │ │Batch Impl  │ │Statistics  │           │
│ │Algorithm    │ │            │ │Collector   │           │
│ └──────┬──────┘ └──────┬──────┘ └──────┬──────┘           │
├────────┼──────────────┼──────────────┼─────────────────────┤
│        │              │              │                     │
│ ┌──────┴─────────────────────────────────────┐             │
│ │         Redis Client Wrapper               │             │
│ │    (Connection Pool, Script Cache)         │             │
│ └──────┬─────────────────────────────────────┘             │
├────────┼───────────────────────────────────────────────────┤
│        │                                                   │
│ ┌──────┴──────┐                                             │
│ │    Redis    │                                             │
│ │  Cluster    │                                             │
│ └─────────────┘                                             │
└─────────────────────────────────────────────────────────────┘
```

### 核心组件

#### 1. 限流器 (RateLimiter)

主限流器接口，提供核心的限流功能：

```go
type RateLimiter interface {
    Allow(ctx context.Context, resource string, ruleName string) (bool, error)
    AllowN(ctx context.Context, resource string, ruleName string, n int64) (bool, error)
    BatchAllow(ctx context.Context, requests []RateLimitRequest) ([]bool, error)
    Close() error
}
```

#### 2. 令牌桶算法 (Token Bucket)

基于 Redis Lua 脚本的原子性令牌桶实现：

```go
// 核心算法参数
type Rule struct {
    Rate        float64       // 令牌产生速率 (tokens/second)
    Capacity    int64         // 桶容量 (最大突发流量)
    Burst       int64         // 突发容量 (可选)
    Window      time.Duration // 时间窗口 (用于滑动窗口)
    Description string        // 规则描述
}
```

#### 3. 配置管理 (Config Management)

与 etcd 集成的动态配置系统：

```
/config/{env}/{service}/ratelimit/{ruleName}
{
    "rate": 10.0,
    "capacity": 20,
    "burst": 5,
    "window": "1s",
    "description": "API限流规则"
}
```

#### 4. 批量操作 (Batch Operations)

支持批量限流请求，减少网络开销：

```go
type BatchIdempotent interface {
    BatchCheck(ctx context.Context, keys []string) (map[string]bool, error)
    BatchAllow(ctx context.Context, requests []RateLimitRequest) ([]bool, error)
}
```

## 技术实现

### 令牌桶算法

#### 核心原理

令牌桶算法维护一个固定容量的桶，以恒定速率向桶中添加令牌：

1. **令牌生成**: 以固定速率 `rate` 产生令牌
2. **令牌消耗**: 请求到达时从桶中取出相应数量的令牌
3. **桶容量**: 当桶满时，多余的令牌会被丢弃
4. **限流判断**: 如果桶中没有足够令牌，请求被拒绝

#### Lua 脚本实现

```lua
-- 令牌桶算法 Lua 脚本
local key = KEYS[1]
local rate = tonumber(ARGV[1])      -- 令牌产生速率
local capacity = tonumber(ARGV[2])  -- 桶容量
local now = tonumber(ARGV[3])       -- 当前时间戳
local requested = tonumber(ARGV[4]) -- 请求令牌数

-- 获取当前状态
local bucket = redis.call('hgetall', key)
local tokens
local last_refill_ts

if #bucket == 0 then
    -- 桶不存在，创建新桶
    tokens = capacity
    last_refill_ts = now
else
    -- 解析现有状态
    for i = 1, #bucket, 2 do
        if bucket[i] == 'tokens' then
            tokens = tonumber(bucket[i+1])
        elseif bucket[i] == 'last_refill_ts' then
            last_refill_ts = tonumber(bucket[i+1])
        end
    end
end

-- 计算时间间隔并补充令牌
local elapsed = (now - last_refill_ts) / 1e9  -- 转换为秒
local new_tokens = elapsed * rate
tokens = math.min(capacity, tokens + new_tokens)
last_refill_ts = now

-- 判断是否允许请求
local allowed = 0
if tokens >= requested then
    tokens = tokens - requested
    allowed = 1
end

-- 更新状态
redis.call('hset', key, 'tokens', tokens, 'last_refill_ts', last_refill_ts)

return {allowed, math.floor(tokens)}
```

### 配置加载机制

#### 动态配置刷新

```go
func (l *limiter) startRuleRefresher() {
    go func() {
        ticker := time.NewTicker(l.opts.RuleRefreshInterval)
        defer ticker.Stop()
        
        for {
            select {
            case <-l.ctx.Done():
                return
            case <-ticker.C:
                if err := l.loadRules(); err != nil {
                    l.logger.Error(