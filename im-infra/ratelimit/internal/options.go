package internal

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	coordination "github.com/ceyewan/gochat/im-infra/coord"
)

// Options 用于配置 RateLimiter 的行为
type Options struct {
	// CacheClient 缓存客户端，用于存储令牌桶数据
	CacheClient cache.Cache

	// CoordinationClient 协调客户端，用于配置管理
	CoordinationClient coordination.Provider

	// RuleRefreshInterval 规则刷新间隔，默认为1分钟
	RuleRefreshInterval time.Duration

	// DefaultRules 默认规则，当配置中心不可用时使用
	DefaultRules map[string]Rule

	// EnableMetrics 是否启用指标收集，默认为true
	EnableMetrics bool

	// EnableStatistics 是否启用统计功能，默认为true
	EnableStatistics bool

	// BatchSize 批量处理大小，默认为100
	BatchSize int

	// ScriptCacheEnabled 是否启用脚本缓存，默认为true
	ScriptCacheEnabled bool

	// KeyPrefix 限流键前缀，默认为"ratelimit"
	KeyPrefix string

	// DefaultTTL 默认过期时间，用于清理无用的限流数据
	DefaultTTL time.Duration

	// FailurePolicy 失败策略：允许（allow）或拒绝（deny），默认为允许
	FailurePolicy FailurePolicy

	// MaxRetries 最大重试次数，默认为3
	MaxRetries int

	// RetryDelay 重试延迟，默认为100ms
	RetryDelay time.Duration
}

// Rule 定义了单个限流规则
type Rule struct {
	// Rate 每秒生成的令牌数，可以为小数
	Rate float64 `json:"rate"`

	// Capacity 令牌桶的最大容量，即允许的突发请求峰值
	Capacity int64 `json:"capacity"`
}

// FailurePolicy 定义失败时的策略
type FailurePolicy int

const (
	// FailurePolicyAllow 失败时允许请求通过（默认策略）
	FailurePolicyAllow FailurePolicy = iota
	// FailurePolicyDeny 失败时拒绝请求
	FailurePolicyDeny
)

// Option 是一个函数，用于修改 Options 结构体
type Option func(*Options)

// WithCacheClient 设置自定义的缓存客户端
func WithCacheClient(client cache.Cache) Option {
	return func(o *Options) {
		o.CacheClient = client
	}
}

// WithCoordinationClient 设置自定义的协调客户端
func WithCoordinationClient(client coordination.Provider) Option {
	return func(o *Options) {
		o.CoordinationClient = client
	}
}

// WithRuleRefreshInterval 设置规则刷新间隔
func WithRuleRefreshInterval(interval time.Duration) Option {
	return func(o *Options) {
		o.RuleRefreshInterval = interval
	}
}

// WithDefaultRules 设置备用规则
func WithDefaultRules(rules map[string]Rule) Option {
	return func(o *Options) {
		o.DefaultRules = rules
	}
}

// WithMetricsEnabled 设置是否启用指标收集
func WithMetricsEnabled(enabled bool) Option {
	return func(o *Options) {
		o.EnableMetrics = enabled
	}
}

// WithStatisticsEnabled 设置是否启用统计功能
func WithStatisticsEnabled(enabled bool) Option {
	return func(o *Options) {
		o.EnableStatistics = enabled
	}
}

// WithBatchSize 设置批量处理大小
func WithBatchSize(size int) Option {
	return func(o *Options) {
		o.BatchSize = size
	}
}

// WithScriptCacheEnabled 设置是否启用脚本缓存
func WithScriptCacheEnabled(enabled bool) Option {
	return func(o *Options) {
		o.ScriptCacheEnabled = enabled
	}
}

// WithKeyPrefix 设置限流键前缀
func WithKeyPrefix(prefix string) Option {
	return func(o *Options) {
		o.KeyPrefix = prefix
	}
}

// WithDefaultTTL 设置默认过期时间
func WithDefaultTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.DefaultTTL = ttl
	}
}

// WithFailurePolicy 设置失败策略
func WithFailurePolicy(policy FailurePolicy) Option {
	return func(o *Options) {
		o.FailurePolicy = policy
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(retries int) Option {
	return func(o *Options) {
		o.MaxRetries = retries
	}
}

// WithRetryDelay 设置重试延迟
func WithRetryDelay(delay time.Duration) Option {
	return func(o *Options) {
		o.RetryDelay = delay
	}
}

// applyDefaults 应用默认配置
func (o *Options) applyDefaults() {
	if o.RuleRefreshInterval == 0 {
		o.RuleRefreshInterval = time.Minute
	}

	if o.DefaultRules == nil {
		o.DefaultRules = make(map[string]Rule)
	}

	if o.BatchSize == 0 {
		o.BatchSize = 100
	}

	if o.KeyPrefix == "" {
		o.KeyPrefix = "ratelimit"
	}

	if o.DefaultTTL == 0 {
		o.DefaultTTL = 24 * time.Hour // 默认24小时过期
	}

	if o.MaxRetries == 0 {
		o.MaxRetries = 3
	}

	if o.RetryDelay == 0 {
		o.RetryDelay = 100 * time.Millisecond
	}

	// 设置默认的功能开关状态
	// 注意：这些值在没有显式设置时将使用默认值
}
