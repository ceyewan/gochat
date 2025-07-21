package etcd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// etcdDistributedLock 实现 DistributedLock 接口
type etcdDistributedLock struct {
	connMgr    ConnectionManager
	leaseMgr   LeaseManager
	logger     Logger
	lockPrefix string

	// 锁管理
	locks map[string]*lockEntry
	mu    sync.RWMutex
}

// lockEntry 锁条目
type lockEntry struct {
	Key       string
	Session   *concurrency.Session
	Mutex     *concurrency.Mutex
	LeaseID   clientv3.LeaseID
	Owner     string
	CreatedAt time.Time
	TTL       time.Duration
}

// NewDistributedLock 创建新的分布式锁管理器
func NewDistributedLock(connMgr ConnectionManager, leaseMgr LeaseManager, logger Logger, lockPrefix string) DistributedLock {
	etcdLogger := clog.Module("etcd")
	etcdLogger.Info("创建分布式锁管理器", clog.String("lock_prefix", lockPrefix))

	if logger == nil {
		logger = NewClogAdapter(etcdLogger)
	}

	return &etcdDistributedLock{
		connMgr:    connMgr,
		leaseMgr:   leaseMgr,
		logger:     logger,
		lockPrefix: lockPrefix,
		locks:      make(map[string]*lockEntry),
	}
}

// Lock 获取分布式锁
func (dl *etcdDistributedLock) Lock(ctx context.Context, key string, ttl time.Duration) error {
	dl.logger.Infof("尝试获取分布式锁: %s, TTL: %v", key, ttl)

	if !dl.connMgr.IsConnected() {
		dl.logger.Error("etcd 连接未建立")
		return ErrNotConnected
	}

	lockKey := dl.getLockKey(key)
	dl.logger.Debugf("锁键: %s", lockKey)

	// 检查是否已经持有锁
	dl.mu.RLock()
	if _, exists := dl.locks[lockKey]; exists {
		dl.mu.RUnlock()
		dl.logger.Warnf("锁已被持有: %s", key)
		return WrapLockError(ErrLockAlreadyHeld, fmt.Sprintf("lock already held for key: %s", key))
	}
	dl.mu.RUnlock()

	client := dl.connMgr.GetClient()
	if client == nil {
		dl.logger.Error("etcd 客户端为空")
		return ErrNotConnected
	}

	// 创建租约
	dl.logger.Debugf("为锁创建租约: %s, TTL: %d 秒", key, int64(ttl.Seconds()))
	leaseID, err := dl.leaseMgr.CreateLease(ctx, int64(ttl.Seconds()))
	if err != nil {
		dl.logger.Errorf("创建锁租约失败: %v, key: %s", err, key)
		return WrapLockError(err, "failed to create lease for lock")
	}
	dl.logger.Debugf("锁租约创建成功: %s, lease: %d", key, leaseID)

	// 启动租约保活
	dl.logger.Debug("启动锁租约保活")
	_, err = dl.leaseMgr.KeepAlive(ctx, leaseID)
	if err != nil {
		dl.logger.Errorf("启动锁租约保活失败: %v, key: %s", err, key)
		dl.leaseMgr.RevokeLease(ctx, leaseID)
		return WrapLockError(err, "failed to start lease keepalive")
	}

	// 创建会话
	session, err := concurrency.NewSession(client, concurrency.WithLease(leaseID))
	if err != nil {
		dl.leaseMgr.RevokeLease(ctx, leaseID)
		return WrapLockError(err, "failed to create session")
	}

	// 创建互斥锁
	mutex := concurrency.NewMutex(session, lockKey)

	// 尝试获取锁
	err = mutex.Lock(ctx)
	if err != nil {
		session.Close()
		dl.leaseMgr.RevokeLease(ctx, leaseID)
		return WrapLockError(err, "failed to acquire lock")
	}

	// 记录锁信息
	dl.mu.Lock()
	dl.locks[lockKey] = &lockEntry{
		Key:       lockKey,
		Session:   session,
		Mutex:     mutex,
		LeaseID:   leaseID,
		Owner:     dl.generateOwnerID(),
		CreatedAt: time.Now(),
		TTL:       ttl,
	}
	dl.mu.Unlock()

	dl.logger.Infof("Successfully acquired lock for key: %s", key)
	return nil
}

