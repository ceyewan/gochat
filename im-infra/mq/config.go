package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ConfigManager provides centralized configuration management
type ConfigManager struct {
	config *QueueConfig
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config: &QueueConfig{},
	}
}

// LoadFromFile loads configuration from JSON file
func (cm *ConfigManager) LoadFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config QueueConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	cm.config = &config
	return nil
}

// LoadFromEnv loads configuration from environment variables
func (cm *ConfigManager) LoadFromEnv() error {
	queueType := os.Getenv("MESSAGE_QUEUE_TYPE")
	if queueType == "" {
		queueType = "redis" // Default
	}

	config := &QueueConfig{
		Type: QueueType(queueType),
	}

	switch config.Type {
	case QueueTypeRedis:
		config.RedisConfig = loadRedisConfigFromEnv()
	case QueueTypeKafka:
		config.KafkaConfig = loadKafkaConfigFromEnv()
	}

	cm.config = config
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *QueueConfig {
	return cm.config
}

// loadRedisConfigFromEnv loads Redis configuration from environment
func loadRedisConfigFromEnv() *RedisQueueConfig {
	maxLen := int64(1000)
	if val := os.Getenv("REDIS_MAX_LEN"); val != "" {
		fmt.Sscanf(val, "%d", &maxLen)
	}

	claimMinIdle := 30
	if val := os.Getenv("REDIS_CLAIM_MIN_IDLE_SECONDS"); val != "" {
		fmt.Sscanf(val, "%d", &claimMinIdle)
	}

	claimCount := 10
	if val := os.Getenv("REDIS_CLAIM_COUNT"); val != "" {
		fmt.Sscanf(val, "%d", &claimCount)
	}

	return &RedisQueueConfig{
		StreamName:    getEnvOrDefault("REDIS_STREAM_NAME", "gochat:message:stream"),
		ConsumerGroup: getEnvOrDefault("REDIS_CONSUMER_GROUP", "gochat:consumer:group"),
		ConsumerName:  getEnvOrDefault("REDIS_CONSUMER_NAME", "consumer"),
		MaxLen:        maxLen,
		ClaimMinIdle:  claimMinIdle,
		ClaimCount:    claimCount,
	}
}

// loadKafkaConfigFromEnv loads Kafka configuration from environment
func loadKafkaConfigFromEnv() *KafkaConfig {
	config := getDefaultKafkaConfig()

	// Load brokers
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		config.Brokers = []string{brokers}
	}

	// Load producer settings
	if val := os.Getenv("KAFKA_PRODUCER_BATCH_SIZE"); val != "" {
		fmt.Sscanf(val, "%d", &config.ProducerConfig.BatchSize)
	}
	if val := os.Getenv("KAFKA_PRODUCER_LINGER_MS"); val != "" {
		lingerMs := 0
		fmt.Sscanf(val, "%d", &lingerMs)
		config.ProducerConfig.Linger = time.Duration(lingerMs) * time.Millisecond
	}
	if val := os.Getenv("KAFKA_PRODUCER_MAX_RETRIES"); val != "" {
		fmt.Sscanf(val, "%d", &config.ProducerConfig.MaxRetries)
	}

	// Load consumer settings
	if val := os.Getenv("KAFKA_CONSUMER_MAX_POLL_RECORDS"); val != "" {
		maxPoll := int32(100)
		fmt.Sscanf(val, "%d", &maxPoll)
		config.ConsumerConfig.MaxPollRecords = maxPoll
	}
	if val := os.Getenv("KAFKA_CONSUMER_SESSION_TIMEOUT_MS"); val != "" {
		sessionTimeout := 30000
		fmt.Sscanf(val, "%d", &sessionTimeout)
		config.ConsumerConfig.SessionTimeout = time.Duration(sessionTimeout) * time.Millisecond
	}

	// Load topic configuration
	if val := os.Getenv("KAFKA_TOPIC_PARTITIONS"); val != "" {
		partitions := int32(12)
		fmt.Sscanf(val, "%d", &partitions)
		config.TopicConfig.NumPartitions = partitions
	}
	if val := os.Getenv("KAFKA_TOPIC_REPLICATION_FACTOR"); val != "" {
		replication := int16(3)
		fmt.Sscanf(val, "%d", &replication)
		config.TopicConfig.ReplicationFactor = replication
	}

	// Load security settings
	if tls := os.Getenv("KAFKA_TLS_ENABLED"); tls == "true" {
		config.SecurityConfig.TLS = true
	}
	if username := os.Getenv("KAFKA_SASL_USERNAME"); username != "" {
		config.SecurityConfig.SASL = &SASLConfig{
			Mechanism: getEnvOrDefault("KAFKA_SASL_MECHANISM", "PLAIN"),
			Username:  username,
			Password:  os.Getenv("KAFKA_SASL_PASSWORD"),
		}
	}

	return config
}

// getEnvOrDefault returns environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SaveConfig saves configuration to file
func (cm *ConfigManager) SaveConfig(filePath string) error {
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateSampleConfig creates a sample configuration file
func CreateSampleConfig(filePath string) error {
	config := &QueueConfig{
		Type: QueueTypeKafka,
		KafkaConfig: &KafkaConfig{
			Brokers: []string{"localhost:9092"},
			ProducerConfig: ProducerConfig{
				MaxRetries:      3,
				RequiredAcks:    "all",
				BatchSize:       100,
				Linger:          "5ms",
				CompressionType: "lz4",
			},
			ConsumerConfig: ConsumerConfig{
				GroupID:           "im-service",
				SessionTimeout:    "30s",
				HeartbeatInterval: "3s",
				MaxPollRecords:    100,
				AutoCommit:        false,
				IsolationLevel:    "read_committed",
			},
			TopicConfig: TopicConfig{
				NumPartitions:     12,
				ReplicationFactor: 3,
				RetentionMs:       7 * 24 * 3600 * 1000,
				SegmentMs:         3600 * 1000,
				MinInsyncReplicas: 2,
			},
		},
	}

	cm := &ConfigManager{config: config}
	return cm.SaveConfig(filePath)
}

// ValidateConfig validates the configuration
func ValidateConfig(config *QueueConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	switch config.Type {
	case QueueTypeRedis:
		if config.RedisConfig == nil {
			return fmt.Errorf("redis configuration is required")
		}
	case QueueTypeKafka:
		if config.KafkaConfig == nil {
			return fmt.Errorf("kafka configuration is required")
		}
		if len(config.KafkaConfig.Brokers) == 0 {
			return fmt.Errorf("kafka brokers are required")
		}
	default:
		return fmt.Errorf("unsupported queue type: %s", config.Type)
	}

	return nil
}