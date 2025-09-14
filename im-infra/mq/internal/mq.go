package internal

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// mq MQ实例的实现
type mq struct {
	// 配置
	config Config

	// 组件实例
	producer       Producer
	consumer       Consumer
	connectionPool ConnectionPool

	// 状态管理
	closed int32
	mu     sync.RWMutex

	// 日志器
	logger clog.Logger
}

// NewMQ 创建新的MQ实例
func NewMQ(cfg Config) (MQ, error) {
	// 将用户配置与默认配置合并
	mergedCfg := MergeWithDefaults(cfg)

	if err := validateConfig(mergedCfg); err != nil {
		return nil, NewConfigError("MQ配置无效", err)
	}

	logger := clog.Namespace("mq")

	// 创建连接池
	connectionPool, err := NewConnectionPool(mergedCfg)
	if err != nil {
		return nil, err
	}

	// 创建生产者
	producer, err := NewProducer(mergedCfg.ProducerConfig)
	if err != nil {
		connectionPool.Close()
		return nil, err
	}

	// 创建消费者
	consumer, err := NewConsumer(mergedCfg.ConsumerConfig)
	if err != nil {
		producer.Close()
		connectionPool.Close()
		return nil, err
	}

	mqInstance := &mq{
		config:         mergedCfg,
		producer:       producer,
		consumer:       consumer,
		connectionPool: connectionPool,
		logger:         logger,
	}

	logger.Info("MQ实例创建成功",
		clog.Strings("brokers", cfg.Brokers),
		clog.String("client_id", cfg.ClientID),
		clog.String("security_protocol", cfg.SecurityProtocol))

	return mqInstance, nil
}

// Producer 获取生产者实例
func (m *mq) Producer() Producer {
	if atomic.LoadInt32(&m.closed) == 1 {
		m.logger.Warn("MQ实例已关闭，无法获取生产者")
		return nil
	}

	return m.producer
}

// Consumer 获取消费者实例
func (m *mq) Consumer() Consumer {
	if atomic.LoadInt32(&m.closed) == 1 {
		m.logger.Warn("MQ实例已关闭，无法获取消费者")
		return nil
	}

	return m.consumer
}

// ConnectionPool 获取连接池实例
func (m *mq) ConnectionPool() ConnectionPool {
	if atomic.LoadInt32(&m.closed) == 1 {
		m.logger.Warn("MQ实例已关闭，无法获取连接池")
		return nil
	}

	return m.connectionPool
}

// Close 关闭MQ实例，释放所有资源
func (m *mq) Close() error {
	if !atomic.CompareAndSwapInt32(&m.closed, 0, 1) {
		return nil // 已经关闭
	}

	m.logger.Info("开始关闭MQ实例")

	var errors []error

	// 关闭消费者
	if m.consumer != nil {
		if err := m.consumer.Close(); err != nil {
			errors = append(errors, err)
			m.logger.Error("关闭消费者失败", clog.Err(err))
		}
	}

	// 关闭生产者
	if m.producer != nil {
		if err := m.producer.Close(); err != nil {
			errors = append(errors, err)
			m.logger.Error("关闭生产者失败", clog.Err(err))
		}
	}

	// 关闭连接池
	if m.connectionPool != nil {
		if err := m.connectionPool.Close(); err != nil {
			errors = append(errors, err)
			m.logger.Error("关闭连接池失败", clog.Err(err))
		}
	}

	if len(errors) > 0 {
		m.logger.Error("MQ实例关闭时发生错误", clog.Int("error_count", len(errors)))
		return errors[0] // 返回第一个错误
	}

	m.logger.Info("MQ实例已成功关闭")
	return nil
}

// Ping 检查连接健康状态
func (m *mq) Ping(ctx context.Context) error {
	if atomic.LoadInt32(&m.closed) == 1 {
		return NewConnectionError("MQ实例已关闭", ErrConnectionClosed)
	}

	// 检查连接池健康状态
	if err := m.connectionPool.HealthCheck(ctx); err != nil {
		return err
	}

	m.logger.Debug("MQ健康检查通过")
	return nil
}

