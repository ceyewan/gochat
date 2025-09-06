package internal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// RuleConfig 限流规则配置
type RuleConfig struct {
	Rate        float64 `json:"rate"`        // 令牌产生速率 (tokens/second)
	Capacity    int64   `json:"capacity"`    // 桶容量
	Description string  `json:"description"` // 规则描述
}

// loadRules 从配置中心加载所有规则
func (l *limiter) loadRules() error {
	configPath := l.buildConfigPath()
	l.logger.Info("从配置中心加载限流规则", clog.String("path", configPath))

	keys, err := l.opts.CoordinationClient.Config().List(l.ctx, configPath)
	if err != nil {
		l.logger.Error("无法从配置中心列出配置键", clog.String("path", configPath), clog.Err(err))
		return fmt.Errorf("无法列出配置: %w", err)
	}

	newRules := make(map[string]Rule)
	for _, key := range keys {
		// 提取规则名称（去掉路径前缀）
		ruleName := strings.TrimPrefix(key, configPath+"/")
		if ruleName == key {
			// 如果没有前缀匹配，跳过
			continue
		}

		var ruleConfig RuleConfig
		err := l.opts.CoordinationClient.Config().Get(l.ctx, key, &ruleConfig)
		if err != nil {
			l.logger.Warn("获取规则失败", clog.String("key", key), clog.Err(err))
			continue
		}

		// 转换为内部规则格式
		rule := Rule{
			Rate:     ruleConfig.Rate,
			Capacity: ruleConfig.Capacity,
		}

		newRules[ruleName] = rule
		l.logger.Debug("成功加载规则",
			clog.String("ruleName", ruleName),
			clog.Float64("rate", rule.Rate),
			clog.Int64("capacity", rule.Capacity))
	}

	l.mu.Lock()
	l.rules = newRules
	l.mu.Unlock()

	l.logger.Info("限流规则加载完成", clog.Int("count", len(newRules)))
	return nil
}

// startRuleRefresher 启动一个后台 goroutine 来监听配置变更并刷新规则
func (l *limiter) startRuleRefresher() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.logger.Error("规则刷新器发生panic", clog.Any("recover", r))
			}
		}()

		// 使用定时器定期刷新规则
		ticker := time.NewTicker(l.opts.RuleRefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-l.ctx.Done():
				l.logger.Info("规则刷新协程已停止")
				return
			case <-ticker.C:
				if err := l.loadRules(); err != nil {
					l.logger.Error("自动刷新规则失败", clog.Err(err))
				}
			}
		}
	}()

	// 尝试启动配置监听（如果支持的话）
	l.startConfigWatcher()
}

// startConfigWatcher 启动配置监听器（如果配置中心支持）
func (l *limiter) startConfigWatcher() {
	if l.opts.CoordinationClient == nil {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.logger.Error("配置监听器发生panic", clog.Any("recover", r))
			}
		}()

		configPath := l.buildConfigPath()

		// 创建一个用于监听的空接口变量
		var watchValue interface{}

		watcher, err := l.opts.CoordinationClient.Config().WatchPrefix(l.ctx, configPath, &watchValue)
		if err != nil {
			l.logger.Error("无法创建配置监听器", clog.String("path", configPath), clog.Err(err))
			return
		}

		l.logger.Info("配置监听器已启动", clog.String("path", configPath))

		for {
			select {
			case <-l.ctx.Done():
				watcher.Close()
				l.logger.Info("配置监听器已停止")
				return
			case event, ok := <-watcher.Chan():
				if !ok {
					l.logger.Warn("配置监听器通道已关闭，将尝试重建")
					// 尝试重建监听器
					time.Sleep(5 * time.Second)
					watcher.Close()
					watcher, err = l.opts.CoordinationClient.Config().WatchPrefix(l.ctx, configPath, &watchValue)
					if err != nil {
						l.logger.Error("无法重建配置监听器", clog.String("path", configPath), clog.Err(err))
						return
					}
					continue
				}

				l.logger.Info("检测到配置变更事件",
					clog.String("type", string(event.Type)),
					clog.String("key", event.Key))

				// 重新加载规则
				if err := l.loadRules(); err != nil {
					l.logger.Error("配置变更后重新加载规则失败", clog.Err(err))
				} else {
					l.logger.Info("配置变更后规则重新加载成功")
				}
			}
		}
	}()
}

// buildConfigPath 构建用于获取配置的路径
// 格式: /config/{env}/{serviceName}/ratelimit
func (l *limiter) buildConfigPath() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev" // 默认环境
	}
	return fmt.Sprintf("/config/%s/%s/ratelimit", env, l.serviceName)
}

