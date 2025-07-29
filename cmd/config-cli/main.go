package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
		Short: "A powerful configuration management CLI tool",
		Long: `config-cli is a safe and feature-rich configuration management tool
that supports deep merging, atomic updates, and various operations.`,
	}

	// 全局标志
	rootCmd.PersistentFlags().StringSliceVar(&endpoints, "endpoints", []string{"localhost:2379"}, "etcd endpoints")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "etcd username")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "etcd password")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "operation timeout")

	// 添加子命令
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(setCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(replaceCmd())
	rootCmd.AddCommand(watchCmd())
	rootCmd.AddCommand(listCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// createCoordinator 创建协调器实例
func createCoordinator() (coord.Provider, error) {
	config := coord.CoordinatorConfig{
		Endpoints: endpoints,
		Username:  username,
		Password:  password,
		Timeout:   timeout,
	}

	return coord.New(config)
}

// initCmd 初始化配置命令
func initCmd() *cobra.Command {
	var configPath string
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "init [env] [service] [component]",
		Short: "Initialize configurations from local files to config center",
		Long: `Initialize configurations by scanning local JSON files and uploading them to the config center.
File structure should be: <config-path>/{env}/{service}/{component}.json

Examples:
  config-cli init                           # Initialize all configurations
  config-cli init dev                       # Initialize all dev environment configs
  config-cli init dev im-infra              # Initialize all dev/im-infra configs
  config-cli init dev im-infra clog         # Initialize specific config`,
		Args: cobra.MaximumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 解析参数
			var env, service, component string
			if len(args) >= 1 {
				env = args[0]
			}
			if len(args) >= 2 {
				service = args[1]
			}
			if len(args) >= 3 {
				component = args[2]
			}

			// 验证配置目录结构
			if err := ValidateConfigStructure(configPath); err != nil {
				return fmt.Errorf("配置目录验证失败: %w", err)
			}

			// 扫描配置文件
			configs, err := ScanConfigs(configPath, env, service, component)
			if err != nil {
				return fmt.Errorf("扫描配置文件失败: %w", err)
			}

			if len(configs) == 0 {
				fmt.Println("没有找到匹配的配置文件")
				return nil
			}

			// 显示配置摘要
			PrintConfigSummary(configs)

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
			coordinator, err := createCoordinator()
			if err != nil {
				return fmt.Errorf("创建协调器失败: %w", err)
			}
			defer coordinator.Close()

			configCenter := coordinator.Config()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// 批量写入配置
			return batchWriteConfigs(ctx, configCenter, configs)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config-path", "c", "../../config", "配置文件根目录路径")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "干运行模式，只显示将要执行的操作")
	cmd.Flags().BoolVar(&force, "force", false, "强制执行，不询问确认")

	return cmd
}

// getCmd 获取配置命令
func getCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			coordinator, err := createCoordinator()
			if err != nil {
				return fmt.Errorf("failed to create coordinator: %w", err)
			}
			defer coordinator.Close()

			configCenter := coordinator.Config()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			var value map[string]interface{}
			err = configCenter.Get(ctx, key, &value)
			if err != nil {
				return fmt.Errorf("failed to get config: %w", err)
			}

			// 格式化输出
			output, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	return cmd
}

// setCmd 设置配置命令（深度合并）
func setCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <json_value>",
		Short: "Set configuration value with deep merge",
		Long: `Set configuration value using deep merge strategy.
Only the specified fields will be updated, existing fields will be preserved.
Uses Compare-And-Set for atomic updates.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			jsonValue := args[1]

			// 解析 JSON 更新值
			var updateValue map[string]interface{}
			if err := json.Unmarshal([]byte(jsonValue), &updateValue); err != nil {
				return fmt.Errorf("invalid JSON value: %w", err)
			}

			coordinator, err := createCoordinator()
			if err != nil {
				return fmt.Errorf("failed to create coordinator: %w", err)
			}
			defer coordinator.Close()

			configCenter := coordinator.Config()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// 使用 CAS 进行原子更新
			return updateWithCAS(ctx, configCenter, key, updateValue, deepMerge)
		},
	}

	return cmd
}

// deleteCmd 删除配置字段命令
func deleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <key> <field_path>",
		Short: "Delete a specific field from configuration",
		Long: `Delete a specific field from configuration using dot notation.
Example: delete /config/dev/app field1.subfield2`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			fieldPath := args[1]

			coordinator, err := createCoordinator()
			if err != nil {
				return fmt.Errorf("failed to create coordinator: %w", err)
			}
			defer coordinator.Close()

			configCenter := coordinator.Config()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// 使用 CAS 进行原子删除
			return updateWithCAS(ctx, configCenter, key, fieldPath, deleteField)
		},
	}

	return cmd
}

// replaceCmd 完全替换配置命令
func replaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replace <key> <json_value>",
		Short: "Replace entire configuration",
		Long: `Replace the entire configuration with the provided JSON value.
This will completely overwrite the existing configuration.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			jsonValue := args[1]

			// 解析 JSON 值
			var newValue map[string]interface{}
			if err := json.Unmarshal([]byte(jsonValue), &newValue); err != nil {
				return fmt.Errorf("invalid JSON value: %w", err)
			}

			coordinator, err := createCoordinator()
			if err != nil {
				return fmt.Errorf("failed to create coordinator: %w", err)
			}
			defer coordinator.Close()

			configCenter := coordinator.Config()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// 直接设置新值
			if err := configCenter.Set(ctx, key, newValue); err != nil {
				return fmt.Errorf("failed to replace config: %w", err)
			}

			fmt.Printf("✅ Configuration replaced successfully: %s\n", key)
			return nil
		},
	}

	return cmd
}

