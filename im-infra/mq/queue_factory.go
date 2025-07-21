package queue

import (
	"fmt"
	"sync"
)

// QueueType defines the type of message queue backend
type QueueType string

const (
	QueueTypeRedis QueueType = "redis"
	QueueTypeKafka QueueType = "kafka"
)

// QueueConfig holds configuration for different queue implementations
type QueueConfig struct {
	Type QueueType
	
	// Redis configuration
	RedisConfig *RedisQueueConfig
	
	// Kafka configuration
	KafkaConfig *KafkaConfig
}

type RedisQueueConfig struct {
	StreamName    string
	ConsumerGroup string
	ConsumerName  string
	MaxLen        int64
	ClaimMinIdle  int
	ClaimCount    int
}

// Global queue instances and factory
var (
	queueFactory     *QueueFactory
	queueFactoryOnce sync.Once
)

// QueueFactory provides a unified interface for creating queue instances
type QueueFactory struct {
	configs map[QueueType]*QueueConfig
	queues  map[string]MessageQueue
	mu      sync.RWMutex
}

// InitializeFactory initializes the global queue factory
func InitializeFactory(config *QueueConfig) error {
	var initErr error
	
	queueFactoryOnce.Do(func() {
		factory := &QueueFactory{
			configs: make(map[QueueType]*QueueConfig),
			queues:  make(map[string]MessageQueue),
		}
		
		// Store the configuration
		factory.configs[config.Type] = config
		
		// Initialize the default queue
		err := factory.initializeDefaultQueue(config)
		if err != nil {
			initErr = err
			return
		}
		
		queueFactory = factory
	})
	
	return initErr
}

// initializeDefaultQueue creates the default queue based on configuration
func (f *QueueFactory) initializeDefaultQueue(config *QueueConfig) error {
	switch config.Type {
	case QueueTypeRedis:
		if config.RedisConfig == nil {
			config.RedisConfig = &RedisQueueConfig{
				StreamName:    DefaultQueueName,
				ConsumerGroup: DefaultConsumerGroup,
				ConsumerName:  DefaultConsumerName,
				MaxLen:        DefaultMaxLen,
				ClaimMinIdle:  int(DefaultClaimMinIdle.Seconds()),
				ClaimCount:    DefaultClaimCount,
			}
		}
		
		queue := NewRedisQueue(config.RedisConfig.StreamName)
		if err := queue.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize redis queue: %w", err)
		}
		
		f.queues["default"] = queue
		DefaultQueue = queue
		
	case QueueTypeKafka:
		if config.KafkaConfig == nil {
			config.KafkaConfig = getDefaultKafkaConfig()
		}
		
		queue, err := NewKafkaQueue(
			"im-messages",
			"im-service-group",
			"im-consumer-1",
			config.KafkaConfig,
		)
		if err != nil {
			return fmt.Errorf("failed to create kafka queue: %w", err)
		}
		
		if err := queue.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize kafka queue: %w", err)
		}
		
		f.queues["default"] = queue
		DefaultQueue = queue
		
	default:
		return fmt.Errorf("unsupported queue type: %s", config.Type)
	}
	
	return nil
}

// GetQueue returns a queue instance by name
func (f *QueueFactory) GetQueue(name string) (MessageQueue, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	queue, exists := f.queues[name]
	if !exists {
		return nil, fmt.Errorf("queue not found: %s", name)
	}
	
	return queue, nil
}

// CreateQueue creates a new queue instance with specific configuration
func (f *QueueFactory) CreateQueue(name string, config *QueueConfig) (MessageQueue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Check if queue already exists
	if _, exists := f.queues[name]; exists {
		return nil, fmt.Errorf("queue already exists: %s", name)
	}
	
	var queue MessageQueue
	var err error
	
	switch config.Type {
	case QueueTypeRedis:
		redisConfig := config.RedisConfig
		if redisConfig == nil {
			redisConfig = &RedisQueueConfig{
				StreamName:    name,
				ConsumerGroup: DefaultConsumerGroup,
				ConsumerName:  DefaultConsumerName,
				MaxLen:        DefaultMaxLen,
				ClaimMinIdle:  int(DefaultClaimMinIdle.Seconds()),
				ClaimCount:    DefaultClaimCount,
			}
		}
		
		queue = NewRedisQueue(redisConfig.StreamName)
		err = queue.Initialize()
		
	case QueueTypeKafka:
		kafkaConfig := config.KafkaConfig
		if kafkaConfig == nil {
			kafkaConfig = getDefaultKafkaConfig()
		}
		
		queue, err = NewKafkaQueue(
			name,
			fmt.Sprintf("%s-group", name),
			fmt.Sprintf("%s-consumer", name),
			kafkaConfig,
		)
		if err == nil {
			err = queue.Initialize()
		}
		
	default:
		return nil, fmt.Errorf("unsupported queue type: %s", config.Type)
	}
	
	if err != nil {
		return nil, err
	}
	
	f.queues[name] = queue
	return queue, nil
}

// CloseAll closes all queue instances
func (f *QueueFactory) CloseAll() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	var errs []error
	for name, queue := range f.queues {
		if err := queue.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close queue %s: %w", name, err))
		}
		delete(f.queues, name)
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors closing queues: %v", errs)
	}
	
	return nil
}

// GetDefaultQueue returns the default queue instance
func GetDefaultQueue() (MessageQueue, error) {
	if queueFactory.queues["default"] == nil {
		return nil, fmt.Errorf("default queue not initialized")
	}
	return queueFactory.queues["default"], nil
}

// GetFactory returns the global queue factory instance
func GetFactory() *QueueFactory {
	return queueFactory
}

// InitializeDefaultQueue initializes the default queue with basic configuration
func InitializeDefaultQueue(queueType QueueType) error {
	config := &QueueConfig{
		Type: queueType,
	}
	
	return InitializeFactory(config)
}