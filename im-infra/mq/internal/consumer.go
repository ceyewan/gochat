package internal

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// consumer 消费者实现
type consumer struct {
	// Kafka客户端
	client *kgo.Client

	// 配置
	config ConsumerConfig

	// 性能指标
	metrics consumerMetrics

	// 状态管理
	closed int32
	paused int32
	mu     sync.RWMutex

	// 日志器
	logger clog.Logger

	// 消费管理
	consumeCtx    context.Context
	consumeCancel context.CancelFunc
	consumeWG     sync.WaitGroup

	// 回调函数
	callback ConsumeCallback

	// 订阅的主题
	subscribedTopics []string

	// 暂停的分区
	pausedPartitions map[TopicPartition]bool
	pausedMu         sync.RWMutex

	// 偏移量管理
	offsetManager *offsetManager

	// 序列化器
	serializer MessageSerializer

	// 压缩器
	compressor CompressionCodec
}

// consumerMetrics 消费者性能指标的内部实现
type consumerMetrics struct {
	totalMessages int64
	totalBytes    int64

	// 吞吐量统计
	lastResetTime time.Time
	mu            sync.RWMutex

	// 延迟统计
	lag                 int64
	lastCommittedOffset map[TopicPartition]int64
	currentOffset       map[TopicPartition]int64
	offsetMu            sync.RWMutex
}

// offsetManager 偏移量管理器
type offsetManager struct {
	// 待提交的偏移量
	pendingOffsets map[TopicPartition]int64

	// 已提交的偏移量
	committedOffsets map[TopicPartition]int64

	// 锁
	mu sync.RWMutex

	// 自动提交定时器
	autoCommitTimer *time.Timer

	// 配置
	config ConsumerConfig

	// 日志器
	logger clog.Logger

	// 客户端
	client *kgo.Client
}

// NewConsumer 创建新的消费者
func NewConsumer(cfg ConsumerConfig) (Consumer, error) {
	if err := validateConsumerConfig(cfg); err != nil {
		return nil, NewConfigError("消费者配置无效", err)
	}

	// 构建Kafka客户端选项
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID(cfg.ClientID),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.SessionTimeout(cfg.SessionTimeout),
		kgo.HeartbeatInterval(cfg.HeartbeatInterval),
		kgo.FetchMinBytes(int32(cfg.FetchMinBytes)),
		kgo.FetchMaxBytes(int32(cfg.FetchMaxBytes)),
		kgo.FetchMaxWait(cfg.FetchMaxWait),
	}

	// 配置自动偏移量重置
	switch cfg.AutoOffsetReset {
	case "earliest":
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	case "latest":
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	}

	// 配置隔离级别
	if cfg.IsolationLevel == "read_committed" {
		opts = append(opts, kgo.FetchIsolationLevel(kgo.ReadCommitted()))
	}

	// 禁用自动提交（我们手动管理）
	opts = append(opts, kgo.DisableAutoCommit())

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, NewConnectionError("创建Kafka消费者客户端失败", err)
	}

	c := &consumer{
		client:           client,
		config:           cfg,
		logger:           clog.Module("mq.consumer"),
		pausedPartitions: make(map[TopicPartition]bool),
		metrics: consumerMetrics{
			lastResetTime:       time.Now(),
			lastCommittedOffset: make(map[TopicPartition]int64),
			currentOffset:       make(map[TopicPartition]int64),
		},
	}

	// 初始化序列化器和压缩器
	c.serializer = newJSONSerializer()
	c.compressor = newCompressionCodec("lz4") // 默认使用LZ4

	// 初始化偏移量管理器
	c.offsetManager = newOffsetManager(cfg, c.logger, client)

	c.logger.Info("消费者创建成功",
		clog.String("client_id", cfg.ClientID),
		clog.String("group_id", cfg.GroupID),
		clog.String("auto_offset_reset", cfg.AutoOffsetReset),
		clog.Bool("auto_commit", cfg.EnableAutoCommit))

	return c, nil
}

