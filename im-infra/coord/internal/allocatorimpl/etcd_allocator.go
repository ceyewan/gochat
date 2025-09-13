package allocatorimpl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/allocator"
)

const (
	// ID 分配器的根路径
	allocatorRoot = "/im-infra/allocators"
	// 默认租约 TTL
	defaultLeaseTTL = 30 * time.Second
	// 续租间隔
	keepAliveInterval = 10 * time.Second
)

// etcdInstanceIDAllocator 基于租约的实例 ID 分配器实现
type etcdInstanceIDAllocator struct {
	client       *clientv3.Client
	serviceName  string
	maxID        int
	logger       clog.Logger
	basePath     string
	session      *concurrency.Session
	sessionMu    sync.RWMutex
	leaseID      clientv3.LeaseID
	allocatedIDs map[int]struct{} // 追踪已分配的 ID，用于快速检查
	idsMu        sync.RWMutex
	closed       bool
	done         chan struct{}
}

// allocatedID 已分配 ID 的具体实现
type allocatedID struct {
	id        int
	allocator *etcdInstanceIDAllocator
	leaseID   clientv3.LeaseID
	session   *concurrency.Session
	logger    clog.Logger
	released  bool
	mu        sync.RWMutex
	closeOnce sync.Once
}

var _ allocator.InstanceIDAllocator = (*etcdInstanceIDAllocator)(nil)
var _ allocator.AllocatedID = (*allocatedID)(nil)

// NewEtcdInstanceIDAllocator 创建新的实例 ID 分配器
func NewEtcdInstanceIDAllocator(client *clientv3.Client, serviceName string, maxID int, logger clog.Logger) (allocator.InstanceIDAllocator, error) {
	allocator := &etcdInstanceIDAllocator{
		client:       client,
		serviceName:  serviceName,
		maxID:        maxID,
		logger:       logger.With(clog.String("service", serviceName)),
		basePath:     fmt.Sprintf("%s/%s/ids", allocatorRoot, serviceName),
		allocatedIDs: make(map[int]struct{}),
		done:         make(chan struct{}),
	}

	// 初始化会话
	if err := allocator.initSession(); err != nil {
		return nil, fmt.Errorf("failed to initialize allocator session: %w", err)
	}

	return allocator, nil
}

// initSession 初始化 etcd 会话
func (a *etcdInstanceIDAllocator) initSession() error {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	if a.session != nil {
		return nil
	}

	// 创建会话
	session, err := concurrency.NewSession(a.client, concurrency.WithTTL(int(defaultLeaseTTL/time.Second)))
	if err != nil {
		return fmt.Errorf("failed to create etcd session: %w", err)
	}

	a.session = session
	a.leaseID = session.Lease()

	// 启动会话保活
	go a.keepSessionAlive()

	a.logger.Info("allocator session initialized", clog.Int64("lease_id", int64(a.leaseID)))
	return nil
}

// keepSessionAlive 保持会话活跃
func (a *etcdInstanceIDAllocator) keepSessionAlive() {
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.done:
			return
		case <-ticker.C:
			a.sessionMu.RLock()
			if a.session == nil || a.closed {
				a.sessionMu.RUnlock()
				return
			}
			sessionCopy := a.session
			a.sessionMu.RUnlock()

			// 检查会话是否还活跃
			// 通过检查会话是否过期来判断其活跃状态
			select {
			case <-sessionCopy.Done():
				a.logger.Error("session expired during keepalive check")
				// 尝试重新建立会话
				if err := a.tryRecreateSession(); err != nil {
					if !errors.Is(err, errAllocatorClosed) {
						a.logger.Error("failed to recreate session", clog.Err(err))
					}
				}
			default:
				// 会话仍然活跃，无需操作
			}
		}
	}
}

// tryRecreateSession 尝试重新创建会话
func (a *etcdInstanceIDAllocator) tryRecreateSession() error {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	if a.closed {
		return errAllocatorClosed
	}

	// 关闭旧会话
	if a.session != nil {
		a.session.Close()
	}

	// 创建新会话
	session, err := concurrency.NewSession(a.client, concurrency.WithTTL(int(defaultLeaseTTL/time.Second)))
	if err != nil {
		return fmt.Errorf("failed to recreate session: %w", err)
	}

	a.session = session
	a.leaseID = session.Lease()

	// 清理已分配的 ID 映射（因为会话已改变，所有之前分配的 ID 都已释放）
	a.idsMu.Lock()
	a.allocatedIDs = make(map[int]struct{})
	a.idsMu.Unlock()

	a.logger.Info("session recreated", clog.Int64("lease_id", int64(a.leaseID)))
	return nil
}

