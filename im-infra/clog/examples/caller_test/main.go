package main

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	// 测试 1: 直接调用全局方法
	clog.Info("直接调用全局方法") // 应该显示这一行的位置

	// 测试 2: 模块日志
	userModule := clog.Module("user")
	userModule.Info("模块日志") // 应该显示这一行的位置

	// 测试 3: Context 日志
	ctx := context.WithValue(context.Background(), "traceID", "test-123")
	clog.C(ctx).Info("Context 日志") // 应该显示这一行的位置

	// 测试 4: 链式调用
	clog.C(ctx).Module("order").Info("链式调用") // 应该显示这一行的位置

	// 测试 5: 自定义 logger
	customLogger := clog.New()
	customLogger.Info("自定义 logger") // 应该显示这一行的位置
}
