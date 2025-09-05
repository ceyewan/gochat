package main

import (
	"context"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/db"
)

// User 示例用户模型
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:100;not null"`
	Email     string `gorm:"uniqueIndex;size:100"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	// 1. 初始化 coord 实例
	coordInstance, _ := coord.New(coord.Config{
		Endpoints: []string{"localhost:2379"}, // etcd endpoints
		Timeout:   5 * time.Second,
	})

	// 2. 获取配置中心
	configCenter := coordInstance.ConfigCenter()

	// 3. 设置 db 配置中心（这会让 db 从配置中心获取配置）
	db.SetupConfigCenterFromCoord(configCenter, "dev", "gochat", "db")

	// 4. 使用默认数据库实例（会自动从配置中心获取配置）
	database := db.GetDB()

	// 5. 自动迁移
	err := db.AutoMigrate(&User{})
	if err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	// 6. 创建记录
	user := &User{
		Name:  "Alice",
		Email: "alice@example.com",
	}
	result := database.Create(user)
	if result.Error != nil {
		log.Fatal("创建用户失败:", result.Error)
	}

	log.Printf("用户创建成功，ID: %d", user.ID)

	// 7. 使用模块化实例（每个模块可以有不同的配置）
	userDB := db.Module("user")   // 配置路径: /config/dev/gochat/db-user
	orderDB := db.Module("order") // 配置路径: /config/dev/gochat/db-order

	// 8. 检查连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		log.Printf("默认数据库连接检查失败: %v", err)
	} else {
		log.Println("默认数据库连接正常")
	}

	// 9. 获取连接池统计信息
	stats := db.Stats()
	log.Printf("连接池统计: OpenConnections=%d, InUse=%d, Idle=%d",
		stats.OpenConnections, stats.InUse, stats.Idle)

	// 10. 演示配置重载
	log.Println("重新加载配置...")
	db.ReloadConfig()

	// 11. 关闭连接
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("关闭数据库连接失败: %v", err)
		}
	}()

	// 使用模块实例进行操作
	_ = userDB
	_ = orderDB

	log.Println("示例执行完成")
}