// AcquireID 获取一个实例 ID
func (a *etcdInstanceIDAllocator) AcquireID(ctx context.Context) (allocator.AllocatedID, error) {
	if a.closed {
		return nil, fmt.Errorf("allocator is closed")
	}

	// 从 1 开始尝试获取 ID，直到找到可用的
	for id := 1; id <= a.maxID; id++ {
		allocatedID, err := a.tryAcquireID(ctx, id)
		if err == nil {
			return allocatedID, nil
		}

		// 如果是 ID 已被占用，继续尝试下一个
		if err == errIDOccupied {
			continue
		}

		// 其他错误，直接返回
		return nil, err
	}

	return nil, fmt.Errorf("no available ID found (max: %d)", a.maxID)
}

// tryAcquireID 尝试获取指定的 ID
func (a *etcdInstanceIDAllocator) tryAcquireID(ctx context.Context, id int) (allocator.AllocatedID, error) {
	a.sessionMu.RLock()
	if a.session == nil {
		a.sessionMu.RUnlock()
		return nil, fmt.Errorf("session not initialized")
	}
	session := a.session
	a.sessionMu.RUnlock()

	key := fmt.Sprintf("%s/%d", a.basePath, id)

	// 使用事务来确保原子性操作
	// 1. 检查 key 是否已存在
	// 2. 如果不存在，创建临时节点并与租约绑定
	txn := a.client.Txn(ctx)
	txn = txn.If(
		clientv3.Compare(clientv3.ModRevision(key), "=", 0),
	).Then(
		clientv3.OpPut(key, fmt.Sprintf("%d", id), clientv3.WithLease(a.leaseID)),
	)

	resp, err := txn.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire ID %d: %w", id, err)
	}

	if !resp.Succeeded {
		return nil, errIDOccupied
	}

	// 添加到已分配的 ID 映射
	a.idsMu.Lock()
	a.allocatedIDs[id] = struct{}{}
	a.idsMu.Unlock()

	// 创建已分配 ID 对象
	allocatedID := &allocatedID{
		id:        id,
		allocator: a,
		leaseID:   a.leaseID,
		session:   session,
		logger:    a.logger.With(clog.Int("id", id)),
	}

	a.logger.Info("ID acquired", clog.Int("id", id))
	return allocatedID, nil
}

var errIDOccupied = fmt.Errorf("ID already occupied")

var errAllocatorClosed = errors.New("allocator closed")

// ID 返回分配的 ID
func (id *allocatedID) ID() int {
	return id.id
}

// Close 释放 ID
func (id *allocatedID) Close(ctx context.Context) error {
	var err error
	id.closeOnce.Do(func() {
		err = id.release(ctx)
	})
	return err
}

// release 执行实际的释放操作
func (id *allocatedID) release(ctx context.Context) error {
	id.mu.Lock()
	defer id.mu.Unlock()

	if id.released {
		return nil
	}

	key := fmt.Sprintf("%s/%d", id.allocator.basePath, id.id)

	// 删除 key
	_, err := id.allocator.client.Delete(ctx, key)
	if err != nil {
		id.logger.Error("failed to release ID", clog.Err(err))
		return fmt.Errorf("failed to release ID %d: %w", id.id, err)
	}

	// 从已分配的 ID 映射中移除
	id.allocator.idsMu.Lock()
	delete(id.allocator.allocatedIDs, id.id)
	id.allocator.idsMu.Unlock()

	id.released = true
	id.logger.Info("ID released", clog.Int("id", id.id))
	return nil
}

// Close 关闭分配器
func (a *etcdInstanceIDAllocator) Close() error {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	if a.closed {
		return nil
	}

	a.closed = true
	close(a.done)

	// 关闭会话，这将自动释放所有分配的 ID
	if a.session != nil {
		a.session.Close()
		a.session = nil
	}

	a.logger.Info("allocator closed")
	return nil
}

// GetAllocatedIDs 获取当前已分配的 ID（主要用于测试和监控）
func (a *etcdInstanceIDAllocator) GetAllocatedIDs() []int {
	a.idsMu.RLock()
	defer a.idsMu.RUnlock()

	ids := make([]int, 0, len(a.allocatedIDs))
	for id := range a.allocatedIDs {
		ids = append(ids, id)
	}
	return ids
}

// IsIDAllocated 检查指定 ID 是否已被分配
func (a *etcdInstanceIDAllocator) IsIDAllocated(ctx context.Context, id int) (bool, error) {
	key := fmt.Sprintf("%s/%d", a.basePath, id)

	resp, err := a.client.Get(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check ID %d: %w", id, err)
	}

	return len(resp.Kvs) > 0, nil
}
