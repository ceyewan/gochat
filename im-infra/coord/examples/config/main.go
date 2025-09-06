package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/coord/config"
)

// AppConfig is a sample struct for configuration.
type AppConfig struct {
	AppName    string `json:"app_name"`
	Version    string `json:"version"`
	MaxConns   int    `json:"max_conns"`
	EnableTLS  bool   `json:"enable_tls"`
	LogLevel   string `json:"log_level"`
	TimeoutSec int    `json:"timeout_sec"`
}

func main() {
	// clog 包是零配置的，不需要显式初始化
	cfg := coord.DefaultConfig()
	provider, err := coord.New(context.Background(), cfg)
	if err != nil {
		clog.Error("failed to create coordinator", clog.Err(err))
		os.Exit(1)
	}
	defer provider.Close()

	configCenter := provider.Config()
	ctx := context.Background()

	// 1. 设置和获取基本类型配置
	testBasicTypes(ctx, configCenter)

	// 2. 设置和获取结构体配置
	testStructType(ctx, configCenter)

	// 3. 演示 Watch 和 WatchPrefix
	testWatch(ctx, configCenter)

	fmt.Println("\nConfig example finished.")
}

func testBasicTypes(ctx context.Context, cc config.ConfigCenter) {
	clog.Info("--- Testing Basic Types ---")
	key := "app/log_level"
	value := "debug"

	// Set
	if err := cc.Set(ctx, key, value); err != nil {
		clog.Error("Failed to set log_level", clog.Err(err))
		return
	}
	clog.Info("Set config successfully", clog.String("key", key), clog.String("value", value))

	// Get
	var logLevel string
	if err := cc.Get(ctx, key, &logLevel); err != nil {
		clog.Error("Failed to get log_level", clog.Err(err))
		return
	}
	clog.Info("Get config successfully", clog.String("key", key), clog.String("retrieved_value", logLevel))

	if logLevel != value {
		clog.Error("Value mismatch!", clog.String("expected", value), clog.String("got", logLevel))
	}
}

func testStructType(ctx context.Context, cc config.ConfigCenter) {
	clog.Info("--- Testing Struct Type ---")
	key := "app/settings"
	appCfg := AppConfig{
		AppName:    "GoChat",
		Version:    "1.2.3",
		MaxConns:   1024,
		EnableTLS:  true,
		LogLevel:   "info",
		TimeoutSec: 30,
	}

	// Set
	if err := cc.Set(ctx, key, &appCfg); err != nil {
		clog.Error("Failed to set app config", clog.Err(err))
		return
	}
	clog.Info("Set struct config successfully", clog.String("key", key))

	// Get
	var retrievedCfg AppConfig
	if err := cc.Get(ctx, key, &retrievedCfg); err != nil {
		clog.Error("Failed to get app config", clog.Err(err))
		return
	}
	clog.Info("Get struct config successfully", clog.String("key", key), clog.Any("value", retrievedCfg))

	if retrievedCfg.AppName != appCfg.AppName {
		clog.Error("Struct value mismatch!")
	}
}

func testWatch(ctx context.Context, cc config.ConfigCenter) {
	clog.Info("--- Testing Watch ---")
	var wg sync.WaitGroup
	wg.Add(1)

	prefix := "notifications/email"
	key1 := prefix + "/server"
	key2 := prefix + "/port"

	// Watcher goroutine
	go func() {
		defer wg.Done()
		// 监听 "notifications/" 前缀下的所有变化
		// 使用 interface{} 来处理不同类型的配置值
		var watchValue interface{}
		watcher, err := cc.WatchPrefix(ctx, "notifications", &watchValue)
		if err != nil {
			clog.Error("Failed to start watcher", clog.Err(err))
			return
		}
		defer watcher.Close()
		clog.Info("Watcher started on prefix", clog.String("prefix", "notifications/"))

		timeout := time.After(5 * time.Second)
		for i := 0; i < 4; i++ { // 等待 4 个事件
			select {
			case event := <-watcher.Chan():
				clog.Info("Received config event",
					clog.String("type", string(event.Type)),
					clog.String("key", event.Key),
					clog.Any("value", event.Value),
				)
			case <-timeout:
				clog.Error("Watcher timed out")
				return
			}
		}
	}()

	// 等待 watcher 启动
	time.Sleep(1 * time.Second)

	// 触发事件
	clog.Info("Setting initial values to trigger watch events...")
	_ = cc.Set(ctx, key1, "smtp.example.com")
	_ = cc.Set(ctx, key2, 587)

	clog.Info("Updating value to trigger watch event...")
	_ = cc.Set(ctx, key2, 465) // Update

	clog.Info("Deleting value to trigger watch event...")
	_ = cc.Delete(ctx, key1) // Delete

	wg.Wait()
}
