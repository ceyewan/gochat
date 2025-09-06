package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ceyewan/gochat/im-infra/uid"
)

func main() {
	cfg := uid.DefaultConfig()

	generator, err := uid.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to create UID generator: %v", err)
	}
	defer generator.Close()

	fmt.Println("=== Snowflake IDs ===")
	for i := 0; i < 5; i++ {
		id := generator.GenerateInt64()
		fmt.Printf("Snowflake ID %d: %d\n", i+1, id)
	}

	fmt.Println("\n=== UUID Strings ===")
	for i := 0; i < 5; i++ {
		uuid := generator.GenerateString()
		fmt.Printf("UUID %d: %s\n", i+1, uuid)
	}

	fmt.Println("\n=== Snowflake String Mode ===")
	cfg.EnableUUID = false
	snowflakeGenerator, err := uid.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to create snowflake generator: %v", err)
	}
	defer snowflakeGenerator.Close()

	for i := 0; i < 5; i++ {
		id := snowflakeGenerator.GenerateString()
		fmt.Printf("Snowflake String %d: %s\n", i+1, id)
	}
}
