package db_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/db"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 集成测试主函数
func TestMain(m *testing.M) {
	// 检查是否跳过集成测试
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "1" {
		fmt.Println("跳过数据库集成测试")
		os.Exit(0)
	}

	// 运行所有测试
	exitCode := m.Run()

	// 清理工作（如果需要）
	cleanupTestDatabase()

	os.Exit(exitCode)
}

// 获取测试数据库配置
func getTestConfig() db.Config {
	// 从环境变量获取数据库配置，如果没有则使用默认值
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local"
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = dsn
	cfg.MaxOpenConns = 10
	cfg.MaxIdleConns = 5
	cfg.AutoCreateDatabase = true // 允许自动创建测试数据库

	return cfg
}

// 清理测试数据库
func cleanupTestDatabase() {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		return
	}

	cfg := getTestConfig()
	logger := clog.Namespace("cleanup")

	// 尝试创建连接来清理
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		return // 连接失败，忽略清理
	}
	defer provider.Close()

	ctx := context.Background()
	db := provider.DB(ctx)

	// 删除所有测试表
	tables := []string{
		"test_users",
		"test_orders",
		"test_transactions",
		"test_accounts",
		"sharded_users_00", "sharded_users_01", "sharded_users_02", "sharded_users_03",
		"sharded_orders_00", "sharded_orders_01", "sharded_orders_02", "sharded_orders_03",
		"sharded_orders_04", "sharded_orders_05", "sharded_orders_06", "sharded_orders_07",
	}

	for _, table := range tables {
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	}
}

