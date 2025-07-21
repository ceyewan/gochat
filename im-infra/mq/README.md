# Distributed Message Queue System

A high-performance, distributed message queue system designed for real-time messaging applications. Supports both Redis (for development/testing) and Kafka (for production) backends with exactly-once processing guarantees.

## Features

- **Multiple Backend Support**: Redis Streams and Apache Kafka
- **Exactly-Once Processing**: Message deduplication and idempotent processing
- **Low-Latency Optimized**: Configured for text messaging with minimal latency
- **Easy Integration**: Simple API that works with any Go project
- **Flexible Configuration**: Environment variables, config files, or programmatic setup
- **Production Ready**: Error handling, retry logic, and graceful shutdown

## Quick Start

### 1. Basic Usage

```go
// Initialize with Redis (development)
import "gochat/tools/queue"

// Simple initialization
err := queue.InitializeDefaultQueue(queue.QueueTypeRedis)
if err != nil {
    log.Fatal(err)
}

// Use the global queue
msg := &queue.QueueMsg{
    Op:         1,
    InstanceId: "server-1",
    Msg:        []byte("Hello World"),
    UserId:     123,
    RoomId:     456,
}

// Publish
err = queue.DefaultQueue.PublishMessage(msg)

// Consume
err = queue.DefaultQueue.ConsumeMessages(1*time.Second, func(msg *queue.QueueMsg) error {
    fmt.Printf("Received: %s\n", string(msg.Msg))
    return nil
})
```

### 2. Production Kafka Setup

```go
// Environment-based configuration
export MESSAGE_QUEUE_TYPE=kafka
export KAFKA_BROKERS=localhost:9092
export KAFKA_TOPIC_PARTITIONS=12
export KAFKA_PRODUCER_BATCH_SIZE=100
export KAFKA_PRODUCER_LINGER_MS=5

// Initialize with environment config
err := queue.InitializeDefaultQueue(queue.QueueTypeKafka)
```

### 3. Advanced Configuration

```go
// Custom configuration
config := &queue.QueueConfig{
    Type: queue.QueueTypeKafka,
    KafkaConfig: &queue.KafkaConfig{
        Brokers: []string{"localhost:9092"},
        ProducerConfig: queue.ProducerConfig{
            MaxRetries:      3,
            RequiredAcks:    "all",
            BatchSize:       100,
            Linger:          "5ms",
            CompressionType: "lz4",
        },
    },
}

err := queue.InitializeFactory(config)
```

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Application   │───▶│  Message Queue   │───▶│   Application   │
│   (Producer)    │    │    Backend       │    │   (Consumer)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                       ┌──────────────────┐
                       │ Deduplication    │
                       │ & Exactly-Once  │
                       └──────────────────┘
```

## Configuration Options

### Environment Variables

#### Redis Configuration
```bash
MESSAGE_QUEUE_TYPE=redis
REDIS_STREAM_NAME=myapp:messages
REDIS_CONSUMER_GROUP=myapp:consumers
REDIS_CONSUMER_NAME=consumer-1
REDIS_MAX_LEN=1000
REDIS_CLAIM_MIN_IDLE_SECONDS=30
REDIS_CLAIM_COUNT=10
```

#### Kafka Configuration
```bash
MESSAGE_QUEUE_TYPE=kafka
KAFKA_BROKERS=localhost:9092,localhost:9093
KAFKA_TOPIC_PARTITIONS=12
KAFKA_REPLICATION_FACTOR=3
KAFKA_PRODUCER_BATCH_SIZE=100
KAFKA_PRODUCER_LINGER_MS=5
KAFKA_CONSUMER_MAX_POLL_RECORDS=100
KAFKA_CONSUMER_SESSION_TIMEOUT_MS=30000
KAFKA_TLS_ENABLED=false
KAFKA_SASL_MECHANISM=PLAIN
KAFKA_SASL_USERNAME=user
KAFKA_SASL_PASSWORD=pass
```

### Configuration File

Create `queue-config.json`:

```json
{
  "type": "kafka",
  "kafka_config": {
    "brokers": ["localhost:9092"],
    "producer_config": {
      "max_retries": 3,
      "required_acks": "all",
      "batch_size": 100,
      "linger": "5ms",
      "compression_type": "lz4"
    },
    "consumer_config": {
      "group_id": "im-service",
      "session_timeout": "30s",
      "heartbeat_interval": "3s",
      "max_poll_records": 100,
      "auto_commit": false,
      "isolation_level": "read_committed"
    },
    "topic_config": {
      "num_partitions": 12,
      "replication_factor": 3,
      "retention_ms": 604800000,
      "segment_ms": 3600000,
      "min_insync_replicas": 2
    }
  }
}
```

## Integration Examples

### 1. Gateway Service

```go
// Gateway publishes messages to upstream topic
func handleIncomingMessage(userID, roomID int, message []byte) error {
    msg := &queue.QueueMsg{
        Op:         1, // SEND_MESSAGE
        InstanceId: "gateway-1",
        Msg:        message,
        UserId:     userID,
        RoomId:     roomID,
    }
    
    return queue.DefaultQueue.PublishMessage(msg)
}
```

### 2. Logic Service

```go
// Logic service consumes messages and processes them
func startMessageProcessor() error {
    deduplicator := queue.NewMessageDeduplicator(nil)
    defer deduplicator.Stop()
    
    processor := queue.NewMessageProcessorWithDeduplication(
        deduplicator,
        func(msg *queue.QueueMsg) error {
            // Process message (save to DB, broadcast, etc.)
            return processMessage(msg)
        },
        nil,
    )
    
    return queue.DefaultQueue.ConsumeMessages(1*time.Second, processor.Process)
}
```

### 3. Multiple Topics

```go
// Create separate queues for different message types
factory := queue.GetFactory()

