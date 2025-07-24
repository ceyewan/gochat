package lock

import (
	"context"
	"path"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coordination"
	"github.com/ceyewan/gochat/im-infra/coordination/pkg/client"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// EtcdDistributedLock 基于 etcd 的分布式锁实现
type EtcdDistributedLock struct {
	client *client.EtcdClient
	prefix string
	logger clog.Logger
}

// NewEtcdDistributedLock 创建新的分布式锁实例
func NewEtcdDistributedLock(client *client.EtcdClient, prefix string) *EtcdDistributedLock {
	if prefix == "" {
		prefix = "/locks"
	}

	return &EtcdDistributedLock{
		client: client,
		prefix: prefix,
		logger: clog.Module("coordination.lock"),
	}
}

// Acquire 获取互斥锁
func (l *EtcdDistributedLock) Acquire(ctx context.Context, key string, ttl time.Duration) (coordination.Lock, error) {
	return l.acquireLock(ctx, key, ttl, true)
}

// TryAcquire 尝试获取锁（非阻塞）
func (l *EtcdDistributedLock) TryAcquire(ctx context.Context, key string, ttl time.Duration) (coordination.Lock, error) {
	return l.acquireLock(ctx, key, ttl, false)
}

// acquireLock 内部获取锁的实现
func (l *EtcdDistributedLock) acquireLock(ctx context.Context, key string, ttl time.Duration, blocking bool) (coordination.Lock, error) {
	if key == "" {
		return nil, coordination.NewCoordinationError(
			coordination.ErrCodeValidation,
			"lock key cannot be empty",
			nil,
		)
	}

	if ttl <= 0 {
		return nil, coordination.NewCoordinationError(
			coordination.ErrCodeValidation,
			"lock ttl must be positive",
			nil,
		)
	}

	lockKey := path.Join(l.prefix, key)

	l.logger.Info("attempting to acquire lock",
		clog.String("key", lockKey),
		clog.Duration("ttl", ttl),
		clog.Bool("blocking", blocking))

	// 创建租约
	leaseResp, err := l.client.Grant(ctx, int64(ttl.Seconds()))
	if err != nil {
		l.logger.Error("failed to create lease for lock",
			clog.String("key", lockKey),
			clog.Err(err))
		return nil, err
	}

	leaseID := leaseResp.ID

	// 尝试获取锁
	var acquired bool
	if blocking {
		acquired, err = l.acquireBlocking(ctx, lockKey, leaseID)
	} else {
		acquired, err = l.acquireNonBlocking(ctx, lockKey, leaseID)
	}

	if err != nil {
		// 如果获取锁失败，撤销租约
		l.client.Revoke(context.Background(), leaseID)
		l.logger.Error("failed to acquire lock",
			clog.String("key", lockKey),
			clog.Err(err))
		return nil, err
	}

	if !acquired {
		// 如果没有获取到锁，撤销租约
		l.client.Revoke(context.Background(), leaseID)
		return nil, coordination.NewCoordinationError(
			coordination.ErrCodeConflict,
			"lock is already held by another client",
			nil,
		)
	}

	// 启动租约续期
	keepAliveCh, err := l.client.KeepAlive(ctx, leaseID)
	if err != nil {
		l.client.Revoke(context.Background(), leaseID)
		l.logger.Error("failed to start lease keep alive",
			clog.String("key", lockKey),
			clog.Err(err))
		return nil, err
	}

	// 创建锁对象
	lock := &EtcdLock{
		client:      l.client,
		key:         lockKey,
		leaseID:     leaseID,
		keepAliveCh: keepAliveCh,
		logger:      l.logger,
		originalTTL: ttl,
	}

	// 启动后台 goroutine 处理 keep alive 响应
	go lock.handleKeepAlive()

	l.logger.Info("lock acquired successfully",
		clog.String("key", lockKey),
		clog.Int64("lease_id", int64(leaseID)))

	return lock, nil
}

// acquireBlocking 阻塞式获取锁
func (l *EtcdDistributedLock) acquireBlocking(ctx context.Context, key string, leaseID clientv3.LeaseID) (bool, error) {
	// 创建会话
	session, err := concurrency.NewSession(l.client.Client(), concurrency.WithLease(leaseID))
	if err != nil {
		return false, coordination.NewCoordinationError(
			coordination.ErrCodeConnection,
			"failed to create etcd session",
			err,
		)
	}
	defer session.Close()

	// 创建互斥锁
	mutex := concurrency.NewMutex(session, key)

	// 获取锁
	if err := mutex.Lock(ctx); err != nil {
		if err == context.DeadlineExceeded || err == context.Canceled {
			return false, coordination.NewCoordinationError(
				coordination.ErrCodeTimeout,
				"lock acquisition timeout",
				err,
			)
		}
		return false, coordination.NewCoordinationError(
			coordination.ErrCodeConnection,
			"failed to acquire lock",
			err,
		)
	}

	return true, nil
}

// acquireNonBlocking 非阻塞式获取锁
func (l *EtcdDistributedLock) acquireNonBlocking(ctx context.Context, key string, leaseID clientv3.LeaseID) (bool, error) {
	// 使用原子操作尝试创建锁
	txn := l.client.Txn(ctx)
	txn = txn.If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0))
	txn = txn.Then(clientv3.OpPut(key, "", clientv3.WithLease(leaseID)))

	resp, err := txn.Commit()
	if err != nil {
		return false, coordination.NewCoordinationError(
			coordination.ErrCodeConnection,
			"failed to execute lock transaction",
			err,
		)
	}

	return resp.Succeeded, nil
}

