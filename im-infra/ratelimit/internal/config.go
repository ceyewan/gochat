package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// loadRules 从配置中心加载所有规则。
func (l *limiter) loadRules() error {
	configPath := l.buildConfigPath()
	l.logger.Info("从配置中心加载限流规则", clog.String("path", configPath))

	keys, err := l.opts.CoordinationClient.Config().List(l.ctx, configPath)
	if err != nil {
		l.logger.Error("无法从 etcd 列出配置键", clog.String("path", configPath), clog.Err(err))
		return fmt.Errorf("无法列出配置: %w", err)
	}

	newRules := make(map[string]Rule)
	for _, key := range keys {
		ruleName := strings.TrimPrefix(key, configPath+"/")
		value, err := l.opts.CoordinationClient.Config().Get(l.ctx, key)
		if err != nil {
			l.logger.Warn("获取规则失败", clog.String("key", key), clog.Err(err))
			continue
		}

		var rule Rule
		valueStr, ok := value.(string)
		if !ok {
			l.logger.Warn("配置值类型不为 string", clog.String("key", key), clog.Any("value", value))
			continue
		}

		if err := json.Unmarshal([]byte(valueStr), &rule); err != nil {
			l.logger.Warn("解析规则失败", clog.String("key", key), clog.String("value", valueStr), clog.Err(err))
			continue
		}
		newRules[ruleName] = rule
		l.logger.Debug("成功加载规则", clog.String("ruleName", ruleName), clog.Float64("rate", rule.Rate), clog.Int64("capacity", rule.Capacity))
	}

	l.mu.Lock()
	l.rules = newRules
	l.mu.Unlock()

	l.logger.Info("限流规则加载完成", clog.Int("count", len(newRules)))
	return nil
}

// startRuleRefresher 启动一个后台 goroutine 来监听配置变更并刷新规则。
func (l *limiter) startRuleRefresher() {
	go func() {
		configPath := l.buildConfigPath()
		watcher, err := l.opts.CoordinationClient.Config().Watch(l.ctx, configPath)
		if err != nil {
			l.logger.Error("无法监听配置变更", clog.String("path", configPath), clog.Err(err))
			return
		}

		for {
			select {
			case <-l.ctx.Done():
				l.logger.Info("规则刷新协程已停止")
				return
			case event, ok := <-watcher:
				if !ok {
					l.logger.Warn("配置中心的 watch 通道已关闭，将尝试重建")
					newWatcher, err := l.opts.CoordinationClient.Config().Watch(l.ctx, configPath)
					if err != nil {
						l.logger.Error("无法重建配置监听", clog.String("path", configPath), clog.Err(err))
					} else {
						watcher = newWatcher
					}
					continue
				}
				l.logger.Info("检测到配置变更事件", clog.String("type", string(event.Type)), clog.String("key", event.Key))
				if err := l.loadRules(); err != nil {
					l.logger.Error("自动刷新规则失败", clog.Err(err))
				}
			}
		}
	}()
}

// buildConfigPath 构建用于获取配置的 etcd 路径。
// 格式: /configimpl/{env}/{serviceName}/ratelimit
func (l *limiter) buildConfigPath() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev" // 默认环境
	}
	return fmt.Sprintf("/configimpl/%s/%s/ratelimit", env, l.serviceName)
}

// getRule 获取一个规则。
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
