package lockimpl

import (
	"context"
	"path"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	"github.com/ceyewan/gochat/im-infra/coord/lock"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// EtcdLockFactory 是用于创建基于 etcd 的分布式锁的工厂。
// 实现了 lock.DistributedLock 接口。
type EtcdLockFactory struct {
	client *client.EtcdClient // etcd 客户端
	prefix string             // 锁的前缀
	logger clog.Logger        // 日志记录器
}

// NewEtcdLockFactory 创建一个 etcd 分布式锁工厂
func NewEtcdLockFactory(c *client.EtcdClient, prefix string, logger clog.Logger) *EtcdLockFactory {
	if prefix == "" {
		prefix = "/locks"
	}
	if logger == nil {
		logger = clog.Module("coordination.lock")
	}
	return &EtcdLockFactory{
		client: c,
		prefix: prefix,
		logger: logger,
	}
}

// Acquire 获取一个新锁，阻塞直到锁被获取或 context 被取消
func (f *EtcdLockFactory) Acquire(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error) {
	return f.acquire(ctx, key, ttl, true)
}

// TryAcquire 尝试获取新锁，不阻塞
func (f *EtcdLockFactory) TryAcquire(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error) {
	return f.acquire(ctx, key, ttl, false)
}

// acquire 内部实现，支持阻塞和非阻塞获取锁
func (f *EtcdLockFactory) acquire(ctx context.Context, key string, ttl time.Duration, blocking bool) (lock.Lock, error) {
	if key == "" {
		return nil, client.NewError(client.ErrCodeValidation, "lock key cannot be empty", nil)
	}
	if ttl <= 0 {
		return nil, client.NewError(client.ErrCodeValidation, "lock ttl must be positive", nil)
	}

	// 创建会话，包含租约并自动续约。锁释放时关闭会话。
	session, err := concurrency.NewSession(f.client.Client(), concurrency.WithTTL(int(ttl.Seconds())))
	if err != nil {
		return nil, client.NewError(client.ErrCodeConnection, "failed to create etcd session", err)
	}

	lockKey := path.Join(f.prefix, key)
	mutex := concurrency.NewMutex(session, lockKey)

	f.logger.Debug("尝试获取锁",
		clog.String("key", lockKey),
		clog.Int64("lease", int64(session.Lease())),
		clog.Bool("blocking", blocking))

	var lockErr error
	if blocking {
		// 阻塞直到获取锁或 context 被取消
		lockErr = mutex.Lock(ctx)
	} else {
		// 非阻塞尝试获取锁，立即返回
		lockErr = mutex.TryLock(ctx)
	}

	if lockErr != nil {
		_ = session.Close() // 尝试关闭会话，释放资源
		if lockErr == concurrency.ErrLocked {
			return nil, client.NewError(client.ErrCodeConflict, "lock is already held", lockErr)
		}
		return nil, client.NewError(client.ErrCodeConnection, "failed to acquire lock", lockErr)
	}

	f.logger.Info("锁获取成功",
		clog.String("key", lockKey),
		clog.Int64("lease", int64(session.Lease())))

	return &etcdLock{
		session: session,
		mutex:   mutex,
		client:  f.client,
		logger:  f.logger,
	}, nil
}

// etcdLock 表示已持有的分布式锁
type etcdLock struct {
	session *concurrency.Session // etcd 会话，管理租约
	mutex   *concurrency.Mutex   // etcd 互斥锁
	client  *client.EtcdClient   // etcd 客户端
	logger  clog.Logger          // 日志记录器
}

// Unlock 释放锁
func (l *etcdLock) Unlock(ctx context.Context) error {
	l.logger.Debug("释放锁",
		clog.String("key", l.mutex.Key()),
		clog.Int64("lease", int64(l.session.Lease())))

	// 先解锁互斥锁
	if err := l.mutex.Unlock(ctx); err != nil {
		// 即使解锁失败，也必须关闭会话以释放租约
		_ = l.session.Close()
		return client.NewError(client.ErrCodeConnection, "failed to unlock mutex", err)
	}

	// 关闭会话，撤销租约，最终释放锁
	if err := l.session.Close(); err != nil {
		return client.NewError(client.ErrCodeConnection, "failed to close session", err)
	}

	l.logger.Info("锁释放成功", clog.String("key", l.mutex.Key()))
	return nil
}

// TTL 返回锁租约的剩余存活时间
func (l *etcdLock) TTL(ctx context.Context) (time.Duration, error) {
	// 通过会话获取租约 ID
	resp, err := l.client.Client().TimeToLive(ctx, l.session.Lease())
	if err != nil {
		return 0, client.NewError(client.ErrCodeConnection, "failed to get lock TTL", err)
	}

	if resp.TTL <= 0 {
		// 如果租约刚好过期会出现这种情况
		return 0, client.NewError(client.ErrCodeNotFound, "lock has expired", nil)
	}

	return time.Duration(resp.TTL) * time.Second, nil
}

// Key 返回锁在 etcd 中的完整键路径
func (l *etcdLock) Key() string {
	return l.mutex.Key()
}
