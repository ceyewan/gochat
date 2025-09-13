package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/coord/config"
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
	return nil
}

// clogUpdater 实现了 config.Updater 接口，用于更新 clog 配置
type clogUpdater struct{}

func (u *clogUpdater) OnConfigUpdate(oldConfig, newConfig *clog.Config) error {
	log.Println("clog config updated, re-initializing logger...")
	return clog.Init(context.Background(), newConfig)
}

func main() {
	log.Println("=== 通用配置管理器示例 ===")

	// 1. 初始化 coord 实例
	coordConfig := coord.GetDefaultConfig("development")
	coordConfig.Endpoints = []string{"localhost:2379"}
	coordInstance, err := coord.New(context.Background(), coordConfig)
	if err != nil {
		log.Fatalf("Failed to create coord instance: %v", err)
	}
	defer coordInstance.Close()

	configCenter := coordInstance.Config()

	// 2. 示例1：管理 clog 配置，支持热更新 (暂时注释掉，因为类型系统较复杂)
	// log.Println("\n--- clog 配置管理示例 ---")
	// clogManager := config.NewManager(
	// 	configCenter, "dev", "gochat", "clog",
	// 	clog.GetDefaultConfig("development"),
	// 	config.WithUpdater[*clog.Config](&clogUpdater{}),
	// 	config.WithLogger[*clog.Config](clog.Namespace("config-manager-clog")),
	// )
	// clogManager.Start()
	// defer clogManager.Stop()
	// clog.Info("clog a info message")

	// 3. 示例2：自定义应用配置管理
	log.Println("\n--- 自定义应用配置管理示例 ---")
	defaultAppConfig := MyAppConfig{AppName: "gochat", Port: 8080}
	appConfigManager := config.NewManager(
		configCenter, "dev", "gochat", "app",
		defaultAppConfig,
		config.WithValidator[MyAppConfig](&myAppConfigValidator{}),
		config.WithUpdater[MyAppConfig](&myAppConfigUpdater{}),
		config.WithLogger[MyAppConfig](clog.Namespace("config-manager-app")),
	)
	appConfigManager.Start()
	defer appConfigManager.Stop()

	currentConfig := appConfigManager.GetCurrentConfig()
	log.Printf("当前应用配置: %+v", *currentConfig)

	// 4. 示例3：演示配置热更新
	log.Println("\n--- 配置热更新演示 ---")
	log.Println("可以使用 config-cli 工具来更新 clog 和 app 的配置, 例如:")
	log.Printf("  ./config-cli sync dev gochat app --force (需要先创建 config/dev/gochat/app.json)")
	log.Println("配置管理器会自动检测并应用更新, 等待几秒查看效果...")
	log.Println("注意: db 包当前不支持热更新。")
	time.Sleep(10 * time.Second)

	log.Println("示例执行完成")
}
