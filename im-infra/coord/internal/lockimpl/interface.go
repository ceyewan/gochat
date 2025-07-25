package lockimpl

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord/lock"
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// Acquire 获取互斥锁
	Acquire(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error)

	// TryAcquire 尝试获取锁（非阻塞）
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error)
}
