package internal

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/twmb/franz-go/pkg/kgo"
)

// connectionPool 连接池实现
type connectionPool struct {
	// 配置
	config Config

	// 连接池
	connections chan *pooledConnection

	// 统计信息
	stats poolStats

	// 状态管理
	closed int32
	mu     sync.RWMutex

	// 日志器
	logger clog.Logger

	// 健康检查
	healthCheckTicker *time.Ticker
	healthCheckDone   chan struct{}
}

// pooledConnection 池化连接
type pooledConnection struct {
	// Kafka客户端
	client *kgo.Client

	// 创建时间
	createdAt time.Time

	// 最后使用时间
	lastUsedAt time.Time

	// 使用次数
	useCount int64

	// 是否健康
	healthy bool

	// 互斥锁
	mu sync.RWMutex
}

// poolStats 连接池统计信息的内部实现
type poolStats struct {
	totalConnections   int32
	activeConnections  int32
	idleConnections    int32
	maxConnections     int32
	connectionsCreated int64
	connectionsClosed  int64
	connectionErrors   int64
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool(cfg Config) (ConnectionPool, error) {
	if err := validatePoolConfig(cfg.PoolConfig); err != nil {
		return nil, NewConfigError("连接池配置无效", err)
	}

	pool := &connectionPool{
		config:      cfg,
		connections: make(chan *pooledConnection, cfg.PoolConfig.MaxConnections),
		stats: poolStats{
			maxConnections: int32(cfg.PoolConfig.MaxConnections),
		},
		logger:          clog.Namespace("mq.pool"),
		healthCheckDone: make(chan struct{}),
	}

	// 启动健康检查
	pool.startHealthCheck()

	// 预创建最小空闲连接
	if err := pool.preCreateConnections(); err != nil {
		pool.logger.Warn("预创建连接失败", clog.Err(err))
	}

	pool.logger.Info("连接池创建成功",
		clog.Int("max_connections", cfg.PoolConfig.MaxConnections),
		clog.Int("min_idle", cfg.PoolConfig.MinIdleConnections),
		clog.Int("max_idle", cfg.PoolConfig.MaxIdleConnections))

	return pool, nil
}

// GetConnection 获取连接
func (p *connectionPool) GetConnection(ctx context.Context) (interface{}, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, NewConnectionError("连接池已关闭", ErrConnectionClosed)
	}

	// 尝试从池中获取连接
	select {
	case conn := <-p.connections:
		if p.isConnectionValid(conn) {
			conn.mu.Lock()
			conn.lastUsedAt = time.Now()
			conn.useCount++
			conn.mu.Unlock()

			atomic.AddInt32(&p.stats.idleConnections, -1)
			atomic.AddInt32(&p.stats.activeConnections, 1)

			p.logger.Debug("从池中获取连接",
				clog.Int64("use_count", conn.useCount),
				clog.Duration("age", time.Since(conn.createdAt)))

			return conn.client, nil
		} else {
			// 连接无效，关闭并创建新连接
			p.closeConnection(conn)
		}
	case <-ctx.Done():
		return nil, NewTimeoutError("获取连接超时", ctx.Err())
	default:
		// 池中没有可用连接，尝试创建新连接
	}

	// 检查是否可以创建新连接
	if atomic.LoadInt32(&p.stats.totalConnections) >= p.stats.maxConnections {
		return nil, NewConnectionError("连接池已满", ErrConnectionPoolExhausted)
	}

	// 创建新连接
	conn, err := p.createConnection()
	if err != nil {
		return nil, err
	}

	atomic.AddInt32(&p.stats.activeConnections, 1)

	p.logger.Debug("创建新连接",
		clog.Int32("total_connections", atomic.LoadInt32(&p.stats.totalConnections)))

	return conn.client, nil
}

// ReleaseConnection 释放连接
func (p *connectionPool) ReleaseConnection(conn interface{}) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return NewConnectionError("连接池已关闭", ErrConnectionClosed)
	}

	client, ok := conn.(*kgo.Client)
	if !ok {
		return NewConnectionError("无效的连接类型", nil)
	}

	// 查找对应的池化连接
	pooledConn := p.findPooledConnection(client)
	if pooledConn == nil {
		p.logger.Warn("未找到对应的池化连接")
		return nil
	}

	atomic.AddInt32(&p.stats.activeConnections, -1)

	// 检查连接是否仍然有效
	if !p.isConnectionValid(pooledConn) {
		p.closeConnection(pooledConn)
		return nil
	}

	// 检查是否超过最大空闲连接数
	if atomic.LoadInt32(&p.stats.idleConnections) >= int32(p.config.PoolConfig.MaxIdleConnections) {
		p.closeConnection(pooledConn)
		return nil
	}

	// 将连接放回池中
	select {
	case p.connections <- pooledConn:
		atomic.AddInt32(&p.stats.idleConnections, 1)
		p.logger.Debug("连接已归还到池中")
	default:
		// 池已满，关闭连接
		p.closeConnection(pooledConn)
	}

	return nil
}

// GetStats 获取连接池统计信息
func (p *connectionPool) GetStats() PoolStats {
	return PoolStats{
		TotalConnections:   int(atomic.LoadInt32(&p.stats.totalConnections)),
		ActiveConnections:  int(atomic.LoadInt32(&p.stats.activeConnections)),
		IdleConnections:    int(atomic.LoadInt32(&p.stats.idleConnections)),
		MaxConnections:     int(p.stats.maxConnections),
		ConnectionsCreated: atomic.LoadInt64(&p.stats.connectionsCreated),
		ConnectionsClosed:  atomic.LoadInt64(&p.stats.connectionsClosed),
		ConnectionErrors:   atomic.LoadInt64(&p.stats.connectionErrors),
	}
}

