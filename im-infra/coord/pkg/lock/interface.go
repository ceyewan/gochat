package lock

import (
	"context"
	"time"
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// Acquire 获取互斥锁
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)

	// TryAcquire 尝试获取锁（非阻塞）
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}
