// Package ratelimit 提供了一个基于 Redis 的分布式限流组件。
//
// # 核心特性
//   - 基于令牌桶算法，支持平滑和突发流量。
//   - 通过 Redis 实现分布式限流，适用于微服务集群。
//   - 依赖 coordination 组件实现动态配置，可实时调整限流规则。
//   - 与 clog 组件集成，提供结构化日志。
//   - 采用与项目内其他 infra 组件一致的设计模式，通过 internal 封装实现细节。
package ratelimit

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/ratelimit/internal"
)

// RateLimiter 是限流器的主接口 (类型别名)
// 它定义了检查请求是否被允许的核心方法。
type RateLimiter = internal.RateLimiter

// New 创建一个新的限流器实例。
// serviceName 用于构建从 coordination 服务获取配置的路径。
// 例如，如果 serviceName 是 "im-gateway"，它会尝试从 "/configimpl/{env}/im-gateway/ratelimit/..." 获取规则。
func New(ctx context.Context, serviceName string, opts ...Option) (RateLimiter, error) {
	return internal.New(ctx, serviceName, opts...)
}

// Default 返回一个使用默认配置的全局单例限流器实例。
// serviceName 默认为 "default"。
func Default() RateLimiter {
	return internal.Default()
}
