package main

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	// 演示全局日志方法
	demonstrateGlobalLogging()

	// 演示模块日志器
	demonstrateModuleLogging()

	// 演示并发安全性
	demonstrateConcurrency()
}

func demonstrateGlobalLogging() {
	clog.Info("=== 全局日志方法演示 ===")

	// 基本的全局日志方法
	clog.Debug("这是一个调试消息")
	clog.Info("这是一个信息消息", clog.String("component", "demo"))
	clog.Warn("这是一个警告消息", clog.String("level", "warning"))
	clog.Error("这是一个错误消息", clog.Int("error_code", 500))

	// 带 context 的全局日志方法
	ctx := context.Background()
	clog.InfoContext(ctx, "带上下文的信息消息", clog.Int("user_id", 12345))
	clog.WarnContext(ctx, "带上下文的警告消息", clog.String("session_id", "sess_123"))

	clog.Info("全局日志方法演示完成")
}

func demonstrateModuleLogging() {
	clog.Info("=== 模块日志器演示 ===")

	// 创建不同的模块日志器
	dbLogger := clog.Module("database")
	apiLogger := clog.Module("api")
	authLogger := clog.Module("auth")

	// 数据库模块日志
	dbLogger.Info("数据库连接已建立", clog.String("host", "localhost"), clog.Int("port", 5432))
	dbLogger.Debug("执行查询", clog.String("query", "SELECT * FROM users"))
	dbLogger.Warn("查询耗时较长", clog.String("duration", "2.5s"))

	// API 模块日志
	apiLogger.Info("API 服务启动", clog.Int("port", 8080))
	apiLogger.Info("处理请求", clog.String("method", "GET"), clog.String("path", "/api/users"))
	apiLogger.Error("请求处理失败", clog.String("error", "database timeout"))

	// 认证模块日志
	authLogger.Info("用户登录", clog.String("username", "alice"), clog.String("ip", "192.168.1.100"))
	authLogger.Warn("登录失败", clog.String("username", "bob"), clog.String("reason", "invalid password"))

	// 带 context 的模块日志
	ctx := context.Background()
	dbLogger.InfoContext(ctx, "事务提交", clog.String("transaction_id", "tx_456"))
	apiLogger.InfoContext(ctx, "响应发送", clog.Int("status_code", 200), clog.String("response_time", "150ms"))

	clog.Info("模块日志器演示完成")
}

func demonstrateConcurrency() {
	clog.Info("=== 并发安全性演示 ===")

	// 启动多个 goroutine 同时使用全局日志和模块日志
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			// 使用全局日志
			clog.Info("并发全局日志", clog.Int("goroutine_id", id))

			// 使用模块日志器
			logger := clog.Module("concurrent")
			logger.Info("并发模块日志", clog.Int("goroutine_id", id), clog.Int64("timestamp", time.Now().Unix()))

			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	clog.Info("并发安全性演示完成")
}
