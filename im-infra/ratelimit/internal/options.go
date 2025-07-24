package internal

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/coordination"
)

// Options 用于配置 RateLimiter 的行为。
type Options struct {
	CacheClient         cache.Cache
	CoordinationClient  coordination.Coordinator
	RuleRefreshInterval time.Duration
	DefaultRules        map[string]Rule
}

// Rule 定义了单个限流规则。
type Rule struct {
	Rate     float64 `json:"rate"`
	Capacity int64   `json:"capacity"`
}

// Option 是一个函数，用于修改 Options 结构体。
type Option func(*Options)

// WithCacheClient 设置自定义的缓存客户端。
func WithCacheClient(client cache.Cache) Option {
	return func(o *Options) {
		o.CacheClient = client
	}
}

// WithCoordinationClient 设置自定义的协调客户端。
func WithCoordinationClient(client coordination.Coordinator) Option {
	return func(o *Options) {
		o.CoordinationClient = client
	}
}

// WithRuleRefreshInterval 设置规则刷新间隔。
func WithRuleRefreshInterval(interval time.Duration) Option {
	return func(o *Options) {
		o.RuleRefreshInterval = interval
	}
}

// WithDefaultRules 设置备用规则。
func WithDefaultRules(rules map[string]Rule) Option {
	return func(o *Options) {
		o.DefaultRules = rules
	}
}
