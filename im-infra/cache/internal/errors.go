package internal

import "errors"

var (
	// ErrBloomFilterNotSupported 表示 Redis 服务器不支持布隆过滤器命令。
	ErrBloomFilterNotSupported = errors.New("redis server does not support bloom filter commands (RedisBloom module may not be installed)")
	// ErrCacheMiss 表示在缓存中未找到指定的 key。
	// 所有 Get 操作在缓存未命中时，都应返回此错误。
	ErrCacheMiss = errors.New("cache: key not found")
)