// TestFullIntegration 完整的集成测试
func TestFullIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	cfg := getTestConfig()
	logger := clog.Namespace("integration-test")

	// 创建 Provider
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		t.Fatalf("创建 Provider 失败: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// 1. 测试连接
	t.Run("Connection", func(t *testing.T) {
		err := provider.Ping(ctx)
		assert.NoError(t, err)
	})

	// 2. 测试基本 CRUD
	t.Run("BasicCRUD", func(t *testing.T) {
		db := provider.DB(ctx)

		// 创建表
		err := provider.AutoMigrate(ctx, &TestUser{})
		require.NoError(t, err)

		// 创建用户
		user := &TestUser{
			Username: "integration_test_user",
			Email:    "integration@test.com",
		}

		err = db.Create(user).Error
		require.NoError(t, err)
		assert.NotZero(t, user.ID)

		// 查询用户
		var foundUser TestUser
		err = db.First(&foundUser, user.ID).Error
		require.NoError(t, err)
		assert.Equal(t, user.Username, foundUser.Username)
		assert.Equal(t, user.Email, foundUser.Email)

		// 更新用户
		newEmail := "updated@test.com"
		err = db.Model(&foundUser).Update("email", newEmail).Error
		require.NoError(t, err)

		// 验证更新
		err = db.First(&foundUser, user.ID).Error
		require.NoError(t, err)
		assert.Equal(t, newEmail, foundUser.Email)

		// 删除用户
		err = db.Delete(&foundUser).Error
		require.NoError(t, err)

		// 验证删除
		var count int64
		err = db.Model(&TestUser{}).Where("id = ?", user.ID).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	// 3. 测试事务
	t.Run("TransactionOperations", func(t *testing.T) {
		db := provider.DB(ctx)

		// 创建测试表
		err := provider.AutoMigrate(ctx, &TestAccount{})
		require.NoError(t, err)

		// 初始化账户
		account1 := &TestAccount{UserID: 1, Balance: 1000.0}
		account2 := &TestAccount{UserID: 2, Balance: 500.0}

		err = db.Create(&[]TestAccount{*account1, *account2}).Error
		require.NoError(t, err)

		// 执行转账事务
		transferAmount := 200.0
		err = provider.Transaction(ctx, func(tx *gorm.DB) error {
			// 扣款
			result := tx.Model(&TestAccount{}).Where("user_id = ?", 1).
				Update("balance", gorm.Expr("balance - ?", transferAmount))
			if result.Error != nil || result.RowsAffected == 0 {
				return fmt.Errorf("扣款失败")
			}

			// 加款
			result = tx.Model(&TestAccount{}).Where("user_id = ?", 2).
				Update("balance", gorm.Expr("balance + ?", transferAmount))
			if result.Error != nil || result.RowsAffected == 0 {
				return fmt.Errorf("加款失败")
			}

			// 记录交易
			transaction := &TestTransaction{
				FromUserID: 1,
				ToUserID:   2,
				Amount:     transferAmount,
			}
			return tx.Create(transaction).Error
		})

		require.NoError(t, err, "转账事务应该成功")

		// 验证余额
		var updatedAccount1, updatedAccount2 TestAccount
		err = db.First(&updatedAccount1, 1).Error
		require.NoError(t, err)
		assert.Equal(t, 800.0, updatedAccount1.Balance)

		err = db.First(&updatedAccount2, 2).Error
		require.NoError(t, err)
		assert.Equal(t, 700.0, updatedAccount2.Balance)

		// 验证交易记录
		var transactionCount int64
		err = db.Model(&TestTransaction{}).Where("from_user_id = ? AND to_user_id = ?", 1, 2).
			Count(&transactionCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), transactionCount)
	})

	// 4. 测试分片功能
	t.Run("ShardingOperations", func(t *testing.T) {
		// 创建分片配置
		shardingConfig := db.NewShardingConfig("user_id", 4)
		shardingConfig.Tables = map[string]*db.TableShardingConfig{
			"sharded_orders": {},
		}

		cfg.Sharding = shardingConfig

		// 创建新的 Provider 用于分片测试
		shardedProvider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
		if err != nil {
			t.Fatalf("创建分片 Provider 失败: %v", err)
		}
		defer shardedProvider.Close()

		shardedDB := shardedProvider.DB(ctx)

		// 创建分片表
		err = shardedProvider.AutoMigrate(ctx, &TestOrder{})
		require.NoError(t, err)

		// 创建测试订单
		orders := []TestOrder{
			{UserID: 1, Product: "Product A", Amount: 100.0, Status: "completed"},
			{UserID: 5, Product: "Product B", Amount: 200.0, Status: "pending"},
			{UserID: 9, Product: "Product C", Amount: 300.0, Status: "completed"},
			{UserID: 2, Product: "Product D", Amount: 150.0, Status: "shipped"},
		}

		for _, order := range orders {
			err = shardedDB.Create(&order).Error
			require.NoError(t, err, "创建订单失败: user_id=%d", order.UserID)
		}

		// 验证分片：查询特定用户的订单
		testCases := []struct {
			userID      int64
			expectedShard int
		}{
			{1, 1 % 4}, // user_id=1 应该在分片 1
			{5, 5 % 4}, // user_id=5 应该在分片 1
			{9, 9 % 4}, // user_id=9 应该在分片 1
			{2, 2 % 4}, // user_id=2 应该在分片 2
		}

		for _, tc := range testCases {
			var userOrders []TestOrder
			err = shardedDB.Where("user_id = ?", tc.userID).Find(&userOrders).Error
			require.NoError(t, err, "查询用户 %d 的订单失败", tc.userID)
			assert.NotEmpty(t, userOrders, "用户 %d 应该有订单", tc.userID)

			// 验证所有订单都属于同一个用户
			for _, order := range userOrders {
				assert.Equal(t, tc.userID, order.UserID)
			}
		}

		// 测试不带分片键的查询（应该返回空结果）
		var allOrders []TestOrder
		err = shardedDB.Find(&allOrders).Error
		require.NoError(t, err)
		assert.Empty(t, allOrders, "不带分片键的查询应该返回空结果")
	})

	// 5. 测试并发操作
	t.Run("ConcurrentOperations", func(t *testing.T) {
		db := provider.DB(ctx)

		// 创建测试表
		err := provider.AutoMigrate(ctx, &TestCounter{})
		require.NoError(t, err)

		// 初始化计数器
		counter := &TestCounter{ID: 1, Value: 0}
		err = db.Create(counter).Error
		require.NoError(t, err)

		// 并发增加计数器
		const numGoroutines = 10
		const incrementsPerGoroutine = 100

		done := make(chan bool, numGoroutines)
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() { done <- true }()

				for j := 0; j < incrementsPerGoroutine; j++ {
					err := provider.Transaction(ctx, func(tx *gorm.DB) error {
						return tx.Model(&TestCounter{}).Where("id = ?", 1).
							Update("value", gorm.Expr("value + 1")).Error
					})
					if err != nil {
						errors <- err
						return
					}
				}
			}()
		}

		// 等待所有 goroutine 完成
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// 正常完成
			case err := <-errors:
				t.Errorf("并发操作失败: %v", err)
			case <-time.After(30 * time.Second):
				t.Fatal("并发操作超时")
			}
		}

		// 验证最终结果
		var finalCounter TestCounter
		err = db.First(&finalCounter, 1).Error
		require.NoError(t, err)
		expectedValue := numGoroutines * incrementsPerGoroutine
		assert.Equal(t, expectedValue, finalCounter.Value)
	})

	// 6. 测试连接池统计
	t.Run("ConnectionPoolStats", func(t *testing.T) {
		db := provider.DB(ctx)
		sqlDB, err := db.DB()
		require.NoError(t, err)

		stats := sqlDB.Stats()
		assert.GreaterOrEqual(t, stats.OpenConnections, 0)
		assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1)
		assert.GreaterOrEqual(t, stats.Idle, 0)

		t.Logf("连接池统计: Open=%d, Idle=%d, InUse=%d",
			stats.OpenConnections, stats.Idle, stats.InUse)
	})
}

