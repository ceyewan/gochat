package main

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	fmt.Println("=== clog Options 模式示例 ===")

	// 示例 1: 使用 WithNamespace 选项初始化全局 logger
	fmt.Println("\n--- 示例 1: 使用 WithNamespace 初始化全局 logger ---")
	config1 := clog.GetDefaultConfig("production")
	
	if err := clog.Init(context.Background(), config1, clog.WithNamespace("im-gateway")); err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		return
	}

	clog.Info("全局 logger 初始化成功")
	clog.Namespace("user").Info("用户模块消息")
	clog.Namespace("payment").Namespace("processor").Info("支付处理器消息")

	// 示例 2: 创建独立的 logger 实例，使用多个选项
	fmt.Println("\n--- 示例 2: 创建独立的 logger 实例 ---")
	config2 := &clog.Config{
		Level:     "debug",
		Format:    "json",
		Output:    "stdout",
		AddSource: true,  // 确保 JSON 格式显示源码信息
	}

	logger, err := clog.New(context.Background(), config2, 
		clog.WithNamespace("order-service"),
	)
	if err != nil {
		fmt.Printf("创建 logger 失败: %v\n", err)
		return
	}

	logger.Info("独立 logger 创建成功")
	logger.Namespace("database").Info("数据库操作")
	logger.Namespace("cache").Warn("缓存警告")

	// 示例 3: 上下文结合新的 options 模式
	fmt.Println("\n--- 示例 3: 上下文与 options 结合 ---")
	ctx := clog.WithTraceID(context.Background(), "example-trace-12345")
	
	// 使用全局 logger 的上下文感知
	clog.WithContext(ctx).Info("带 traceID 的日志消息")
	clog.WithContext(ctx).Namespace("auth").Info("认证模块消息")

	// 使用独立 logger 的上下文感知（通过全局函数）
	clog.WithContext(ctx).Info("通过全局函数获取的上下文 logger")
	clog.WithContext(ctx).Namespace("payment").Info("支付处理模块")

	// 示例 4: 链式调用与命名空间
	fmt.Println("\n--- 示例 4: 链式命名空间调用 ---")
	
	// 从全局 logger 开始
	baseLogger := clog.Namespace("api")
	userLogger := baseLogger.Namespace("v1").Namespace("users")
	orderLogger := baseLogger.Namespace("v1").Namespace("orders")

	userLogger.Info("用户 API 调用")
	orderLogger.Warn("订单 API 警告")

	// 从独立 logger 开始
	serviceLogger := logger.Namespace("service")
	dbLogger := serviceLogger.Namespace("database")
	cacheLogger := serviceLogger.Namespace("cache")

	dbLogger.Info("数据库连接")
	cacheLogger.Debug("缓存查询")

	fmt.Println("\n=== 所有 options 示例演示完成 ===")
}