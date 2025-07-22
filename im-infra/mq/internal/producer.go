package internal

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/twmb/franz-go/pkg/kgo"
)

// producer 生产者实现
type producer struct {
	// Kafka客户端
	client *kgo.Client

	// 配置
	config ProducerConfig

	// 性能指标
	metrics producerMetrics

	// 状态管理
	closed int32
	mu     sync.RWMutex

	// 日志器
	logger clog.Logger

	// 压缩器
	compressor CompressionCodec

	// 序列化器
	serializer MessageSerializer

	// 批处理管理
	batchManager *batchManager
}

// producerMetrics 生产者性能指标的内部实现
type producerMetrics struct {
	totalMessages   int64
	totalBytes      int64
	successMessages int64
	failedMessages  int64

	// 延迟统计
	totalLatency time.Duration
	maxLatency   time.Duration
	minLatency   time.Duration
	latencyCount int64

	// 吞吐量统计
	lastResetTime time.Time
	mu            sync.RWMutex
}

// batchManager 批处理管理器
type batchManager struct {
	// 当前批次
	currentBatch *MessageBatch

	// 批次锁
	mu sync.Mutex

	// 批次定时器
	timer *time.Timer

	// 发送通道
	sendChan chan *MessageBatch

	// 停止信号
	stopChan chan struct{}

	// 配置
	config ProducerConfig

	// 日志器
	logger clog.Logger
}

// NewProducer 创建新的生产者
func NewProducer(cfg ProducerConfig) (Producer, error) {
	if err := validateProducerConfig(cfg); err != nil {
		return nil, NewConfigError("生产者配置无效", err)
	}

	// 构建Kafka客户端选项
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID(cfg.ClientID),
		kgo.RequiredAcks(convRequiredAcks(cfg.RequiredAcks)),
		kgo.ProducerBatchMaxBytes(int32(cfg.BatchSize)),
		kgo.ProducerLinger(time.Duration(cfg.LingerMs) * time.Millisecond),
		kgo.RequestTimeoutOverhead(cfg.RequestTimeout),
		kgo.RequestRetries(cfg.MaxRetries),
		kgo.RetryBackoffFn(func(tries int) time.Duration {
			return cfg.RetryBackoff
		}),
	}

	// 配置幂等性
	if !cfg.EnableIdempotence {
		opts = append(opts, kgo.DisableIdempotentWrite())
	}

	// 配置压缩
	switch cfg.Compression {
	case "gzip":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.GzipCompression()))
	case "snappy":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.SnappyCompression()))
	case "lz4":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.Lz4Compression()))
	case "zstd":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.ZstdCompression()))
	default:
		opts = append(opts, kgo.ProducerBatchCompression(kgo.NoCompression()))
	}

	// 配置最大飞行中请求数
	opts = append(opts, kgo.MaxProduceRequestsInflightPerBroker(cfg.MaxInFlightRequests))

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, NewConnectionError("创建Kafka生产者客户端失败", err)
	}

	p := &producer{
		client: client,
		config: cfg,
		logger: clog.Module("mq.producer"),
		metrics: producerMetrics{
			lastResetTime: time.Now(),
			minLatency:    time.Hour, // 初始化为一个大值
		},
	}

	// 初始化压缩器
	p.compressor = newCompressionCodec(cfg.Compression)

	// 初始化序列化器
	p.serializer = newJSONSerializer()

	// 初始化批处理管理器
	p.batchManager = newBatchManager(cfg, p.logger)

	// 启动批处理管理器
	go p.batchManager.start(p.sendBatchInternal)

	p.logger.Info("生产者创建成功",
		clog.String("client_id", cfg.ClientID),
		clog.String("compression", cfg.Compression),
		clog.Bool("idempotence", cfg.EnableIdempotence),
		clog.Int("batch_size", cfg.BatchSize),
		clog.Int("linger_ms", cfg.LingerMs))

	return p, nil
}

