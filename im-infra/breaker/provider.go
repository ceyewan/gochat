package breaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// gobreakerAdapter 是 sony/gobreaker 库的适配器
type gobreakerAdapter struct {
	breaker *gobreaker.CircuitBreaker
	name    string
	logger  Logger
}

// provider 是 Provider 接口的具体实现
type provider struct {
	config        *Config
	breakers      map[string]Breaker
	defaultPolicy *Policy
	logger        Logger
	coordProvider CoordProvider
	cancelFunc    context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	closed        bool
}

// New 创建一个新的熔断器 Provider
// 它会自动从 coord 加载所有策略，并监听后续变更
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error) {
	// 验证配置
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	if config.ServiceName == "" {
		return nil, errors.New("serviceName cannot be empty")
	}
	if config.PoliciesPath == "" {
		return nil, errors.New("policiesPath cannot be empty")
	}

	// 解析选项
	options := &providerOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// 设置默认日志器
	if options.logger == nil {
		options.logger = &noopLogger{}
	}

	// 创建子上下文用于后台任务
	childCtx, cancel := context.WithCancel(ctx)

	// 初始化使用默认策略
	policy := GetDefaultPolicy()

	// 创建 provider 实例
	p := &provider{
		config:        config,
		breakers:      make(map[string]Breaker),
		defaultPolicy: policy,
		logger:        options.logger,
		coordProvider: options.coordProvider,
		cancelFunc:    cancel,
		closed:        false,
	}

	// Temporarily disable logging to test
	// p.logger.Info("breaker provider created",
	// 	clog.String("service", config.ServiceName),
	// 	clog.String("policies_path", config.PoliciesPath))

	// 如果有配置中心，启动配置监听
	if options.coordProvider != nil {
		if err := p.startConfigWatcher(childCtx); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to start config watcher: %w", err)
		}
	} else {
		// p.logger.Warn("no coord provider provided, using default policy only")
	}

	return p, nil
}

// GetBreaker 获取或创建一个指定名称的熔断器实例
// name 是被保护资源的唯一标识，例如 "grpc:user-service" 或 "http:payment-api"
// 如果配置中心没有该名称的策略，会使用默认策略
func (p *provider) GetBreaker(name string) Breaker {
	fmt.Printf("DEBUG: GetBreaker called with name: %s\n", name)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		fmt.Printf("DEBUG: Provider is closed\n")
		p.logger.Warn("provider is closed, returning noop breaker")
		return &noopBreaker{}
	}

	fmt.Printf("DEBUG: About to call getOrCreateBreaker\n")
	return p.getOrCreateBreaker(name, p.defaultPolicy)
}

// Close 关闭 Provider，停止所有后台任务
func (p *provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.logger.Info("closing breaker provider")

	// 停止配置监听
	p.cancelFunc()

	// 等待所有后台任务完成
	p.wg.Wait()

	// 清理所有熔断器
	p.breakers = make(map[string]Breaker)

	p.closed = true
	p.logger.Info("breaker provider closed successfully")
	return nil
}

// startConfigWatcher 启动配置监听器
func (p *provider) startConfigWatcher(ctx context.Context) error {
	p.logger.Info("starting config watcher",
		clog.String("path", p.config.PoliciesPath))

	// 首次加载所有策略
	if err := p.loadAllPolicies(ctx); err != nil {
		p.logger.Error("failed to load initial policies", clog.Err(err))
		return err
	}

	// 启动监听
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		// 监听策略路径下的所有变化
		var policy Policy
		watcher, err := p.coordProvider.WatchPrefix(ctx, p.config.PoliciesPath, &policy)
		if err != nil {
			p.logger.Error("failed to start config watcher", clog.Err(err))
			return
		}
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event := <-watcher.Chan():
				p.handleConfigEvent(event)
			}
		}
	}()

	return nil
}

