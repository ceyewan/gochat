package internal

import (
	"context"
	"time"
)

// RateLimiter 是限流器的主接口
// 它定义了检查请求是否被允许的核心方法
type RateLimiter interface {
	// Allow 检查给定资源的单个请求是否被允许
	Allow(ctx context.Context, resource string, ruleName string) (bool, error)

	// AllowN 检查给定资源的N个请求是否被允许
	AllowN(ctx context.Context, resource string, ruleName string, n int64) (bool, error)

	// BatchAllow 批量处理限流请求
	BatchAllow(ctx context.Context, requests []RateLimitRequest) ([]bool, error)

	// GetStatistics 获取限流统计信息
	GetStatistics(ctx context.Context, resource string, ruleName string) (*RateLimitStatistics, error)

	// Close 关闭限流器并释放资源
	Close() error
}

// RateLimiterManager 扩展接口，提供管理功能
type RateLimiterManager interface {
	RateLimiter

	// SetRule 动态设置限流规则
	SetRule(ctx context.Context, ruleName string, rule Rule) error

	// ListRules 获取当前所有规则
	ListRules() map[string]Rule

	// DeleteRule 删除限流规则
	DeleteRule(ctx context.Context, ruleName string) error

	// ExportRules 导出规则到配置中心
	ExportRules(ctx context.Context) error

	// ReloadRules 重新加载配置中心的规则
	ReloadRules() error

	// GetServiceName 获取服务名称
	GetServiceName() string
}

// BatchIdempotent 定义批量幂等操作的接口
type BatchIdempotent interface {
	// BatchCheck 批量检查键是否存在
	BatchCheck(ctx context.Context, keys []string) (map[string]bool, error)

	// BatchSet 批量设置幂等标记
	BatchSet(ctx context.Context, keys []string, ttl time.Duration) (map[string]bool, error)

	// BatchDelete 批量删除幂等标记
	BatchDelete(ctx context.Context, keys []string) error
}

// ResultIdempotent 定义带结果存储的幂等操作接口
type ResultIdempotent interface {
	// SetWithResult 设置幂等标记并存储操作结果
	SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error)

	// GetResult 获取存储的操作结果
	GetResult(ctx context.Context, key string) (interface{}, error)

	// GetResultWithStatus 获取结果和状态
	GetResultWithStatus(ctx context.Context, key string) (*IdempotentResult, error)
}

// AdvancedIdempotent 定义高级幂等操作的接口
type AdvancedIdempotent interface {
	// Do 执行幂等操作，如果已执行过则跳过，否则执行函数
	Do(ctx context.Context, key string, f func() error) error

	// Execute 执行幂等操作并返回结果
	Execute(ctx context.Context, key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error)

	// ExecuteSimple 执行简单幂等操作
	ExecuteSimple(ctx context.Context, key string, ttl time.Duration, callback func() error) error
}

// Metrics 定义指标收集接口
type Metrics interface {
	// RecordRequest 记录请求指标
	RecordRequest(ctx context.Context, service, rule, resource string, allowed bool, remainingTokens, totalRequests, allowedRequests int64)

	// RecordLatency 记录延迟指标
	RecordLatency(ctx context.Context, service, rule string, duration time.Duration)

	// GetMetrics 获取指标数据
	GetMetrics(ctx context.Context) (map[string]interface{}, error)
}

// IdempotentResult 定义幂等操作的结果
type IdempotentResult struct {
	// Status 幂等状态
	Status IdempotentStatus
	// Result 操作结果
	Result interface{}
	// Error 错误信息
	Error error
	// CreatedAt 创建时间
	CreatedAt time.Time
	// UpdatedAt 更新时间
	UpdatedAt time.Time
}

// IdempotentStatus 定义幂等状态
type IdempotentStatus int

const (
	// StatusNotExecuted 未执行状态
	StatusNotExecuted IdempotentStatus = iota
	// StatusExecuting 执行中状态
	StatusExecuting
	// StatusExecuted 已执行状态
	StatusExecuted
	// StatusFailed 执行失败状态
	StatusFailed
)

// RateLimitRequest 批量限流请求
type RateLimitRequest struct {
	Resource string
	RuleName string
	Count    int64
}

// RateLimitStatistics 限流统计信息
type RateLimitStatistics struct {
	Resource        string    `json:"resource"`
	RuleName        string    `json:"rule_name"`
	TotalRequests   int64     `json:"total_requests"`
	AllowedRequests int64     `json:"allowed_requests"`
	DeniedRequests  int64     `json:"denied_requests"`
	CurrentTokens   int64     `json:"current_tokens"`
	SuccessRate     float64   `json:"success_rate"`
	LastUpdated     time.Time `json:"last_updated"`
	WindowStart     time.Time `json:"window_start"`
}

// TokenBucketState 令牌桶状态
type TokenBucketState struct {
	Tokens         float64   `json:"tokens"`
	LastRefillTime time.Time `json:"last_refill_time"`
	Rate           float64   `json:"rate"`
	Capacity       int64     `json:"capacity"`
}

// RuleValidator 规则验证器接口
type RuleValidator interface {
	// ValidateRule 验证规则
	ValidateRule(rule Rule) error
}

// ConfigProvider 配置提供者接口
type ConfigProvider interface {
	// GetRules 获取所有规则
	GetRules(ctx context.Context) (map[string]Rule, error)

	// GetRule 获取单个规则
	GetRule(ctx context.Context, ruleName string) (Rule, bool, error)

	// SetRule 设置规则
	SetRule(ctx context.Context, ruleName string, rule Rule) error

	// DeleteRule 删除规则
	DeleteRule(ctx context.Context, ruleName string) error

	// Watch 监听配置变化
	Watch(ctx context.Context) (<-chan ConfigEvent, error)
}

// ConfigEvent 配置变更事件
type ConfigEvent struct {
	Type     ConfigEventType
	RuleName string
	Rule     Rule
}

// ConfigEventType 配置事件类型
type ConfigEventType int

const (
	ConfigEventTypeAdd ConfigEventType = iota
	ConfigEventTypeUpdate
	ConfigEventTypeDelete
)

// CacheKeyBuilder 缓存键构建器接口
type CacheKeyBuilder interface {
	// BuildKey 构建缓存键
	BuildKey(service, rule, resource string) string

	// ParseKey 解析缓存键
	ParseKey(key string) (service, rule, resource string, err error)
}

// ScriptExecutor 脚本执行器接口
type ScriptExecutor interface {
	// Execute 执行脚本
	Execute(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)

	// ExecuteWithSHA 使用 SHA 执行脚本
	ExecuteWithSHA(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error)

	// LoadScript 加载脚本
	LoadScript(ctx context.Context, script string) (string, error)
}
