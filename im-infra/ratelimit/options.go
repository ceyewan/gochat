package ratelimit

import "github.com/ceyewan/gochat/im-infra/ratelimit/internal"

// Options 是 RateLimiter 的配置选项 (类型别名)。
type Options = internal.Options

// Rule 定义了单个限流规则 (类型别名)。
type Rule = internal.Rule

// Option 是一个用于修改 Options 的函数 (类型别名)。
type Option = internal.Option

// WithCacheClient 设置自定义的缓存客户端。
var WithCacheClient = internal.WithCacheClient

// WithCoordinationClient 设置自定义的协调客户端。
var WithCoordinationClient = internal.WithCoordinationClient

// WithRuleRefreshInterval 设置规则刷新间隔。
var WithRuleRefreshInterval = internal.WithRuleRefreshInterval

// WithDefaultRules 设置备用规则。
var WithDefaultRules = internal.WithDefaultRules