// EtcdLock 基于 etcd 的锁实现
type EtcdLock struct {
	client      *client.EtcdClient
	key         string
	leaseID     clientv3.LeaseID
	keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse
	logger      clog.Logger
	originalTTL time.Duration
	closed      bool
}

// Unlock 释放锁
func (l *EtcdLock) Unlock(ctx context.Context) error {
	if l.closed {
		return coordination.NewCoordinationError(
			coordination.ErrCodeValidation,
			"lock is already closed",
			nil,
		)
	}

	l.logger.Info("releasing lock",
		clog.String("key", l.key),
		clog.Int64("lease_id", int64(l.leaseID)))

	// 撤销租约，这会自动删除锁
	_, err := l.client.Revoke(ctx, l.leaseID)
	if err != nil {
		l.logger.Error("failed to revoke lease",
			clog.String("key", l.key),
			clog.Int64("lease_id", int64(l.leaseID)),
			clog.Err(err))
		return coordination.NewCoordinationError(
			coordination.ErrCodeConnection,
			"failed to release lock",
			err,
		)
	}

	l.closed = true
	l.logger.Info("lock released successfully",
		clog.String("key", l.key))

	return nil
}

// Renew 续期锁
func (l *EtcdLock) Renew(ctx context.Context, ttl time.Duration) error {
	if l.closed {
		return coordination.NewCoordinationError(
			coordination.ErrCodeValidation,
			"lock is already closed",
			nil,
		)
	}

	if ttl <= 0 {
		return coordination.NewCoordinationError(
			coordination.ErrCodeValidation,
			"ttl must be positive",
			nil,
		)
	}

	l.logger.Info("renewing lock",
		clog.String("key", l.key),
		clog.Duration("ttl", ttl))

	// etcd 的租约续期是通过 KeepAlive 自动处理的
	// 这里我们只是更新 TTL 记录用于 TTL() 方法
	l.originalTTL = ttl

	l.logger.Info("lock renewed successfully",
		clog.String("key", l.key),
		clog.Duration("new_ttl", ttl))

	return nil
}

// TTL 获取锁的剩余有效时间
func (l *EtcdLock) TTL(ctx context.Context) (time.Duration, error) {
	if l.closed {
		return 0, coordination.NewCoordinationError(
			coordination.ErrCodeValidation,
			"lock is already closed",
			nil,
		)
	}

	// 查询租约信息
	resp, err := l.client.Client().TimeToLive(ctx, l.leaseID)
	if err != nil {
		l.logger.Error("failed to get lease TTL",
			clog.String("key", l.key),
			clog.Int64("lease_id", int64(l.leaseID)),
			clog.Err(err))
		return 0, coordination.NewCoordinationError(
			coordination.ErrCodeConnection,
			"failed to get lock TTL",
			err,
		)
	}

	if resp.TTL <= 0 {
		return 0, coordination.NewCoordinationError(
			coordination.ErrCodeNotFound,
			"lock has expired",
			nil,
		)
	}

	return time.Duration(resp.TTL) * time.Second, nil
}

// Key 获取锁的键
func (l *EtcdLock) Key() string {
	return l.key
}

// handleKeepAlive 处理租约续期响应
func (l *EtcdLock) handleKeepAlive() {
	for resp := range l.keepAliveCh {
		if resp == nil {
			l.logger.Warn("keep alive channel closed",
				clog.String("key", l.key),
				clog.Int64("lease_id", int64(l.leaseID)))
			break
		}

		l.logger.Debug("lease keep alive response received",
			clog.String("key", l.key),
			clog.Int64("lease_id", int64(l.leaseID)),
			clog.Int64("ttl", resp.TTL))
	}
}
