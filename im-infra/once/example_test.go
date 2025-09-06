package idempotent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
)

// Example demonstrates basic usage of the idempotent component
func Example() {
	ctx := context.Background()

	// Use global methods
	key := "example:operation:123"

	// Clean up after example
	defer Delete(ctx, key)

	// Check if operation was already executed
	exists, err := Check(ctx, key)
	if err != nil {
		log.Fatal(err)
	}

	if exists {
		fmt.Println("Operation already executed")
		return
	}

	// Execute operation with idempotency
	success, err := Set(ctx, key, time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	if success {
		fmt.Println("First execution, performing operation")
		// Perform actual operation here
	} else {
		fmt.Println("Operation already in progress")
	}

	// Output: First execution, performing operation
}

// ExampleExecute demonstrates the Execute method
func ExampleExecute() {
	ctx := context.Background()
	key := "example:execute:456"

	// Clean up after example
	defer Delete(ctx, key)

	// Execute with result caching
	result, err := Execute(ctx, key, time.Hour, func() (interface{}, error) {
		// Simulate some work
		return map[string]interface{}{
			"status":  "completed",
			"user_id": 456,
			"message": "Operation successful",
		}, nil
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result: %+v\n", result)

	// Execute again - should return cached result
	cachedResult, err := Execute(ctx, key, time.Hour, func() (interface{}, error) {
		// This won't be executed
		return nil, fmt.Errorf("should not be called")
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Cached result: %+v\n", cachedResult)
}

// ExampleDo demonstrates the Do method
func ExampleDo() {
	ctx := context.Background()
	key := "example:do:789"

	// Clean up after example
	defer Delete(ctx, key)

	// Execute idempotent operation
	err := Do(ctx, key, func() error {
		fmt.Println("Performing idempotent operation")
		// Perform your actual operation here
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	// Execute again - should skip
	err = Do(ctx, key, func() error {
		fmt.Println("This won't be executed")
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	// Output: Performing idempotent operation
}

// Example_customClient demonstrates using a custom client
func Example_customClient() {
	ctx := context.Background()

	// Create custom configuration
	cfg := NewConfigBuilder().
		KeyPrefix("myapp").
		DefaultTTL(30 * time.Minute).
		CacheConfig(cache.DefaultConfig()).
		Build()

	client, err := New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	key := "example:custom:999"

	// Clean up after example
	defer client.Delete(ctx, key)

	// Use custom client
	success, err := client.Set(ctx, key, time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	if success {
		fmt.Println("Custom client: Operation set successfully")
	}

	// Output: Custom client: Operation set successfully
}

// ExampleSetWithResult demonstrates storing operation results
func ExampleSetWithResult() {
	ctx := context.Background()
	key := "example:result:111"

	// Clean up after example
	defer Delete(ctx, key)

	// Store result with idempotent operation
	result := map[string]interface{}{
		"order_id":     "12345",
		"status":       "processed",
		"amount":       99.99,
		"processed_at": time.Now().Format(time.RFC3339),
	}

	success, err := SetWithResult(ctx, key, result, time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	if success {
		fmt.Println("Result stored successfully")
	}

	// Retrieve stored result
	storedResult, err := GetResult(ctx, key)
	if err != nil {
		log.Fatal(err)
	}

	if storedResult != nil {
		fmt.Printf("Stored result: %+v\n", storedResult)
	}
}