// validateConfig 验证MQ配置
func validateConfig(cfg Config) error {
	if len(cfg.Brokers) == 0 {
		return NewConfigError("Broker地址列表不能为空", nil)
	}

	if cfg.ClientID == "" {
		return NewConfigError("客户端ID不能为空", nil)
	}

	// 验证生产者配置
	if err := validateProducerConfig(cfg.ProducerConfig); err != nil {
		return err
	}

	// 验证消费者配置
	if err := validateConsumerConfig(cfg.ConsumerConfig); err != nil {
		return err
	}

	// 验证连接池配置
	if err := validatePoolConfig(cfg.PoolConfig); err != nil {
		return err
	}

	// 验证性能配置
	if err := validatePerformanceConfig(cfg.Performance); err != nil {
		return err
	}

	return nil
}

// validatePerformanceConfig 验证性能配置
func validatePerformanceConfig(cfg PerformanceConfig) error {
	if cfg.TargetLatencyMicros <= 0 {
		return NewConfigError("目标延迟必须大于0", nil)
	}

	if cfg.TargetThroughputPerSec <= 0 {
		return NewConfigError("目标吞吐量必须大于0", nil)
	}

	if cfg.SmallMessageThresholdBytes <= 0 {
		return NewConfigError("小消息阈值必须大于0", nil)
	}

	return nil
}

// MQManager MQ管理器，用于管理多个MQ实例
type MQManager struct {
	instances map[string]MQ
	mu        sync.RWMutex
	logger    clog.Logger
}

// NewMQManager 创建MQ管理器
func NewMQManager() *MQManager {
	return &MQManager{
		instances: make(map[string]MQ),
		logger:    clog.Namespace("mq.manager"),
	}
}

// CreateInstance 创建MQ实例
func (mm *MQManager) CreateInstance(name string, cfg Config) (MQ, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.instances[name]; exists {
		return nil, NewConfigError("MQ实例已存在", nil)
	}

	instance, err := NewMQ(cfg)
	if err != nil {
		return nil, err
	}

	mm.instances[name] = instance
	mm.logger.Info("创建MQ实例成功", clog.String("name", name))

	return instance, nil
}

// GetInstance 获取MQ实例
func (mm *MQManager) GetInstance(name string) (MQ, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	instance, exists := mm.instances[name]
	return instance, exists
}

// RemoveInstance 移除MQ实例
func (mm *MQManager) RemoveInstance(name string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	instance, exists := mm.instances[name]
	if !exists {
		return NewConfigError("MQ实例不存在", nil)
	}

	if err := instance.Close(); err != nil {
		mm.logger.Error("关闭MQ实例失败", clog.String("name", name), clog.Err(err))
		return err
	}

	delete(mm.instances, name)
	mm.logger.Info("移除MQ实例成功", clog.String("name", name))

	return nil
}

// ListInstances 列出所有MQ实例名称
func (mm *MQManager) ListInstances() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	names := make([]string, 0, len(mm.instances))
	for name := range mm.instances {
		names = append(names, name)
	}

	return names
}

// CloseAll 关闭所有MQ实例
func (mm *MQManager) CloseAll() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	var errors []error

	for name, instance := range mm.instances {
		if err := instance.Close(); err != nil {
			errors = append(errors, err)
			mm.logger.Error("关闭MQ实例失败", clog.String("name", name), clog.Err(err))
		}
	}

	mm.instances = make(map[string]MQ)

	if len(errors) > 0 {
		return errors[0]
	}

	mm.logger.Info("所有MQ实例已关闭")
	return nil
}

// HealthCheck 检查所有MQ实例的健康状态
func (mm *MQManager) HealthCheck(ctx context.Context) map[string]error {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	results := make(map[string]error)

	for name, instance := range mm.instances {
		if err := instance.Ping(ctx); err != nil {
			results[name] = err
		} else {
			results[name] = nil
		}
	}

	return results
}

// GetStats 获取所有MQ实例的统计信息
func (mm *MQManager) GetStats() map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	stats := make(map[string]interface{})

	for name, instance := range mm.instances {
		instanceStats := map[string]interface{}{
			"producer_metrics": instance.Producer().GetMetrics(),
			"consumer_metrics": instance.Consumer().GetMetrics(),
			"pool_stats":       instance.ConnectionPool().GetStats(),
		}
		stats[name] = instanceStats
	}

	return stats
}