// loadAllPolicies 加载所有策略
func (p *provider) loadAllPolicies(ctx context.Context) error {
	// 列出所有策略文件
	keys, err := p.coordProvider.List(ctx, p.config.PoliciesPath)
	if err != nil {
		p.logger.Error("failed to list policy files", clog.Err(err))
		return err
	}

	p.logger.Info("found policy files", clog.Int("count", len(keys)))

	// 加载每个策略文件
	for _, key := range keys {
		if err := p.loadPolicy(ctx, key); err != nil {
			p.logger.Error("failed to load policy", clog.String("key", key), clog.Err(err))
			// 继续加载其他策略，不因为单个策略失败而中断
		}
	}

	return nil
}

// loadPolicy 加载单个策略
func (p *provider) loadPolicy(ctx context.Context, key string) error {
	var policy Policy
	if err := p.coordProvider.Get(ctx, key, &policy); err != nil {
		return err
	}

	// 验证策略
	if policy.FailureThreshold <= 0 {
		policy.FailureThreshold = GetDefaultPolicy().FailureThreshold
	}
	if policy.SuccessThreshold <= 0 {
		policy.SuccessThreshold = GetDefaultPolicy().SuccessThreshold
	}
	if policy.OpenStateTimeout <= 0 {
		policy.OpenStateTimeout = GetDefaultPolicy().OpenStateTimeout
	}

	p.logger.Info("policy loaded",
		clog.String("key", key),
		clog.Int("failure_threshold", policy.FailureThreshold),
		clog.Int("success_threshold", policy.SuccessThreshold),
		clog.Duration("open_state_timeout", policy.OpenStateTimeout))

	// 如果是默认策略文件，更新默认策略
	if key == p.config.PoliciesPath+"default.json" {
		p.defaultPolicy = &policy
	}

	return nil
}

// handleConfigEvent 处理配置变更事件
func (p *provider) handleConfigEvent(event ConfigEvent[any]) {
	switch event.Type {
	case EventTypePut:
		if policy, ok := event.Value.(*Policy); ok {
			p.handlePolicyUpdate(policy, event.Key)
		} else {
			p.logger.Warn("received non-policy config event", clog.String("key", event.Key))
		}
	case EventTypeDelete:
		p.logger.Info("policy deleted", clog.String("key", event.Key))
		// 可以在这里处理策略删除的逻辑
	}
}

// handlePolicyUpdate 处理策略更新
func (p *provider) handlePolicyUpdate(policy *Policy, key string) {
	// 验证策略
	if policy.FailureThreshold <= 0 {
		policy.FailureThreshold = GetDefaultPolicy().FailureThreshold
	}
	if policy.SuccessThreshold <= 0 {
		policy.SuccessThreshold = GetDefaultPolicy().SuccessThreshold
	}
	if policy.OpenStateTimeout <= 0 {
		policy.OpenStateTimeout = GetDefaultPolicy().OpenStateTimeout
	}

	p.logger.Info("policy updated",
		clog.String("key", key),
		clog.Int("failure_threshold", policy.FailureThreshold),
		clog.Int("success_threshold", policy.SuccessThreshold),
		clog.Duration("open_state_timeout", policy.OpenStateTimeout))

	// 如果是默认策略文件，更新默认策略
	if key == p.config.PoliciesPath+"default.json" {
		p.defaultPolicy = policy
		// 重新创建所有熔断器实例
		p.recreateAllBreakers()
	}
}

// getOrCreateBreaker 获取或创建一个熔断器实例
// 调用此方法时必须已经持有写锁
func (p *provider) getOrCreateBreaker(name string, policy *Policy) Breaker {
	fmt.Printf("DEBUG: getOrCreateBreaker called with name: %s\n", name)

	// 检查是否已存在
	if breaker, exists := p.breakers[name]; exists {
		fmt.Printf("DEBUG: Found existing breaker\n")
		return breaker
	}

	fmt.Printf("DEBUG: About to load breaker policy\n")
	// 尝试从配置中心加载该熔断器的特定策略
	breakerPolicy := p.loadBreakerPolicy(name)
	if breakerPolicy == nil {
		fmt.Printf("DEBUG: Using default policy\n")
		breakerPolicy = policy
	}

	fmt.Printf("DEBUG: About to create new breaker adapter\n")
	// 创建新的熔断器实例
	adapter := p.newGobreakerAdapter(name, breakerPolicy)
	p.breakers[name] = adapter

	// Temporarily disable logging
	// p.logger.Info("circuit breaker created",
	// 	clog.String("name", name),
	// 	clog.Int("failure_threshold", breakerPolicy.FailureThreshold),
	// 	clog.Int("success_threshold", breakerPolicy.SuccessThreshold),
	// 	clog.Duration("open_state_timeout", breakerPolicy.OpenStateTimeout))

	fmt.Printf("DEBUG: Breaker created successfully\n")
	return adapter
}