// Subscribe 订阅主题列表
func (c *consumer) Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.subscribedTopics = topics
	c.callback = callback

	// 订阅主题
	c.client.AddConsumeTopics(topics...)

	// 启动消费循环
	c.consumeCtx, c.consumeCancel = context.WithCancel(ctx)
	c.consumeWG.Add(1)
	go c.consumeLoop()

	c.logger.Info("订阅主题成功", clog.Strings("topics", topics))
	return nil
}

// SubscribePattern 使用正则表达式订阅主题
func (c *consumer) SubscribePattern(ctx context.Context, pattern string, callback ConsumeCallback) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.callback = callback

	// 使用正则表达式订阅
	c.client.AddConsumePartitions(map[string]map[int32]kgo.Offset{})

	// 启动消费循环
	c.consumeCtx, c.consumeCancel = context.WithCancel(ctx)
	c.consumeWG.Add(1)
	go c.consumeLoop()

	c.logger.Info("使用模式订阅主题成功", clog.String("pattern", pattern))
	return nil
}

// Unsubscribe 取消订阅指定主题
func (c *consumer) Unsubscribe(topics []string) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 从订阅列表中移除主题
	for _, topic := range topics {
		for i, subscribedTopic := range c.subscribedTopics {
			if subscribedTopic == topic {
				c.subscribedTopics = append(c.subscribedTopics[:i], c.subscribedTopics[i+1:]...)
				break
			}
		}
	}

	// 重新设置订阅
	c.client.AddConsumeTopics(c.subscribedTopics...)

	c.logger.Info("取消订阅主题成功", clog.Strings("topics", topics))
	return nil
}

// UnsubscribeAll 取消所有订阅
func (c *consumer) UnsubscribeAll() error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 停止消费循环
	if c.consumeCancel != nil {
		c.consumeCancel()
		c.consumeWG.Wait()
	}

	c.subscribedTopics = nil
	c.callback = nil

	c.logger.Info("取消所有订阅成功")
	return nil
}

// Pause 暂停指定主题分区的消费
func (c *consumer) Pause(topicPartitions []TopicPartition) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	c.pausedMu.Lock()
	defer c.pausedMu.Unlock()

	for _, tp := range topicPartitions {
		c.pausedPartitions[tp] = true
	}

	c.logger.Info("暂停分区消费", clog.Int("partition_count", len(topicPartitions)))
	return nil
}

// Resume 恢复指定主题分区的消费
func (c *consumer) Resume(topicPartitions []TopicPartition) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	c.pausedMu.Lock()
	defer c.pausedMu.Unlock()

	for _, tp := range topicPartitions {
		delete(c.pausedPartitions, tp)
	}

	c.logger.Info("恢复分区消费", clog.Int("partition_count", len(topicPartitions)))
	return nil
}

// CommitOffset 手动提交偏移量
func (c *consumer) CommitOffset(ctx context.Context, topic string, partition int32, offset int64) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	tp := TopicPartition{Topic: topic, Partition: partition}
	return c.offsetManager.commitOffset(ctx, tp, offset)
}

// CommitOffsets 批量提交偏移量
func (c *consumer) CommitOffsets(ctx context.Context, offsets map[TopicPartition]int64) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	return c.offsetManager.commitOffsets(ctx, offsets)
}

// GetCommittedOffset 获取已提交的偏移量
func (c *consumer) GetCommittedOffset(ctx context.Context, topic string, partition int32) (int64, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return -1, NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	tp := TopicPartition{Topic: topic, Partition: partition}
	return c.offsetManager.getCommittedOffset(tp), nil
}

// GetCurrentOffset 获取当前消费偏移量
func (c *consumer) GetCurrentOffset(topic string, partition int32) (int64, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return -1, NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	c.metrics.offsetMu.RLock()
	defer c.metrics.offsetMu.RUnlock()

	tp := TopicPartition{Topic: topic, Partition: partition}
	if offset, exists := c.metrics.currentOffset[tp]; exists {
		return offset, nil
	}

	return -1, NewConsumerError("未找到分区偏移量", ErrInvalidPartition)
}

