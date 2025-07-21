package queue

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Simple logger for kafka operations
var kafkaLogger = log.New(log.Writer(), "[KAFKA] ", log.LstdFlags)

// KafkaQueue implements MessageQueue interface using Apache Kafka
// Optimized for low-latency text messaging with exactly-once semantics

type KafkaQueue struct {
	producer      *kgo.Client
	consumer      *kgo.Client
	ctx           context.Context
	cancel        context.CancelFunc
	config        *KafkaConfig
	topic         string
	consumerGroup string
	consumerID    string
	mu            sync.RWMutex
}

// KafkaConfig holds configuration for Kafka clients
type KafkaConfig struct {
	Brokers []string
	
	// Producer settings optimized for text messaging
	ProducerConfig struct {
		MaxRetries      int
		RequiredAcks    string
		BatchSize       int
		Linger          string
		CompressionType string
	}
	
	// Consumer settings optimized for low latency
	ConsumerConfig struct {
		GroupID           string
		SessionTimeout    string
		HeartbeatInterval string
		MaxPollRecords    int
		AutoCommit        bool
		IsolationLevel    string
	}
	
	// Topic configuration
	TopicConfig struct {
		NumPartitions     int
		ReplicationFactor int
		RetentionMs       int64
		SegmentMs         int64
		MinInsyncReplicas int
	}
	
	// Security
	SecurityConfig struct {
		TLS  bool
		SASL *SASLConfig
	}
}

// SASLConfig holds SASL authentication configuration
type SASLConfig struct {
	Mechanism string
	Username  string
	Password  string
}