// SendSync 同步发送单条消息
func (p *producer) SendSync(ctx context.Context, topic string, message []byte) error {
	return p.SendSyncWithHeaders(ctx, topic, nil, message, nil)
}

// SendSyncWithKey 同步发送带键的消息
func (p *producer) SendSyncWithKey(ctx context.Context, topic string, key []byte, message []byte) error {
	return p.SendSyncWithHeaders(ctx, topic, key, message, nil)
}

// SendSyncWithHeaders 同步发送带头部信息的消息
func (p *producer) SendSyncWithHeaders(ctx context.Context, topic string, key []byte, message []byte, headers map[string][]byte) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return NewProducerError("生产者已关闭", ErrProducerClosed)
	}

	startTime := time.Now()

	// 验证消息大小
	if len(message) > p.config.MaxMessageBytes {
		atomic.AddInt64(&p.metrics.failedMessages, 1)
		return NewProducerError("消息过大", ErrMessageTooLarge)
	}

	// 构建Kafka记录
	record := &kgo.Record{
		Topic: topic,
		Key:   key,
		Value: message,
	}

	// 添加头部信息
	if headers != nil {
		for k, v := range headers {
			record.Headers = append(record.Headers, kgo.RecordHeader{
				Key:   k,
				Value: v,
			})
		}
	}

	// 同步发送
	results := p.client.ProduceSync(ctx, record)
	if err := results.FirstErr(); err != nil {
		atomic.AddInt64(&p.metrics.failedMessages, 1)
		p.logger.Error("同步发送消息失败",
			clog.String("topic", topic),
			clog.Int("message_size", len(message)),
			clog.ErrorValue(err))
		return NewProducerError("发送消息失败", err)
	}

	// 更新指标
	latency := time.Since(startTime)
	p.updateMetrics(int64(len(message)), latency, true)

	p.logger.Debug("同步发送消息成功",
		clog.String("topic", topic),
		clog.Int("message_size", len(message)),
		clog.Duration("latency", latency))

	return nil
}

// SendAsync 异步发送单条消息
func (p *producer) SendAsync(ctx context.Context, topic string, message []byte, callback func(error)) {
	p.SendAsyncWithHeaders(ctx, topic, nil, message, nil, callback)
}

// SendAsyncWithKey 异步发送带键的消息
func (p *producer) SendAsyncWithKey(ctx context.Context, topic string, key []byte, message []byte, callback func(error)) {
	p.SendAsyncWithHeaders(ctx, topic, key, message, nil, callback)
}

// SendAsyncWithHeaders 异步发送带头部信息的消息
func (p *producer) SendAsyncWithHeaders(ctx context.Context, topic string, key []byte, message []byte, headers map[string][]byte, callback func(error)) {
	if atomic.LoadInt32(&p.closed) == 1 {
		if callback != nil {
			callback(NewProducerError("生产者已关闭", ErrProducerClosed))
		}
		return
	}

	startTime := time.Now()

	// 验证消息大小
	if len(message) > p.config.MaxMessageBytes {
		atomic.AddInt64(&p.metrics.failedMessages, 1)
		if callback != nil {
			callback(NewProducerError("消息过大", ErrMessageTooLarge))
		}
		return
	}

	// 构建Kafka记录
	record := &kgo.Record{
		Topic: topic,
		Key:   key,
		Value: message,
	}

	// 添加头部信息
	if headers != nil {
		for k, v := range headers {
			record.Headers = append(record.Headers, kgo.RecordHeader{
				Key:   k,
				Value: v,
			})
		}
	}

	// 异步发送
	p.client.Produce(ctx, record, func(r *kgo.Record, err error) {
		latency := time.Since(startTime)

		if err != nil {
			atomic.AddInt64(&p.metrics.failedMessages, 1)
			p.logger.Error("异步发送消息失败",
				clog.String("topic", topic),
				clog.Int("message_size", len(message)),
				clog.Duration("latency", latency),
				clog.ErrorValue(err))

			if callback != nil {
				callback(NewProducerError("发送消息失败", err))
			}
		} else {
			// 更新指标
			p.updateMetrics(int64(len(message)), latency, true)

			p.logger.Debug("异步发送消息成功",
				clog.String("topic", topic),
				clog.Int("message_size", len(message)),
				clog.Duration("latency", latency),
				clog.Int32("partition", r.Partition),
				clog.Int64("offset", r.Offset))

			if callback != nil {
				callback(nil)
			}
		}
	})
}

