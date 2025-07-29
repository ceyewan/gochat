package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
)

// 配置更新工具
// 使用方法：go run update.go <env> <service> <component> <json_config>
// 例如：go run update.go dev im-infra clog '{"level":"debug","format":"json"}'

func main() {
	// 解析命令行参数
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run update.go <env> <service> <component> <json_config>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println(`  go run update.go dev im-infra clog '{"level":"debug","format":"json"}'`)
		fmt.Println(`  go run update.go prod im-infra cache '{"addr":"redis-cluster:6379","poolSize":100}'`)
		fmt.Println()
		fmt.Println("Note: The json_config should be a valid JSON string that will be merged with existing config")
		os.Exit(1)
	}

	env := os.Args[1]
	service := os.Args[2]
	component := os.Args[3]
	jsonConfig := os.Args[4]

	fmt.Printf("=== 更新配置 %s/%s/%s ===\n", env, service, component)

	// 解析 JSON 配置
	var updateConfig map[string]interface{}
	err := json.Unmarshal([]byte(jsonConfig), &updateConfig)
	if err != nil {
		log.Fatalf("Failed to parse JSON config: %v", err)
	}

	fmt.Printf("更新内容: %s\n", jsonConfig)

	// 创建协调器连接
	coordinator, err := coord.New()
	if err != nil {
		log.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// 获取配置中心
	configCenter := coordinator.Config()

	// 构建配置键
	key := fmt.Sprintf("/config/%s/%s/%s", env, service, component)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取现有配置
	var existingConfig map[string]interface{}
	err = configCenter.Get(ctx, key, &existingConfig)
	if err != nil {
		fmt.Printf("⚠️  获取现有配置失败，将创建新配置: %v\n", err)
		existingConfig = make(map[string]interface{})
	} else {
		fmt.Println("✅ 获取现有配置成功")
	}

	// 合并配置
	for k, v := range updateConfig {
		existingConfig[k] = v
	}

	// 更新配置到配置中心
	err = configCenter.Set(ctx, key, existingConfig)
	if err != nil {
		log.Fatalf("Failed to update config in config center: %v", err)
	}

	fmt.Printf("✅ 成功更新配置到配置中心: %s\n", key)

	// 验证配置是否更新成功
	var retrievedConfig map[string]interface{}
	err = configCenter.Get(ctx, key, &retrievedConfig)
	if err != nil {
		log.Fatalf("Failed to retrieve config for verification: %v", err)
	}

	// 显示更新后的配置
	updatedJSON, err := json.MarshalIndent(retrievedConfig, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal updated config: %v", err)
	}

	fmt.Println("\n=== 更新后的完整配置 ===")
	fmt.Println(string(updatedJSON))

	fmt.Println("\n✅ 配置更新完成！")
	fmt.Println("任何使用此配置的运行中应用程序将自动更新。")
}
