package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog 性能测试示例 ===")

	// 1. 基准性能测试
	runBasicPerformanceTest()

	// 2. 并发性能测试
	runConcurrencyPerformanceTest()

	// 3. 模块日志器缓存性能测试
	runModuleCachingPerformanceTest()

	// 4. With 方法性能测试
	runWithMethodPerformanceTest()

	// 5. Field 类型性能测试
	runFieldTypePerformanceTest()

	// 6. Context 性能影响测试
	runContextPerformanceTest()

	// 7. 内存使用测试
	runMemoryUsageTest()

	fmt.Println("\n=== 性能测试完成 ===")
}

// runBasicPerformanceTest 基准性能测试
func runBasicPerformanceTest() {
	fmt.Println("\n1. 基准性能测试:")

	const iterations = 100000
	logger := clog.Module("performance_test")

	// 测试基本日志记录性能
	start := time.Now()
	for i := 0; i < iterations; i++ {
		logger.Info("性能测试消息", clog.Int("iteration", i))
	}
	basicDuration := time.Since(start)

	// 测试带多个字段的日志记录性能
	start = time.Now()
	for i := 0; i < iterations; i++ {
		logger.Info("多字段性能测试",
			clog.Int("iteration", i),
			clog.String("test_type", "performance"),
			clog.Bool("success", true),
			clog.Float64("score", 95.5))
	}
	multiFieldDuration := time.Since(start)

	clog.Info("基准性能测试结果",
		clog.Int("iterations", iterations),
		clog.Duration("basic_duration", basicDuration),
		clog.Duration("multi_field_duration", multiFieldDuration),
		clog.Float64("basic_ops_per_sec", float64(iterations)/basicDuration.Seconds()),
		clog.Float64("multi_field_ops_per_sec", float64(iterations)/multiFieldDuration.Seconds()))
}

// runConcurrencyPerformanceTest 并发性能测试
func runConcurrencyPerformanceTest() {
	fmt.Println("\n2. 并发性能测试:")

	const goroutines = 100
	const messagesPerGoroutine = 1000
	const totalMessages = goroutines * messagesPerGoroutine

	logger := clog.Module("concurrency_test")
	var wg sync.WaitGroup

	start := time.Now()

	// 启动多个 goroutine 并发写入日志
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			goroutineLogger := logger.With(clog.Int("goroutine_id", id))

			for j := 0; j < messagesPerGoroutine; j++ {
				goroutineLogger.Info("并发测试消息",
					clog.Int("message_id", j),
					clog.Int64("timestamp", time.Now().UnixNano()))
			}
		}(i)
	}

	wg.Wait()
	concurrentDuration := time.Since(start)

	// 单线程基准测试
	start = time.Now()
	for i := 0; i < totalMessages; i++ {
		logger.Info("单线程测试消息",
			clog.Int("message_id", i),
			clog.Int64("timestamp", time.Now().UnixNano()))
	}
	singleThreadDuration := time.Since(start)

	clog.Info("并发性能测试结果",
		clog.Int("goroutines", goroutines),
		clog.Int("messages_per_goroutine", messagesPerGoroutine),
		clog.Int("total_messages", totalMessages),
		clog.Duration("concurrent_duration", concurrentDuration),
		clog.Duration("single_thread_duration", singleThreadDuration),
		clog.Float64("concurrent_ops_per_sec", float64(totalMessages)/concurrentDuration.Seconds()),
		clog.Float64("single_thread_ops_per_sec", float64(totalMessages)/singleThreadDuration.Seconds()),
		clog.Float64("concurrency_speedup", float64(singleThreadDuration)/float64(concurrentDuration)))
}

// runModuleCachingPerformanceTest 模块日志器缓存性能测试
func runModuleCachingPerformanceTest() {
	fmt.Println("\n3. 模块日志器缓存性能测试:")

	const iterations = 50000

	// 测试缓存的模块日志器性能（推荐做法）
	cachedLogger := clog.Module("cached_test")

	start := time.Now()
	for i := 0; i < iterations; i++ {
		cachedLogger.Info("缓存日志器测试", clog.Int("iteration", i))
	}
	cachedDuration := time.Since(start)

	// 测试每次创建模块日志器的性能（不推荐做法）
	start = time.Now()
	for i := 0; i < iterations; i++ {
		clog.Module("non_cached_test").Info("非缓存日志器测试", clog.Int("iteration", i))
	}
	nonCachedDuration := time.Since(start)

	clog.Info("模块缓存性能测试结果",
		clog.Int("iterations", iterations),
		clog.Duration("cached_duration", cachedDuration),
		clog.Duration("non_cached_duration", nonCachedDuration),
		clog.Float64("cached_ops_per_sec", float64(iterations)/cachedDuration.Seconds()),
		clog.Float64("non_cached_ops_per_sec", float64(iterations)/nonCachedDuration.Seconds()),
		clog.Float64("caching_speedup", float64(nonCachedDuration)/float64(cachedDuration)))
}

