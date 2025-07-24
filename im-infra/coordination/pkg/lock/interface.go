package lock

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination"
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// Acquire 获取互斥锁
	Acquire(ctx context.Context, key string, ttl time.Duration) (coordination.Lock, error)

	// TryAcquire 尝试获取锁（非阻塞）
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (coordination.Lock, error)
}

// Lock 锁对象接口
type Lock interface {
	// Unlock 释放锁
	Unlock(ctx context.Context) error

	// Renew 续期锁
	Renew(ctx context.Context, ttl time.Duration) error

	// TTL 获取锁的剩余有效时间
	TTL(ctx context.Context) (time.Duration, error)

	// Key 获取锁的键
	Key() string
}
