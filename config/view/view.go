package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
)

// 配置查看工具
// 使用方法：
//   go run view.go                      # 列出所有配置
//   go run view.go dev                  # 列出指定环境的所有配置
//   go run view.go dev im-infra         # 列出指定环境和服务的所有配置
//   go run view.go dev im-infra clog    # 查看指定配置的详细内容

func main() {
	fmt.Println("=== GoChat 配置查看工具 ===")

	// 解析命令行参数
	var env, service, component string
	switch len(os.Args) {
	case 1:
		fmt.Println("列出所有配置...")
	case 2:
		env = os.Args[1]
		fmt.Printf("列出环境 '%s' 的所有配置...\n", env)
	case 3:
		env = os.Args[1]
		service = os.Args[2]
		fmt.Printf("列出环境 '%s' 服务 '%s' 的所有配置...\n", env, service)
	case 4:
		env = os.Args[1]
		service = os.Args[2]
		component = os.Args[3]
		fmt.Printf("查看配置 %s/%s/%s 的详细内容...\n", env, service, component)
	default:
		fmt.Println("Usage:")
		fmt.Println("  go run view.go                      # 列出所有配置")
		fmt.Println("  go run view.go <env>                # 列出指定环境的所有配置")
		fmt.Println("  go run view.go <env> <service>      # 列出指定环境和服务的所有配置")
		fmt.Println("  go run view.go <env> <service> <component>  # 查看指定配置的详细内容")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run view.go")
		fmt.Println("  go run view.go dev")
		fmt.Println("  go run view.go prod im-infra")
		fmt.Println("  go run view.go dev im-infra clog")
		os.Exit(1)
	}

	// 创建协调器连接
	coordinator, err := coord.New()
	if err != nil {
		log.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// 获取配置中心
	configCenter := coordinator.Config()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if component != "" {
		// 查看具体配置的详细内容
		viewSpecificConfig(ctx, configCenter, env, service, component)
	} else {
		// 列出配置
		listConfigs(ctx, configCenter, env, service)
	}
}

// viewSpecificConfig 查看具体配置的详细内容
func viewSpecificConfig(ctx context.Context, configCenter interface{}, env, service, component string) {
	key := fmt.Sprintf("/config/%s/%s/%s", env, service, component)

	// 获取配置
	var config interface{}
	err := configCenter.(interface {
		Get(ctx context.Context, key string, v interface{}) error
	}).Get(ctx, key, &config)

	if err != nil {
		fmt.Printf("❌ 获取配置失败: %v\n", err)
		return
	}

	// 格式化输出
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("❌ 格式化配置失败: %v\n", err)
		return
	}

	fmt.Printf("\n=== 配置详情: %s ===\n", key)
	fmt.Println(string(configJSON))
}

// listConfigs 列出配置
func listConfigs(ctx context.Context, configCenter interface{}, env, service string) {
	// 构建前缀
	var prefix string
	if env == "" {
		prefix = "/config/"
	} else if service == "" {
		prefix = fmt.Sprintf("/config/%s/", env)
	} else {
		prefix = fmt.Sprintf("/config/%s/%s/", env, service)
	}

	// 列出配置键
	keys, err := configCenter.(interface {
		List(ctx context.Context, prefix string) ([]string, error)
	}).List(ctx, prefix)

	if err != nil {
		fmt.Printf("❌ 列出配置失败: %v\n", err)
		return
	}

	if len(keys) == 0 {
		fmt.Println("没有找到匹配的配置")
		return
	}

	fmt.Printf("\n找到 %d 个配置:\n", len(keys))
	fmt.Println(strings.Repeat("=", 60))

	for _, key := range keys {
		// 解析配置键
		parts := strings.Split(strings.TrimPrefix(key, "/config/"), "/")
		if len(parts) >= 3 {
			fmt.Printf("📁 %s/%s/%s\n", parts[0], parts[1], parts[2])
			fmt.Printf("   键: %s\n", key)
		} else {
			fmt.Printf("📁 %s\n", key)
		}
		fmt.Println(strings.Repeat("-", 40))
	}

	fmt.Printf("\n提示: 使用 'go run view.go <env> <service> <component>' 查看具体配置内容\n")
}
