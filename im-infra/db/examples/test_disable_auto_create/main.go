package main

import (
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
)

func main() {
	clog.Info("=== 测试禁用自动创建数据库功能 ===")

	// 测试禁用自动创建数据库
	cfg := db.Config{
		DSN:                fmt.Sprintf("root:mysql@tcp(localhost:3306)/test_disabled_auto_create_%d?charset=utf8mb4&parseTime=True&loc=Local", time.Now().Unix()),
		Driver:             "mysql",
		AutoCreateDatabase: false, // 禁用自动创建
		LogLevel:           "info",
	}

	clog.Info("尝试连接到不存在的数据库（禁用自动创建）", clog.String("dsn", cfg.DSN))

	// 创建数据库实例，应该失败
	database, err := db.New(cfg)
	if err != nil {
		clog.Info("预期的失败：数据库不存在且未启用自动创建", clog.ErrorValue(err))
		clog.Info("=== 测试成功：禁用自动创建功能正常工作 ===")
		return
	}
	defer database.Close()

	clog.Error("意外成功：应该失败但却成功了")
}
