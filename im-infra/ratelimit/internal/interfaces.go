package internal

import "context"

// RateLimiter 是限流器的主接口。
// 它定义了检查请求是否被允许的核心方法。
type RateLimiter interface {
	// Allow 检查给定资源的请求是否被允许。
	// resource: 资源的唯一标识符，例如 "user:123" 或 "ip:1.2.3.4"。
	// ruleName: 规则名称，用于从配置中查找对应的速率和容量，例如 "user_message_freq"。
	Allow(ctx context.Context, resource string, ruleName string) (bool, error)

	// Close 释放限流器持有的资源，例如停止后台的规则刷新 goroutine。
	Close() error
}
