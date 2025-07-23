package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// distributedLock 是 DistributedLock 接口的内部实现
type distributedLock struct {
	client    *clientv3.Client
	config    DistributedLockConfig
	logger    clog.Logger
	sessions  map[string]*concurrency.Session // 会话映射
	sessionMu sync.RWMutex
	closed    bool
	closeMu   sync.RWMutex
}

// newDistributedLock 创建新的分布式锁实例
func newDistributedLock(client *clientv3.Client, config DistributedLockConfig, logger clog.Logger) DistributedLock {
	return &distributedLock{
		client:   client,
		config:   config,
		logger:   logger,
		sessions: make(map[string]*concurrency.Session),
	}
}

// Acquire 获取基础分布式锁
func (dl *distributedLock) Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	dl.closeMu.RLock()
	defer dl.closeMu.RUnlock()

	if dl.closed {
		return nil, fmt.Errorf("distributed lock is closed")
	}

	// 创建会话
	session, err := dl.getOrCreateSession(ctx, key, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// 构建锁键名
	lockKey := dl.buildLockKey(key)

	// 创建互斥锁
	mutex := concurrency.NewMutex(session, lockKey)

	// 获取锁
	start := time.Now()
	err = mutex.Lock(ctx)
	if err != nil {
		dl.logger.Error("获取分布式锁失败",
			clog.Err(err),
			clog.String("key", key),
			clog.Duration("ttl", ttl),
		)
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	duration := time.Since(start)
	dl.logger.Debug("获取分布式锁成功",
		clog.String("key", key),
		clog.Duration("ttl", ttl),
		clog.Duration("acquire_time", duration),
	)

	lock := &basicLock{
		key:     key,
		mutex:   mutex,
		session: session,
		logger:  dl.logger,
		ttl:     ttl,
	}

	// 启动自动续期
	go lock.autoRenew(ctx, dl.config.RenewInterval)

	return lock, nil
}

// AcquireReentrant 获取可重入分布式锁
func (dl *distributedLock) AcquireReentrant(ctx context.Context, key string, ttl time.Duration) (ReentrantLock, error) {
	dl.closeMu.RLock()
	defer dl.closeMu.RUnlock()

	if dl.closed {
		return nil, fmt.Errorf("distributed lock is closed")
	}

	if !dl.config.EnableReentrant {
		return nil, fmt.Errorf("reentrant lock is not enabled")
	}

	// 创建会话
	session, err := dl.getOrCreateSession(ctx, key, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// 构建锁键名
	lockKey := dl.buildLockKey(key)

	// 创建互斥锁
	mutex := concurrency.NewMutex(session, lockKey)

	// 获取锁
	start := time.Now()
	err = mutex.Lock(ctx)
	if err != nil {
		dl.logger.Error("获取可重入锁失败",
			clog.Err(err),
			clog.String("key", key),
			clog.Duration("ttl", ttl),
		)
		return nil, fmt.Errorf("failed to acquire reentrant lock: %w", err)
	}

	duration := time.Since(start)
	dl.logger.Debug("获取可重入锁成功",
		clog.String("key", key),
		clog.Duration("ttl", ttl),
		clog.Duration("acquire_time", duration),
	)

	lock := &reentrantLock{
		basicLock: basicLock{
			key:     key,
			mutex:   mutex,
			session: session,
			logger:  dl.logger,
			ttl:     ttl,
		},
		acquireCount: 1,
	}

	// 启动自动续期
	go lock.autoRenew(ctx, dl.config.RenewInterval)

	return lock, nil
}

// AcquireReadLock 获取读锁
func (dl *distributedLock) AcquireReadLock(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	dl.closeMu.RLock()
	defer dl.closeMu.RUnlock()

	if dl.closed {
		return nil, fmt.Errorf("distributed lock is closed")
	}

	// 读锁使用特殊的键前缀
	readKey := fmt.Sprintf("%s:read", key)
	return dl.Acquire(ctx, readKey, ttl)
}

// AcquireWriteLock 获取写锁
func (dl *distributedLock) AcquireWriteLock(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	dl.closeMu.RLock()
	defer dl.closeMu.RUnlock()

	if dl.closed {
		return nil, fmt.Errorf("distributed lock is closed")
	}

	// 写锁使用特殊的键前缀
	writeKey := fmt.Sprintf("%s:write", key)
	return dl.Acquire(ctx, writeKey, ttl)
}

// 辅助方法

// getOrCreateSession 获取或创建会话
func (dl *distributedLock) getOrCreateSession(ctx context.Context, key string, ttl time.Duration) (*concurrency.Session, error) {
	dl.sessionMu.Lock()
	defer dl.sessionMu.Unlock()

	sessionKey := fmt.Sprintf("%s:%d", key, int64(ttl.Seconds()))

	if session, exists := dl.sessions[sessionKey]; exists {
		// 检查会话是否仍然有效
		select {
		case <-session.Done():
			// 会话已失效，删除并创建新的
			delete(dl.sessions, sessionKey)
		default:
			// 会话仍然有效，直接返回
			return session, nil
		}
	}

	// 创建新会话
	session, err := concurrency.NewSession(dl.client,
		concurrency.WithTTL(int(dl.config.SessionTTL.Seconds())),
		concurrency.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	dl.sessions[sessionKey] = session

	// 监听会话关闭
	go func() {
		<-session.Done()
		dl.sessionMu.Lock()
		delete(dl.sessions, sessionKey)
		dl.sessionMu.Unlock()
		dl.logger.Debug("会话已关闭", clog.String("session_key", sessionKey))
	}()

	dl.logger.Debug("创建新会话",
		clog.String("session_key", sessionKey),
		clog.Duration("session_ttl", dl.config.SessionTTL),
	)

	return session, nil
}

// buildLockKey 构建锁键名
func (dl *distributedLock) buildLockKey(key string) string {
	return fmt.Sprintf("%s/%s", dl.config.KeyPrefix, key)
}

// Close 关闭分布式锁
func (dl *distributedLock) Close() error {
	dl.closeMu.Lock()
	defer dl.closeMu.Unlock()

	if dl.closed {
		return nil
	}

	dl.closed = true

	// 关闭所有会话
	dl.sessionMu.Lock()
	for _, session := range dl.sessions {
		session.Close()
	}
	dl.sessions = make(map[string]*concurrency.Session)
	dl.sessionMu.Unlock()

	dl.logger.Info("分布式锁已关闭")
	return nil
}

// basicLock 基础锁实现
type basicLock struct {
	key      string
	mutex    *concurrency.Mutex
	session  *concurrency.Session
	logger   clog.Logger
	ttl      time.Duration
	released bool
	mu       sync.Mutex
}

// Release 释放锁
func (bl *basicLock) Release(ctx context.Context) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if bl.released {
		return fmt.Errorf("lock already released")
	}

	err := bl.mutex.Unlock(ctx)
	if err != nil {
		bl.logger.Error("释放锁失败",
			clog.Err(err),
			clog.String("key", bl.key),
		)
		return fmt.Errorf("failed to release lock: %w", err)
	}

	bl.released = true
	bl.logger.Debug("释放锁成功", clog.String("key", bl.key))
	return nil
}

// Renew 续期锁
func (bl *basicLock) Renew(ctx context.Context, ttl time.Duration) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if bl.released {
		return fmt.Errorf("cannot renew released lock")
	}

	// etcd concurrency 包会自动处理租约续期
	bl.ttl = ttl
	bl.logger.Debug("锁续期成功",
		clog.String("key", bl.key),
		clog.Duration("new_ttl", ttl),
	)
	return nil
}

// IsHeld 检查锁是否仍被持有
func (bl *basicLock) IsHeld(ctx context.Context) (bool, error) {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if bl.released {
		return false, nil
	}

	// 检查会话是否仍然活跃
	select {
	case <-bl.session.Done():
		return false, nil
	default:
		return true, nil
	}
}

// Key 返回锁的键名
func (bl *basicLock) Key() string {
	return bl.key
}

// TTL 返回锁的剩余生存时间
func (bl *basicLock) TTL(ctx context.Context) (time.Duration, error) {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if bl.released {
		return 0, fmt.Errorf("lock is released")
	}

	// 这里返回配置的 TTL，实际的 TTL 由 etcd 管理
	return bl.ttl, nil
}

// autoRenew 自动续期
func (bl *basicLock) autoRenew(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bl.session.Done():
			return
		case <-ticker.C:
			bl.mu.Lock()
			if bl.released {
				bl.mu.Unlock()
				return
			}
			bl.mu.Unlock()

			// etcd concurrency 包会自动处理租约续期
			bl.logger.Debug("锁自动续期", clog.String("key", bl.key))
		}
	}
}