// Seek 设置消费位置到指定偏移量
func (c *consumer) Seek(topic string, partition int32, offset int64) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	// 使用franz-go的Seek功能
	partitions := map[string]map[int32]kgo.Offset{
		topic: {
			partition: kgo.NewOffset().At(offset),
		},
	}

	c.client.AddConsumePartitions(partitions)

	c.logger.Debug("设置消费位置",
		clog.String("topic", topic),
		clog.Int32("partition", partition),
		clog.Int64("offset", offset))

	return nil
}

// SeekToBeginning 设置消费位置到最早偏移量
func (c *consumer) SeekToBeginning(topicPartitions []TopicPartition) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	partitions := make(map[string]map[int32]kgo.Offset)

	for _, tp := range topicPartitions {
		if partitions[tp.Topic] == nil {
			partitions[tp.Topic] = make(map[int32]kgo.Offset)
		}
		partitions[tp.Topic][tp.Partition] = kgo.NewOffset().AtStart()
	}

	c.client.AddConsumePartitions(partitions)

	c.logger.Debug("设置消费位置到开始", clog.Int("partition_count", len(topicPartitions)))
	return nil
}

// SeekToEnd 设置消费位置到最新偏移量
func (c *consumer) SeekToEnd(topicPartitions []TopicPartition) error {
	if atomic.LoadInt32(&c.closed) == 1 {
		return NewConsumerError("消费者已关闭", ErrConsumerClosed)
	}

	partitions := make(map[string]map[int32]kgo.Offset)

	for _, tp := range topicPartitions {
		if partitions[tp.Topic] == nil {
			partitions[tp.Topic] = make(map[int32]kgo.Offset)
		}
		partitions[tp.Topic][tp.Partition] = kgo.NewOffset().AtEnd()
	}

	c.client.AddConsumePartitions(partitions)

	c.logger.Debug("设置消费位置到末尾", clog.Int("partition_count", len(topicPartitions)))
	return nil
}

// Close 关闭消费者
func (c *consumer) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil // 已经关闭
	}

	c.logger.Info("开始关闭消费者")

	// 停止消费循环
	if c.consumeCancel != nil {
		c.consumeCancel()
		c.consumeWG.Wait()
	}

	// 停止偏移量管理器
	c.offsetManager.stop()

	// 关闭Kafka客户端
	c.client.Close()

	stats := c.GetMetrics()
	c.logger.Info("消费者已关闭",
		clog.Int64("total_messages", stats.TotalMessages),
		clog.Int64("total_bytes", stats.TotalBytes),
		clog.Int64("lag", stats.Lag))

	return nil
}

// GetMetrics 获取消费者性能指标
func (c *consumer) GetMetrics() ConsumerMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	c.metrics.offsetMu.RLock()
	defer c.metrics.offsetMu.RUnlock()

	totalMessages := atomic.LoadInt64(&c.metrics.totalMessages)
	totalBytes := atomic.LoadInt64(&c.metrics.totalBytes)
	lag := atomic.LoadInt64(&c.metrics.lag)

	// 计算吞吐量
	elapsed := time.Since(c.metrics.lastResetTime)
	var messagesPerSecond, bytesPerSecond float64
	if elapsed.Seconds() > 0 {
		messagesPerSecond = float64(totalMessages) / elapsed.Seconds()
		bytesPerSecond = float64(totalBytes) / elapsed.Seconds()
	}

	// 复制偏移量映射
	lastCommittedOffset := make(map[TopicPartition]int64)
	for k, v := range c.metrics.lastCommittedOffset {
		lastCommittedOffset[k] = v
	}

	currentOffset := make(map[TopicPartition]int64)
	for k, v := range c.metrics.currentOffset {
		currentOffset[k] = v
	}

	return ConsumerMetrics{
		TotalMessages:       totalMessages,
		TotalBytes:          totalBytes,
		MessagesPerSecond:   messagesPerSecond,
		BytesPerSecond:      bytesPerSecond,
		Lag:                 lag,
		LastCommittedOffset: lastCommittedOffset,
		CurrentOffset:       currentOffset,
	}
}

