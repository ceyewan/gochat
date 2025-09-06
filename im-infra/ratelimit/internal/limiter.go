package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
	coordination "github.com/ceyewan/gochat/im-infra/coord"
)

// limiter 是 RateLimiter 接口的内部实现
type limiter struct {
	serviceName string
	opts        Options
	logger      clog.Logger
	mu          sync.RWMutex
	rules       map[string]Rule
	ctx         context.Context
	cancel      context.CancelFunc
	bucket      *tokenBucket
}

var (
	defaultLimiter     RateLimiter
	defaultLimiterOnce sync.Once
)

// New 创建一个新的限流器实例
func New(ctx context.Context, serviceName string, opts ...Option) (RateLimiter, error) {
	// 应用默认选项
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	// 应用默认配置
	options.applyDefaults()

	// 如果没有提供客户端，则使用默认的
	if options.CacheClient == nil {
		defaultCacheClient, err := cache.New(ctx, cache.DefaultConfig())
		if err != nil {
			return nil, fmt.Errorf("failed to create default cache client: %w", err)
		}
		options.CacheClient = defaultCacheClient
	}
	if options.CoordinationClient == nil {
		defaultCoordClient, err := coordination.New(ctx, coordination.DefaultConfig())
		if err != nil {
			return nil, fmt.Errorf("failed to create default coordination client: %w", err)
		}
		options.CoordinationClient = defaultCoordClient
	}

	limiterCtx, cancel := context.WithCancel(ctx)

	l := &limiter{
		serviceName: serviceName,
		opts:        options,
		logger:      clog.Module("ratelimit"),
		rules:       make(map[string]Rule),
		ctx:         limiterCtx,
		cancel:      cancel,
		bucket:      newTokenBucket(options.CacheClient),
	}

	// 初始加载规则
	if err := l.loadRules(); err != nil {
		l.logger.Warn("初始化加载规则失败，使用默认规则", clog.Err(err))
		l.mu.Lock()
		l.rules = options.DefaultRules
		l.mu.Unlock()
	}

	// 启动后台规则刷新
	l.startRuleRefresher()

	return l, nil
}

// Default 返回全局单例限流器
func Default() RateLimiter {
	defaultLimiterOnce.Do(func() {
		var err error
		defaultLimiter, err = New(context.Background(), "default")
		if err != nil {
			clog.Error("创建默认限流器失败", clog.Err(err))
		}
	})
	return defaultLimiter
}

// Allow 检查给定资源的单个请求是否被允许
func (l *limiter) Allow(ctx context.Context, resource string, ruleName string) (bool, error) {
	return l.AllowN(ctx, resource, ruleName, 1)
}

// AllowN 检查给定资源的N个请求是否被允许
func (l *limiter) AllowN(ctx context.Context, resource string, ruleName string, n int64) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	// 获取规则
	rule, ok := l.getRule(ruleName)
	if !ok {
		l.logger.Warn("未找到限流规则，默认允许",
			clog.String("ruleName", ruleName),
			clog.String("resource", resource))
		return true, nil
	}

	// 构建 Redis Key
	key := fmt.Sprintf("ratelimit:%s:%s:%s", l.serviceName, ruleName, resource)

	// 执行令牌桶算法
	allowed, _, _, _, err := l.bucket.take(ctx, key, rule, n)
	if err != nil {
		l.logger.Error("执行限流脚本失败，默认允许",
			clog.String("key", key),
			clog.Int64("requested", n),
			clog.Err(err))
		// 出错时默认允许，保证系统可用性
		return true, err
	}

	l.logger.Debug("限流检查完成",
		clog.String("key", key),
		clog.Bool("allowed", allowed),
		clog.Int64("requested", n))

	return allowed, nil
}

// BatchAllow 批量处理限流请求
func (l *limiter) BatchAllow(ctx context.Context, requests []RateLimitRequest) ([]bool, error) {
	if len(requests) == 0 {
		return []bool{}, nil
	}

	results := make([]bool, len(requests))
	for i, req := range requests {
		count := req.Count
		if count <= 0 {
			count = 1
		}

		allowed, err := l.AllowN(ctx, req.Resource, req.RuleName, count)
		if err != nil {
			return nil, fmt.Errorf("批量请求第%d个失败: %w", i, err)
		}
		results[i] = allowed
	}

	return results, nil
}

// GetStatistics 获取限流统计信息
func (l *limiter) GetStatistics(ctx context.Context, resource string, ruleName string) (*RateLimitStatistics, error) {
	key := fmt.Sprintf("ratelimit:%s:%s:%s", l.serviceName, ruleName, resource)

	bucketStats, err := l.bucket.getStatistics(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("获取统计信息失败: %w", err)
	}

	stats := &RateLimitStatistics{
		Resource:        resource,
		RuleName:        ruleName,
		TotalRequests:   bucketStats.TotalRequests,
		AllowedRequests: bucketStats.AllowedRequests,
		DeniedRequests:  bucketStats.DeniedRequests,
		CurrentTokens:   bucketStats.CurrentTokens,
		SuccessRate:     bucketStats.SuccessRate,
		LastUpdated:     time.Now(),
	}

	return stats, nil
}

// Close 停止后台goroutine并释放资源
func (l *limiter) Close() error {
	l.cancel()
	l.logger.Info("限流器已关闭")
	return nil
}

// getRule 获取限流规则
// func (l *limiter) getRule(name string) (Rule, bool) {
// 	l.mu.RLock()
// 	defer l.mu.RUnlock()

// 	rule, ok := l.rules[name]
// 	if !ok {
// 		// 如果在内存中找不到，尝试从默认选项中查找
// 		rule, ok = l.opts.DefaultRules[name]
// 	}
// 	return rule, ok
// }

// SetRule 动态设置限流规则（公开方法）
func (l *limiter) SetRule(ctx context.Context, ruleName string, rule Rule) error {
	return l.setRule(ctx, ruleName, rule)
}

// ListRules 获取当前所有规则（公开方法）
func (l *limiter) ListRules() map[string]Rule {
	return l.listRules()
}

// DeleteRule 删除限流规则（公开方法）
func (l *limiter) DeleteRule(ctx context.Context, ruleName string) error {
	return l.deleteRule(ctx, ruleName)
}

// ExportRules 导出规则到配置中心（公开方法）
func (l *limiter) ExportRules(ctx context.Context) error {
	return l.exportRules(ctx)
}

// ReloadRules 重新加载配置中心的规则（公开方法）
func (l *limiter) ReloadRules() error {
	return l.loadRules()
}

// GetServiceName 获取服务名称
func (l *limiter) GetServiceName() string {
	return l.serviceName
}
