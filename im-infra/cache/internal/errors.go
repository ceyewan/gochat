package internal

import "errors"

var (
	// ErrBloomFilterNotSupported 表示 Redis 服务器不支持布隆过滤器命令。
	ErrBloomFilterNotSupported = errors.New("redis server does not support bloom filter commands (RedisBloom module may not be installed)")
)
