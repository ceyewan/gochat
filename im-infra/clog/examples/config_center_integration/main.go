package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
)

// 演示 clog 与配置中心的集成，包括两阶段启动和配置热更新
func main() {
	fmt.Println("=== clog 配置中心集成示例 ===")

	// 阶段一：降级启动 - 使用默认配置确保基础日志功能可用
	fmt.Println("阶段一：降级启动...")
	clog.Info("应用启动 - 使用默认配置", clog.String("stage", "fallback"))

	// 创建协调器（这时 clog 已经可以工作）
	coordinator, err := coord.New()
	if err != nil {
		log.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// 阶段二：配置中心集成 - 从配置中心获取配置并热更新
	fmt.Println("阶段二：配置中心集成...")

	// 设置配置中心作为 clog 的配置源
	clog.SetupConfigCenterFromCoord(coordinator.Config(), "dev", "im-infra", "clog")

	// 重新初始化 clog，这次会从配置中心读取配置
	err = clog.Init()
	if err != nil {
		// 如果配置中心不可用，会继续使用当前配置，不会中断服务
		clog.Warn("配置中心不可用，继续使用当前配置", clog.Err(err))
	} else {
		clog.Info("配置中心集成成功", clog.String("stage", "config-center"))
	}

	// 演示不同的日志使用方式
	demonstrateLogging()

	// 演示配置热更新
	demonstrateConfigHotUpdate(coordinator)

	fmt.Println("示例完成")
}

func demonstrateLogging() {
	fmt.Println("\n--- 演示不同的日志使用方式 ---")

	// 1. 全局日志
	clog.Info("全局日志示例", clog.String("type", "global"))

	// 2. 模块日志
	userModule := clog.Module("user")
	userModule.Info("用户模块日志", clog.String("userID", "123"))

	orderModule := clog.Module("order")
	orderModule.Info("订单模块日志", clog.String("orderID", "456"))

	// 3. 带 Context 的日志（自动 TraceID）
	ctx := context.WithValue(context.Background(), "traceID", "trace-abc-123")
	clog.C(ctx).Info("带 TraceID 的日志", clog.String("action", "process_request"))

	// 4. 依赖注入模式（推荐）
	service := NewUserService()
	service.ProcessUser("john_doe")
}

func demonstrateConfigHotUpdate(coordinator coord.Provider) {
	fmt.Println("\n--- 演示配置热更新 ---")

	ctx := context.Background()
	configCenter := coordinator.Config()

	// 当前配置
	currentConfig := clog.GetCurrentConfig()
	fmt.Printf("当前配置: Level=%s, Format=%s, AddSource=%t\n",
		currentConfig.Level, currentConfig.Format, currentConfig.AddSource)

	// 记录一条当前配置的日志
	clog.Info("使用当前配置记录日志", clog.String("stage", "before-update"))

	// 模拟配置更新 - 确保与当前配置不同
	var newConfig clog.Config
	if currentConfig.Format == "json" {
		// 如果当前是 JSON 格式，切换到 console 格式
		newConfig = clog.Config{
			Level:       "warn",    // 从 debug 改为 warn
			Format:      "console", // 从 json 改为 console
			Output:      "stdout",
			AddSource:   false, // 从 true 改为 false
			EnableColor: true,  // 启用颜色
			RootPath:    "gochat",
		}
	} else {
		// 如果当前是 console 格式，切换到 JSON 格式
		newConfig = clog.Config{
			Level:       "error", // 改为 error 级别
			Format:      "json",  // 改为 json 格式
			Output:      "stdout",
			AddSource:   true,
			EnableColor: false,
			RootPath:    "gochat",
		}
	}

	fmt.Printf("准备更新到: Level=%s, Format=%s, AddSource=%t\n",
		newConfig.Level, newConfig.Format, newConfig.AddSource)

	// 更新配置到配置中心
	key := "/config/dev/im-infra/clog"
	err := configCenter.Set(ctx, key, newConfig)
	if err != nil {
		clog.Error("更新配置失败", clog.Err(err))
		return
	}

	clog.Info("配置已更新到配置中心", clog.String("key", key))

	// 等待配置热更新生效
	fmt.Println("等待配置热更新生效...")
	time.Sleep(3 * time.Second)

	// 验证配置是否已更新
	updatedConfig := clog.GetCurrentConfig()
	fmt.Printf("更新后配置: Level=%s, Format=%s, AddSource=%t\n",
		updatedConfig.Level, updatedConfig.Format, updatedConfig.AddSource)

	// 验证配置是否真的发生了变化
	if updatedConfig.Level != newConfig.Level || updatedConfig.Format != newConfig.Format {
		fmt.Printf("❌ 配置更新失败！期望: Level=%s, Format=%s，实际: Level=%s, Format=%s\n",
			newConfig.Level, newConfig.Format, updatedConfig.Level, updatedConfig.Format)
	} else {
		fmt.Printf("✅ 配置更新成功！\n")
	}

	// 使用新配置记录不同级别的日志来验证
	clog.Debug("这是 DEBUG 级别日志", clog.String("config", "updated"))
	clog.Info("这是 INFO 级别日志", clog.String("config", "updated"))
	clog.Warn("这是 WARN 级别日志", clog.String("config", "updated"))
	clog.Error("这是 ERROR 级别日志", clog.String("config", "updated"))
}

// UserService 演示依赖注入模式
type UserService struct {
	logger clog.Logger
}

func NewUserService() *UserService {
	// 创建独立的 logger 实例，推荐用于生产环境
	logger, err := clog.New()
	if err != nil {
		// 如果创建失败，使用模块 logger 作为备选
		logger = clog.Module("user-service")
	}

	return &UserService{
		logger: logger,
	}
}

func (s *UserService) ProcessUser(userID string) {
	s.logger.Info("处理用户请求",
		clog.String("userID", userID),
		clog.String("service", "user-service"))

	// 模拟一些处理逻辑
	s.logger.Debug("验证用户权限", clog.String("userID", userID))
	s.logger.Debug("加载用户数据", clog.String("userID", userID))
	s.logger.Info("用户处理完成", clog.String("userID", userID))
}