// SendBatchSync 同步发送消息批次
func (p *producer) SendBatchSync(ctx context.Context, batch MessageBatch) ([]ProduceResult, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, NewProducerError("生产者已关闭", ErrProducerClosed)
	}

	if len(batch.Messages) == 0 {
		return []ProduceResult{}, nil
	}

	startTime := time.Now()
	results := make([]ProduceResult, len(batch.Messages))

	// 构建Kafka记录
	records := make([]*kgo.Record, len(batch.Messages))
	for i, msg := range batch.Messages {
		if len(msg.Value) > p.config.MaxMessageBytes {
			results[i] = ProduceResult{
				Topic:     msg.Topic,
				Partition: -1,
				Offset:    -1,
				Error:     NewProducerError("消息过大", ErrMessageTooLarge),
				Latency:   0,
			}
			continue
		}

		record := &kgo.Record{
			Topic: msg.Topic,
			Key:   msg.Key,
			Value: msg.Value,
		}

		// 添加头部信息
		if msg.Headers != nil {
			for k, v := range msg.Headers {
				record.Headers = append(record.Headers, kgo.RecordHeader{
					Key:   k,
					Value: v,
				})
			}
		}

		records[i] = record
	}

	// 同步发送批次
	produceResults := p.client.ProduceSync(ctx, records...)

	// 处理结果
	totalBytes := int64(0)
	successCount := int64(0)
	failedCount := int64(0)

	for i, result := range produceResults {
		latency := time.Since(startTime)
		messageSize := len(batch.Messages[i].Value)
		totalBytes += int64(messageSize)

		if result.Err != nil {
			failedCount++
			results[i] = ProduceResult{
				Topic:     batch.Messages[i].Topic,
				Partition: -1,
				Offset:    -1,
				Error:     NewProducerError("发送消息失败", result.Err),
				Latency:   latency,
			}
		} else {
			successCount++
			results[i] = ProduceResult{
				Topic:     result.Record.Topic,
				Partition: result.Record.Partition,
				Offset:    result.Record.Offset,
				Error:     nil,
				Latency:   latency,
			}
		}
	}

	// 更新指标
	atomic.AddInt64(&p.metrics.totalBytes, totalBytes)
	atomic.AddInt64(&p.metrics.successMessages, successCount)
	atomic.AddInt64(&p.metrics.failedMessages, failedCount)
	atomic.AddInt64(&p.metrics.totalMessages, int64(len(batch.Messages)))

	p.logger.Info("批次发送完成",
		clog.Int("total_messages", len(batch.Messages)),
		clog.Int64("success_count", successCount),
		clog.Int64("failed_count", failedCount),
		clog.Int64("total_bytes", totalBytes),
		clog.Duration("total_latency", time.Since(startTime)))

	return results, nil
}

// SendBatchAsync 异步发送消息批次
func (p *producer) SendBatchAsync(ctx context.Context, batch MessageBatch, callback func([]ProduceResult, error)) {
	if atomic.LoadInt32(&p.closed) == 1 {
		if callback != nil {
			callback(nil, NewProducerError("生产者已关闭", ErrProducerClosed))
		}
		return
	}

	go func() {
		results, err := p.SendBatchSync(ctx, batch)
		if callback != nil {
			callback(results, err)
		}
	}()
}

// Flush 刷新所有待发送的消息
func (p *producer) Flush(ctx context.Context) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return NewProducerError("生产者已关闭", ErrProducerClosed)
	}

	// 刷新批处理管理器
	p.batchManager.flush()

	// 刷新Kafka客户端
	if err := p.client.Flush(ctx); err != nil {
		return NewProducerError("刷新失败", err)
	}

	p.logger.Debug("生产者刷新完成")
	return nil
}

