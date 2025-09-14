package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/uid"
)

func main() {
	logger := clog.Namespace("uid-example")

	cfg := uid.Config{
		WorkerID:     1,
		DatacenterID: 1,
		EnableUUID:   true,
	}

	generator, err := uid.New(context.Background(), cfg,
		uid.WithLogger(logger),
		uid.WithComponentName("advanced-uid"))
	if err != nil {
		log.Fatalf("Failed to create UID generator: %v", err)
	}
	defer generator.Close()

	fmt.Println("=== Concurrent ID Generation Test ===")
	concurrentTest(generator)

	fmt.Println("\n=== Performance Benchmark ===")
	performanceTest(generator)

	fmt.Println("\n=== Multiple Worker Configuration ===")
	multiWorkerTest()
}

func concurrentTest(generator uid.UID) {
	const numGoroutines = 10
	const idsPerGoroutine = 1000

	idMap := sync.Map{}
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id := generator.GenerateInt64()
				if _, loaded := idMap.LoadOrStore(id, true); loaded {
					fmt.Printf("Duplicate ID detected: %d\n", id)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	count := 0
	idMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	fmt.Printf("Generated %d unique IDs in %v\n", count, duration)
	fmt.Printf("Rate: %.2f IDs/second\n", float64(count)/duration.Seconds())
}

func performanceTest(generator uid.UID) {
	const iterations = 100000

	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = generator.GenerateInt64()
	}
	duration := time.Since(start)

	fmt.Printf("Generated %d Snowflake IDs in %v\n", iterations, duration)
	fmt.Printf("Rate: %.2f IDs/second\n", float64(iterations)/duration.Seconds())

	start = time.Now()
	for i := 0; i < iterations; i++ {
		_ = generator.GenerateString()
	}
	duration = time.Since(start)

	fmt.Printf("Generated %d string IDs in %v\n", iterations, duration)
	fmt.Printf("Rate: %.2f IDs/second\n", float64(iterations)/duration.Seconds())
}

func multiWorkerTest() {
	ctx := context.Background()

	workers := []struct {
		workerID     int64
		datacenterID int64
	}{
		{1, 1},
		{2, 1},
		{1, 2},
		{2, 2},
	}

	var generators []uid.UID
	for _, w := range workers {
		cfg := uid.Config{
			WorkerID:     w.workerID,
			DatacenterID: w.datacenterID,
			EnableUUID:   false,
		}

		gen, err := uid.New(ctx, cfg)
		if err != nil {
			log.Fatalf("Failed to create generator for worker %d-%d: %v",
				w.workerID, w.datacenterID, err)
		}
		defer gen.Close()
		generators = append(generators, gen)
	}

	fmt.Println("Testing multiple workers generating IDs concurrently...")

	var wg sync.WaitGroup
	idMap := sync.Map{}

	for i, gen := range generators {
		wg.Add(1)
		go func(idx int, generator uid.UID) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				id := generator.GenerateInt64()
				key := fmt.Sprintf("%d-%d", idx, id)
				if _, loaded := idMap.LoadOrStore(key, true); loaded {
					fmt.Printf("Duplicate ID detected from generator %d: %d\n", idx, id)
				}
			}
		}(i, gen)
	}

	wg.Wait()

	count := 0
	idMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	fmt.Printf("All workers generated %d unique IDs\n", count)
}