// User messages queue
userQueue, _ := factory.CreateQueue("user-messages", &queue.QueueConfig{
    Type: queue.QueueTypeKafka,
    KafkaConfig: &queue.KafkaConfig{
        Brokers: []string{"localhost:9092"},
    },
})

// System notifications queue
systemQueue, _ := factory.CreateQueue("system-notifications", &queue.QueueConfig{
    Type: queue.QueueTypeKafka,
    KafkaConfig: &queue.KafkaConfig{
        Brokers: []string{"localhost:9092"},
    },
})
```

## Performance Optimization

### For Text Messaging (Optimized Settings)

#### Kafka Configuration
- **Batch Size**: 100 messages (balance between latency and throughput)
- **Linger Time**: 5ms (minimal delay for real-time messaging)
- **Compression**: LZ4 (fast compression for text)
- **Partitions**: 12 (good parallelism for text messaging)
- **Replication**: 3 (high availability)

#### Redis Configuration
- **Max Length**: 1000 messages per stream
- **Consumer Groups**: Separate groups for different services
- **Claim Settings**: 30s idle time, 10 messages per claim

### Monitoring and Health Checks

```go
// Health check
err := queue.DefaultQueue.(*queue.KafkaQueue).HealthCheck()
if err != nil {
    log.Printf("Health check failed: %v", err)
}

// Get statistics
stats := queue.DefaultQueue.(*queue.KafkaQueue).GetStats()
fmt.Printf("Queue stats: %+v\n", stats)
```

## Migration Guide

### From Redis to Kafka

1. **Phase 1**: Update configuration
   ```go
   // Change from
   err := queue.InitializeDefaultQueue(queue.QueueTypeRedis)
   
   // To
   err := queue.InitializeDefaultQueue(queue.QueueTypeKafka)
   ```

2. **Phase 2**: Gradual rollout with feature flags
   ```go
   // Use config to switch backends
   queueType := os.Getenv("MESSAGE_QUEUE_TYPE")
   err := queue.InitializeDefaultQueue(queue.QueueType(queueType))
   ```

3. **Phase 3**: Monitor and validate
   - Check message delivery times
   - Verify exactly-once processing
   - Monitor Kafka cluster health

## Testing

```bash
# Run tests
go test -v ./...

# Run integration tests (requires Redis/Kafka)
go test -v ./... -tags=integration
```

## Troubleshooting

### Common Issues

1. **Connection Issues**
   - Check Redis/Kafka server availability
   - Verify network connectivity
   - Check authentication credentials

2. **Performance Issues**
   - Monitor batch sizes and linger times
   - Check partition count for Kafka
   - Review consumer group configuration

3. **Message Loss**
   - Verify exactly-once processing is enabled
   - Check deduplication configuration
   - Monitor offset commits

### Debug Mode

```go
// Enable debug logging
os.Setenv("LOG_LEVEL", "debug")
```

## API Reference

### MessageQueue Interface

```go
type MessageQueue interface {
    Initialize() error
    Close() error
    PublishMessage(message *QueueMsg) error
    ConsumeMessages(timeout time.Duration, callback func(*QueueMsg) error) error
}
```

### QueueMsg Structure

```go
type QueueMsg struct {
    Op         int    `json:"op"`           // Operation type
    InstanceId string `json:"instance_id"`  // Server identifier
    Msg        []byte `json:"msg"`          // Message content
    UserId     int    `json:"user_id"`      // User identifier
    RoomId     int    `json:"room_id"`      // Room identifier
    Timestamp  int64  `json:"timestamp"`    // Unix timestamp
}
```

## License

This message queue system is designed for use in Go applications and provides a robust foundation for distributed messaging systems.