package internal

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// RecordProducerMetrics 记录生产者指标
	RecordProducerMetrics(metrics ProducerMetrics)
	
	// RecordConsumerMetrics 记录消费者指标
	RecordConsumerMetrics(metrics ConsumerMetrics)
	
	// RecordConnectionPoolStats 记录连接池统计
	RecordConnectionPoolStats(stats PoolStats)
	
	// RecordLatency 记录延迟
	RecordLatency(operation string, latency time.Duration)
	
	// RecordThroughput 记录吞吐量
	RecordThroughput(operation string, count int64, bytes int64)
	
	// RecordError 记录错误
	RecordError(operation string, errorType string, err error)
	
	// GetMetrics 获取所有指标
	GetMetrics() map[string]interface{}
	
	// Reset 重置指标
	Reset()
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	// CheckHealth 执行健康检查
	CheckHealth(ctx context.Context) HealthStatus
	
	// RegisterCheck 注册健康检查项
	RegisterCheck(name string, check HealthCheckFunc)
	
	// UnregisterCheck 取消注册健康检查项
	UnregisterCheck(name string)
}

// HealthCheckFunc 健康检查函数类型
type HealthCheckFunc func(ctx context.Context) error

// HealthStatus 健康状态
type HealthStatus struct {
	// Overall 整体健康状态
	Overall bool
	
	// Checks 各项检查结果
	Checks map[string]HealthCheckResult
	
	// Timestamp 检查时间戳
	Timestamp time.Time
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	// Healthy 是否健康
	Healthy bool
	
	// Error 错误信息
	Error string
	
	// Duration 检查耗时
	Duration time.Duration
}

// defaultMetricsCollector 默认指标收集器实现
type defaultMetricsCollector struct {
	// 生产者指标
	producerMetrics ProducerMetrics
	producerMu      sync.RWMutex
	
	// 消费者指标
	consumerMetrics ConsumerMetrics
	consumerMu      sync.RWMutex
	
	// 连接池统计
	poolStats PoolStats
	poolMu    sync.RWMutex
	
	// 延迟统计
	latencyStats map[string]*latencyMetrics
	latencyMu    sync.RWMutex
	
	// 吞吐量统计
	throughputStats map[string]*throughputMetrics
	throughputMu    sync.RWMutex
	
	// 错误统计
	errorStats map[string]*errorMetrics
	errorMu    sync.RWMutex
	
	// 日志器
	logger clog.Logger
}

// latencyMetrics 延迟指标
type latencyMetrics struct {
	count       int64
	totalNanos  int64
	minNanos    int64
	maxNanos    int64
	lastUpdated time.Time
}

// throughputMetrics 吞吐量指标
type throughputMetrics struct {
	count       int64
	bytes       int64
	lastUpdated time.Time
	startTime   time.Time
}