// HealthCheck 执行连接健康检查
func (p *connectionPool) HealthCheck(ctx context.Context) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return NewConnectionError("连接池已关闭", ErrConnectionClosed)
	}

	// 获取一个连接进行健康检查
	conn, err := p.GetConnection(ctx)
	if err != nil {
		return err
	}
	defer p.ReleaseConnection(conn)

	// 执行ping操作
	client := conn.(*kgo.Client)
	if err := client.Ping(ctx); err != nil {
		atomic.AddInt64(&p.stats.connectionErrors, 1)
		return NewConnectionError("健康检查失败", err)
	}

	return nil
}

// Close 关闭连接池
func (p *connectionPool) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil // 已经关闭
	}

	p.logger.Info("开始关闭连接池")

	// 停止健康检查
	close(p.healthCheckDone)
	if p.healthCheckTicker != nil {
		p.healthCheckTicker.Stop()
	}

	// 关闭所有连接
	close(p.connections)
	for conn := range p.connections {
		p.closeConnection(conn)
	}

	p.logger.Info("连接池已关闭",
		clog.Int64("total_created", atomic.LoadInt64(&p.stats.connectionsCreated)),
		clog.Int64("total_closed", atomic.LoadInt64(&p.stats.connectionsClosed)))

	return nil
}

// createConnection 创建新连接
func (p *connectionPool) createConnection() (*pooledConnection, error) {
	// 构建Kafka客户端选项
	opts := []kgo.Opt{
		kgo.SeedBrokers(p.config.Brokers...),
		kgo.ClientID(p.config.ClientID),
		kgo.ConnIdleTimeout(p.config.Connection.KeepAlive),
		kgo.RequestTimeoutOverhead(p.config.Connection.ReadTimeout),
	}

	// 添加安全配置
	if p.config.SecurityProtocol != "PLAINTEXT" {
		// TODO: 添加SSL/SASL配置
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		atomic.AddInt64(&p.stats.connectionErrors, 1)
		return nil, NewConnectionError("创建Kafka客户端失败", err)
	}

	conn := &pooledConnection{
		client:     client,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
		healthy:    true,
	}

	atomic.AddInt32(&p.stats.totalConnections, 1)
	atomic.AddInt64(&p.stats.connectionsCreated, 1)

	return conn, nil
}

// isConnectionValid 检查连接是否有效
func (p *connectionPool) isConnectionValid(conn *pooledConnection) bool {
	conn.mu.RLock()
	defer conn.mu.RUnlock()

	if !conn.healthy {
		return false
	}

	// 检查连接年龄
	if time.Since(conn.createdAt) > p.config.PoolConfig.ConnectionMaxLifetime {
		return false
	}

	// 检查空闲时间
	if time.Since(conn.lastUsedAt) > p.config.PoolConfig.ConnectionMaxIdleTime {
		return false
	}

	return true
}

// closeConnection 关闭连接
func (p *connectionPool) closeConnection(conn *pooledConnection) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.client != nil {
		conn.client.Close()
		conn.client = nil
	}

	conn.healthy = false
	atomic.AddInt32(&p.stats.totalConnections, -1)
	atomic.AddInt64(&p.stats.connectionsClosed, 1)
}

// findPooledConnection 查找池化连接
func (p *connectionPool) findPooledConnection(client *kgo.Client) *pooledConnection {
	// 这是一个简化实现，实际应该维护一个映射表
	// 由于这里只是演示，我们创建一个临时的池化连接
	return &pooledConnection{
		client:     client,
		lastUsedAt: time.Now(),
		healthy:    true,
	}
}

// preCreateConnections 预创建连接
func (p *connectionPool) preCreateConnections() error {
	for i := 0; i < p.config.PoolConfig.MinIdleConnections; i++ {
		conn, err := p.createConnection()
		if err != nil {
			return err
		}

		select {
		case p.connections <- conn:
			atomic.AddInt32(&p.stats.idleConnections, 1)
		default:
			p.closeConnection(conn)
			break
		}
	}

	return nil
}

// startHealthCheck 启动健康检查
func (p *connectionPool) startHealthCheck() {
	if p.config.PoolConfig.HealthCheckInterval <= 0 {
		return
	}

	p.healthCheckTicker = time.NewTicker(p.config.PoolConfig.HealthCheckInterval)

	go func() {
		for {
			select {
			case <-p.healthCheckTicker.C:
				p.performHealthCheck()
			case <-p.healthCheckDone:
				return
			}
		}
	}()
}

// performHealthCheck 执行健康检查
func (p *connectionPool) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.HealthCheck(ctx); err != nil {
		p.logger.Warn("健康检查失败", clog.Err(err))
	}
}

// validatePoolConfig 验证连接池配置
func validatePoolConfig(cfg PoolConfig) error {
	if cfg.MaxConnections <= 0 {
		return NewConfigError("最大连接数必须大于0", nil)
	}

	if cfg.MinIdleConnections < 0 {
		return NewConfigError("最小空闲连接数不能小于0", nil)
	}

	if cfg.MaxIdleConnections < cfg.MinIdleConnections {
		return NewConfigError("最大空闲连接数不能小于最小空闲连接数", nil)
	}

	if cfg.MaxIdleConnections > cfg.MaxConnections {
		return NewConfigError("最大空闲连接数不能大于最大连接数", nil)
	}

	return nil
}