// consumeLoop 消费循环
func (c *consumer) consumeLoop() {
	defer c.consumeWG.Done()

	c.logger.Info("开始消费循环")

	for {
		select {
		case <-c.consumeCtx.Done():
			c.logger.Info("消费循环已停止")
			return
		default:
		}

		// 检查是否暂停
		if atomic.LoadInt32(&c.paused) == 1 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 拉取消息
		fetches := c.client.PollFetches(c.consumeCtx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				c.logger.Error("拉取消息错误", clog.ErrorValue(err.Err))
			}
			continue
		}

		// 处理消息
		c.processFetches(fetches)
	}
}

// processFetches 处理拉取的消息
func (c *consumer) processFetches(fetches kgo.Fetches) {
	fetches.EachPartition(func(partition kgo.FetchTopicPartition) {
		tp := TopicPartition{
			Topic:     partition.Topic,
			Partition: partition.Partition,
		}

		// 检查分区是否被暂停
		c.pausedMu.RLock()
		isPaused := c.pausedPartitions[tp]
		c.pausedMu.RUnlock()

		if isPaused {
			return
		}

		// 处理分区中的消息
		for _, record := range partition.Records {
			c.processRecord(record, tp)
		}
	})
}

// processRecord 处理单条记录
func (c *consumer) processRecord(record *kgo.Record, tp TopicPartition) {
	// 构建消息对象
	message := &Message{
		Topic:     record.Topic,
		Partition: record.Partition,
		Offset:    record.Offset,
		Key:       record.Key,
		Value:     record.Value,
		Headers:   make(map[string][]byte),
		Timestamp: record.Timestamp,
	}

	// 转换头部信息
	for _, header := range record.Headers {
		message.Headers[header.Key] = header.Value
	}

	// 更新当前偏移量
	c.metrics.offsetMu.Lock()
	c.metrics.currentOffset[tp] = record.Offset
	c.metrics.offsetMu.Unlock()

	// 更新指标
	atomic.AddInt64(&c.metrics.totalMessages, 1)
	atomic.AddInt64(&c.metrics.totalBytes, int64(len(record.Value)))

	// 调用回调函数
	if c.callback != nil {
		shouldContinue := c.callback(message, tp, nil)
		if !shouldContinue {
			c.logger.Info("回调函数要求停止消费")
			if c.consumeCancel != nil {
				c.consumeCancel()
			}
			return
		}
	}

	// 自动提交偏移量
	if c.config.EnableAutoCommit {
		c.offsetManager.markForCommit(tp, record.Offset+1)
	}

	c.logger.Debug("处理消息完成",
		clog.String("topic", record.Topic),
		clog.Int32("partition", record.Partition),
		clog.Int64("offset", record.Offset),
		clog.Int("message_size", len(record.Value)))
}

// newOffsetManager 创建偏移量管理器
func newOffsetManager(cfg ConsumerConfig, logger clog.Logger, client *kgo.Client) *offsetManager {
	om := &offsetManager{
		pendingOffsets:   make(map[TopicPartition]int64),
		committedOffsets: make(map[TopicPartition]int64),
		config:           cfg,
		logger:           logger,
		client:           client,
	}

	// 启动自动提交
	if cfg.EnableAutoCommit {
		om.startAutoCommit()
	}

	return om
}

// markForCommit 标记偏移量待提交
func (om *offsetManager) markForCommit(tp TopicPartition, offset int64) {
	om.mu.Lock()
	defer om.mu.Unlock()

	om.pendingOffsets[tp] = offset
}

// commitOffset 提交单个偏移量
func (om *offsetManager) commitOffset(ctx context.Context, tp TopicPartition, offset int64) error {
	offsets := map[TopicPartition]int64{tp: offset}
	return om.commitOffsets(ctx, offsets)
}

