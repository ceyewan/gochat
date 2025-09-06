package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
	"gorm.io/gorm"
)

// Order 订单模型 - 用于分片
type Order struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null;index"`
	OrderNo   string `gorm:"uniqueIndex;size:50;not null"`
	ProductID uint   `gorm:"not null"`
	Amount    int64  `gorm:"not null;comment:订单金额(分)"`
	Status    string `gorm:"size:20;not null;default:pending"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Message 消息模型 - 用于分片
type Message struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null;index"`
	Content   string `gorm:"type:text;not null"`
	Type      string `gorm:"size:20;not null"`
	Status    string `gorm:"size:20;not null;default:unread"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	ctx := context.Background()

	// 创建自定义日志器
	logger := clog.Module("db-sharding-example")

	// 创建 MySQL 配置
	cfg := db.MySQLConfig("root:mysql@tcp(localhost:3306)/gochat_sharding?charset=utf8mb4&parseTime=True&loc=Local")

	// === 配置分片规则 ===
	logger.Info("配置分片规则")

	// 创建分片配置
	shardingConfig := db.NewShardingConfig("user_id", 4) // 基于 user_id 分成 4 个分片

	// 为不同表配置分片
	shardingConfig.Tables = map[string]*db.TableShardingConfig{
		"orders": {
			ShardingKey:       "user_id",
			NumberOfShards:    4,
			ShardingAlgorithm: "hash",
		},
		"messages": {
			ShardingKey:       "user_id",
			NumberOfShards:    4,
			ShardingAlgorithm: "hash",
		},
	}

	// 将分片配置添加到数据库配置
	cfg.Sharding = shardingConfig

	logger.Info("分片配置完成",
		clog.String("shardingKey", shardingConfig.ShardingKey),
		clog.Int("numberOfShards", shardingConfig.NumberOfShards),
		clog.String("algorithm", shardingConfig.ShardingAlgorithm))

	// 使用 New 函数创建数据库实例，并注入 Logger
	database, err := db.New(ctx, cfg, db.WithLogger(logger), db.WithComponentName("sharding-example"))
	if err != nil {
		log.Fatalf("创建数据库实例失败: %v", err)
	}
	defer database.Close()

	logger.Info("开始分片操作示例")

	// 自动迁移 - 这会创建分片表
	if err := database.AutoMigrate(&Order{}, &Message{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	gormDB := database.GetDB()

	// === 示例1: 插入分片数据 ===
	logger.Info("=== 示例1: 插入分片数据 ===")

	// 创建多个用户的订单数据
	orders := []Order{
		{UserID: 1, OrderNo: "ORD001", ProductID: 100, Amount: 99900},  // 分片到 orders_0
		{UserID: 2, OrderNo: "ORD002", ProductID: 101, Amount: 199900}, // 分片到 orders_1
		{UserID: 3, OrderNo: "ORD003", ProductID: 102, Amount: 299900}, // 分片到 orders_2
		{UserID: 4, OrderNo: "ORD004", ProductID: 103, Amount: 399900}, // 分片到 orders_3
		{UserID: 5, OrderNo: "ORD005", ProductID: 104, Amount: 499900}, // 分片到 orders_0 (5 % 4 = 1)
		{UserID: 6, OrderNo: "ORD006", ProductID: 105, Amount: 599900}, // 分片到 orders_1 (6 % 4 = 2)
		{UserID: 7, OrderNo: "ORD007", ProductID: 106, Amount: 699900}, // 分片到 orders_2 (7 % 4 = 3)
		{UserID: 8, OrderNo: "ORD008", ProductID: 107, Amount: 799900}, // 分片到 orders_3 (8 % 4 = 0)
	}

	for _, order := range orders {
		if err := gormDB.WithContext(ctx).Create(&order).Error; err != nil {
			logger.Error("创建订单失败", clog.Err(err))
		} else {
			logger.Info("订单创建成功",
				clog.String("orderNo", order.OrderNo),
				clog.Uint("userID", order.UserID),
				clog.Int64("amount", order.Amount))
		}
	}

	// 创建多个用户的消息数据
	messages := []Message{
		{UserID: 1, Content: "用户1的消息1", Type: "text"},
		{UserID: 1, Content: "用户1的消息2", Type: "text"},
		{UserID: 2, Content: "用户2的消息1", Type: "image"},
		{UserID: 2, Content: "用户2的消息2", Type: "text"},
		{UserID: 3, Content: "用户3的消息1", Type: "video"},
		{UserID: 4, Content: "用户4的消息1", Type: "text"},
		{UserID: 5, Content: "用户5的消息1", Type: "text"},
		{UserID: 6, Content: "用户6的消息1", Type: "file"},
	}

	for _, message := range messages {
		if err := gormDB.WithContext(ctx).Create(&message).Error; err != nil {
			logger.Error("创建消息失败", clog.Err(err))
		} else {
			logger.Info("消息创建成功",
				clog.Uint("userID", message.UserID),
				clog.String("type", message.Type),
				clog.String("content", message.Content[:min(20, len(message.Content))]))
		}
	}

	// === 示例2: 查询分片数据 ===
	logger.Info("=== 示例2: 查询分片数据 ===")

	// 查询特定用户的订单（必须包含分片键）
	userID := uint(1)
	var userOrders []Order
	if err := gormDB.WithContext(ctx).Where("user_id = ?", userID).Find(&userOrders).Error; err != nil {
		logger.Error("查询用户订单失败", clog.Err(err))
	} else {
		logger.Info("查询用户订单成功",
			clog.Uint("userID", userID),
			clog.Int("count", len(userOrders)))
		for _, order := range userOrders {
			logger.Info("订单详情",
				clog.String("orderNo", order.OrderNo),
				clog.Int64("amount", order.Amount),
				clog.String("status", order.Status))
		}
	}

	// 查询特定用户的消息
	var userMessages []Message
	if err := gormDB.WithContext(ctx).Where("user_id = ?", userID).Find(&userMessages).Error; err != nil {
		logger.Error("查询用户消息失败", clog.Err(err))
	} else {
		logger.Info("查询用户消息成功",
			clog.Uint("userID", userID),
			clog.Int("count", len(userMessages)))
		for _, message := range userMessages {
			logger.Info("消息详情",
				clog.String("type", message.Type),
				clog.String("status", message.Status),
				clog.String("content", message.Content))
		}
	}

	// === 示例3: 更新分片数据 ===
	logger.Info("=== 示例3: 更新分片数据 ===")

	// 更新特定用户的订单状态
	if err := gormDB.WithContext(ctx).Model(&Order{}).Where("user_id = ? AND order_no = ?", 1, "ORD001").Update("status", "completed").Error; err != nil {
		logger.Error("更新订单状态失败", clog.Err(err))
	} else {
		logger.Info("订单状态更新成功")
	}

	// 更新特定用户的消息状态
	if err := gormDB.WithContext(ctx).Model(&Message{}).Where("user_id = ?", 1).Update("status", "read").Error; err != nil {
		logger.Error("更新消息状态失败", clog.Err(err))
	} else {
		logger.Info("消息状态更新成功")
	}

	// === 示例4: 分片数据的聚合查询 ===
	logger.Info("=== 示例4: 分片数据的聚合查询 ===")

	// 查询用户的订单总金额
	type OrderSum struct {
		UserID      uint  `gorm:"column:user_id"`
		TotalAmount int64 `gorm:"column:total_amount"`
		OrderCount  int64 `gorm:"column:order_count"`
	}

	var orderSums []OrderSum
	if err := gormDB.WithContext(ctx).Model(&Order{}).
		Select("user_id, SUM(amount) as total_amount, COUNT(*) as order_count").
		Where("user_id IN ?", []uint{1, 2, 3, 4}).
		Group("user_id").
		Find(&orderSums).Error; err != nil {
		logger.Error("聚合查询失败", clog.Err(err))
	} else {
		logger.Info("订单聚合查询成功", clog.Int("userCount", len(orderSums)))
		for _, sum := range orderSums {
			logger.Info("用户订单统计",
				clog.Uint("userID", sum.UserID),
				clog.Int64("totalAmount", sum.TotalAmount),
				clog.Int64("orderCount", sum.OrderCount))
		}
	}

	// === 示例5: 跨分片事务操作 ===
	logger.Info("=== 示例5: 跨分片事务操作 ===")

	// 注意：跨分片事务会有性能影响，应该尽量避免
	err = database.Transaction(func(tx *gorm.DB) error {
		// 创建多个用户的订单（可能跨分片）
		newOrders := []Order{
			{UserID: 9, OrderNo: "ORD009", ProductID: 108, Amount: 100000},
			{UserID: 10, OrderNo: "ORD010", ProductID: 109, Amount: 200000},
		}

		for _, order := range newOrders {
			if err := tx.Create(&order).Error; err != nil {
				return err
			}
		}

		// 创建对应的消息
		newMessages := []Message{
			{UserID: 9, Content: "订单 ORD009 已创建", Type: "notification"},
			{UserID: 10, Content: "订单 ORD010 已创建", Type: "notification"},
		}

		for _, message := range newMessages {
			if err := tx.Create(&message).Error; err != nil {
				return err
			}
		}

		logger.Info("跨分片事务操作成功")
		return nil
	})

	if err != nil {
		logger.Error("跨分片事务失败", clog.Err(err))
	} else {
		logger.Info("跨分片事务操作完成")
	}

	// === 示例6: 分片数据的批量操作 ===
	logger.Info("=== 示例6: 分片数据的批量操作 ===")

	// 批量更新特定用户的消息
	userIDs := []uint{1, 2, 3}
	for _, uid := range userIDs {
		result := gormDB.WithContext(ctx).Model(&Message{}).
			Where("user_id = ? AND type = ?", uid, "text").
			Update("status", "archived")

		if result.Error != nil {
			logger.Error("批量更新消息失败",
				clog.Uint("userID", uid),
				clog.Err(result.Error))
		} else {
			logger.Info("批量更新消息成功",
				clog.Uint("userID", uid),
				clog.Int64("affected", result.RowsAffected))
		}
	}

	// === 示例7: 分片性能测试 ===
	logger.Info("=== 示例7: 分片性能测试 ===")

	// 批量插入测试
	start := time.Now()
	batchSize := 1000
	testOrders := make([]Order, batchSize)

	for i := 0; i < batchSize; i++ {
		testOrders[i] = Order{
			UserID:    uint(i%100 + 1), // 100 个不同的用户
			OrderNo:   fmt.Sprintf("BATCH_ORD_%d", i),
			ProductID: uint(i%50 + 1),
			Amount:    int64((i + 1) * 100),
			Status:    "pending",
		}
	}

	// 分批插入
	batchInsertSize := 100
	for i := 0; i < batchSize; i += batchInsertSize {
		end := i + batchInsertSize
		if end > batchSize {
			end = batchSize
		}

		if err := gormDB.WithContext(ctx).CreateInBatches(testOrders[i:end], batchInsertSize).Error; err != nil {
			logger.Error("批量插入失败", clog.Err(err))
		} else {
			logger.Debug("批量插入成功",
				clog.Int("batch", i/batchInsertSize+1),
				clog.Int("size", end-i))
		}
	}

	duration := time.Since(start)
	logger.Info("批量插入性能测试完成",
		clog.Int("totalRecords", batchSize),
		clog.Duration("duration", duration),
		clog.Float64("recordsPerSecond", float64(batchSize)/duration.Seconds()))

	// === 查看最终结果 ===
	logger.Info("=== 查看最终结果 ===")

	// 统计各个用户的数据量
	type UserStats struct {
		UserID       uint  `gorm:"column:user_id"`
		OrderCount   int64 `gorm:"column:order_count"`
		MessageCount int64 `gorm:"column:message_count"`
	}

	// 由于是分片表，需要分别查询再聚合
	var topUsers []uint
	if err := gormDB.WithContext(ctx).Model(&Order{}).
		Select("user_id").
		Group("user_id").
		Order("COUNT(*) DESC").
		Limit(10).
		Pluck("user_id", &topUsers).Error; err != nil {
		logger.Error("查询热门用户失败", clog.Err(err))
	} else {
		logger.Info("数据分布统计", clog.Int("activeUsers", len(topUsers)))
		for _, userID := range topUsers {
			var orderCount, messageCount int64
			gormDB.WithContext(ctx).Model(&Order{}).Where("user_id = ?", userID).Count(&orderCount)
			gormDB.WithContext(ctx).Model(&Message{}).Where("user_id = ?", userID).Count(&messageCount)

			logger.Info("用户数据统计",
				clog.Uint("userID", userID),
				clog.Int64("orders", orderCount),
				clog.Int64("messages", messageCount))
		}
	}

	logger.Info("分片操作示例完成")
}

// min 函数用于计算两个整数的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
