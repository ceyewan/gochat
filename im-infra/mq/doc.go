// Package queue provides a comprehensive message queue system for Go applications
// supporting both Redis and Kafka backends with exactly-once processing guarantees.

package queue

// This file serves as the main package documentation and import summary

// Usage:
//
// 1. Basic Redis setup for development:
//    err := InitializeDefaultQueue(QueueTypeRedis)
//    if err != nil { log.Fatal(err) }
//
// 2. Kafka setup for production:
//    err := InitializeDefaultQueue(QueueTypeKafka)
//    if err != nil { log.Fatal(err) }
//
// 3. Advanced configuration:
//    config := &QueueConfig{
//        Type: QueueTypeKafka,
//        KafkaConfig: &KafkaConfig{...},
//    }
//    err := InitializeFactory(config)
//
// 4. With deduplication:
//    deduplicator := NewMessageDeduplicator(nil)
//    processor := NewMessageProcessorWithDeduplication(deduplicator, handler, nil)
//    DefaultQueue.ConsumeMessages(timeout, processor.Process)

// Architecture:
//
//   ┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
//   │   Application   │───▶│  Message Queue   │───▶│   Application   │
//   │   (Producer)    │    │    Backend       │    │   (Consumer)    │
//   └─────────────────┘    └──────────────────┘    └─────────────────┘
//                                   │
//                          ┌──────────────────┐
//                          │ Deduplication    │
//                          │ & Exactly-Once  │
//                          └──────────────────┘

// Features:
// - Multiple backend support (Redis Stream, Apache Kafka)
// - Exactly-once processing guarantees
// - Low-latency optimization for text messaging
// - Flexible configuration management
// - Graceful shutdown handling
// - Health checks and monitoring
// - Message deduplication and retry logic