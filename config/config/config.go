package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
)

// ConfigInfo 配置信息
type ConfigInfo struct {
	Env       string      `json:"env"`
	Service   string      `json:"service"`
	Component string      `json:"component"`
	Key       string      `json:"key"`
	Config    interface{} `json:"config"`
}

// 统一的配置管理工具
// 使用方法：
//   go run config.go                    # 初始化所有配置
//   go run config.go dev                # 初始化指定环境的所有配置
//   go run config.go dev im-infra       # 初始化指定环境和服务的所有配置
//   go run config.go dev im-infra clog  # 初始化指定的单个配置

func main() {
	fmt.Println("=== GoChat 配置中心管理工具 ===")

	// 解析命令行参数
	var env, service, component string
	switch len(os.Args) {
	case 1:
		// 初始化所有配置
		fmt.Println("初始化所有环境的所有配置...")
	case 2:
		env = os.Args[1]
		fmt.Printf("初始化环境 '%s' 的所有配置...\n", env)
	case 3:
		env = os.Args[1]
		service = os.Args[2]
		fmt.Printf("初始化环境 '%s' 服务 '%s' 的所有配置...\n", env, service)
	case 4:
		env = os.Args[1]
		service = os.Args[2]
		component = os.Args[3]
		fmt.Printf("初始化配置 %s/%s/%s\n", env, service, component)
	default:
		fmt.Println("Usage:")
		fmt.Println("  go run config.go                    # 初始化所有配置")
		fmt.Println("  go run config.go <env>              # 初始化指定环境的所有配置")
		fmt.Println("  go run config.go <env> <service>    # 初始化指定环境和服务的所有配置")
		fmt.Println("  go run config.go <env> <service> <component>  # 初始化指定的单个配置")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run config.go")
		fmt.Println("  go run config.go dev")
		fmt.Println("  go run config.go prod im-infra")
		fmt.Println("  go run config.go dev im-infra clog")
		os.Exit(1)
	}

	// 扫描配置文件
	configs, err := scanConfigs(env, service, component)
	if err != nil {
		log.Fatalf("Failed to scan configs: %v", err)
	}

	if len(configs) == 0 {
		fmt.Println("没有找到匹配的配置文件")
		return
	}

	fmt.Printf("找到 %d 个配置文件\n", len(configs))

	// 创建协调器连接
	coordinator, err := coord.New()
	if err != nil {
		log.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// 获取配置中心
	configCenter := coordinator.Config()

	// 批量写入配置
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	successCount := 0
	for _, config := range configs {
		err := configCenter.Set(ctx, config.Key, config.Config)
		if err != nil {
			fmt.Printf("❌ 写入失败 %s: %v\n", config.Key, err)
			continue
		}

		// 验证配置是否写入成功
		var retrieved interface{}
		err = configCenter.Get(ctx, config.Key, &retrieved)
		if err != nil {
			fmt.Printf("❌ 验证失败 %s: %v\n", config.Key, err)
			continue
		}

		fmt.Printf("✅ 成功写入 %s\n", config.Key)
		successCount++
	}

	fmt.Printf("\n=== 配置初始化完成 ===\n")
	fmt.Printf("总计: %d 个配置\n", len(configs))
	fmt.Printf("成功: %d 个配置\n", successCount)
	fmt.Printf("失败: %d 个配置\n", len(configs)-successCount)
}

// scanConfigs 扫描配置文件
func scanConfigs(env, service, component string) ([]ConfigInfo, error) {
	var configs []ConfigInfo

	// 遍历配置目录
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理 JSON 文件
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		// 解析路径：env/service/component.json
		parts := strings.Split(path, string(filepath.Separator))
		if len(parts) != 3 {
			return nil
		}

		fileEnv := parts[0]
		fileService := parts[1]
		fileComponent := strings.TrimSuffix(parts[2], ".json")

		// 过滤条件
		if env != "" && fileEnv != env {
			return nil
		}
		if service != "" && fileService != service {
			return nil
		}
		if component != "" && fileComponent != component {
			return nil
		}

		// 读取配置文件
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("⚠️  读取文件失败 %s: %v\n", path, err)
			return nil
		}

		// 解析 JSON
		var config interface{}
		err = json.Unmarshal(data, &config)
		if err != nil {
			fmt.Printf("⚠️  解析 JSON 失败 %s: %v\n", path, err)
			return nil
		}

		// 构建配置信息
		key := fmt.Sprintf("/config/%s/%s/%s", fileEnv, fileService, fileComponent)
		configInfo := ConfigInfo{
			Env:       fileEnv,
			Service:   fileService,
			Component: fileComponent,
			Key:       key,
			Config:    config,
		}

		configs = append(configs, configInfo)
		return nil
	})

	return configs, err
}
