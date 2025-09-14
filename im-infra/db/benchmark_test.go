package db_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/db"
	"github.com/ceyewan/gochat/im-infra/clog"
	"gorm.io/gorm"
)

// BenchmarkGetDefaultConfig 配置创建基准测试
func BenchmarkGetDefaultConfig(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = db.GetDefaultConfig("development")
		}
	})
}

// BenchmarkNewProvider Provider 创建基准测试
func BenchmarkNewProvider(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"
	cfg.MaxOpenConns = 20
	cfg.MaxIdleConns = 10

	logger := clog.Namespace("benchmark")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
			if err != nil {
				continue // 跳过连接失败的情况
			}
			provider.Close()
		}
	})
}

// BenchmarkDBMethod DB 方法调用基准测试
func BenchmarkDBMethod(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("benchmark")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		b.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			db := provider.DB(ctx)
			if db == nil {
				b.Fatal("无法获取数据库实例")
			}
		}
	})
}

// BenchmarkSimpleQuery 简单查询基准测试
func BenchmarkSimpleQuery(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("benchmark")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		b.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// 准备测试数据
	db := provider.DB(ctx)
	_ = db.Exec("CREATE TABLE IF NOT EXISTS benchmark_users (id BIGINT PRIMARY KEY, name VARCHAR(100))")
	_ = db.Exec("TRUNCATE TABLE benchmark_users")

	// 插入测试数据
	for i := 0; i < 1000; i++ {
		_ = db.Exec("INSERT INTO benchmark_users VALUES (?, ?)", i, fmt.Sprintf("user_%d", i))
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var name string
			err := db.Raw("SELECT name FROM benchmark_users WHERE id = ?", 0).Scan(&name).Error
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkTransaction 事务基准测试
func BenchmarkTransaction(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("benchmark")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		b.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := provider.Transaction(ctx, func(tx *gorm.DB) error {
			return tx.Exec("SELECT 1").Error
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAutoMigrate 自动迁移基准测试
func BenchmarkAutoMigrate(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
		if err != nil {
			b.Skipf("无法连接到数据库: %v", err)
		}

		ctx := context.Background()
		err = provider.AutoMigrate(ctx, &BenchmarkUser{})
		if err != nil {
			b.Fatal(err)
		}

		provider.Close()

		// 清理表
		cleanupDB, err := db.New(context.Background(), cfg, db.WithLogger(logger))
		if err == nil {
			db := cleanupDB.DB(ctx)
			_ = db.Exec("DROP TABLE IF EXISTS benchmark_users")
			cleanupDB.Close()
		}
	}
}

// BenchmarkShardingAlgorithm 分片算法基准测试
func BenchmarkShardingAlgorithm(b *testing.B) {
	testValues := []interface{}{
		int64(123456789),
		"user_123456",
		"abcdefghijklmnopqrstuvwxyz",
		int32(98765),
		uint64(5555555555),
		"short_string",
		"very_long_string_with_many_characters_for_testing_purposes",
		int64(0),
		int64(-12345),
		"negative_string",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, value := range testValues {
			_, err := simulateShardingAlgorithm(value, 16)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkConcurrentTransactions 并发事务基准测试
func BenchmarkConcurrentTransactions(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"
	cfg.MaxOpenConns = 50
	cfg.MaxIdleConns = 25

	logger := clog.Namespace("benchmark")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		b.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// 准备测试表
	db := provider.DB(ctx)
	_ = provider.AutoMigrate(ctx, &BenchmarkCounter{})
	_ = db.Exec("TRUNCATE TABLE benchmark_counters")
	_ = db.Create(&BenchmarkCounter{ID: 1, Value: 0})

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := provider.Transaction(ctx, func(tx *gorm.DB) error {
				return tx.Model(&BenchmarkCounter{}).Where("id = ?", 1).
					Update("value", gorm.Expr("value + 1")).Error
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConnectionPool 连接池性能基准测试
func BenchmarkConnectionPool(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"

	// 测试不同的连接池配置
	poolConfigs := []struct {
		name         string
		maxOpenConns int
		maxIdleConns int
	}{
		{"SmallPool", 5, 2},
		{"MediumPool", 20, 10},
		{"LargePool", 100, 50},
	}

	for _, config := range poolConfigs {
		b.Run(config.name, func(b *testing.B) {
			cfg.MaxOpenConns = config.maxOpenConns
			cfg.MaxIdleConns = config.maxIdleConns

			logger := clog.Namespace("pool-benchmark")
			provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
			if err != nil {
				b.Skipf("无法连接到数据库: %v", err)
			}
			defer provider.Close()

			ctx := context.Background()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					db := provider.DB(ctx)
					_ = db.Exec("SELECT 1")
				}
			})
		})
	}
}

// BenchmarkWithContext 带上下文的操作基准测试
func BenchmarkWithContext(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("context-benchmark")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		b.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	b.Run("WithCancel", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			_ = provider.DB(ctx)
			cancel()
		}
	})

	b.Run("WithTimeout", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = provider.DB(ctx)
			cancel()
		}
	})

	b.Run("WithValue", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.WithValue(context.Background(), "benchmark-key", "benchmark-value")
			_ = provider.DB(ctx)
		}
	})
}

// BenchmarkShardingQuery 分片查询基准测试
func BenchmarkShardingQuery(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	// 创建分片配置
	shardingConfig := db.NewShardingConfig("user_id", 16)
	shardingConfig.Tables = map[string]*db.TableShardingConfig{
		"benchmark_orders": {},
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"
	cfg.Sharding = shardingConfig

	logger := clog.Namespace("sharding-benchmark")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		b.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()
	db := provider.DB(ctx)

	// 准备测试数据
	_ = provider.AutoMigrate(ctx, &BenchmarkOrder{})
	for i := 0; i < 1000; i++ {
		userID := int64(i % 100) // 100个不同的用户
		order := &BenchmarkOrder{
			UserID:  userID,
			Product: fmt.Sprintf("product_%d", i),
			Amount:  float64(i) + 0.99,
		}
		_ = db.Create(order)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 查询特定用户的订单
			userID := int64(i % 100)
			i++
			var orders []BenchmarkOrder
			err := db.Where("user_id = ?", userID).Limit(10).Find(&orders).Error
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMemoryUsage 内存使用基准测试
func BenchmarkMemoryUsage(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("memory-benchmark")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		b.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()
	db := provider.DB(ctx)

	// 准备大量数据
	_ = provider.AutoMigrate(ctx, &BenchmarkData{})
	for i := 0; i < 10000; i++ {
		data := &BenchmarkData{
			Name:  fmt.Sprintf("data_%d", i),
			Value: i,
		}
		_ = db.Create(data)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var results []BenchmarkData
		err := db.Where("value > ?", i%1000).Limit(100).Find(&results).Error
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 基准测试模型定义
type BenchmarkUser struct {
	ID   int64  `gorm:"primaryKey"`
	Name string `gorm:"size:100"`
}

type BenchmarkCounter struct {
	ID    int `gorm:"primaryKey"`
	Value int `gorm:"not null"`
}

type BenchmarkOrder struct {
	ID      int64   `gorm:"primaryKey"`
	UserID  int64   `gorm:"index;not null"`
	Product string  `gorm:"size:200"`
	Amount  float64 `gorm:"not null"`
}

type BenchmarkData struct {
	ID    int64  `gorm:"primaryKey"`
	Name  string `gorm:"size:100"`
	Value int    `gorm:"index"`
}