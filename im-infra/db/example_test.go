package db_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/db"
	"github.com/ceyewan/gochat/im-infra/clog"
	"gorm.io/gorm"
)

// User 用户模型示例
type User struct {
	ID        int64     `gorm:"primaryKey"`
	Username  string    `gorm:"uniqueIndex;size:50;not null"`
	Email     string    `gorm:"uniqueIndex;size:255;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// Example 演示 db 组件的基本用法
func Example() {
	// 1. 创建配置
	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local"

	// 2. 创建 Provider
	logger := clog.Namespace("example")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		log.Fatalf("创建数据库 Provider 失败: %v", err)
	}
	defer provider.Close()

	// 3. 执行 CRUD 操作
	ctx := context.Background()
	db := provider.DB(ctx)

	// 清理可能存在的测试用户
	db.Where("username = ?", "johndoe").Delete(&User{})

	// 创建用户
	user := &User{
		Username: "johndoe",
		Email:    "john@example.com",
	}
	if err := db.Create(user).Error; err != nil {
		log.Printf("创建用户失败: %v", err)
		return
	}

	// 查询用户
	var foundUser User
	if err := db.First(&foundUser, user.ID).Error; err != nil {
		log.Printf("查询用户失败: %v", err)
		return
	}

	fmt.Printf("找到用户: %s\n", foundUser.Username)

	// Output: 找到用户: johndoe
}

// ExampleProvider 演示 Provider 接口的核心功能
func ExampleProvider() {
	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("provider-example")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Close()

	ctx := context.Background()

	// 清理可能存在的测试用户
	db := provider.DB(ctx)
	db.Where("username = ?", "alice").Delete(&User{})

	// 使用 Transaction 执行事务
	err = provider.Transaction(ctx, func(tx *gorm.DB) error {
		user := &User{
			Username: "alice",
			Email:    "alice@example.com",
		}
		return tx.Create(user).Error
	})

	if err != nil {
		log.Printf("事务执行失败: %v", err)
		return
	}

	fmt.Println("事务执行成功")

	// Output: 事务执行成功
}

// ExampleGetDefaultConfig 演示默认配置的使用
func ExampleGetDefaultConfig() {
	// 获取开发环境配置
	devCfg := db.GetDefaultConfig("development")
	fmt.Printf("开发环境 - 最大连接数: %d, 日志级别: %s\n",
		devCfg.MaxOpenConns, devCfg.LogLevel)

	// 获取生产环境配置
	prodCfg := db.GetDefaultConfig("production")
	fmt.Printf("生产环境 - 最大连接数: %d, 日志级别: %s\n",
		prodCfg.MaxOpenConns, prodCfg.LogLevel)

	// Output:
	// 开发环境 - 最大连接数: 25, 日志级别: info
	// 生产环境 - 最大连接数: 100, 日志级别: warn
}

// ExampleNewShardingConfig 演示分片配置的创建
func ExampleNewShardingConfig() {
	// 创建分片配置
	shardingConfig := db.NewShardingConfig("user_id", 16)

	// 为特定表配置分片
	shardingConfig.Tables = map[string]*db.TableShardingConfig{
		"orders": {
			NumberOfShards: 8, // orders 表使用 8 个分片
		},
		"messages": {}, // messages 表使用默认 16 个分片
	}

	fmt.Printf("分片键: %s, 分片数: %d\n",
		shardingConfig.ShardingKey, shardingConfig.NumberOfShards)
	fmt.Printf("orders 表分片数: %d\n",
		shardingConfig.Tables["orders"].NumberOfShards)

	// Output:
	// 分片键: user_id, 分片数: 16
	// orders 表分片数: 8
}

// ExampleWithLogger 演示日志配置的使用
func ExampleWithLogger() {
	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local"

	// 创建带有自定义日志的 Provider
	logger := clog.Namespace("myapp").With(clog.String("component", "database"))
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Close()

	ctx := context.Background()

	// 检查连接
	err = provider.Ping(ctx)
	if err != nil {
		log.Printf("数据库连接失败: %v", err)
		return
	}

	fmt.Println("数据库连接成功，日志已配置")

	// Output: 数据库连接成功，日志已配置
}

// Example_autoMigrate 演示自动迁移功能
func Example_autoMigrate() {
	cfg := db.GetDefaultConfig("development")
	cfg.DSN = "gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local"

	logger := clog.Namespace("migrate-example")
	provider, err := db.New(context.Background(), cfg, db.WithLogger(logger))
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Close()

	ctx := context.Background()

	// 定义一个新的模型用于迁移测试
	type Product struct {
		ID        int64     `gorm:"primaryKey"`
		Name      string    `gorm:"size:100;not null"`
		Price     float64   `gorm:"not null"`
		CreatedAt time.Time `gorm:"autoCreateTime"`
		UpdatedAt time.Time `gorm:"autoUpdateTime"`
	}

	// 自动迁移表结构
	err = provider.AutoMigrate(ctx, &Product{})
	if err != nil {
		log.Printf("自动迁移失败: %v", err)
		return
	}

	fmt.Println("表结构自动迁移完成")

	// Output: 表结构自动迁移完成
}