// errorMetrics 错误指标
type errorMetrics struct {
	count       int64
	lastError   string
	lastUpdated time.Time
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() MetricsCollector {
	return &defaultMetricsCollector{
		latencyStats:    make(map[string]*latencyMetrics),
		throughputStats: make(map[string]*throughputMetrics),
		errorStats:      make(map[string]*errorMetrics),
		logger:          clog.Module("mq.metrics"),
	}
}

// RecordProducerMetrics 记录生产者指标
func (mc *defaultMetricsCollector) RecordProducerMetrics(metrics ProducerMetrics) {
	mc.producerMu.Lock()
	defer mc.producerMu.Unlock()
	
	mc.producerMetrics = metrics
	mc.logger.Debug("记录生产者指标",
		clog.Int64("total_messages", metrics.TotalMessages),
		clog.Int64("success_messages", metrics.SuccessMessages),
		clog.Int64("failed_messages", metrics.FailedMessages),
		clog.Float64("messages_per_second", metrics.MessagesPerSecond))
}

// RecordConsumerMetrics 记录消费者指标
func (mc *defaultMetricsCollector) RecordConsumerMetrics(metrics ConsumerMetrics) {
	mc.consumerMu.Lock()
	defer mc.consumerMu.Unlock()
	
	mc.consumerMetrics = metrics
	mc.logger.Debug("记录消费者指标",
		clog.Int64("total_messages", metrics.TotalMessages),
		clog.Float64("messages_per_second", metrics.MessagesPerSecond),
		clog.Int64("lag", metrics.Lag))
}

// RecordConnectionPoolStats 记录连接池统计
func (mc *defaultMetricsCollector) RecordConnectionPoolStats(stats PoolStats) {
	mc.poolMu.Lock()
	defer mc.poolMu.Unlock()
	
	mc.poolStats = stats
	mc.logger.Debug("记录连接池统计",
		clog.Int("total_connections", stats.TotalConnections),
		clog.Int("active_connections", stats.ActiveConnections),
		clog.Int("idle_connections", stats.IdleConnections))
}

// RecordLatency 记录延迟
func (mc *defaultMetricsCollector) RecordLatency(operation string, latency time.Duration) {
	mc.latencyMu.Lock()
	defer mc.latencyMu.Unlock()
	
	if mc.latencyStats[operation] == nil {
		mc.latencyStats[operation] = &latencyMetrics{
			minNanos: latency.Nanoseconds(),
			maxNanos: latency.Nanoseconds(),
		}
	}
	
	stats := mc.latencyStats[operation]
	nanos := latency.Nanoseconds()
	
	atomic.AddInt64(&stats.count, 1)
	atomic.AddInt64(&stats.totalNanos, nanos)
	
	// 更新最小值
	for {
		current := atomic.LoadInt64(&stats.minNanos)
		if nanos >= current || atomic.CompareAndSwapInt64(&stats.minNanos, current, nanos) {
			break
		}
	}
	
	// 更新最大值
	for {
		current := atomic.LoadInt64(&stats.maxNanos)
		if nanos <= current || atomic.CompareAndSwapInt64(&stats.maxNanos, current, nanos) {
			break
		}
	}
	
	stats.lastUpdated = time.Now()
}

// RecordThroughput 记录吞吐量
func (mc *defaultMetricsCollector) RecordThroughput(operation string, count int64, bytes int64) {
	mc.throughputMu.Lock()
	defer mc.throughputMu.Unlock()
	
	if mc.throughputStats[operation] == nil {
		mc.throughputStats[operation] = &throughputMetrics{
			startTime: time.Now(),
		}
	}
	
	stats := mc.throughputStats[operation]
	atomic.AddInt64(&stats.count, count)
	atomic.AddInt64(&stats.bytes, bytes)
	stats.lastUpdated = time.Now()
}

// RecordError 记录错误
func (mc *defaultMetricsCollector) RecordError(operation string, errorType string, err error) {
	key := operation + ":" + errorType
	
	mc.errorMu.Lock()
	defer mc.errorMu.Unlock()
	
	if mc.errorStats[key] == nil {
		mc.errorStats[key] = &errorMetrics{}
	}
	
	stats := mc.errorStats[key]
	atomic.AddInt64(&stats.count, 1)
	stats.lastError = err.Error()
	stats.lastUpdated = time.Now()
	
	mc.logger.Warn("记录错误",
		clog.String("operation", operation),
		clog.String("error_type", errorType),
		clog.ErrorValue(err))
}

// GetMetrics 获取所有指标
func (mc *defaultMetricsCollector) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	
	// 生产者指标
	mc.producerMu.RLock()
	metrics["producer"] = mc.producerMetrics
	mc.producerMu.RUnlock()
	
	// 消费者指标
	mc.consumerMu.RLock()
	metrics["consumer"] = mc.consumerMetrics
	mc.consumerMu.RUnlock()
	
	// 连接池统计
	mc.poolMu.RLock()
	metrics["connection_pool"] = mc.poolStats
	mc.poolMu.RUnlock()
	
	// 延迟统计
	mc.latencyMu.RLock()
	latencyMetrics := make(map[string]interface{})
	for operation, stats := range mc.latencyStats {
		count := atomic.LoadInt64(&stats.count)
		totalNanos := atomic.LoadInt64(&stats.totalNanos)
		minNanos := atomic.LoadInt64(&stats.minNanos)
		maxNanos := atomic.LoadInt64(&stats.maxNanos)
		
		var avgNanos int64
		if count > 0 {
			avgNanos = totalNanos / count
		}
		
		latencyMetrics[operation] = map[string]interface{}{
			"count":        count,
			"avg_latency":  time.Duration(avgNanos),
			"min_latency":  time.Duration(minNanos),
			"max_latency":  time.Duration(maxNanos),
			"last_updated": stats.lastUpdated,
		}
	}
	metrics["latency"] = latencyMetrics
	mc.latencyMu.RUnlock()
	
	// 吞吐量统计
	mc.throughputMu.RLock()
	throughputMetrics := make(map[string]interface{})
	for operation, stats := range mc.throughputStats {
		count := atomic.LoadInt64(&stats.count)
		bytes := atomic.LoadInt64(&stats.bytes)
		
		elapsed := time.Since(stats.startTime)
		var countPerSecond, bytesPerSecond float64
		if elapsed.Seconds() > 0 {
			countPerSecond = float64(count) / elapsed.Seconds()
			bytesPerSecond = float64(bytes) / elapsed.Seconds()
		}
		
		throughputMetrics[operation] = map[string]interface{}{
			"count":            count,
			"bytes":            bytes,
			"count_per_second": countPerSecond,
			"bytes_per_second": bytesPerSecond,
			"last_updated":     stats.lastUpdated,
		}
	}
	metrics["throughput"] = throughputMetrics
	mc.throughputMu.RUnlock()
	
	// 错误统计
	mc.errorMu.RLock()
	errorMetrics := make(map[string]interface{})
	for key, stats := range mc.errorStats {
		count := atomic.LoadInt64(&stats.count)
		errorMetrics[key] = map[string]interface{}{
			"count":        count,
			"last_error":   stats.lastError,
			"last_updated": stats.lastUpdated,
		}
	}
	metrics["errors"] = errorMetrics
	mc.errorMu.RUnlock()
	
	return metrics
}

