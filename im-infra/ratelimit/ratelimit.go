// Package ratelimit 提供了一个基于 Redis 的分布式限流组件。
//
// # 核心特性
//   - 基于令牌桶算法，支持平滑和突发流量。
//   - 通过 Redis 实现分布式限流，适用于微服务集群。
//   - 依赖 coord 组件实现动态配置，可实时调整限流规则。
//   - 与 cache 组件集成，提供抽象的缓存接口。
//   - 采用与项目内其他 infra 组件一致的设计模式，通过 internal 封装实现细节。
package ratelimit

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/ratelimit/internal"
)

// RateLimiter 是限流器的主接口 (类型别名)
// 它定义了检查请求是否被允许的核心方法。
type RateLimiter = internal.RateLimiter

// RateLimiterManager 扩展接口，提供管理功能 (类型别名)
type RateLimiterManager = internal.RateLimiterManager

// RateLimitRequest 批量限流请求 (类型别名)
type RateLimitRequest = internal.RateLimitRequest

// RateLimitStatistics 限流统计信息 (类型别名)
type RateLimitStatistics = internal.RateLimitStatistics

// New 创建一个新的限流器实例。
// serviceName 用于构建从 coord 服务获取配置的路径。
// 例如，如果 serviceName 是 "im-gateway"，它会尝试从 "/config/{env}/im-gateway/ratelimit/..." 获取规则。
func New(ctx context.Context, serviceName string, opts ...Option) (RateLimiter, error) {
	return internal.New(ctx, serviceName, opts...)
}

// NewManager 创建一个带管理功能的限流器实例。
// 除了基本的限流功能外，还提供动态配置、规则管理等高级功能。
func NewManager(ctx context.Context, serviceName string, opts ...Option) (RateLimiterManager, error) {
	limiter, err := internal.New(ctx, serviceName, opts...)
	if err != nil {
		return nil, err
	}

	// 确保返回的是 RateLimiterManager 接口
	if manager, ok := limiter.(internal.RateLimiterManager); ok {
		return manager, nil
	}

	// 这种情况理论上不会发生，但为了类型安全
	return limiter.(internal.RateLimiterManager), nil
}

// Default 返回一个使用默认配置的全局单例限流器实例。
// serviceName 默认为 "default"。
func Default() RateLimiter {
	return internal.Default()
}

// SimpleRateLimiter 创建一个简单的限流器，仅使用内存中的规则，不依赖配置中心。
// 适用于测试或简单场景。
func SimpleRateLimiter(ctx context.Context, serviceName string, rules map[string]Rule, opts ...Option) (RateLimiter, error) {
	opts = append(opts, WithDefaultRules(rules))
	return New(ctx, serviceName, opts...)
}

// ValidateRule 验证规则的有效性
func ValidateRule(rule Rule) error {
	if rule.Rate <= 0 {
		return ErrInvalidRate
	}
	if rule.Capacity <= 0 {
		return ErrInvalidCapacity
	}
	return nil
}

// CreateDefaultRules 创建一组常用的默认规则
func CreateDefaultRules() map[string]Rule {
	return map[string]Rule{
		"api_default":     {Rate: 100, Capacity: 200}, // API 默认限流：100 req/s，突发 200
		"user_action":     {Rate: 10, Capacity: 20},   // 用户操作：10 req/s，突发 20
		"login":           {Rate: 5, Capacity: 10},    // 登录限流：5 req/s，突发 10
		"register":        {Rate: 1, Capacity: 3},     // 注册限流：1 req/s，突发 3
		"password_reset":  {Rate: 0.1, Capacity: 1},   // 密码重置：0.1 req/s，突发 1
		"sms_send":        {Rate: 0.5, Capacity: 2},   // 短信发送：0.5 req/s，突发 2
		"email_send":      {Rate: 1, Capacity: 5},     // 邮件发送：1 req/s，突发 5
		"file_upload":     {Rate: 2, Capacity: 10},    // 文件上传：2 req/s，突发 10
		"ws_message":      {Rate: 50, Capacity: 100},  // WebSocket 消息：50 req/s，突发 100
		"heavy_operation": {Rate: 1, Capacity: 2},     // 重型操作：1 req/s，突发 2
	}
}

// GetRuleByScenario 根据场景获取推荐的限流规则
func GetRuleByScenario(scenario string) (Rule, bool) {
	rules := map[string]Rule{
		// Web API 场景
		"web_api_high":   {Rate: 1000, Capacity: 2000}, // 高频 API
		"web_api_medium": {Rate: 100, Capacity: 200},   // 中频 API
		"web_api_low":    {Rate: 10, Capacity: 20},     // 低频 API

		// 用户操作场景
		"user_read":      {Rate: 50, Capacity: 100}, // 用户读操作
		"user_write":     {Rate: 10, Capacity: 20},  // 用户写操作
		"user_sensitive": {Rate: 1, Capacity: 3},    // 敏感操作

		// 系统资源场景
		"cpu_intensive": {Rate: 5, Capacity: 10},  // CPU 密集型
		"io_intensive":  {Rate: 20, Capacity: 40}, // IO 密集型
		"memory_heavy":  {Rate: 2, Capacity: 5},   // 内存密集型

		// 外部服务场景
		"third_party_api": {Rate: 10, Capacity: 20}, // 第三方 API
		"payment":         {Rate: 1, Capacity: 2},   // 支付相关
		"notification":    {Rate: 5, Capacity: 15},  // 通知发送

		// 安全场景
		"auth_attempt":   {Rate: 3, Capacity: 5},  // 认证尝试
		"captcha":        {Rate: 5, Capacity: 10}, // 验证码
		"security_check": {Rate: 1, Capacity: 2},  // 安全检查
	}

	rule, exists := rules[scenario]
	return rule, exists
}

// BuildResourceKey 构建资源键的辅助函数
func BuildResourceKey(resourceType, identifier string) string {
	return resourceType + ":" + identifier
}

// BuildUserResourceKey 构建用户相关资源键
func BuildUserResourceKey(userID string) string {
	return BuildResourceKey("user", userID)
}

// BuildIPResourceKey 构建 IP 相关资源键
func BuildIPResourceKey(ip string) string {
	return BuildResourceKey("ip", ip)
}

// BuildAPIResourceKey 构建 API 相关资源键
func BuildAPIResourceKey(endpoint string) string {
	return BuildResourceKey("api", endpoint)
}

// BuildDeviceResourceKey 构建设备相关资源键
func BuildDeviceResourceKey(deviceID string) string {
	return BuildResourceKey("device", deviceID)
}
