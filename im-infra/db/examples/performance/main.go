package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
	"gorm.io/gorm"
)

// Product 产品模型 - 用于性能测试
type Product struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"size:100;not null;index"`
	Category    string `gorm:"size:50;not null;index"`
	Price       int64  `gorm:"not null;comment:价格(分)"`
	Stock       int    `gorm:"not null;default:0"`
	Description string `gorm:"type:text"`
	Status      string `gorm:"size:20;not null;default:active;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Review 评论模型 - 用于关联查询性能测试
type Review struct {
	ID        uint   `gorm:"primaryKey"`
	ProductID uint   `gorm:"not null;index"`
	UserID    uint   `gorm:"not null;index"`
	Rating    int    `gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Content   string `gorm:"type:text"`
	Status    string `gorm:"size:20;not null;default:published;index"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Product Product `gorm:"foreignKey:ProductID"`
}

// PerformanceResult 性能测试结果
type PerformanceResult struct {
	Operation    string
	RecordCount  int
	Duration     time.Duration
	ThroughputPS float64
	AvgLatencyMS float64
}

func main() {
	ctx := context.Background()

	// 创建自定义日志器
	logger := clog.Module("db-performance-example")

	// 创建 MySQL 配置 - 优化连接池设置以提高性能
	cfg := db.MySQLConfig("gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local")

	// 优化连接池设置
	cfg.MaxOpenConns = 50                  // 增加最大连接数
	cfg.MaxIdleConns = 25                  // 增加最大空闲连接数
	cfg.ConnMaxLifetime = time.Hour        // 连接最大生存时间
	cfg.ConnMaxIdleTime = 30 * time.Minute // 连接最大空闲时间

	logger.Info("数据库性能优化配置",
		clog.Int("maxOpenConns", cfg.MaxOpenConns),
		clog.Int("maxIdleConns", cfg.MaxIdleConns),
		clog.Duration("connMaxLifetime", cfg.ConnMaxLifetime))

	// 使用 New 函数创建数据库实例，并注入 Logger
	database, err := db.New(ctx, cfg, db.WithLogger(logger), db.WithComponentName("performance-example"))
	if err != nil {
		log.Fatalf("创建数据库实例失败: %v", err)
	}
	defer database.Close()

	logger.Info("开始数据库性能测试")

	// 自动迁移
	if err := database.AutoMigrate(&Product{}, &Review{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	gormDB := database.GetDB()

	// 清理测试数据
	gormDB.Where("1 = 1").Delete(&Review{})
	gormDB.Where("1 = 1").Delete(&Product{})

	// 存储性能测试结果
	var results []PerformanceResult

	// === 性能测试1: 批量插入测试 ===
	logger.Info("=== 性能测试1: 批量插入测试 ===")

	// 测试不同批量大小的插入性能
	batchSizes := []int{100, 500, 1000, 2000}

	for _, batchSize := range batchSizes {
		products := generateTestProducts(batchSize)

		start := time.Now()

		// 使用 CreateInBatches 进行批量插入
		if err := gormDB.WithContext(ctx).CreateInBatches(products, 100).Error; err != nil {
			logger.Error("批量插入失败", clog.Err(err), clog.Int("batchSize", batchSize))
			continue
		}

		duration := time.Since(start)
		throughput := float64(batchSize) / duration.Seconds()
		avgLatency := duration.Seconds() * 1000 / float64(batchSize)

		result := PerformanceResult{
			Operation:    fmt.Sprintf("批量插入_%d条", batchSize),
			RecordCount:  batchSize,
			Duration:     duration,
			ThroughputPS: throughput,
			AvgLatencyMS: avgLatency,
		}
		results = append(results, result)

		logger.Info("批量插入性能测试完成",
			clog.Int("batchSize", batchSize),
			clog.Duration("duration", duration),
			clog.Float64("throughputPS", throughput),
			clog.Float64("avgLatencyMS", avgLatency))

		// 清理数据
		gormDB.Where("1 = 1").Delete(&Product{})
	}

	// === 性能测试2: 并发插入测试 ===
	logger.Info("=== 性能测试2: 并发插入测试 ===")

	concurrencyLevels := []int{1, 5, 10, 20}
	recordsPerGoroutine := 100

	for _, concurrency := range concurrencyLevels {
		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				products := generateTestProductsWithPrefix(recordsPerGoroutine, fmt.Sprintf("Worker%d", workerID))

				if err := gormDB.WithContext(ctx).CreateInBatches(products, 50).Error; err != nil {
					logger.Error("并发插入失败", clog.Err(err), clog.Int("workerID", workerID))
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)
		totalRecords := concurrency * recordsPerGoroutine
		throughput := float64(totalRecords) / duration.Seconds()

		result := PerformanceResult{
			Operation:    fmt.Sprintf("并发插入_%d协程", concurrency),
			RecordCount:  totalRecords,
			Duration:     duration,
			ThroughputPS: throughput,
			AvgLatencyMS: duration.Seconds() * 1000 / float64(totalRecords),
		}
		results = append(results, result)

		logger.Info("并发插入性能测试完成",
			clog.Int("concurrency", concurrency),
			clog.Int("totalRecords", totalRecords),
			clog.Duration("duration", duration),
			clog.Float64("throughputPS", throughput))

		// 清理数据
		gormDB.Where("1 = 1").Delete(&Product{})
	}

	// === 性能测试3: 查询性能测试 ===
	logger.Info("=== 性能测试3: 查询性能测试 ===")

	// 先插入测试数据
	testProducts := generateTestProducts(10000)
	if err := gormDB.WithContext(ctx).CreateInBatches(testProducts, 500).Error; err != nil {
		log.Fatalf("插入测试数据失败: %v", err)
	}

	// 获取实际插入的产品ID列表
	var productIDs []uint
	if err := gormDB.WithContext(ctx).Model(&Product{}).Pluck("id", &productIDs).Error; err != nil {
		log.Fatalf("获取产品ID失败: %v", err)
	}

	logger.Info("产品数据插入完成",
		clog.Int("productCount", len(productIDs)),
		clog.Uint("firstID", productIDs[0]),
		clog.Uint("lastID", productIDs[len(productIDs)-1]))

	// 插入评论数据，使用实际的产品ID
	testReviews := generateTestReviewsWithProductIDs(20000, productIDs)
	if err := gormDB.WithContext(ctx).CreateInBatches(testReviews, 500).Error; err != nil {
		log.Fatalf("插入评论数据失败: %v", err)
	}

	// 测试不同类型的查询
	queryTests := []struct {
		name  string
		query func() error
	}{
		{
			name: "简单ID查询",
			query: func() error {
				var product Product
				// 使用实际存在的ID进行查询
				if len(productIDs) > 0 {
					return gormDB.WithContext(ctx).First(&product, productIDs[0]).Error
				}
				return gormDB.WithContext(ctx).First(&product, 1).Error
			},
		},
		{
			name: "索引字段查询",
			query: func() error {
				var products []Product
				return gormDB.WithContext(ctx).Where("category = ?", "Electronics").Limit(100).Find(&products).Error
			},
		},
		{
			name: "范围查询",
			query: func() error {
				var products []Product
				return gormDB.WithContext(ctx).Where("price BETWEEN ? AND ?", 10000, 50000).Limit(100).Find(&products).Error
			},
		},
		{
			name: "关联查询",
			query: func() error {
				var reviews []Review
				return gormDB.WithContext(ctx).Preload("Product").Where("rating >= ?", 4).Limit(100).Find(&reviews).Error
			},
		},
		{
			name: "聚合查询",
			query: func() error {
				var result struct {
					Category string
					Count    int64
					AvgPrice float64
				}
				return gormDB.WithContext(ctx).Model(&Product{}).
					Select("category, COUNT(*) as count, AVG(price) as avg_price").
					Where("status = ?", "active").
					Group("category").
					Order("count DESC").
					Limit(1).
					Scan(&result).Error
			},
		},
	}

	for _, test := range queryTests {
		// 预热查询
		test.query()

		// 性能测试
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			if err := test.query(); err != nil {
				logger.Error("查询失败", clog.Err(err), clog.String("testName", test.name))
				break
			}
		}

		duration := time.Since(start)
		throughput := float64(iterations) / duration.Seconds()
		avgLatency := duration.Seconds() * 1000 / float64(iterations)

		result := PerformanceResult{
			Operation:    test.name,
			RecordCount:  iterations,
			Duration:     duration,
			ThroughputPS: throughput,
			AvgLatencyMS: avgLatency,
		}
		results = append(results, result)

		logger.Info("查询性能测试完成",
			clog.String("queryType", test.name),
			clog.Int("iterations", iterations),
			clog.Duration("duration", duration),
			clog.Float64("throughputPS", throughput),
			clog.Float64("avgLatencyMS", avgLatency))
	}

	// === 性能测试4: 更新性能测试 ===
	logger.Info("=== 性能测试4: 更新性能测试 ===")

	updateTests := []struct {
		name   string
		update func() error
	}{
		{
			name: "单条记录更新",
			update: func() error {
				return gormDB.WithContext(ctx).Model(&Product{}).Where("id = ?", 1).Update("stock", 100).Error
			},
		},
		{
			name: "批量更新",
			update: func() error {
				return gormDB.WithContext(ctx).Model(&Product{}).Where("category = ?", "Electronics").Update("status", "updated").Error
			},
		},
		{
			name: "条件更新",
			update: func() error {
				return gormDB.WithContext(ctx).Model(&Product{}).Where("price < ?", 20000).Update("status", "discount").Error
			},
		},
	}

	for _, test := range updateTests {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			if err := test.update(); err != nil {
				logger.Error("更新失败", clog.Err(err), clog.String("testName", test.name))
				break
			}
		}

		duration := time.Since(start)
		throughput := float64(iterations) / duration.Seconds()
		avgLatency := duration.Seconds() * 1000 / float64(iterations)

		result := PerformanceResult{
			Operation:    test.name,
			RecordCount:  iterations,
			Duration:     duration,
			ThroughputPS: throughput,
			AvgLatencyMS: avgLatency,
		}
		results = append(results, result)

		logger.Info("更新性能测试完成",
			clog.String("updateType", test.name),
			clog.Int("iterations", iterations),
			clog.Duration("duration", duration),
			clog.Float64("throughputPS", throughput),
			clog.Float64("avgLatencyMS", avgLatency))
	}

	// === 性能测试5: 事务性能测试 ===
	logger.Info("=== 性能测试5: 事务性能测试 ===")

	transactionStart := time.Now()
	transactionCount := 100

	for i := 0; i < transactionCount; i++ {
		err := database.Transaction(func(tx *gorm.DB) error {
			// 在事务中执行多个操作
			product := Product{
				Name:        fmt.Sprintf("Transaction Product %d", i),
				Category:    "Transaction",
				Price:       int64(10000 + i*100),
				Stock:       100,
				Description: "Transaction test product",
				Status:      "active",
			}

			if err := tx.Create(&product).Error; err != nil {
				return err
			}

			review := Review{
				ProductID: product.ID,
				UserID:    uint(i + 1),
				Rating:    5,
				Content:   fmt.Sprintf("Transaction review %d", i),
				Status:    "published",
			}

			return tx.Create(&review).Error
		})

		if err != nil {
			logger.Error("事务执行失败", clog.Err(err), clog.Int("transaction", i))
		}
	}

	transactionDuration := time.Since(transactionStart)
	transactionThroughput := float64(transactionCount) / transactionDuration.Seconds()

	result := PerformanceResult{
		Operation:    "事务操作",
		RecordCount:  transactionCount,
		Duration:     transactionDuration,
		ThroughputPS: transactionThroughput,
		AvgLatencyMS: transactionDuration.Seconds() * 1000 / float64(transactionCount),
	}
	results = append(results, result)

	logger.Info("事务性能测试完成",
		clog.Int("transactionCount", transactionCount),
		clog.Duration("duration", transactionDuration),
		clog.Float64("throughputPS", transactionThroughput))

	// === 连接池状态监控 ===
	logger.Info("=== 连接池状态监控 ===")

	stats := database.Stats()
	logger.Info("连接池统计信息",
		clog.Int("openConnections", stats.OpenConnections),
		clog.Int("inUse", stats.InUse),
		clog.Int("idle", stats.Idle),
		clog.Int64("waitCount", stats.WaitCount),
		clog.Duration("waitDuration", stats.WaitDuration),
		clog.Int64("maxIdleClosed", stats.MaxIdleClosed),
		clog.Int64("maxIdleTimeClosed", stats.MaxIdleTimeClosed),
		clog.Int64("maxLifetimeClosed", stats.MaxLifetimeClosed))

	// === 输出性能测试报告 ===
	logger.Info("=== 性能测试报告 ===")

	fmt.Printf("\n性能测试报告:\n")
	fmt.Printf("%-30s | %-10s | %-12s | %-15s | %-15s\n",
		"操作类型", "记录数", "耗时", "吞吐量(ops/s)", "平均延迟(ms)")
	fmt.Printf("%s\n", strings.Repeat("-", 85))

	for _, result := range results {
		fmt.Printf("%-30s | %-10d | %-12s | %-15.2f | %-15.2f\n",
			result.Operation,
			result.RecordCount,
			result.Duration.String(),
			result.ThroughputPS,
			result.AvgLatencyMS)
	}

	logger.Info("数据库性能测试完成")
}

