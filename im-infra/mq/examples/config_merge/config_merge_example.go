package main

import (
	"fmt"
	"log"

	"github.com/ceyewan/gochat/im-infra/mq"
)

func main() {
	fmt.Println("Testing MQ configimpl merging...")

	// Test 1: Partial configimpl with only main fields
	fmt.Println("\n=== Test 1: Partial configimpl with main fields ===")
	cfg1 := mq.Config{
		Brokers:  []string{"localhost:19092"},
		ClientID: "test-client",
		ProducerConfig: mq.ProducerConfig{
			Compression: "lz4",
			BatchSize:   32768,
		},
		ConsumerConfig: mq.ConsumerConfig{
			GroupID: "test-group",
		},
	}

	merged1 := mq.MergeWithDefaults(cfg1)
	fmt.Printf("Original ProducerConfig.Brokers: %v\n", cfg1.ProducerConfig.Brokers)
	fmt.Printf("Merged ProducerConfig.Brokers: %v\n", merged1.ProducerConfig.Brokers)
	fmt.Printf("Original ConsumerConfig.Brokers: %v\n", cfg1.ConsumerConfig.Brokers)
	fmt.Printf("Merged ConsumerConfig.Brokers: %v\n", merged1.ConsumerConfig.Brokers)
	fmt.Printf("Merged ProducerConfig.ClientID: %s\n", merged1.ProducerConfig.ClientID)
	fmt.Printf("Merged ConsumerConfig.ClientID: %s\n", merged1.ConsumerConfig.ClientID)

	// Test 2: Try to create MQ instance with partial configimpl
	fmt.Println("\n=== Test 2: Creating MQ instance with partial configimpl ===")
	mqInstance, err := mq.New(cfg1)
	if err != nil {
		log.Printf("Failed to create MQ instance: %v", err)
	} else {
		fmt.Println("✅ Successfully created MQ instance with partial configimpl!")
		mqInstance.Close()
	}

	// Test 3: Empty configimpl should get all defaults
	fmt.Println("\n=== Test 3: Empty configimpl gets all defaults ===")
	emptyCfg := mq.Config{}
	mergedEmpty := mq.MergeWithDefaults(emptyCfg)
	fmt.Printf("Default Brokers: %v\n", mergedEmpty.Brokers)
	fmt.Printf("Default ClientID: %s\n", mergedEmpty.ClientID)
	fmt.Printf("Default ProducerConfig.Brokers: %v\n", mergedEmpty.ProducerConfig.Brokers)
	fmt.Printf("Default ConsumerConfig.Brokers: %v\n", mergedEmpty.ConsumerConfig.Brokers)

	// Test 4: Show that the original chat example configimpl now works
	fmt.Println("\n=== Test 4: Original chat example configimpl ===")
	chatCfg := mq.Config{
		Brokers:  []string{"localhost:19092"},
		ClientID: "chat-example",
		ProducerConfig: mq.ProducerConfig{
			Compression:       "lz4",
			BatchSize:         16384,
			LingerMs:          5,
			EnableIdempotence: true,
		},
		ConsumerConfig: mq.ConsumerConfig{
			GroupID:         "chat-example-group",
			AutoOffsetReset: "latest",
		},
	}

	chatMqInstance, err := mq.New(chatCfg)
	if err != nil {
		log.Printf("Failed to create chat MQ instance: %v", err)
	} else {
		fmt.Println("✅ Successfully created MQ instance with chat example configimpl!")
		chatMqInstance.Close()
	}

	fmt.Println("\n✅ All tests completed successfully!")
}
