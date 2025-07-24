package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coordination"
)

// limiter 是 RateLimiter 接口的内部实现。
type limiter struct {
	serviceName string
	opts        Options
	logger      clog.Logger
	mu          sync.RWMutex
	rules       map[string]Rule
	ctx         context.Context
	cancel      context.CancelFunc
}

var (
	defaultLimiter     RateLimiter
	defaultLimiterOnce sync.Once
)

// New 创建一个新的限流器实例。
func New(ctx context.Context, serviceName string, opts ...Option) (RateLimiter, error) {
	// 默认选项
	options := Options{
		RuleRefreshInterval: time.Minute,
	}
	for _, o := range opts {
		o(&options)
	}

	// 如果没有提供客户端，则使用默认的
	if options.CacheClient == nil {
		options.CacheClient = cache.Default()
	}
	if options.CoordinationClient == nil {
		options.CoordinationClient = coordination.Default()
	}

	limiterCtx, cancel := context.WithCancel(ctx)

	l := &limiter{
		serviceName: serviceName,
		opts:        options,
		logger:      clog.Module("ratelimit"),
		rules:       make(map[string]Rule),
		ctx:         limiterCtx,
		cancel:      cancel,
	}

	// 初始化加载一次规则
	if err := l.loadRules(); err != nil {
		// 即使初次加载失败，也只记录日志而不阻塞启动
		// 组件可以依赖默认规则或等待下一次刷新
		l.logger.Error("初始化加载规则失败", clog.Err(err))
	}

	// 启动后台 goroutine 来持续刷新规则
	l.startRuleRefresher()

	return l, nil
}

// Default 返回全局单例限流器。
func Default() RateLimiter {
	defaultLimiterOnce.Do(func() {
		var err error
		defaultLimiter, err = New(context.Background(), "default")
		if err != nil {
			clog.Error("创建默认限流器失败", clog.Err(err))
			// 在真实应用中，这里可能需要 panic 或采取其他恢复措施
		}
	})
	return defaultLimiter
}

// Allow 检查给定资源的请求是否被允许。
func (l *limiter) Allow(ctx context.Context, resource string, ruleName string) (bool, error) {
	// 1. 获取规则
	rule, ok := l.getRule(ruleName)
	if !ok {
		// 如果规则未定义，默认允许通过，并记录警告
		l.logger.Warn("未找到限流规则，默认允许", clog.String("ruleName", ruleName))
		return true, nil
	}

	// 2. 构建 Redis Key
	// 格式: {prefix}:ratelimit:{serviceName}:{ruleName}:{resource}
	key := fmt.Sprintf("ratelimit:%s:%s:%s", l.serviceName, ruleName, resource)

	// 3. 执行 Lua 脚本
	allowed, err := l.executeTokenBucketScript(ctx, key, rule)
	if err != nil {
		l.logger.Error("执行限流脚本失败，默认允许",
			clog.String("key", key),
			clog.Err(err),
		)
		// 在脚本执行失败的情况下，为了系统可用性，选择默认允许通过。
		return true, err
	}

	return allowed, nil
}

// Close 停止后台的规则刷新 goroutine。
func (l *limiter) Close() error {
	l.cancel()
	return nil
}