// runWithMethodPerformanceTest With 方法性能测试
func runWithMethodPerformanceTest() {
	fmt.Println("\n4. With 方法性能测试:")

	const iterations = 30000

	// 测试使用 With 方法预设字段的性能（推荐做法）
	withLogger := clog.New().With(
		clog.String("service", "test-service"),
		clog.String("version", "1.0.0"),
		clog.String("environment", "test"))

	start := time.Now()
	for i := 0; i < iterations; i++ {
		withLogger.Info("With方法测试", clog.Int("iteration", i))
	}
	withMethodDuration := time.Since(start)

	// 测试每次重复添加相同字段的性能（不推荐做法）
	start = time.Now()
	for i := 0; i < iterations; i++ {
		clog.Info("重复字段测试",
			clog.String("service", "test-service"),
			clog.String("version", "1.0.0"),
			clog.String("environment", "test"),
			clog.Int("iteration", i))
	}
	repeatedFieldsDuration := time.Since(start)

	clog.Info("With方法性能测试结果",
		clog.Int("iterations", iterations),
		clog.Duration("with_method_duration", withMethodDuration),
		clog.Duration("repeated_fields_duration", repeatedFieldsDuration),
		clog.Float64("with_method_ops_per_sec", float64(iterations)/withMethodDuration.Seconds()),
		clog.Float64("repeated_fields_ops_per_sec", float64(iterations)/repeatedFieldsDuration.Seconds()),
		clog.Float64("with_method_speedup", float64(repeatedFieldsDuration)/float64(withMethodDuration)))
}

// runFieldTypePerformanceTest Field 类型性能测试
func runFieldTypePerformanceTest() {
	fmt.Println("\n5. Field 类型性能测试:")

	const iterations = 20000
	logger := clog.Module("field_test")

	// 测试不同类型字段的性能
	fieldTests := []struct {
		name string
		fn   func(int)
	}{
		{"String字段", func(i int) {
			logger.Info("String字段测试", clog.String("value", fmt.Sprintf("test-%d", i)))
		}},
		{"Int字段", func(i int) {
			logger.Info("Int字段测试", clog.Int("value", i))
		}},
		{"Bool字段", func(i int) {
			logger.Info("Bool字段测试", clog.Bool("value", i%2 == 0))
		}},
		{"Float64字段", func(i int) {
			logger.Info("Float64字段测试", clog.Float64("value", float64(i)*3.14))
		}},
		{"Duration字段", func(i int) {
			logger.Info("Duration字段测试", clog.Duration("value", time.Duration(i)*time.Millisecond))
		}},
		{"Time字段", func(i int) {
			logger.Info("Time字段测试", clog.Time("value", time.Now()))
		}},
		{"Any字段", func(i int) {
			logger.Info("Any字段测试", clog.Any("value", map[string]int{"iteration": i}))
		}},
	}

	for _, test := range fieldTests {
		start := time.Now()
		for i := 0; i < iterations; i++ {
			test.fn(i)
		}
		duration := time.Since(start)

		clog.Info("字段类型性能测试",
			clog.String("field_type", test.name),
			clog.Int("iterations", iterations),
			clog.Duration("duration", duration),
			clog.Float64("ops_per_sec", float64(iterations)/duration.Seconds()))
	}
}

// runContextPerformanceTest Context 性能影响测试
func runContextPerformanceTest() {
	fmt.Println("\n6. Context 性能影响测试:")

	const iterations = 25000
	logger := clog.Module("context_test")

	// 测试普通日志方法性能
	start := time.Now()
	for i := 0; i < iterations; i++ {
		logger.Info("普通日志测试", clog.Int("iteration", i))
	}
	normalDuration := time.Since(start)

	// 测试带 Context 的日志方法性能
	ctx := context.WithValue(context.Background(), "trace_id", "perf-test-001")
	start = time.Now()
	for i := 0; i < iterations; i++ {
		logger.InfoContext(ctx, "Context日志测试", clog.Int("iteration", i))
	}
	contextDuration := time.Since(start)

	clog.Info("Context性能影响测试结果",
		clog.Int("iterations", iterations),
		clog.Duration("normal_duration", normalDuration),
		clog.Duration("context_duration", contextDuration),
		clog.Float64("normal_ops_per_sec", float64(iterations)/normalDuration.Seconds()),
		clog.Float64("context_ops_per_sec", float64(iterations)/contextDuration.Seconds()),
		clog.Float64("context_overhead_ratio", float64(contextDuration)/float64(normalDuration)))
}

// runMemoryUsageTest 内存使用测试
func runMemoryUsageTest() {
	fmt.Println("\n7. 内存使用测试:")

	// 强制垃圾回收并获取初始内存状态
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	const iterations = 10000
	logger := clog.Module("memory_test")

	// 执行大量日志操作
	for i := 0; i < iterations; i++ {
		logger.Info("内存测试",
			clog.Int("iteration", i),
			clog.String("data", fmt.Sprintf("test-data-%d", i)),
			clog.Bool("success", true),
			clog.Float64("score", float64(i)*1.23))
	}

	// 强制垃圾回收并获取最终内存状态
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	clog.Info("内存使用测试结果",
		clog.Int("iterations", iterations),
		clog.Uint64("initial_alloc_mb", m1.Alloc/1024/1024),
		clog.Uint64("final_alloc_mb", m2.Alloc/1024/1024),
		clog.Uint64("total_alloc_mb", m2.TotalAlloc/1024/1024),
		clog.Uint64("memory_diff_mb", (m2.Alloc-m1.Alloc)/1024/1024),
		clog.Uint32("gc_cycles", m2.NumGC-m1.NumGC),
		clog.Float64("avg_memory_per_log_bytes", float64(m2.TotalAlloc-m1.TotalAlloc)/float64(iterations)))

	// 创建多个模块日志器测试内存占用
	runtime.GC()
	runtime.ReadMemStats(&m1)

	const moduleCount = 1000
	modules := make([]interface{}, moduleCount) // 防止被GC回收

	for i := 0; i < moduleCount; i++ {
		modules[i] = clog.Module(fmt.Sprintf("module-%d", i))
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	clog.Info("模块日志器内存占用测试",
		clog.Int("module_count", moduleCount),
		clog.Uint64("memory_per_module_bytes", (m2.Alloc-m1.Alloc)/uint64(moduleCount)),
		clog.Uint64("total_modules_memory_kb", (m2.Alloc-m1.Alloc)/1024))
}