// NewKafkaQueue creates a new Kafka queue instance
func NewKafkaQueue(topic, consumerGroup, consumerID string, config *KafkaConfig) (*KafkaQueue, error) {
	if config == nil {
		config = getDefaultKafkaConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	queue := &KafkaQueue{
		ctx:           ctx,
		cancel:        cancel,
		config:        config,
		topic:         topic,
		consumerGroup: consumerGroup,
		consumerID:    consumerID,
	}

	return queue, nil
}

// getDefaultKafkaConfig returns optimized configuration for text messaging
func getDefaultKafkaConfig() *KafkaConfig {
	config := &KafkaConfig{
		Brokers: []string{"localhost:9092"},
	}
	
	// Producer optimized for low-latency text messages
	config.ProducerConfig.MaxRetries = 3
	config.ProducerConfig.RequiredAcks = "all"
	config.ProducerConfig.BatchSize = 100
	config.ProducerConfig.Linger = "5ms"
	config.ProducerConfig.CompressionType = "lz4"
	
	// Consumer optimized for low-latency processing
	config.ConsumerConfig.GroupID = "im-service"
	config.ConsumerConfig.SessionTimeout = "30s"
	config.ConsumerConfig.HeartbeatInterval = "3s"
	config.ConsumerConfig.MaxPollRecords = 100
	config.ConsumerConfig.AutoCommit = false
	config.ConsumerConfig.IsolationLevel = "read_committed"
	
	// Topic configuration for text messaging
	config.TopicConfig.NumPartitions = 12
	config.TopicConfig.ReplicationFactor = 3
	config.TopicConfig.RetentionMs = 7 * 24 * 3600 * 1000 // 7 days
	config.TopicConfig.SegmentMs = 3600 * 1000             // 1 hour segments
	config.TopicConfig.MinInsyncReplicas = 2
	
	return config
}

// Initialize sets up Kafka producer and consumer clients
func (q *KafkaQueue) Initialize() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Parse durations
	producerLinger, err := time.ParseDuration(q.config.ProducerConfig.Linger)
	if err != nil {
		producerLinger = 5 * time.Millisecond
	}
	
	consumerSessionTimeout, err := time.ParseDuration(q.config.ConsumerConfig.SessionTimeout)
	if err != nil {
		consumerSessionTimeout = 30 * time.Second
	}
	
	consumerHeartbeatInterval, err := time.ParseDuration(q.config.ConsumerConfig.HeartbeatInterval)
	if err != nil {
		consumerHeartbeatInterval = 3 * time.Second
	}

	// Initialize producer
	producerOpts := []kgo.Opt{
		kgo.SeedBrokers(q.config.Brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordRetries(q.config.ProducerConfig.MaxRetries),
		kgo.ProducerBatchCompression(
			kgo.NoCompression(),
			kgo.Lz4Compression(),
			kgo.GzipCompression(),
			kgo.SnappyCompression(),
			kgo.ZstdCompression(),
		),
		kgo.ProducerLinger(producerLinger),
	}

	// Add security if configured
	if q.config.SecurityConfig.TLS {
		producerOpts = append(producerOpts, kgo.DialTLS())
	}
	if q.config.SecurityConfig.SASL != nil {
		saslMechanism := q.getSASLMechanism(q.config.SecurityConfig.SASL)
		producerOpts = append(producerOpts, kgo.SASL(saslMechanism))
	}

	// Add logging
	producerOpts = append(producerOpts, kgo.WithLogger(kgo.BasicLogger(
		log.Writer(),
		kgo.LogLevelInfo,
		func() string { return "[KAFKA-PRODUCER] " },
	)))

	producer, err := kgo.NewClient(producerOpts...)
	if err != nil {
		return fmt.Errorf("failed to create kafka producer: %w", err)
	}

	// Test producer connection
	ctx, cancel := context.WithTimeout(q.ctx, 10*time.Second)
	defer cancel()
	
	if err := producer.Ping(ctx); err != nil {
		producer.Close()
		return fmt.Errorf("failed to ping kafka brokers: %w", err)
	}

	// Initialize consumer
	consumerOpts := []kgo.Opt{
		kgo.SeedBrokers(q.config.Brokers...),
		kgo.ConsumerGroup(q.consumerGroup),
		kgo.ConsumeTopics(q.topic),
		kgo.SessionTimeout(consumerSessionTimeout),
		kgo.HeartbeatInterval(consumerHeartbeatInterval),
		kgo.FetchMaxWait(time.Duration(q.config.ConsumerConfig.MaxPollRecords) * time.Millisecond),
		kgo.DisableAutoCommit(), // Manual offset commit
	}

	// Add security if configured
	if q.config.SecurityConfig.TLS {
		consumerOpts = append(consumerOpts, kgo.DialTLS())
	}
	if q.config.SecurityConfig.SASL != nil {
		saslMechanism := q.getSASLMechanism(q.config.SecurityConfig.SASL)
		consumerOpts = append(consumerOpts, kgo.SASL(saslMechanism))
	}

	// Add logging
	consumerOpts = append(consumerOpts, kgo.WithLogger(kgo.BasicLogger(
		log.Writer(),
		kgo.LogLevelInfo,
		func() string { return "[KAFKA-CONSUMER] " },
	)))

	consumer, err := kgo.NewClient(consumerOpts...)
	if err != nil {
		producer.Close()
		return fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	// Test consumer connection
	ctx, cancel = context.WithTimeout(q.ctx, 10*time.Second)
	defer cancel()
	
	if err := consumer.Ping(ctx); err != nil {
		producer.Close()
		consumer.Close()
		return fmt.Errorf("failed to ping kafka brokers: %w", err)
	}

	q.producer = producer
	q.consumer = consumer

	kafkaLogger.Printf("Kafka queue initialized successfully - topic: %s, group: %s", q.topic, q.consumerGroup)
	return nil
}

// Close gracefully shuts down the Kafka clients
func (q *KafkaQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.cancel()

	var errs []error
	if q.producer != nil {
		q.producer.Close()
	}
	if q.consumer != nil {
		q.consumer.Close()
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing kafka clients: %v", errs)
	}
	return nil
}

// PublishMessage publishes a message to Kafka
func (q *KafkaQueue) PublishMessage(message *QueueMsg) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.producer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}

	// Serialize message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Create unique message key for deduplication
	messageKey := fmt.Sprintf("%d-%d-%d-%s", 
		message.UserId, 
		message.RoomId, 
		time.Now().UnixNano(),
		generateNonce(8),
	)

	record := &kgo.Record{
		Topic: q.topic,
		Key:   []byte(messageKey),
		Value: messageBytes,
		Headers: []kgo.RecordHeader{
			{Key: "message-type", Value: []byte("text")},
		},
	}

	// Use synchronous produce for immediate feedback
	ctx, cancel := context.WithTimeout(q.ctx, 30*time.Second)
	defer cancel()

	results := q.producer.ProduceSync(ctx, record)
	
	// Check for errors
	if err := results.FirstErr(); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

// ConsumeMessages consumes messages from Kafka
func (q *KafkaQueue) ConsumeMessages(timeout time.Duration, callback func(*QueueMsg) error) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.consumer == nil {
		return fmt.Errorf("kafka consumer not initialized")
	}

	for {
		select {
		case <-q.ctx.Done():
			return nil
		default:
			// Poll for messages
			fetches := q.consumer.PollFetches(context.Background())
			
			// Process errors
			errs := fetches.Errors()
			for _, err := range errs {
				kafkaLogger.Printf("Fetch error: %v", err.Err)
			}
			
			// Process records
			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()
				
				// Skip if context is cancelled
				select {
				case <-q.ctx.Done():
					return nil
				default:
				}

				// Deserialize message
				var msg QueueMsg
				if err := json.Unmarshal(record.Value, &msg); err != nil {
					kafkaLogger.Printf("Failed to deserialize message: %v", err)
					continue
				}

				// Process message with retry logic
				if err := q.processMessageWithRetry(&msg, record, callback); err != nil {
					kafkaLogger.Printf("Failed to process message after retries: %v", err)
					// Don't commit offset - message will be retried
					continue
				}

				// Commit offset after successful processing
				q.consumer.CommitRecords(context.Background(), record)
			}
			
			// Small delay to prevent tight loop
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// processMessageWithRetry handles message processing with retry mechanism
func (q *KafkaQueue) processMessageWithRetry(msg *QueueMsg, record *kgo.Record, callback func(*QueueMsg) error) error {
	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * baseDelay
			time.Sleep(delay)
			kafkaLogger.Printf("Retrying message processing, attempt %d/%d", attempt+1, maxRetries)
		}

		err := callback(msg)
		if err == nil {
			return nil
		}

		// Check if callback requested to stop
		if err == ErrStopConsumer {
			return err
		}

		kafkaLogger.Printf("Message processing failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
	}

	return fmt.Errorf("max retries exceeded")
}

// getSASLMechanism returns appropriate SASL mechanism
func (q *KafkaQueue) getSASLMechanism(config *SASLConfig) kgo.SASLMechanism {
	switch strings.ToUpper(config.Mechanism) {
	case "PLAIN":
		return kgo.SASLPlain(config.Username, config.Password)
	case "SCRAM-SHA-256":
		return kgo.SASLSCRAMSHA256(config.Username, config.Password)
	case "SCRAM-SHA-512":
		return kgo.SASLSCRAMSHA512(config.Username, config.Password)
	default:
		return kgo.SASLPlain(config.Username, config.Password)
	}
}

// HealthCheck checks if the Kafka cluster is reachable
func (q *KafkaQueue) HealthCheck() error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.producer == nil || q.consumer == nil {
		return fmt.Errorf("kafka clients not initialized")
	}

	ctx, cancel := context.WithTimeout(q.ctx, 5*time.Second)
	defer cancel()

	if err := q.producer.Ping(ctx); err != nil {
		return fmt.Errorf("producer health check failed: %w", err)
	}

	if err := q.consumer.Ping(ctx); err != nil {
		return fmt.Errorf("consumer health check failed: %w", err)
	}

	return nil
}

// GetStats returns Kafka client statistics
func (q *KafkaQueue) GetStats() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := make(map[string]interface{})
	
	if q.producer != nil {
		stats["producer"] = map[string]interface{}{
			"topic": q.topic,
			"brokers": q.config.Brokers,
		}
	}
	
	if q.consumer != nil {
		stats["consumer"] = map[string]interface{}{
			"topic": q.topic,
			"group": q.consumerGroup,
			"consumer_id": q.consumerID,
			"brokers": q.config.Brokers,
		}
	}

	return stats
}

// generateNonce generates a random nonce string
func generateNonce(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}