// Reset 重置指标
func (mc *defaultMetricsCollector) Reset() {
	mc.producerMu.Lock()
	mc.producerMetrics = ProducerMetrics{}
	mc.producerMu.Unlock()
	
	mc.consumerMu.Lock()
	mc.consumerMetrics = ConsumerMetrics{}
	mc.consumerMu.Unlock()
	
	mc.poolMu.Lock()
	mc.poolStats = PoolStats{}
	mc.poolMu.Unlock()
	
	mc.latencyMu.Lock()
	mc.latencyStats = make(map[string]*latencyMetrics)
	mc.latencyMu.Unlock()
	
	mc.throughputMu.Lock()
	mc.throughputStats = make(map[string]*throughputMetrics)
	mc.throughputMu.Unlock()
	
	mc.errorMu.Lock()
	mc.errorStats = make(map[string]*errorMetrics)
	mc.errorMu.Unlock()
	
	mc.logger.Info("指标已重置")
}

// defaultHealthChecker 默认健康检查器实现
type defaultHealthChecker struct {
	checks map[string]HealthCheckFunc
	mu     sync.RWMutex
	logger clog.Logger
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker() HealthChecker {
	return &defaultHealthChecker{
		checks: make(map[string]HealthCheckFunc),
		logger: clog.Module("mq.health"),
	}
}

// CheckHealth 执行健康检查
func (hc *defaultHealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	status := HealthStatus{
		Overall:   true,
		Checks:    make(map[string]HealthCheckResult),
		Timestamp: time.Now(),
	}
	
	for name, checkFunc := range hc.checks {
		start := time.Now()
		err := checkFunc(ctx)
		duration := time.Since(start)
		
		result := HealthCheckResult{
			Healthy:  err == nil,
			Duration: duration,
		}
		
		if err != nil {
			result.Error = err.Error()
			status.Overall = false
		}
		
		status.Checks[name] = result
	}
	
	hc.logger.Debug("健康检查完成",
		clog.Bool("overall_healthy", status.Overall),
		clog.Int("check_count", len(status.Checks)))
	
	return status
}

// RegisterCheck 注册健康检查项
func (hc *defaultHealthChecker) RegisterCheck(name string, check HealthCheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.checks[name] = check
	hc.logger.Info("注册健康检查项", clog.String("name", name))
}

// UnregisterCheck 取消注册健康检查项
func (hc *defaultHealthChecker) UnregisterCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	delete(hc.checks, name)
	hc.logger.Info("取消注册健康检查项", clog.String("name", name))
}

// MonitoringManager 监控管理器
type MonitoringManager struct {
	metricsCollector MetricsCollector
	healthChecker    HealthChecker
	
	// 定期收集器
	ticker *time.Ticker
	done   chan struct{}
	
	// 配置
	config MonitoringConfig
	
	// 日志器
	logger clog.Logger
}

// NewMonitoringManager 创建监控管理器
func NewMonitoringManager(cfg MonitoringConfig) *MonitoringManager {
	mm := &MonitoringManager{
		metricsCollector: NewMetricsCollector(),
		healthChecker:    NewHealthChecker(),
		config:           cfg,
		logger:           clog.Module("mq.monitoring"),
		done:             make(chan struct{}),
	}
	
	// 启动定期收集
	if cfg.EnableMetrics && cfg.MetricsInterval > 0 {
		mm.startPeriodicCollection()
	}
	
	return mm
}

// GetMetricsCollector 获取指标收集器
func (mm *MonitoringManager) GetMetricsCollector() MetricsCollector {
	return mm.metricsCollector
}

// GetHealthChecker 获取健康检查器
func (mm *MonitoringManager) GetHealthChecker() HealthChecker {
	return mm.healthChecker
}

// startPeriodicCollection 启动定期收集
func (mm *MonitoringManager) startPeriodicCollection() {
	mm.ticker = time.NewTicker(mm.config.MetricsInterval)
	
	go func() {
		for {
			select {
			case <-mm.ticker.C:
				mm.collectMetrics()
			case <-mm.done:
				return
			}
		}
	}()
}

// collectMetrics 收集指标
func (mm *MonitoringManager) collectMetrics() {
	metrics := mm.metricsCollector.GetMetrics()
	
	mm.logger.Debug("定期收集指标",
		clog.Int("metric_categories", len(metrics)))
	
	// 这里可以添加将指标发送到外部监控系统的逻辑
	// 例如：Prometheus、InfluxDB、CloudWatch等
}

// Stop 停止监控管理器
func (mm *MonitoringManager) Stop() {
	if mm.ticker != nil {
		mm.ticker.Stop()
	}
	
	close(mm.done)
	mm.logger.Info("监控管理器已停止")
}
