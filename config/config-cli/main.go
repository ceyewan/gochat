package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/spf13/cobra"
)

var (
	// 全局配置
	endpoints []string
	username  string
	password  string
	timeout   time.Duration
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "config-cli",
		Short: "简化的配置管理工具",
		Long:  `config-cli 是一个简化的配置管理工具，专注于将 JSON 配置文件原子地写入 etcd 配置中心。`,
	}

	// 全局标志
	rootCmd.PersistentFlags().StringSliceVar(&endpoints, "endpoints", []string{"localhost:2379"}, "etcd endpoints")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "etcd username")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "etcd password")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "operation timeout")

	// 添加子命令
	rootCmd.AddCommand(syncCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// createCoordinator 创建协调器实例
func createCoordinator(ctx context.Context) (coord.Provider, error) {
	config := coord.CoordinatorConfig{
		Endpoints: endpoints,
		Username:  username,
		Password:  password,
		Timeout:   timeout,
	}

	return coord.New(ctx, config)
}

// syncCmd 同步配置命令 - 核心功能
func syncCmd() *cobra.Command {
	var configPath string
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "sync [env]",
		Short: "将 JSON 配置文件同步到 etcd 配置中心",
		Long: `将本地 JSON 配置文件原子地写入 etcd 配置中心。
支持同步所有配置或指定环境的配置。

示例:
		config-cli sync          # 同步所有环境的配置
		config-cli sync dev      # 同步 'dev' 环境的所有配置`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 解析参数
			var env string
			if len(args) >= 1 {
				env = args[0]
			}

			// 扫描配置文件
			configs, err := scanConfigs(configPath, env, "", "")
			if err != nil {
				return fmt.Errorf("扫描配置文件失败: %w", err)
			}

			if len(configs) == 0 {
				fmt.Println("没有找到匹配的配置文件")
				return nil
			}

			// 显示配置摘要
			printConfigSummary(configs)

			if dryRun {
				fmt.Println("\n🔍 干运行模式：不会实际写入配置中心")
				return nil
			}

			// 确认操作
			if !force {
				fmt.Printf("\n❓ 确定要将 %d 个配置写入配置中心吗？(y/N): ", len(configs))
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("操作已取消")
					return nil
				}
			}

			// 创建协调器连接
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			coordinator, err := createCoordinator(ctx)
			if err != nil {
				return fmt.Errorf("创建协调器失败: %w", err)
			}
			defer coordinator.Close()

			// 批量写入配置
			return writeConfigs(ctx, coordinator, configs)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config-path", "c", "..", "配置文件根目录路径")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "干运行模式，只显示将要执行的操作")
	cmd.Flags().BoolVar(&force, "force", false, "强制执行，不询问确认")

	return cmd
}

// ConfigInfo 配置信息结构
type ConfigInfo struct {
	Env       string `json:"env"`
	Service   string `json:"service"`
	Component string `json:"component"`
	Key       string `json:"key"`
	Config    []byte `json:"config"`
	FilePath  string `json:"file_path"`
}

// scanConfigs 扫描配置文件
func scanConfigs(basePath, env, service, component string) ([]ConfigInfo, error) {
	var configs []ConfigInfo

	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理 JSON 文件
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// 解析路径获取环境、服务、组件信息
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		// 路径至少需要是 {env}/{service}/{component}.json 的形式
		if len(parts) < 3 {
			return nil
		}

		fileEnv := parts[0]
		fileService := parts[1]
		fileComponent := strings.TrimSuffix(parts[2], ".json")

		// 如果路径是 {env}/global/{component}.json，则服务名是 "global"
		// 否则，服务名是路径的第二部分
		if fileService == "global" {
			// 允许 global 目录下的任何组件
		}

		// 应用过滤条件
		if env != "" && fileEnv != env {
			return nil
		}
		// service 和 component 参数已移除，不再需要过滤
		// if service != "" && fileService != service {
		// 	return nil
		// }
		// if component != "" && fileComponent != component {
		// 	return nil
		// }

		// 读取配置文件
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
		}

		// 构建 etcd 键
		key := fmt.Sprintf("/config/%s/%s/%s", fileEnv, fileService, fileComponent)

		configs = append(configs, ConfigInfo{
			Env:       fileEnv,
			Service:   fileService,
			Component: fileComponent,
			Key:       key,
			Config:    data,
			FilePath:  path,
		})

		return nil
	})

	return configs, err
}

// printConfigSummary 打印配置摘要
func printConfigSummary(configs []ConfigInfo) {
	fmt.Printf("📋 找到 %d 个配置文件:\n\n", len(configs))

	// 按环境分组显示
	envGroups := make(map[string][]ConfigInfo)
	for _, config := range configs {
		envGroups[config.Env] = append(envGroups[config.Env], config)
	}

	for env, envConfigs := range envGroups {
		fmt.Printf("🌍 环境: %s\n", env)

		// 按服务分组
		serviceGroups := make(map[string][]ConfigInfo)
		for _, config := range envConfigs {
			serviceGroups[config.Service] = append(serviceGroups[config.Service], config)
		}

		for service, serviceConfigs := range serviceGroups {
			fmt.Printf("  📦 服务: %s\n", service)
			for _, config := range serviceConfigs {
				fmt.Printf("    ⚙️  %s -> %s\n", config.Component, config.Key)
			}
		}
		fmt.Println()
	}
}

// writeConfigs 批量写入配置
func writeConfigs(ctx context.Context, coordinator coord.Provider, configs []ConfigInfo) error {
	successCount := 0
	errorCount := 0

	fmt.Println("🚀 开始写入配置...")
	configCenter := coordinator.Config()
	for _, config := range configs {
		fmt.Printf("📝 写入: %s ... ", config.Key)

		// 原子地写入配置（覆盖模式）
		if err := configCenter.Set(ctx, config.Key, config.Config); err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
			errorCount++
			continue
		}

		fmt.Println("✅ 成功")
		successCount++
	}

	fmt.Printf("\n📊 写入完成: %d 成功, %d 失败\n", successCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("有 %d 个配置写入失败", errorCount)
	}

	return nil
}
