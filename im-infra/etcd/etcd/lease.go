package etcd

import (
	"context"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// etcdLeaseManager 实现 LeaseManager 接口
type etcdLeaseManager struct {
	connMgr ConnectionManager
	logger  Logger

	// 租约管理
	leases map[clientv3.LeaseID]*leaseInfo
	mu     sync.RWMutex
}

// leaseInfo 租约信息
type leaseInfo struct {
	ID        clientv3.LeaseID
	TTL       int64
	CreatedAt time.Time
	Cancel    context.CancelFunc
}

// NewLeaseManager 创建新的租约管理器
func NewLeaseManager(connMgr ConnectionManager, logger Logger) LeaseManager {
	return &etcdLeaseManager{
		connMgr: connMgr,
		logger:  logger,
		leases:  make(map[clientv3.LeaseID]*leaseInfo),
	}
}

// CreateLease 创建租约
func (lm *etcdLeaseManager) CreateLease(ctx context.Context, ttl int64) (clientv3.LeaseID, error) {
	if !lm.connMgr.IsConnected() {
		return 0, ErrNotConnected
	}

	client := lm.connMgr.GetClient()
	if client == nil {
		return 0, ErrNotConnected
	}

	// 创建租约
	resp, err := client.Grant(ctx, ttl)
	if err != nil {
		lm.logger.Errorf("Failed to create lease with TTL %d: %v", ttl, err)
		return 0, WrapLeaseError(err, "failed to create lease")
	}

	// 记录租约信息
	lm.mu.Lock()
	lm.leases[resp.ID] = &leaseInfo{
		ID:        resp.ID,
		TTL:       ttl,
		CreatedAt: time.Now(),
	}
	lm.mu.Unlock()

	lm.logger.Infof("Created lease %d with TTL %d seconds", resp.ID, ttl)
	return resp.ID, nil
}

// KeepAlive 保持租约活跃
func (lm *etcdLeaseManager) KeepAlive(ctx context.Context, leaseID clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	if !lm.connMgr.IsConnected() {
		return nil, ErrNotConnected
	}

	client := lm.connMgr.GetClient()
	if client == nil {
		return nil, ErrNotConnected
	}

	// 检查租约是否存在
	lm.mu.RLock()
	lease, exists := lm.leases[leaseID]
	lm.mu.RUnlock()

	if !exists {
		return nil, WrapLeaseError(ErrLeaseNotFound, "lease not found in manager")
	}

	// 如果已经有 KeepAlive，先取消
	if lease.Cancel != nil {
		lease.Cancel()
	}

	// 创建新的 KeepAlive 上下文
	keepAliveCtx, cancel := context.WithCancel(ctx)

	// 启动 KeepAlive
	keepAliveCh, err := client.KeepAlive(keepAliveCtx, leaseID)
	if err != nil {
		cancel()
		lm.logger.Errorf("Failed to start keepalive for lease %d: %v", leaseID, err)
		return nil, WrapLeaseError(err, "failed to start lease keepalive")
	}

	// 更新租约信息
	lm.mu.Lock()
	lease.Cancel = cancel
	lm.mu.Unlock()

	// 启动监控 goroutine
	go lm.monitorKeepAlive(leaseID, keepAliveCh, cancel)

	lm.logger.Infof("Started keepalive for lease %d", leaseID)
	return keepAliveCh, nil
}

// RevokeLease 撤销租约
func (lm *etcdLeaseManager) RevokeLease(ctx context.Context, leaseID clientv3.LeaseID) error {
	if !lm.connMgr.IsConnected() {
		return ErrNotConnected
	}

	client := lm.connMgr.GetClient()
	if client == nil {
		return ErrNotConnected
	}

	// 停止 KeepAlive
	lm.mu.Lock()
	if lease, exists := lm.leases[leaseID]; exists {
		if lease.Cancel != nil {
			lease.Cancel()
		}
		delete(lm.leases, leaseID)
	}
	lm.mu.Unlock()

	// 撤销租约
	_, err := client.Revoke(ctx, leaseID)
	if err != nil {
		lm.logger.Errorf("Failed to revoke lease %d: %v", leaseID, err)
		return WrapLeaseError(err, "failed to revoke lease")
	}

	lm.logger.Infof("Revoked lease %d", leaseID)
	return nil
}

// GetLeaseInfo 获取租约信息
func (lm *etcdLeaseManager) GetLeaseInfo(ctx context.Context, leaseID clientv3.LeaseID) (*clientv3.LeaseTimeToLiveResponse, error) {
	if !lm.connMgr.IsConnected() {
		return nil, ErrNotConnected
	}

	client := lm.connMgr.GetClient()
	if client == nil {
		return nil, ErrNotConnected
	}

	resp, err := client.TimeToLive(ctx, leaseID)
	if err != nil {
		lm.logger.Errorf("Failed to get lease info for %d: %v", leaseID, err)
		return nil, WrapLeaseError(err, "failed to get lease info")
	}

	if resp.TTL == -1 {
		// 租约不存在或已过期
		lm.mu.Lock()
		delete(lm.leases, leaseID)
		lm.mu.Unlock()
		return nil, WrapLeaseError(ErrLeaseNotFound, "lease not found or expired")
	}

	return resp, nil
}

// ListLeases 列出所有租约
func (lm *etcdLeaseManager) ListLeases(ctx context.Context) ([]clientv3.LeaseStatus, error) {
	if !lm.connMgr.IsConnected() {
		return nil, ErrNotConnected
	}

	client := lm.connMgr.GetClient()
	if client == nil {
		return nil, ErrNotConnected
	}

	resp, err := client.Leases(ctx)
	if err != nil {
		lm.logger.Errorf("Failed to list leases: %v", err)
		return nil, WrapLeaseError(err, "failed to list leases")
	}

	return resp.Leases, nil
}

// RefreshLease 刷新租约TTL
func (lm *etcdLeaseManager) RefreshLease(ctx context.Context, leaseID clientv3.LeaseID, ttl int64) error {
	// etcd 不支持直接刷新租约 TTL，需要重新创建
	// 这里提供一个替代方案：撤销旧租约并创建新租约

	// 检查租约是否存在
	lm.mu.RLock()
	_, exists := lm.leases[leaseID]
	lm.mu.RUnlock()

	if !exists {
		return WrapLeaseError(ErrLeaseNotFound, "lease not found in manager")
	}

	// 创建新租约
	newLeaseID, err := lm.CreateLease(ctx, ttl)
	if err != nil {
		return WrapLeaseError(err, "failed to create new lease for refresh")
	}

	// 撤销旧租约
	if err := lm.RevokeLease(ctx, leaseID); err != nil {
		lm.logger.Warnf("Failed to revoke old lease %d during refresh: %v", leaseID, err)
	}

	lm.logger.Infof("Refreshed lease %d -> %d with TTL %d", leaseID, newLeaseID, ttl)
	return nil
}

// monitorKeepAlive 监控 KeepAlive 响应
func (lm *etcdLeaseManager) monitorKeepAlive(leaseID clientv3.LeaseID, keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse, cancel context.CancelFunc) {
	defer cancel()

	for resp := range keepAliveCh {
		if resp == nil {
			lm.logger.Warnf("Received nil keepalive response for lease %d", leaseID)
			continue
		}

		lm.logger.Debugf("Lease %d renewed, TTL: %d", leaseID, resp.TTL)

		// 更新租约信息
		lm.mu.Lock()
		if lease, exists := lm.leases[leaseID]; exists {
			lease.TTL = resp.TTL
		}
		lm.mu.Unlock()
	}

	// 通道关闭，清理租约信息
	lm.logger.Warnf("KeepAlive channel closed for lease %d", leaseID)
	lm.mu.Lock()
	delete(lm.leases, leaseID)
	lm.mu.Unlock()
}

// GetManagedLeases 获取管理器管理的所有租约
func (lm *etcdLeaseManager) GetManagedLeases() map[clientv3.LeaseID]*leaseInfo {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	result := make(map[clientv3.LeaseID]*leaseInfo)
	for id, info := range lm.leases {
		result[id] = &leaseInfo{
			ID:        info.ID,
			TTL:       info.TTL,
			CreatedAt: info.CreatedAt,
		}
	}
	return result
}

// CleanupExpiredLeases 清理过期的租约
func (lm *etcdLeaseManager) CleanupExpiredLeases(ctx context.Context) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	var expiredLeases []clientv3.LeaseID

	for leaseID := range lm.leases {
		// 检查租约是否仍然有效
		info, err := lm.GetLeaseInfo(ctx, leaseID)
		if err != nil || info.TTL == -1 {
			expiredLeases = append(expiredLeases, leaseID)
		}
	}

	// 清理过期租约
	for _, leaseID := range expiredLeases {
		if lease, exists := lm.leases[leaseID]; exists {
			if lease.Cancel != nil {
				lease.Cancel()
			}
			delete(lm.leases, leaseID)
		}
		lm.logger.Infof("Cleaned up expired lease %d", leaseID)
	}

	return nil
}

// Close 关闭租约管理器
func (lm *etcdLeaseManager) Close() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// 取消所有 KeepAlive
	for leaseID, lease := range lm.leases {
		if lease.Cancel != nil {
			lease.Cancel()
		}
		lm.logger.Infof("Stopped keepalive for lease %d", leaseID)
	}

	// 清空租约映射
	lm.leases = make(map[clientv3.LeaseID]*leaseInfo)

	lm.logger.Info("Lease manager closed")
	return nil
}