// TryLock 尝试获取分布式锁，不阻塞
func (dl *etcdDistributedLock) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if !dl.connMgr.IsConnected() {
		return false, ErrNotConnected
	}

	lockKey := dl.getLockKey(key)

	// 检查是否已经持有锁
	dl.mu.RLock()
	if _, exists := dl.locks[lockKey]; exists {
		dl.mu.RUnlock()
		return false, WrapLockError(ErrLockAlreadyHeld, fmt.Sprintf("lock already held for key: %s", key))
	}
	dl.mu.RUnlock()

	client := dl.connMgr.GetClient()
	if client == nil {
		return false, ErrNotConnected
	}

	// 创建租约
	leaseID, err := dl.leaseMgr.CreateLease(ctx, int64(ttl.Seconds()))
	if err != nil {
		return false, WrapLockError(err, "failed to create lease for lock")
	}

	// 创建会话
	session, err := concurrency.NewSession(client, concurrency.WithLease(leaseID))
	if err != nil {
		dl.leaseMgr.RevokeLease(ctx, leaseID)
		return false, WrapLockError(err, "failed to create session")
	}

	// 创建互斥锁
	mutex := concurrency.NewMutex(session, lockKey)

	// 尝试获取锁（非阻塞）
	tryCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err = mutex.Lock(tryCtx)
	if err != nil {
		session.Close()
		dl.leaseMgr.RevokeLease(ctx, leaseID)

		// 检查是否是超时错误
		if tryCtx.Err() == context.DeadlineExceeded {
			return false, nil // 锁被其他进程持有，但不是错误
		}
		return false, WrapLockError(err, "failed to try acquire lock")
	}

	// 启动租约保活
	_, err = dl.leaseMgr.KeepAlive(ctx, leaseID)
	if err != nil {
		mutex.Unlock(ctx)
		session.Close()
		dl.leaseMgr.RevokeLease(ctx, leaseID)
		return false, WrapLockError(err, "failed to start lease keepalive")
	}

	// 记录锁信息
	dl.mu.Lock()
	dl.locks[lockKey] = &lockEntry{
		Key:       lockKey,
		Session:   session,
		Mutex:     mutex,
		LeaseID:   leaseID,
		Owner:     dl.generateOwnerID(),
		CreatedAt: time.Now(),
		TTL:       ttl,
	}
	dl.mu.Unlock()

	dl.logger.Infof("Successfully try-acquired lock for key: %s", key)
	return true, nil
}

// Unlock 释放分布式锁
func (dl *etcdDistributedLock) Unlock(ctx context.Context, key string) error {
	lockKey := dl.getLockKey(key)

	dl.mu.Lock()
	entry, exists := dl.locks[lockKey]
	if !exists {
		dl.mu.Unlock()
		return WrapLockError(ErrLockNotHeld, fmt.Sprintf("lock not held for key: %s", key))
	}
	delete(dl.locks, lockKey)
	dl.mu.Unlock()

	// 释放锁
	err := entry.Mutex.Unlock(ctx)
	if err != nil {
		dl.logger.Warnf("Failed to unlock mutex for key %s: %v", key, err)
	}

	// 关闭会话
	err = entry.Session.Close()
	if err != nil {
		dl.logger.Warnf("Failed to close session for key %s: %v", key, err)
	}

	// 撤销租约
	err = dl.leaseMgr.RevokeLease(ctx, entry.LeaseID)
	if err != nil {
		dl.logger.Warnf("Failed to revoke lease for key %s: %v", key, err)
	}

	dl.logger.Infof("Successfully released lock for key: %s", key)
	return nil
}

// Refresh 刷新锁的TTL
func (dl *etcdDistributedLock) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	lockKey := dl.getLockKey(key)

	dl.mu.RLock()
	entry, exists := dl.locks[lockKey]
	dl.mu.RUnlock()

	if !exists {
		return WrapLockError(ErrLockNotHeld, fmt.Sprintf("lock not held for key: %s", key))
	}

	// 刷新租约TTL
	err := dl.leaseMgr.RefreshLease(ctx, entry.LeaseID, int64(ttl.Seconds()))
	if err != nil {
		return WrapLockError(err, "failed to refresh lock lease")
	}

	// 更新锁信息
	dl.mu.Lock()
	entry.TTL = ttl
	dl.mu.Unlock()

	dl.logger.Infof("Successfully refreshed lock for key: %s, new TTL: %v", key, ttl)
	return nil
}

