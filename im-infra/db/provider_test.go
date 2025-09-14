package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/db"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestProviderInterface 测试 Provider 接口的基本功能
func TestProviderInterface(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要数据库的测试")
	}

	// 创建测试配置
	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local"
	cfg.MaxOpenConns = 5
	cfg.MaxIdleConns = 2

	logger := clog.Namespace("db-test")

	// 创建 Provider
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger), db.WithComponentName("test-provider"))
	if err != nil {
		t.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	t.Run("DBMethod", func(t *testing.T) {
		ctx := context.Background()
		db := provider.DB(ctx)

		require.NotNil(t, db)
		assert.NotNil(t, db.Statement)
	})

	t.Run("Ping", func(t *testing.T) {
		ctx := context.Background()
		err := provider.Ping(ctx)

		assert.NoError(t, err)
	})

	t.Run("ConnectionStats", func(t *testing.T) {
		// 获取底层数据库连接来测试统计信息
		ctx := context.Background()
		gormDB := provider.DB(ctx)
		sqlDB, err := gormDB.DB()
		require.NoError(t, err)

		stats := sqlDB.Stats()
		assert.GreaterOrEqual(t, stats.OpenConnections, 0)
		assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1)
	})

	t.Run("TransactionSuccess", func(t *testing.T) {
		ctx := context.Background()
		var executed bool

		err := provider.Transaction(ctx, func(tx *gorm.DB) error {
			executed = true
			// 执行一个简单的查询
			return tx.Exec("SELECT 1").Error
		})

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("TransactionRollback", func(t *testing.T) {
		ctx := context.Background()
		var executed bool

		err := provider.Transaction(ctx, func(tx *gorm.DB) error {
			executed = true
			// 返回错误以触发回滚
			return assert.AnError
		})

		assert.Error(t, err)
		assert.True(t, executed)
		assert.Equal(t, assert.AnError, err)
	})

	t.Run("Close", func(t *testing.T) {
		// 创建一个新的 provider 来测试关闭
		newProvider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
		require.NoError(t, err)

		err = newProvider.Close()
		assert.NoError(t, err)

		// 关闭后应该无法再使用
		err = newProvider.Ping(context.Background())
		assert.Error(t, err)
	})
}

// TestWithOptions 测试选项功能
func TestWithOptions(t *testing.T) {
	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local"

	t.Run("WithLogger", func(t *testing.T) {
		customLogger := clog.Namespace("custom-db")
		provider, err := db.New(context.Background(), cfg, db.WithLogger(customLogger))

		if err == nil {
			defer provider.Close()
			assert.NotNil(t, provider)
		}
		// 即使连接失败，选项也应该被正确处理
	})

	t.Run("WithComponentName", func(t *testing.T) {
		provider, err := db.New(context.Background(), cfg, db.WithComponentName("my-component"))

		if err == nil {
			defer provider.Close()
			assert.NotNil(t, provider)
		}
	})

	t.Run("MultipleOptions", func(t *testing.T) {
		customLogger := clog.Namespace("multi-test")
		provider, err := db.New(context.Background(), cfg,
			db.WithLogger(customLogger),
			db.WithComponentName("multi-component"))

		if err == nil {
			defer provider.Close()
			assert.NotNil(t, provider)
		}
	})
}

// TestContextPropagation 测试上下文传播
func TestContextPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要数据库的测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("context-test")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		t.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	t.Run("DBWithContext", func(t *testing.T) {
		// 创建带有值的上下文
		ctx := context.WithValue(context.Background(), "test-key", "test-value")

		gormDB := provider.DB(ctx)
		assert.NotNil(t, gormDB)

		// 验证上下文确实被传递（虽然我们无法直接验证，但可以确保方法不panic）
		_ = gormDB.Exec("SELECT 1")
	})

	t.Run("TransactionWithContext", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "transaction-key", "transaction-value")

		err := provider.Transaction(ctx, func(tx *gorm.DB) error {
			// 在事务中使用上下文
			return tx.Exec("SELECT 1").Error
		})

		assert.NoError(t, err)
	})
}

// TestAutoMigrate 测试自动迁移功能
func TestAutoMigrate(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要数据库的测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("migrate-test")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		t.Skipf("无法连接到数据库: %v", err)
	}
	defer provider.Close()

	// 定义测试模型
	type TestUser struct {
		ID       int64  `gorm:"primaryKey"`
		Name     string `gorm:"size:100;not null"`
		Email    string `gorm:"size:255;uniqueIndex"`
		CreateAt time.Time
	}

	type TestProduct struct {
		ID    int64  `gorm:"primaryKey"`
		Name  string `gorm:"size:200;not null"`
		Price float64
	}

	t.Run("AutoMigrateSingleModel", func(t *testing.T) {
		ctx := context.Background()

		err := provider.AutoMigrate(ctx, &TestUser{})
		assert.NoError(t, err)

		// 验证表是否被创建
		gormDB := provider.DB(ctx)
		var result []map[string]interface{}
		err = gormDB.Raw("SHOW TABLES LIKE 'test_users'").Scan(&result).Error
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("AutoMigrateMultipleModels", func(t *testing.T) {
		ctx := context.Background()

		err := provider.AutoMigrate(ctx, &TestProduct{})
		assert.NoError(t, err)

		// 验证表是否被创建
		gormDB := provider.DB(ctx)
		var result []map[string]interface{}
		err = gormDB.Raw("SHOW TABLES LIKE 'test_products'").Scan(&result).Error
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

// BenchmarkProvider 性能基准测试
func BenchmarkProvider_DB(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要数据库的基准测试")
	}

	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "root:mysql@tcp(localhost:3306)/gochat_test?charset=utf8mb4&parseTime=True&loc=Local"
	cfg.MaxOpenConns = 20
	cfg.MaxIdleConns = 10

	logger := clog.Namespace("benchmark")
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
			if db == nil {
				b.Fatal("无法获取数据库实例")
			}
		}
	})
}

func BenchmarkProvider_Transaction(b *testing.B) {
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