// Close 关闭生产者
func (p *producer) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil // 已经关闭
	}

	p.logger.Info("开始关闭生产者")

	// 停止批处理管理器
	p.batchManager.stop()

	// 关闭Kafka客户端
	p.client.Close()

	stats := p.GetMetrics()
	p.logger.Info("生产者已关闭",
		clog.Int64("total_messages", stats.TotalMessages),
		clog.Int64("success_messages", stats.SuccessMessages),
		clog.Int64("failed_messages", stats.FailedMessages),
		clog.Int64("total_bytes", stats.TotalBytes),
		clog.Duration("avg_latency", stats.AverageLatency))

	return nil
}

// GetMetrics 获取生产者性能指标
func (p *producer) GetMetrics() ProducerMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	totalMessages := atomic.LoadInt64(&p.metrics.totalMessages)
	totalBytes := atomic.LoadInt64(&p.metrics.totalBytes)
	successMessages := atomic.LoadInt64(&p.metrics.successMessages)
	failedMessages := atomic.LoadInt64(&p.metrics.failedMessages)

	var avgLatency time.Duration
	if p.metrics.latencyCount > 0 {
		avgLatency = time.Duration(p.metrics.totalLatency.Nanoseconds() / p.metrics.latencyCount)
	}

	// 计算吞吐量
	elapsed := time.Since(p.metrics.lastResetTime)
	var messagesPerSecond, bytesPerSecond float64
	if elapsed.Seconds() > 0 {
		messagesPerSecond = float64(totalMessages) / elapsed.Seconds()
		bytesPerSecond = float64(totalBytes) / elapsed.Seconds()
	}

	return ProducerMetrics{
		TotalMessages:     totalMessages,
		TotalBytes:        totalBytes,
		SuccessMessages:   successMessages,
		FailedMessages:    failedMessages,
		AverageLatency:    avgLatency,
		MaxLatency:        p.metrics.maxLatency,
		MinLatency:        p.metrics.minLatency,
		MessagesPerSecond: messagesPerSecond,
		BytesPerSecond:    bytesPerSecond,
	}
}

// updateMetrics 更新性能指标
func (p *producer) updateMetrics(bytes int64, latency time.Duration, success bool) {
	atomic.AddInt64(&p.metrics.totalBytes, bytes)
	atomic.AddInt64(&p.metrics.totalMessages, 1)

	if success {
		atomic.AddInt64(&p.metrics.successMessages, 1)
	} else {
		atomic.AddInt64(&p.metrics.failedMessages, 1)
	}

	// 更新延迟统计
	p.metrics.mu.Lock()
	p.metrics.totalLatency += latency
	p.metrics.latencyCount++

	if latency > p.metrics.maxLatency {
		p.metrics.maxLatency = latency
	}

	if latency < p.metrics.minLatency {
		p.metrics.minLatency = latency
	}
	p.metrics.mu.Unlock()
}

// sendBatchInternal 内部批次发送方法
func (p *producer) sendBatchInternal(batch *MessageBatch) {
	if len(batch.Messages) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.config.RequestTimeout)
	defer cancel()

	_, err := p.SendBatchSync(ctx, *batch)
	if err != nil {
		p.logger.Error("批次发送失败", clog.ErrorValue(err))
	}
}

// newBatchManager 创建批处理管理器
func newBatchManager(cfg ProducerConfig, logger clog.Logger) *batchManager {
	return &batchManager{
		currentBatch: &MessageBatch{
			Messages:      make([]*Message, 0),
			MaxBatchSize:  cfg.BatchSize,
			MaxBatchCount: 100, // 默认最大批次消息数
			LingerMs:      cfg.LingerMs,
		},
		sendChan: make(chan *MessageBatch, 10),
		stopChan: make(chan struct{}),
		config:   cfg,
		logger:   logger,
	}
}