// IsLocked 检查锁是否被持有
func (dl *etcdDistributedLock) IsLocked(ctx context.Context, key string) (bool, error) {
	if !dl.connMgr.IsConnected() {
		return false, ErrNotConnected
	}

	lockKey := dl.getLockKey(key)

	// 先检查本地是否持有锁
	dl.mu.RLock()
	_, locallyHeld := dl.locks[lockKey]
	dl.mu.RUnlock()

	if locallyHeld {
		return true, nil
	}

	// 检查 etcd 中是否有锁
	client := dl.connMgr.GetClient()
	if client == nil {
		return false, ErrNotConnected
	}

	resp, err := client.Get(ctx, lockKey, clientv3.WithPrefix())
	if err != nil {
		return false, WrapLockError(err, "failed to check lock status")
	}

	return len(resp.Kvs) > 0, nil
}

// GetLockInfo 获取锁的详细信息
func (dl *etcdDistributedLock) GetLockInfo(ctx context.Context, key string) (*LockInfo, error) {
	lockKey := dl.getLockKey(key)

	dl.mu.RLock()
	entry, exists := dl.locks[lockKey]
	dl.mu.RUnlock()

	if !exists {
		return nil, WrapLockError(ErrLockNotHeld, fmt.Sprintf("lock not held for key: %s", key))
	}

	// 获取租约信息
	leaseInfo, err := dl.leaseMgr.GetLeaseInfo(ctx, entry.LeaseID)
	if err != nil {
		return nil, WrapLockError(err, "failed to get lease info")
	}

	return &LockInfo{
		Key:     key,
		Owner:   entry.Owner,
		LeaseID: entry.LeaseID,
		TTL:     leaseInfo.TTL,
		Created: entry.CreatedAt,
	}, nil
}

// getLockKey 获取锁的完整键名
func (dl *etcdDistributedLock) getLockKey(key string) string {
	return fmt.Sprintf("%s/%s", dl.lockPrefix, key)
}

// generateOwnerID 生成锁拥有者ID
func (dl *etcdDistributedLock) generateOwnerID() string {
	// 这里可以使用更复杂的ID生成策略
	return fmt.Sprintf("etcd-lock-%d", time.Now().UnixNano())
}

// GetHeldLocks 获取当前持有的所有锁
func (dl *etcdDistributedLock) GetHeldLocks() []string {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	var keys []string
	for lockKey := range dl.locks {
		// 移除前缀，返回原始键名
		if len(lockKey) > len(dl.lockPrefix)+1 {
			key := lockKey[len(dl.lockPrefix)+1:]
			keys = append(keys, key)
		}
	}
	return keys
}

// CleanupExpiredLocks 清理过期的锁
func (dl *etcdDistributedLock) CleanupExpiredLocks(ctx context.Context) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	var expiredKeys []string

	for lockKey, entry := range dl.locks {
		// 检查租约是否仍然有效
		_, err := dl.leaseMgr.GetLeaseInfo(ctx, entry.LeaseID)
		if err != nil {
			expiredKeys = append(expiredKeys, lockKey)
		}
	}

	// 清理过期锁
	for _, lockKey := range expiredKeys {
		entry := dl.locks[lockKey]

		// 关闭会话
		if entry.Session != nil {
			entry.Session.Close()
		}

		delete(dl.locks, lockKey)
		dl.logger.Infof("Cleaned up expired lock: %s", lockKey)
	}

	return nil
}

// Close 关闭分布式锁管理器
func (dl *etcdDistributedLock) Close() error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// 释放所有锁
	for lockKey, entry := range dl.locks {
		if entry.Mutex != nil {
			entry.Mutex.Unlock(context.Background())
		}
		if entry.Session != nil {
			entry.Session.Close()
		}
		dl.logger.Infof("Released lock during close: %s", lockKey)
	}

	// 清空锁映射
	dl.locks = make(map[string]*lockEntry)

	dl.logger.Info("Distributed lock manager closed")
	return nil
}