// reentrantLock 可重入锁实现
type reentrantLock struct {
	basicLock
	acquireCount int
	countMu      sync.Mutex
}

// AcquireCount 返回当前锁的获取次数
func (rl *reentrantLock) AcquireCount() int {
	rl.countMu.Lock()
	defer rl.countMu.Unlock()
	return rl.acquireCount
}

// Release 释放一次锁，只有当获取次数为0时才真正释放
func (rl *reentrantLock) Release(ctx context.Context) error {
	rl.countMu.Lock()
	defer rl.countMu.Unlock()

	if rl.released {
		return fmt.Errorf("lock already released")
	}

	if rl.acquireCount <= 0 {
		return fmt.Errorf("lock not acquired")
	}

	rl.acquireCount--

	if rl.acquireCount == 0 {
		// 真正释放锁
		return rl.basicLock.Release(ctx)
	}

	rl.logger.Debug("可重入锁部分释放",
		clog.String("key", rl.key),
		clog.Int("remaining_count", rl.acquireCount),
	)
	return nil
}

// Acquire 再次获取锁（可重入）
func (rl *reentrantLock) Acquire(ctx context.Context) error {
	rl.countMu.Lock()
	defer rl.countMu.Unlock()

	if rl.released {
		return fmt.Errorf("cannot acquire released lock")
	}

	// 对于可重入锁，同一个会话可以多次获取
	rl.acquireCount++

	rl.logger.Debug("可重入锁再次获取",
		clog.String("key", rl.key),
		clog.Int("acquire_count", rl.acquireCount),
	)
	return nil
}
