package queue

// Example usage and integration guide for the message queue system
// This file demonstrates how to use the queue system in your application

import (
	"fmt"
	"log"
	"time"
)

// Example 1: Basic usage with Redis (development/testing)
func ExampleRedisUsage() {
	// Initialize with Redis (default)
	err := InitializeDefaultQueue(QueueTypeRedis)
	if err != nil {
		log.Fatal("Failed to initialize Redis queue:", err)
	}
	
	// Use the global DefaultQueue
	msg := &QueueMsg{
		Op:         1,
		InstanceId: "server-1",
		Msg:        []byte("Hello, Redis!"),
		UserId:     123,
		RoomId:     456,
	}
	
	// Publish message
	err = DefaultQueue.PublishMessage(msg)
	if err != nil {
		log.Fatal("Failed to publish message:", err)
	}
	
	// Consume messages
	err = DefaultQueue.ConsumeMessages(1*time.Second, func(msg *QueueMsg) error {
		fmt.Printf("Received message: %s\n", string(msg.Msg))
		return nil
	})
	if err != nil {
		log.Fatal("Failed to consume messages:", err)
	}
}

// Example 2: Basic usage with Kafka (production)
func ExampleKafkaUsage() {
	// Initialize with Kafka
	err := InitializeDefaultQueue(QueueTypeKafka)
	if err != nil {
		log.Fatal("Failed to initialize Kafka queue:", err)
	}
	
	// Use the global DefaultQueue
	msg := &QueueMsg{
		Op:         2,
		InstanceId: "server-1",
		Msg:        []byte("Hello, Kafka!"),
		UserId:     123,
		RoomId:     456,
	}
	
	// Publish message
	err = DefaultQueue.PublishMessage(msg)
	if err != nil {
		log.Fatal("Failed to publish message:", err)
	}
	
	// Consume messages
	err = DefaultQueue.ConsumeMessages(1*time.Second, func(msg *QueueMsg) error {
		fmt.Printf("Received Kafka message: %s\n", string(msg.Msg))
		return nil
	})
	if err != nil {
		log.Fatal("Failed to consume messages:", err)
	}
}

// Example 3: Advanced usage with custom configuration
func ExampleAdvancedUsage() {
	// Initialize with Redis (development)
	err := InitializeDefaultQueue(QueueTypeRedis)
	if err != nil {
		log.Fatal("Failed to initialize Redis queue:", err)
	}
	
	// Create a deduplication processor
	deduplicator := NewMessageDeduplicator(nil)
	defer deduplicator.Stop()
	
	// Create a processor with deduplication
	processor := NewMessageProcessorWithDeduplication(
		deduplicator,
		func(msg *QueueMsg) error {
			fmt.Printf("Processing message: %s\n", string(msg.Msg))
			// Your business logic here
			return nil
		},
		nil, // Use default retry policy
	)
	
	// Use the processor with the queue
	err = DefaultQueue.ConsumeMessages(1*time.Second, processor.Process)
	if err != nil {
		log.Fatal("Failed to consume messages:", err)
	}
}

// Example usage: main function
func main() {
	fmt.Println("Message queue system examples loaded")
	fmt.Println("Choose an initialization method based on your needs")
	fmt.Println("\nExample usage:")
	fmt.Println("1. Basic Redis: InitializeDefaultQueue(QueueTypeRedis)")
	fmt.Println("2. Basic Kafka: InitializeDefaultQueue(QueueTypeKafka)")
	fmt.Println("3. Environment: ConfigManager + LoadFromEnv()")
	fmt.Println("4. File config: ConfigManager + LoadFromFile()")
}