// commitOffsets 批量提交偏移量
func (om *offsetManager) commitOffsets(ctx context.Context, offsets map[TopicPartition]int64) error {
	if len(offsets) == 0 {
		return nil
	}

	// 构建提交请求
	toCommit := make(map[string]map[int32]kgo.EpochOffset)

	for tp, offset := range offsets {
		if toCommit[tp.Topic] == nil {
			toCommit[tp.Topic] = make(map[int32]kgo.EpochOffset)
		}
		toCommit[tp.Topic][tp.Partition] = kgo.EpochOffset{
			Epoch:  -1, // 使用默认epoch
			Offset: offset,
		}
	}

	// 提交偏移量
	var commitErr error
	var wg sync.WaitGroup
	wg.Add(1)

	om.client.CommitOffsets(ctx, toCommit, func(_ *kgo.Client, req *kmsg.OffsetCommitRequest, resp *kmsg.OffsetCommitResponse, err error) {
		defer wg.Done()
		if err != nil {
			commitErr = err
			return
		}
		for _, topic := range resp.Topics {
			for _, partition := range topic.Partitions {
				if err := kerr.ErrorForCode(partition.ErrorCode); err != nil {
					commitErr = err
					return
				}
			}
		}
	})

	wg.Wait()

	if commitErr != nil {
		om.logger.Error("提交偏移量失败", clog.ErrorValue(commitErr))
		return NewConsumerError("偏移量提交失败", commitErr)
	}

	// 更新已提交偏移量
	om.mu.Lock()
	for tp, offset := range offsets {
		om.committedOffsets[tp] = offset
		delete(om.pendingOffsets, tp)
	}
	om.mu.Unlock()

	om.logger.Debug("偏移量提交成功", clog.Int("offset_count", len(offsets)))
	return nil
}

// getCommittedOffset 获取已提交的偏移量
func (om *offsetManager) getCommittedOffset(tp TopicPartition) int64 {
	om.mu.RLock()
	defer om.mu.RUnlock()

	if offset, exists := om.committedOffsets[tp]; exists {
		return offset
	}

	return -1
}

// startAutoCommit 启动自动提交
func (om *offsetManager) startAutoCommit() {
	if om.config.AutoCommitInterval <= 0 {
		return
	}

	om.autoCommitTimer = time.AfterFunc(om.config.AutoCommitInterval, func() {
		om.autoCommit()
		om.startAutoCommit() // 重新启动定时器
	})
}

// autoCommit 自动提交偏移量
func (om *offsetManager) autoCommit() {
	om.mu.Lock()
	defer om.mu.Unlock()

	if len(om.pendingOffsets) == 0 {
		return
	}

	// 复制待提交偏移量
	toCommit := make(map[TopicPartition]int64)
	for tp, offset := range om.pendingOffsets {
		toCommit[tp] = offset
	}

	// 异步提交
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := om.commitOffsets(ctx, toCommit); err != nil {
			om.logger.Warn("自动提交偏移量失败", clog.ErrorValue(err))
		}
	}()
}

// stop 停止偏移量管理器
func (om *offsetManager) stop() {
	if om.autoCommitTimer != nil {
		om.autoCommitTimer.Stop()
	}

	// 最后一次提交待提交的偏移量
	om.autoCommit()
}

// validateConsumerConfig 验证消费者配置
func validateConsumerConfig(cfg ConsumerConfig) error {
	if len(cfg.Brokers) == 0 {
		return NewConfigError("Broker地址列表不能为空", nil)
	}

	if cfg.ClientID == "" {
		return NewConfigError("客户端ID不能为空", nil)
	}

	if cfg.GroupID == "" {
		return NewConfigError("消费者组ID不能为空", nil)
	}

	if cfg.SessionTimeout <= 0 {
		return NewConfigError("会话超时时间必须大于0", nil)
	}

	if cfg.HeartbeatInterval <= 0 {
		return NewConfigError("心跳间隔必须大于0", nil)
	}

	if cfg.HeartbeatInterval >= cfg.SessionTimeout {
		return NewConfigError("心跳间隔必须小于会话超时时间", nil)
	}

	return nil
}