// start 启动批处理管理器
func (bm *batchManager) start(sendFunc func(*MessageBatch)) {
	go func() {
		for {
			select {
			case batch := <-bm.sendChan:
				sendFunc(batch)
			case <-bm.stopChan:
				return
			}
		}
	}()

	// 启动定时器
	bm.startTimer()
}

// stop 停止批处理管理器
func (bm *batchManager) stop() {
	close(bm.stopChan)
	if bm.timer != nil {
		bm.timer.Stop()
	}

	// 发送剩余的批次
	bm.flush()
}

// addMessage 添加消息到批次
func (bm *batchManager) addMessage(msg *Message) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.currentBatch.Messages = append(bm.currentBatch.Messages, msg)

	// 检查是否需要发送批次
	if bm.shouldSendBatch() {
		bm.sendCurrentBatch()
	}
}

// shouldSendBatch 检查是否应该发送批次
func (bm *batchManager) shouldSendBatch() bool {
	if len(bm.currentBatch.Messages) >= bm.currentBatch.MaxBatchCount {
		return true
	}

	// 计算当前批次大小
	totalSize := 0
	for _, msg := range bm.currentBatch.Messages {
		totalSize += len(msg.Value)
		if msg.Key != nil {
			totalSize += len(msg.Key)
		}
	}

	return totalSize >= bm.currentBatch.MaxBatchSize
}

// sendCurrentBatch 发送当前批次
func (bm *batchManager) sendCurrentBatch() {
	if len(bm.currentBatch.Messages) == 0 {
		return
	}

	// 复制当前批次
	batch := &MessageBatch{
		Messages:      make([]*Message, len(bm.currentBatch.Messages)),
		MaxBatchSize:  bm.currentBatch.MaxBatchSize,
		MaxBatchCount: bm.currentBatch.MaxBatchCount,
		LingerMs:      bm.currentBatch.LingerMs,
	}
	copy(batch.Messages, bm.currentBatch.Messages)

	// 重置当前批次
	bm.currentBatch.Messages = bm.currentBatch.Messages[:0]

	// 发送批次
	select {
	case bm.sendChan <- batch:
	default:
		bm.logger.Warn("批次发送通道已满，丢弃批次")
	}

	// 重启定时器
	bm.resetTimer()
}

// flush 刷新所有待发送的消息
func (bm *batchManager) flush() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.sendCurrentBatch()
}

// startTimer 启动定时器
func (bm *batchManager) startTimer() {
	if bm.config.LingerMs <= 0 {
		return
	}

	bm.timer = time.AfterFunc(time.Duration(bm.config.LingerMs)*time.Millisecond, func() {
		bm.mu.Lock()
		defer bm.mu.Unlock()
		bm.sendCurrentBatch()
	})
}

// resetTimer 重置定时器
func (bm *batchManager) resetTimer() {
	if bm.timer != nil {
		bm.timer.Stop()
	}
	bm.startTimer()
}

// validateProducerConfig 验证生产者配置
func validateProducerConfig(cfg ProducerConfig) error {
	if len(cfg.Brokers) == 0 {
		return NewConfigError("Broker地址列表不能为空", nil)
	}

	if cfg.ClientID == "" {
		return NewConfigError("客户端ID不能为空", nil)
	}

	if cfg.BatchSize <= 0 {
		return NewConfigError("批次大小必须大于0", nil)
	}

	if cfg.MaxMessageBytes <= 0 {
		return NewConfigError("最大消息大小必须大于0", nil)
	}

	if cfg.RequestTimeout <= 0 {
		return NewConfigError("请求超时时间必须大于0", nil)
	}

	return nil
}

// convRequiredAcks converts an integer to kgo.Acks.
func convRequiredAcks(acks int) kgo.Acks {
	switch acks {
	case -1:
		return kgo.AllISRAcks()
	case 0:
		return kgo.NoAck()
	case 1:
		return kgo.LeaderAck()
	default:
		return kgo.AllISRAcks()
	}
}