// loadBreakerPolicy 加载特定熔断器的策略
func (p *provider) loadBreakerPolicy(name string) *Policy {
	fmt.Printf("DEBUG: loadBreakerPolicy called with name: %s\n", name)
	if p.coordProvider == nil {
		fmt.Printf("DEBUG: No coord provider, returning nil\n")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := p.config.PoliciesPath + name + ".json"
	var policy Policy
	if err := p.coordProvider.Get(ctx, key, &policy); err != nil {
		p.logger.Debug("failed to load specific policy for breaker",
			clog.String("name", name),
			clog.String("key", key),
			clog.Err(err))
		return nil
	}

	// 验证策略
	if policy.FailureThreshold <= 0 {
		policy.FailureThreshold = GetDefaultPolicy().FailureThreshold
	}
	if policy.SuccessThreshold <= 0 {
		policy.SuccessThreshold = GetDefaultPolicy().SuccessThreshold
	}
	if policy.OpenStateTimeout <= 0 {
		policy.OpenStateTimeout = GetDefaultPolicy().OpenStateTimeout
	}

	return &policy
}

// newGobreakerAdapter 创建一个新的 gobreaker 适配器
func (p *provider) newGobreakerAdapter(name string, policy *Policy) *gobreakerAdapter {
	if p.logger == nil {
		p.logger = &noopLogger{}
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,           // 半开状态只允许一个请求通过
		Interval:    time.Minute, // 使用计数器重置
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= uint32(policy.FailureThreshold)
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			p.logger.Info("circuit breaker state changed",
				clog.String("name", name),
				clog.String("from", from.String()),
				clog.String("to", to.String()))
		},
		Timeout: policy.OpenStateTimeout,
	})

	return &gobreakerAdapter{
		breaker: cb,
		name:    name,
		logger:  p.logger,
	}
}

// recreateAllBreakers 重新创建所有熔断器实例
func (p *provider) recreateAllBreakers() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("recreating all circuit breakers with new policy")

	// 保存旧的名称
	names := make([]string, 0, len(p.breakers))
	for name := range p.breakers {
		names = append(names, name)
	}

	// 清空旧的熔断器
	p.breakers = make(map[string]Breaker)

	// 重新创建所有熔断器
	for _, name := range names {
		adapter := p.newGobreakerAdapter(name, p.defaultPolicy)
		p.breakers[name] = adapter
	}
}

// Do 执行受熔断器保护的操作
func (b *gobreakerAdapter) Do(ctx context.Context, op func() error) error {
	_, err := b.breaker.Execute(func() (interface{}, error) {
		err := op()
		if err != nil {
			b.logger.Debug("operation failed",
				clog.String("breaker", b.name),
				clog.Err(err))
		}
		return nil, err
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			return fmt.Errorf("%w: %s", ErrBreakerOpen, b.name)
		}
		return err
	}

	return nil
}

// noopBreaker 是一个空的熔断器实现，用于在 provider 关闭后返回
type noopBreaker struct{}

func (n *noopBreaker) Do(ctx context.Context, op func() error) error {
	return op() // 直接执行操作，不进行熔断保护
}

// noopLogger 是一个空的日志器实现
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, fields ...Field)          {}
func (n *noopLogger) Info(msg string, fields ...Field)           {}
func (n *noopLogger) Warn(msg string, fields ...Field)           {}
func (n *noopLogger) Error(msg string, fields ...Field)          {}
func (n *noopLogger) Fatal(msg string, fields ...Field)          {}
func (n *noopLogger) With(fields ...Field) Logger                { return n }
func (n *noopLogger) WithOptions(opts ...zap.Option) clog.Logger { return n }
func (n *noopLogger) Namespace(name string) clog.Logger          { return n }
