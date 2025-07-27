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

// EtcdLockFactory is a factory for creating etcd-based distributed locks.
// It implements the lock.DistributedLock interface.
type EtcdLockFactory struct {
	client *client.EtcdClient
	prefix string
	logger clog.Logger
}

// NewEtcdLockFactory creates a new factory for etcd distributed locks.
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

// Acquire obtains a new lock, blocking until the lock is acquired or the context is canceled.
func (f *EtcdLockFactory) Acquire(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error) {
	return f.acquire(ctx, key, ttl, true)
}

// TryAcquire attempts to obtain a new lock without blocking.
func (f *EtcdLockFactory) TryAcquire(ctx context.Context, key string, ttl time.Duration) (lock.Lock, error) {
	return f.acquire(ctx, key, ttl, false)
}

func (f *EtcdLockFactory) acquire(ctx context.Context, key string, ttl time.Duration, blocking bool) (lock.Lock, error) {
	if key == "" {
		return nil, client.NewError(client.ErrCodeValidation, "lock key cannot be empty", nil)
	}
	if ttl <= 0 {
		return nil, client.NewError(client.ErrCodeValidation, "lock ttl must be positive", nil)
	}

	// The session is the key. It bundles a lease and manages its keep-alive.
	// The session will be closed when the lock is released.
	session, err := concurrency.NewSession(f.client.Client(), concurrency.WithTTL(int(ttl.Seconds())))
	if err != nil {
		return nil, client.NewError(client.ErrCodeConnection, "failed to create etcd session", err)
	}

	lockKey := path.Join(f.prefix, key)
	mutex := concurrency.NewMutex(session, lockKey)

	f.logger.Debug("Attempting to acquire lock",
		clog.String("key", lockKey),
		clog.Int64("lease", int64(session.Lease())),
		clog.Bool("blocking", blocking))

	var lockErr error
	if blocking {
		// This blocks until the lock is acquired or context is canceled.
		lockErr = mutex.Lock(ctx)
	} else {
		// This attempts to acquire the lock and returns immediately.
		lockErr = mutex.TryLock(ctx)
	}

	if lockErr != nil {
		_ = session.Close() // Best-effort close the session.
		if lockErr == concurrency.ErrLocked {
			return nil, client.NewError(client.ErrCodeConflict, "lock is already held", lockErr)
		}
		return nil, client.NewError(client.ErrCodeConnection, "failed to acquire lock", lockErr)
	}

	f.logger.Info("Lock acquired successfully",
		clog.String("key", lockKey),
		clog.Int64("lease", int64(session.Lease())))

	return &etcdLock{
		session: session,
		mutex:   mutex,
		client:  f.client,
		logger:  f.logger,
	}, nil
}

// etcdLock represents a held distributed lock.
type etcdLock struct {
	session *concurrency.Session
	mutex   *concurrency.Mutex
	client  *client.EtcdClient
	logger  clog.Logger
}

// Unlock releases the lock.
func (l *etcdLock) Unlock(ctx context.Context) error {
	l.logger.Debug("Releasing lock",
		clog.String("key", l.mutex.Key()),
		clog.Int64("lease", int64(l.session.Lease())))

	// Unlock the mutex first.
	if err := l.mutex.Unlock(ctx); err != nil {
		// Even if unlock fails, we must close the session to release the lease.
		_ = l.session.Close()
		return client.NewError(client.ErrCodeConnection, "failed to unlock mutex", err)
	}

	// Closing the session revokes the lease, which is the final step in releasing the lock.
	if err := l.session.Close(); err != nil {
		return client.NewError(client.ErrCodeConnection, "failed to close session", err)
	}

	l.logger.Info("Lock released successfully", clog.String("key", l.mutex.Key()))
	return nil
}

// TTL returns the remaining time-to-live of the lock's lease.
func (l *etcdLock) TTL(ctx context.Context) (time.Duration, error) {
	// The lease ID is available via the session.
	resp, err := l.client.Client().TimeToLive(ctx, l.session.Lease())
	if err != nil {
		return 0, client.NewError(client.ErrCodeConnection, "failed to get lock TTL", err)
	}

	if resp.TTL <= 0 {
		// This can happen if the lease expires just before this call.
		return 0, client.NewError(client.ErrCodeNotFound, "lock has expired", nil)
	}

	return time.Duration(resp.TTL) * time.Second, nil
}

// Key returns the full key path of the lock in etcd.
func (l *etcdLock) Key() string {
	return l.mutex.Key()
}
