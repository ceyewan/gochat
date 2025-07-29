package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/coord/config"
	"github.com/ceyewan/gochat/im-infra/db"
)

// MyAppConfig 自定义应用配置示例
type MyAppConfig struct {
	AppName     string        `json:"appName"`
	Port        int           `json:"port"`
	Timeout     time.Duration `json:"timeout"`
	EnableDebug bool          `json:"enableDebug"`
}

// myAppConfigValidator 自定义配置验证器
type myAppConfigValidator struct{}

func (v *myAppConfigValidator) Validate(cfg *MyAppConfig) error {
	if cfg.AppName == "" {
		return fmt.Errorf("appName cannot be empty")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}
	return nil
}

// myAppConfigUpdater 自定义配置更新器
type myAppConfigUpdater struct{}

func (u *myAppConfigUpdater) OnConfigUpdate(oldConfig, newConfig *MyAppConfig) error {
	log.Printf("App config updated: %s -> %s", oldConfig.AppName, newConfig.AppName)
	// 这里可以执行配置更新时的自定义逻辑
	return nil
}

// simpleLogger 简单的日志适配器
type simpleLogger struct{}

func (l *simpleLogger) Debug(msg string, fields ...any) { log.Printf("[DEBUG] %s %v", msg, fields) }
func (l *simpleLogger) Info(msg string, fields ...any)  { log.Printf("[INFO] %s %v", msg, fields) }
func (l *simpleLogger) Warn(msg string, fields ...any)  { log.Printf("[WARN] %s %v", msg, fields) }
func (l *simpleLogger) Error(msg string, fields ...any) { log.Printf("[ERROR] %s %v", msg, fields) }

func main() {
	log.Println("=== 通用配置管理器示例 ===")

	// 1. 初始化 coord 实例
	coordInstance, err := coord.New(coord.CoordinatorConfig{
		Endpoints: []string{"localhost:2379"}, // etcd endpoints
		Timeout:   5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to create coord instance: %v", err)
	}

	// 2. 获取配置中心
	configCenter := coordInstance.Config()

	// 3. 示例1：使用新的依赖注入方式创建 clog 配置管理器
	log.Println("\n--- clog 配置管理示例（新的依赖注入方式） ---")

	// 创建 clog 配置管理器
	clogManager := clog.NewConfigManager(configCenter, "dev", "gochat", "clog")
	clogManager.Start()      // 显式启动
	defer clogManager.Stop() // 确保清理

	// 使用 clog
	logger := clog.Module("example")
	logger.Info("clog 配置管理已设置（使用新的依赖注入方式）")

	// 4. 示例2：使用新的依赖注入方式创建 db 配置管理器
	log.Println("\n--- db 配置管理示例（新的依赖注入方式） ---")

	// 创建 db 配置管理器
	dbManager := db.NewConfigManager(configCenter, "dev", "gochat", "db")
	dbManager.Start()      // 显式启动
	defer dbManager.Stop() // 确保清理

	// 使用 db
	database := db.GetDB()
	log.Printf("数据库连接已建立: %v", database != nil)

	// 5. 示例3：向后兼容的全局方式（仍然支持）
	log.Println("\n--- 向后兼容的全局方式 ---")
	clog.SetupConfigCenterFromCoord(configCenter, "dev", "gochat", "clog-global")
	db.SetupConfigCenterFromCoord(configCenter, "dev", "gochat", "db-global")

	// 6. 示例4：自定义应用配置管理（新的生命周期管理）
	log.Println("\n--- 自定义应用配置管理示例 ---")

	// 默认配置
	defaultAppConfig := MyAppConfig{
		AppName:     "gochat",
		Port:        8080,
		Timeout:     30 * time.Second,
		EnableDebug: false,
	}

	// 创建配置管理器（使用新的 API）
	appConfigManager := config.NewManager(
		configCenter,
		"dev", "gochat", "app",
		defaultAppConfig,
		config.WithValidator[MyAppConfig](&myAppConfigValidator{}),
		config.WithUpdater[MyAppConfig](&myAppConfigUpdater{}),
		config.WithLogger[MyAppConfig](&simpleLogger{}),
	)

	// 显式启动配置管理器
	appConfigManager.Start()
	defer appConfigManager.Stop()

	// 获取当前配置
	currentConfig := appConfigManager.GetCurrentConfig()
	log.Printf("当前应用配置: %+v", *currentConfig)

	// 7. 示例5：演示配置热更新
	log.Println("\n--- 配置热更新演示 ---")
	log.Println("可以使用新的 config-cli 工具来更新配置:")
	log.Printf("  config-cli set /config/dev/gochat/app '{\"port\":9090}'")
	log.Println("配置管理器会自动检测并应用更新")

	// 6. 示例4：简单配置管理（无验证器和更新器）
	log.Println("\n--- 简单配置管理示例 ---")

	type SimpleConfig struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	defaultSimpleConfig := SimpleConfig{
		Name:  "default",
		Value: 100,
	}

	simpleConfigManager := config.SimpleManager(
		configCenter,
		"dev", "gochat", "simple",
		defaultSimpleConfig,
		&simpleLogger{},
	)

	simpleConfig := simpleConfigManager.GetCurrentConfig()
	log.Printf("简单配置: %+v", *simpleConfig)

	// 7. 演示配置重载
	log.Println("\n--- 配置重载示例 ---")

	log.Println("重新加载 clog 配置...")
	clog.ReloadConfig()

	log.Println("重新加载 db 配置...")
	db.ReloadConfig()

	log.Println("重新加载应用配置...")
	appConfigManager.ReloadConfig()

	// 8. 清理资源
	log.Println("\n--- 清理资源 ---")

	// 关闭配置管理器
	appConfigManager.Close()
	simpleConfigManager.Close()

	// 关闭 clog 配置管理器
	clog.CloseConfigManager()

	log.Println("示例执行完成")
}