// generateTestProducts 生成测试产品数据
func generateTestProducts(count int) []Product {
	products := make([]Product, count)
	categories := []string{"Electronics", "Books", "Clothing", "Home", "Sports"}

	for i := 0; i < count; i++ {
		products[i] = Product{
			Name:        fmt.Sprintf("Product_%d", i+1),
			Category:    categories[i%len(categories)],
			Price:       int64(1000 + (i%100)*100),
			Stock:       100 + i%50,
			Description: fmt.Sprintf("Description for product %d", i+1),
			Status:      "active",
		}
	}

	return products
}

// generateTestProductsWithPrefix 生成带前缀的测试产品数据
func generateTestProductsWithPrefix(count int, prefix string) []Product {
	products := make([]Product, count)
	categories := []string{"Electronics", "Books", "Clothing", "Home", "Sports"}

	for i := 0; i < count; i++ {
		products[i] = Product{
			Name:        fmt.Sprintf("%s_Product_%d", prefix, i+1),
			Category:    categories[i%len(categories)],
			Price:       int64(1000 + (i%100)*100),
			Stock:       100 + i%50,
			Description: fmt.Sprintf("Description for %s product %d", prefix, i+1),
			Status:      "active",
		}
	}

	return products
}

// generateTestReviews 生成测试评论数据
func generateTestReviews(count int, maxProductID int) []Review {
	reviews := make([]Review, count)

	for i := 0; i < count; i++ {
		reviews[i] = Review{
			ProductID: uint((i % maxProductID) + 1),
			UserID:    uint((i % 1000) + 1),
			Rating:    (i%5 + 1),
			Content:   fmt.Sprintf("Review content for review %d", i+1),
			Status:    "published",
		}
	}

	return reviews
}

// generateTestReviewsWithProductIDs 使用实际产品ID生成测试评论数据
func generateTestReviewsWithProductIDs(count int, productIDs []uint) []Review {
	reviews := make([]Review, count)
	productCount := len(productIDs)

	for i := 0; i < count; i++ {
		reviews[i] = Review{
			ProductID: productIDs[i%productCount], // 使用实际存在的产品ID
			UserID:    uint((i % 1000) + 1),
			Rating:    (i%5 + 1),
			Content:   fmt.Sprintf("Review content for review %d", i+1),
			Status:    "published",
		}
	}

	return reviews
}