// getRule 获取一个规则
func (l *limiter) getRule(name string) (Rule, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	rule, ok := l.rules[name]
	if !ok {
		// 如果在内存中找不到，尝试从默认选项中查找
		rule, ok = l.opts.DefaultRules[name]
	}
	return rule, ok
}

// validateRule 验证规则的有效性
func validateRule(rule Rule) error {
	if rule.Rate <= 0 {
		return fmt.Errorf("rate must be positive, got: %f", rule.Rate)
	}
	if rule.Capacity <= 0 {
		return fmt.Errorf("capacity must be positive, got: %d", rule.Capacity)
	}
	// 检查速率和容量的合理性
	if rule.Rate > 1000000 {
		return fmt.Errorf("rate too high, maximum is 1000000, got: %f", rule.Rate)
	}
	if rule.Capacity > 1000000 {
		return fmt.Errorf("capacity too high, maximum is 1000000, got: %d", rule.Capacity)
	}
	return nil
}

// setRule 动态设置限流规则（内部方法）
func (l *limiter) setRule(ctx context.Context, ruleName string, rule Rule) error {
	if err := validateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	if ruleName == "" {
		return fmt.Errorf("rule name cannot be empty")
	}

	l.mu.Lock()
	if l.rules == nil {
		l.rules = make(map[string]Rule)
	}
	l.rules[ruleName] = rule
	l.mu.Unlock()

	// 如果有配置中心，同时更新到配置中心
	if l.opts.CoordinationClient != nil {
		configPath := l.buildConfigPath()
		ruleKey := fmt.Sprintf("%s/%s", configPath, ruleName)

		ruleConfig := RuleConfig{
			Rate:        rule.Rate,
			Capacity:    rule.Capacity,
			Description: fmt.Sprintf("动态设置的规则：%s", ruleName),
		}

		if err := l.opts.CoordinationClient.Config().Set(ctx, ruleKey, ruleConfig); err != nil {
			l.logger.Warn("无法将规则保存到配置中心",
				clog.String("ruleName", ruleName),
				clog.Err(err))
			// 不返回错误，因为本地已经设置成功
		}
	}

	l.logger.Info("限流规则已设置",
		clog.String("ruleName", ruleName),
		clog.Float64("rate", rule.Rate),
		clog.Int64("capacity", rule.Capacity))

	return nil
}

// listRules 获取当前所有规则（内部方法）
func (l *limiter) listRules() map[string]Rule {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make(map[string]Rule)
	for name, rule := range l.rules {
		result[name] = rule
	}

	// 合并默认规则
	for name, rule := range l.opts.DefaultRules {
		if _, exists := result[name]; !exists {
			result[name] = rule
		}
	}

	return result
}

// deleteRule 删除限流规则（内部方法）
func (l *limiter) deleteRule(ctx context.Context, ruleName string) error {
	if ruleName == "" {
		return fmt.Errorf("rule name cannot be empty")
	}

	l.mu.Lock()
	_, exists := l.rules[ruleName]
	if exists {
		delete(l.rules, ruleName)
	}
	l.mu.Unlock()

	if !exists {
		return fmt.Errorf("rule %s not found", ruleName)
	}

	// 如果有配置中心，同时从配置中心删除
	if l.opts.CoordinationClient != nil {
		configPath := l.buildConfigPath()
		ruleKey := fmt.Sprintf("%s/%s", configPath, ruleName)

		if err := l.opts.CoordinationClient.Config().Delete(ctx, ruleKey); err != nil {
			l.logger.Warn("无法从配置中心删除规则",
				clog.String("ruleName", ruleName),
				clog.Err(err))
			// 不返回错误，因为本地已经删除成功
		}
	}

	l.logger.Info("限流规则已删除", clog.String("ruleName", ruleName))
	return nil
}

// exportRules 导出规则到配置中心（用于初始化或备份）
func (l *limiter) exportRules(ctx context.Context) error {
	if l.opts.CoordinationClient == nil {
		return fmt.Errorf("no coordination client configured")
	}

	configPath := l.buildConfigPath()
	rules := l.listRules()

	exportCount := 0
	for ruleName, rule := range rules {
		ruleKey := fmt.Sprintf("%s/%s", configPath, ruleName)
		ruleConfig := RuleConfig{
			Rate:        rule.Rate,
			Capacity:    rule.Capacity,
			Description: fmt.Sprintf("导出的规则：%s", ruleName),
		}

		if err := l.opts.CoordinationClient.Config().Set(ctx, ruleKey, ruleConfig); err != nil {
			l.logger.Error("导出规则失败",
				clog.String("ruleName", ruleName),
				clog.Err(err))
			return fmt.Errorf("failed to export rule %s: %w", ruleName, err)
		}
		exportCount++
	}

	l.logger.Info("规则导出完成", clog.Int("count", exportCount))
	return nil
}
