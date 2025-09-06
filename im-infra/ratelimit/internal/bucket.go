package internal

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// tokenBucketScript 令牌桶算法的 Lua 脚本
// Keys:
// 1. KEYS[1] - 令牌桶的 key
// Args:
// 1. ARGV[1] - 令牌产生速率 (tokens/second)
// 2. ARGV[2] - 桶容量 (bucket capacity)
// 3. ARGV[3] - 当前时间戳 (nanoseconds)
// 4. ARGV[4] - 请求的令牌数量
// Returns:
// 1. 是否允许 (1=允许, 0=拒绝)
// 2. 剩余令牌数
// 3. 总请求数
// 4. 允许的请求数
const tokenBucketScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

-- 获取当前状态
local bucket = redis.call('hgetall', key)
local tokens
local last_refill_ts
local total_requests = 0
local allowed_requests = 0

if #bucket == 0 then
    -- 桶不存在，创建新桶
    tokens = capacity
    last_refill_ts = now
    total_requests = 0
    allowed_requests = 0
else
    -- 解析现有状态
    for i = 1, #bucket, 2 do
        if bucket[i] == 'tokens' then
            tokens = tonumber(bucket[i+1])
        elseif bucket[i] == 'last_refill_ts' then
            last_refill_ts = tonumber(bucket[i+1])
        elseif bucket[i] == 'total_requests' then
            total_requests = tonumber(bucket[i+1])
        elseif bucket[i] == 'allowed_requests' then
            allowed_requests = tonumber(bucket[i+1])
        end
    end
end

-- 计算时间间隔并补充令牌
local elapsed = (now - last_refill_ts) / 1e9  -- 转换为秒
local new_tokens = elapsed * rate
tokens = math.min(capacity, tokens + new_tokens)
last_refill_ts = now

-- 更新统计信息
total_requests = total_requests + 1

-- 判断是否允许请求
local allowed = 0
if tokens >= requested then
    tokens = tokens - requested
    allowed = 1
    allowed_requests = allowed_requests + 1
end

-- 更新状态
redis.call('hset', key, 'tokens', tokens, 'last_refill_ts', last_refill_ts, 'total_requests', total_requests, 'allowed_requests', allowed_requests)

return {allowed, math.floor(tokens), total_requests, allowed_requests}
`

// tokenBucket 令牌桶实现
type tokenBucket struct {
	cache     cache.Cache
	logger    clog.Logger
	scriptSHA string
	loadOnce  sync.Once
}

// newTokenBucket 创建一个新的令牌桶实例
func newTokenBucket(cache cache.Cache) *tokenBucket {
	return &tokenBucket{
		cache:  cache,
		logger: clog.Module("ratelimit.bucket"),
	}
}

// ensureScript 确保 Lua 脚本已加载
func (tb *tokenBucket) ensureScript(ctx context.Context) error {
	var err error
	tb.loadOnce.Do(func() {
		var sha string
		sha, err = tb.cache.ScriptLoad(ctx, tokenBucketScript)
		if err != nil {
			err = fmt.Errorf("failed to load token bucket script: %w", err)
			return
		}
		tb.scriptSHA = sha
		tb.logger.Info("令牌桶脚本加载成功", clog.String("sha", sha))
	})
	return err
}

// take 尝试从令牌桶获取指定数量的令牌
func (tb *tokenBucket) take(ctx context.Context, key string, rule Rule, count int64) (bool, int64, int64, int64, error) {
	// 确保脚本已加载
	if err := tb.ensureScript(ctx); err != nil {
		return false, 0, 0, 0, err
	}

	now := time.Now().UnixNano()
	args := []interface{}{
		rule.Rate,
		rule.Capacity,
		now,
		count,
	}

	res, err := tb.cache.EvalSha(ctx, tb.scriptSHA, []string{key}, args...)
	if err != nil {
		// 如果脚本未找到，尝试重新加载
		if isScriptNotFoundError(err) {
			tb.scriptSHA = "" // 清除 SHA
			tb.loadOnce = sync.Once{}

			if err := tb.ensureScript(ctx); err != nil {
				return false, 0, 0, 0, err
			}

			res, err = tb.cache.EvalSha(ctx, tb.scriptSHA, []string{key}, args...)
		}

		if err != nil {
			return false, 0, 0, 0, fmt.Errorf("failed to execute token bucket script: %w", err)
		}
	}

	result, ok := res.([]interface{})
	if !ok || len(result) < 4 {
		return false, 0, 0, 0, fmt.Errorf("invalid response from token bucket script: %v", res)
	}

	allowed, ok := result[0].(int64)
	if !ok {
		return false, 0, 0, 0, fmt.Errorf("invalid allowed value: %v", result[0])
	}

	remainingTokens, _ := result[1].(int64)
	totalRequests, _ := result[2].(int64)
	allowedRequests, _ := result[3].(int64)

	return allowed == 1, remainingTokens, totalRequests, allowedRequests, nil
}

// getStatistics 获取令牌桶的统计信息
func (tb *tokenBucket) getStatistics(ctx context.Context, key string) (*BucketStatistics, error) {
	data, err := tb.cache.HGetAll(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket statistics: %w", err)
	}

	stats := &BucketStatistics{}
	for k, v := range data {
		switch k {
		case "tokens":
			stats.CurrentTokens, _ = toInt64(v)
		case "total_requests":
			stats.TotalRequests, _ = toInt64(v)
		case "allowed_requests":
			stats.AllowedRequests, _ = toInt64(v)
		case "last_refill_ts":
			if ts, _ := toInt64(v); ts > 0 {
				stats.LastRefillTime = time.Unix(0, ts)
			}
		}
	}

	stats.DeniedRequests = stats.TotalRequests - stats.AllowedRequests
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.AllowedRequests) / float64(stats.TotalRequests)
	}

	return stats, nil
}

// BucketStatistics 定义令牌桶的统计信息
type BucketStatistics struct {
	CurrentTokens   int64     `json:"current_tokens"`
	TotalRequests   int64     `json:"total_requests"`
	AllowedRequests int64     `json:"allowed_requests"`
	DeniedRequests  int64     `json:"denied_requests"`
	SuccessRate     float64   `json:"success_rate"`
	LastRefillTime  time.Time `json:"last_refill_time"`
}

// isScriptNotFoundError 判断错误是否为脚本未找到
func isScriptNotFoundError(err error) bool {
	return err != nil && err.Error() == "NOSCRIPT" // 使用字符串匹配替代直接依赖 redis 包
}

// toInt64 工具函数：将接口转换为 int64
func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case string:
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i, true
		}
		return 0, false
	default:
		return 0, false
	}
}