// 测试模型定义
type TestUser struct {
	ID        int64     `gorm:"primaryKey"`
	Username  string    `gorm:"uniqueIndex;size:50;not null"`
	Email     string    `gorm:"uniqueIndex;size:255;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type TestAccount struct {
	ID      int64   `gorm:"primaryKey"`
	UserID  int64   `gorm:"uniqueIndex;not null"`
	Balance float64 `gorm:"not null;default:0"`
}

type TestTransaction struct {
	ID         int64     `gorm:"primaryKey"`
	FromUserID int64     `gorm:"not null;index"`
	ToUserID   int64     `gorm:"not null;index"`
	Amount     float64   `gorm:"not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

type TestOrder struct {
	ID        int64     `gorm:"primaryKey"`
	UserID    int64     `gorm:"index;not null"` // 分片键
	Product   string    `gorm:"size:200;not null"`
	Amount    float64   `gorm:"not null"`
	Status    string    `gorm:"size:20;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type TestCounter struct {
	ID    int64 `gorm:"primaryKey"`
	Value int   `gorm:"not null;default:0"`
}

// BenchmarkIntegration 集成测试基准测试
func BenchmarkIntegration(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过集成基准测试")
	}

	cfg := getTestConfig()
	logger := clog.Namespace("benchmark")

	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		b.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// 准备测试数据
	err = provider.AutoMigrate(ctx, &TestUser{})
	require.NoError(b, err)

	b.Run("Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			user := &TestUser{
				Username: fmt.Sprintf("bench_user_%d", i),
				Email:    fmt.Sprintf("bench_%d@example.com", i),
			}

			db := provider.DB(ctx)
			err := db.Create(user).Error
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Query", func(b *testing.B) {
		// 先创建一些测试数据
		db := provider.DB(ctx)
		for i := 0; i < 100; i++ {
			user := &TestUser{
				Username: fmt.Sprintf("query_user_%d", i),
				Email:    fmt.Sprintf("query_%d@example.com", i),
			}
			db.Create(user)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var user TestUser
			err := db.Where("username = ?", fmt.Sprintf("query_user_%d", i%100)).First(&user).Error
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Transaction", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := provider.Transaction(ctx, func(tx *gorm.DB) error {
				return tx.Exec("SELECT 1").Error
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}