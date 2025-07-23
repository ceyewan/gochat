package main

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
	"gorm.io/gorm"
)

// Product 产品模型
type Product struct {
	ID          uint    `gorm:"primaryKey"`
	Name        string  `gorm:"size:100;not null"`
	Description string  `gorm:"type:text"`
	Price       float64 `gorm:"not null"`
	Stock       int     `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func main() {
	// 初始化日志
	clog.Info("=== DB 包基础功能演示 ===")

	ctx := context.Background()

	// 1. 数据库创建演示
	demonstrateDatabaseCreation(ctx)

	// 2. 全局数据库方法演示
	demonstrateGlobalMethods(ctx)

	// 3. 自定义数据库实例演示
	demonstrateCustomInstances(ctx)

	// 4. 多数据库实例演示
	demonstrateMultipleInstances(ctx)

	// 5. 事务操作演示
	demonstrateTransactions(ctx)

	// 6. 分库分表演示
	demonstrateSharding(ctx)

	clog.Info("=== 演示完成 ===")
}

// demonstrateDatabaseCreation 演示数据库自动创建
func demonstrateDatabaseCreation(ctx context.Context) {
	clog.Info("--- 数据库自动创建演示 ---")

	// 新的优雅方式：直接创建数据库实例，如果数据库不存在会自动创建
	cfg := db.Config{
		DSN:                "root:mysql@tcp(localhost:3306)/gochat_example?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:             "mysql",
		AutoCreateDatabase: true, // 启用自动创建（默认就是true）
	}

	database, err := db.New(cfg)
	if err != nil {
		clog.Error("创建数据库实例失败", clog.Err(err))
		return
	}
	defer database.Close()

	clog.Info("数据库实例创建成功（数据库已自动创建）")

	// 测试连接
	err = database.Ping(ctx)
	if err != nil {
		clog.Error("数据库连接测试失败", clog.Err(err))
	} else {
		clog.Info("数据库连接测试成功")
	}
}

// demonstrateGlobalMethods 演示全局数据库方法
func demonstrateGlobalMethods(ctx context.Context) {
	clog.Info("--- 全局数据库方法演示 ---")

	// 获取全局数据库实例
	gormDB := db.GetDB()
	if gormDB != nil {
		clog.Info("获取全局数据库实例成功")
	}

	// 自动迁移
	err := db.AutoMigrate(&Product{})
	if err != nil {
		clog.Error("自动迁移失败", clog.Err(err))
		return
	}
	clog.Info("自动迁移成功")

	// 检查连接
	err = db.Ping(ctx)
	if err != nil {
		clog.Warn("全局数据库连接检查失败", clog.Err(err))
	} else {
		clog.Info("全局数据库连接正常")
	}

	// 创建产品
	product := &Product{
		Name:        "全局方法示例产品",
		Description: "使用全局方法创建的产品",
		Price:       99.99,
		Stock:       5,
	}

	err = gormDB.WithContext(ctx).Create(product).Error
	if err != nil {
		clog.Error("创建产品失败", clog.Err(err))
	} else {
		clog.Info("创建产品成功", clog.String("name", product.Name), clog.Uint("id", product.ID))
	}
}

// demonstrateCustomInstances 演示自定义数据库实例
func demonstrateCustomInstances(ctx context.Context) {
	clog.Info("--- 自定义数据库实例演示 ---")

	// 创建自定义配置
	cfg := db.Config{
		DSN:                "root:mysql@tcp(localhost:3306)/gochat_custom_example?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:             "mysql",
		MaxOpenConns:       20,
		MaxIdleConns:       10,
		ConnMaxLifetime:    time.Hour,
		LogLevel:           "info",
		TablePrefix:        "custom_",
		AutoCreateDatabase: true, // 自动创建数据库
	}

	// 创建自定义数据库实例
	database, err := db.New(cfg)
	if err != nil {
		clog.Error("创建数据库实例失败", clog.Err(err))
		return
	}
	defer database.Close()

	// 自动迁移
	err = database.AutoMigrate(&Product{})
	if err != nil {
		clog.Error("数据库迁移失败", clog.Err(err))
		return
	}

	// 创建产品
	product := &Product{
		Name:        "自定义实例产品",
		Description: "使用自定义数据库实例创建的产品",
		Price:       199.99,
		Stock:       10,
	}

	err = database.GetDB().WithContext(ctx).Create(product).Error
	if err != nil {
		clog.Error("创建产品失败", clog.Err(err))
	} else {
		clog.Info("创建产品成功", clog.String("name", product.Name), clog.Uint("id", product.ID))
	}

	// 查询产品
	var products []Product
	err = database.GetDB().WithContext(ctx).Find(&products).Error
	if err != nil {
		clog.Error("查询产品失败", clog.Err(err))
	} else {
		clog.Info("查询产品成功", clog.Int("count", len(products)))
	}

	// 检查连接池状态
	stats := database.Stats()
	clog.Info("连接池状态",
		clog.Int("openConnections", stats.OpenConnections),
		clog.Int("inUse", stats.InUse),
		clog.Int("idle", stats.Idle),
	)
}

// demonstrateMultipleInstances 演示多个数据库实例
func demonstrateMultipleInstances(ctx context.Context) {
	clog.Info("--- 多数据库实例演示 ---")

	// 创建不同配置的数据库实例
	cfg1 := db.Config{
		DSN:                "root:mysql@tcp(localhost:3306)/gochat_product_db?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:             "mysql",
		TablePrefix:        "product_",
		AutoCreateDatabase: true, // 自动创建数据库
	}

	cfg2 := db.Config{
		DSN:                "root:mysql@tcp(localhost:3306)/gochat_user_db?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:             "mysql",
		TablePrefix:        "user_",
		AutoCreateDatabase: true, // 自动创建数据库
	}

	productDB, err := db.New(cfg1)
	if err != nil {
		clog.Error("创建产品数据库实例失败", clog.Err(err))
		return
	}
	defer productDB.Close()

	userDB, err := db.New(cfg2)
	if err != nil {
		clog.Error("创建用户数据库实例失败", clog.Err(err))
		return
	}
	defer userDB.Close()

	clog.Info("创建多个数据库实例成功")

	// 测试连接
	err = productDB.Ping(ctx)
	if err != nil {
		clog.Warn("产品数据库连接检查失败", clog.Err(err))
	} else {
		clog.Info("产品数据库连接正常")
	}

	err = userDB.Ping(ctx)
	if err != nil {
		clog.Warn("用户数据库连接检查失败", clog.Err(err))
	} else {
		clog.Info("用户数据库连接正常")
	}
}

// demonstrateTransactions 演示事务操作
func demonstrateTransactions(ctx context.Context) {
	clog.Info("--- 事务操作演示 ---")

	// 创建数据库实例
	cfg := db.Config{
		DSN:                "root:mysql@tcp(localhost:3306)/gochat_transaction_example?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:             "mysql",
		LogLevel:           "info",
		AutoCreateDatabase: true, // 自动创建数据库
	}

	database, err := db.New(cfg)
	if err != nil {
		clog.Error("创建数据库实例失败", clog.Err(err))
		return
	}
	defer database.Close()

	// 自动迁移
	database.AutoMigrate(&Product{})

	// 演示事务操作
	err = database.Transaction(func(tx *gorm.DB) error {
		// 在事务中创建多个产品
		products := []Product{
			{Name: "事务产品1", Price: 100.0, Stock: 5},
			{Name: "事务产品2", Price: 200.0, Stock: 3},
			{Name: "事务产品3", Price: 300.0, Stock: 8},
		}

		for _, product := range products {
			if err := tx.Create(&product).Error; err != nil {
				return err // 这会导致事务回滚
			}
		}

		clog.Info("事务中创建了多个产品")
		return nil
	})

	if err != nil {
		clog.Error("事务执行失败", clog.Err(err))
	} else {
		clog.Info("事务执行成功")
	}
}

// demonstrateSharding 演示分库分表
func demonstrateSharding(ctx context.Context) {
	clog.Info("--- 分库分表演示 ---")

	// 创建带分片配置的数据库实例
	shardingConfig := &db.ShardingConfig{
		ShardingKey:       "user_id",
		NumberOfShards:    4,
		ShardingAlgorithm: "hash",
		Tables: map[string]*db.TableShardingConfig{
			"orders": {},
		},
	}

	cfg := db.Config{
		DSN:                "root:mysql@tcp(localhost:3306)/gochat_sharding_example?charset=utf8mb4&parseTime=True&loc=Local",
		Driver:             "mysql",
		Sharding:           shardingConfig,
		AutoCreateDatabase: true, // 自动创建数据库
	}

	database, err := db.New(cfg)
	if err != nil {
		clog.Error("创建分片数据库实例失败", clog.Err(err))
		return
	}
	defer database.Close()

	clog.Info("分片数据库实例创建成功")

	// 注意：实际使用分片功能需要：
	// 1. 确保数据库中存在对应的分片表（如 orders_00, orders_01, orders_02, orders_03）
	// 2. 在查询时包含分片键
	clog.Info("分片功能已配置，实际使用需要创建分片表并在查询时包含分片键")
}

// Order 订单模型（用于分片演示）
type Order struct {
	ID       uint  `gorm:"primaryKey"`
	UserID   int64 `gorm:"not null"` // 分片键
	Amount   float64
	Status   string
	CreateAt time.Time
}
