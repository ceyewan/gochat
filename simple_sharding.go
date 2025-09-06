package main

import (
	"context"
	"log"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
)

// Order 简单订单模型
type Order struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null;index"`
	Amount int64
}

func main() {
	ctx := context.Background()

	// 创建日志器
	logger := clog.Module("simple-sharding")

	// 创建 MySQL 配置
	cfg := db.MySQLConfig("root:mysql@tcp(localhost:3306)/gochat_sharding?charset=utf8mb4&parseTime=True&loc=Local")

	// 配置分片规则 - 基于 user_id 分成 4 个分片
	shardingConfig := db.NewShardingConfig("user_id", 4)
	shardingConfig.Tables = map[string]*db.TableShardingConfig{
		"orders": {
			ShardingKey:       "user_id",
			NumberOfShards:    4,
			ShardingAlgorithm: "hash",
		},
	}
	cfg.Sharding = shardingConfig

	logger.Info("分片配置完成", clog.Int("numberOfShards", 4))

	// 创建数据库实例
	database, err := db.New(ctx, cfg, db.WithLogger(logger))
	if err != nil {
		log.Fatalf("创建数据库实例失败: %v", err)
	}
	defer database.Close()

	// 自动迁移
	if err := database.AutoMigrate(&Order{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	gormDB := database.GetDB()

	// 插入测试数据
	logger.Info("插入测试数据")
	orders := []Order{
		{UserID: 1, Amount: 100},
		{UserID: 2, Amount: 200},
		{UserID: 3, Amount: 300},
		{UserID: 4, Amount: 400},
		{UserID: 5, Amount: 500}, // 会分片到 orders_1 (5%4=1)
	}

	for _, order := range orders {
		if err := gormDB.WithContext(ctx).Create(&order).Error; err != nil {
			logger.Error("创建订单失败", clog.Err(err))
		} else {
			logger.Info("订单创建成功",
				clog.Uint("userID", order.UserID),
				clog.Int64("amount", order.Amount))
		}
	}

	// 查询测试
	logger.Info("查询测试")
	var userOrders []Order
	if err := gormDB.WithContext(ctx).Where("user_id = ?", 1).Find(&userOrders).Error; err != nil {
		logger.Error("查询失败", clog.Err(err))
	} else {
		logger.Info("查询成功", clog.Int("count", len(userOrders)))
	}

	logger.Info("分片测试完成")
}
