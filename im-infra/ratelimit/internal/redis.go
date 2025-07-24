package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// Redis Lua script for token bucket algorithm.
//
// Keys:
// 1. KEYS[1] - The key for the rate limiter hash (e.g., ratelimit:user:123)
//
// Args:
// 1. ARGV[1] - rate (tokens per second)
// 2. ARGV[2] - capacity (bucket size)
// 3. ARGV[3] - now (current unix timestamp in nanoseconds)
// 4. ARGV[4] - requested (number of tokens to take, usually 1)
//
// Returns:
// 1. 1 if allowed, 0 if denied
// 2. The number of remaining tokens in the bucket
const tokenBucketScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local bucket = redis.call('hgetall', key)
local tokens
local last_refill_ts

if #bucket == 0 then
    -- Bucket doesn't exist, create a new one
    tokens = capacity
    last_refill_ts = now
else
    -- Bucket exists, parse its values
    for i = 1, #bucket, 2 do
        if bucket[i] == 'tokens' then
            tokens = tonumber(bucket[i+1])
        elseif bucket[i] == 'last_refill_ts' then
            last_refill_ts = tonumber(bucket[i+1])
        end
    end
end

-- Refill tokens based on elapsed time
local elapsed = (now - last_refill_ts) / 1e9 -- elapsed time in seconds
local new_tokens = elapsed * rate
if new_tokens > 0 then
    tokens = math.min(capacity, tokens + new_tokens)
    last_refill_ts = now
end

local allowed = 0
if tokens >= requested then
    -- Enough tokens, consume them
    tokens = tokens - requested
    allowed = 1
end

-- Update the bucket in Redis
redis.call('hset', key, 'tokens', tokens, 'last_refill_ts', last_refill_ts)

return {allowed, math.floor(tokens)}
`

var (
	// Script SHA, used for efficient execution with EVALSHA
	scriptSHA string
)

// loadScript loads the Lua script into Redis and stores its SHA hash.
func (l *limiter) loadScript(ctx context.Context) error {
	// The script is loaded only once per limiter instance.
	if scriptSHA != "" {
		return nil
	}

	sha, err := l.opts.CacheClient.ScriptLoad(ctx, tokenBucketScript)
	if err != nil {
		return fmt.Errorf("failed to load lua script: %w", err)
	}
	scriptSHA = sha
	l.logger.Info("成功加载限流 Lua 脚本", clog.String("sha", scriptSHA))
	return nil
}

// executeTokenBucketScript runs the token bucket algorithm script in Redis.
func (l *limiter) executeTokenBucketScript(ctx context.Context, key string, rule Rule) (bool, error) {
	if scriptSHA == "" {
		if err := l.loadScript(ctx); err != nil {
			return false, err
		}
	}

	now := time.Now().UnixNano()
	args := []interface{}{
		rule.Rate,
		rule.Capacity,
		now,
		1, // a single request consumes 1 token
	}

	res, err := l.opts.CacheClient.EvalSha(ctx, scriptSHA, []string{key}, args...)
	if err != nil {
		// If the script is not found (e.g., Redis was flushed), reload it and retry.
		if redis.HasErrorPrefix(err, "NOSCRIPT") {
			l.logger.Warn("Lua 脚本未找到，将重新加载", clog.String("sha", scriptSHA))
			scriptSHA = "" // Reset SHA to force reload
			if errLoad := l.loadScript(ctx); errLoad != nil {
				return false, errLoad
			}
			res, err = l.opts.CacheClient.EvalSha(ctx, scriptSHA, []string{key}, args...)
		}
	}

	if err != nil {
		return false, fmt.Errorf("failed to execute lua script: %w", err)
	}

	result, ok := res.([]interface{})
	if !ok || len(result) < 2 {
		return false, fmt.Errorf("invalid response from lua script: %v", res)
	}

	allowed, ok := result[0].(int64)
	if !ok {
		return false, fmt.Errorf("invalid 'allowed' value from lua script: %v", result[0])
	}

	return allowed == 1, nil
}
