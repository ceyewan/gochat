package lock

import (
	"context"
	"time"
)

// DistributedLock 是分布式锁服务的接口
type DistributedLock interface {
	// Acquire 获取互斥锁，如果锁已被占用，会阻塞直到获取成功或 context 取消
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
	// TryAcquire 尝试获取锁（非阻塞），如果锁已被占用，会立即返回错误
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}

// Lock 是一个已获取的锁对象的接口
// 用户通过这个接口与持有的锁进行交互
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