// watchCmd 监听配置变化命令
func watchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <key>",
		Short: "Watch configuration changes in real-time",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			coordinator, err := createCoordinator()
			if err != nil {
				return fmt.Errorf("failed to create coordinator: %w", err)
			}
			defer coordinator.Close()

			configCenter := coordinator.Config()
			ctx := context.Background()

			var value map[string]interface{}
			watcher, err := configCenter.Watch(ctx, key, &value)
			if err != nil {
				return fmt.Errorf("failed to create watcher: %w", err)
			}
			defer watcher.Close()

			fmt.Printf("👀 Watching configuration changes for: %s\n", key)
			fmt.Println("Press Ctrl+C to stop...")

			for {
				select {
				case event, ok := <-watcher.Chan():
					if !ok {
						fmt.Println("Watcher channel closed")
						return nil
					}

					fmt.Printf("\n🔄 Configuration changed [%s]: %s\n", event.Type, event.Key)
					if output, err := json.MarshalIndent(event.Value, "", "  "); err == nil {
						fmt.Println(string(output))
					}

				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
	}

	return cmd
}

// listCmd 列出配置键命令
func listCmd() *cobra.Command {
	var detailed bool
	var format string

	cmd := &cobra.Command{
		Use:   "list [env] [service]",
		Short: "List configuration keys",
		Long: `List configuration keys in the config center.

Examples:
  config-cli list                    # List all configurations
  config-cli list dev                # List all dev environment configs
  config-cli list dev im-infra       # List all dev/im-infra configs
  config-cli list --detailed         # Show detailed information`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var env, service string
			if len(args) >= 1 {
				env = args[0]
			}
			if len(args) >= 2 {
				service = args[1]
			}

			coordinator, err := createCoordinator()
			if err != nil {
				return fmt.Errorf("failed to create coordinator: %w", err)
			}
			defer coordinator.Close()

			configCenter := coordinator.Config()
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// 构建前缀
			prefix := GetConfigPrefix(env, service)

			keys, err := configCenter.List(ctx, prefix)
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}

			if len(keys) == 0 {
				fmt.Printf("没有找到匹配前缀 '%s' 的配置\n", prefix)
				return nil
			}

			// 根据格式显示结果
			switch format {
			case "json":
				return printKeysAsJSON(keys)
			case "table":
				return printKeysAsTable(keys, detailed)
			default:
				return printKeysAsTree(keys, detailed, configCenter, ctx)
			}
		},
	}

	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "显示详细信息")
	cmd.Flags().StringVarP(&format, "format", "f", "tree", "输出格式 (tree|table|json)")

	return cmd
}
