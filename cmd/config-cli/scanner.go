package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ConfigInfo 配置信息
type ConfigInfo struct {
	Env       string      `json:"env"`
	Service   string      `json:"service"`
	Component string      `json:"component"`
	Key       string      `json:"key"`
	Config    interface{} `json:"config"`
	FilePath  string      `json:"file_path"`
}

// ScanConfigs 扫描配置文件
// basePath: 配置文件根目录路径
// env, service, component: 过滤条件，空字符串表示不过滤
func ScanConfigs(basePath, env, service, component string) ([]ConfigInfo, error) {
	var configs []ConfigInfo

	// 确保基础路径存在
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置目录不存在: %s", basePath)
	}

	// 遍历配置目录
	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理 JSON 文件
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		// 解析路径：env/service/component.json
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) != 3 {
			fmt.Printf("⚠️  跳过格式不正确的文件: %s (期望格式: env/service/component.json)\n", relPath)
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
			fmt.Printf("⚠️  读取文件失败 %s: %v\n", relPath, err)
			return nil
		}

		// 解析 JSON
		var config interface{}
		err = json.Unmarshal(data, &config)
		if err != nil {
			fmt.Printf("⚠️  解析 JSON 失败 %s: %v\n", relPath, err)
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
			FilePath:  relPath,
		}

		configs = append(configs, configInfo)
		return nil
	})

	return configs, err
}

// ValidateConfigStructure 验证配置目录结构
func ValidateConfigStructure(basePath string) error {
	// 检查基础目录是否存在
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return fmt.Errorf("配置目录不存在: %s", basePath)
	}

	// 检查是否有有效的配置文件
	configs, err := ScanConfigs(basePath, "", "", "")
	if err != nil {
		return fmt.Errorf("扫描配置文件失败: %w", err)
	}

	if len(configs) == 0 {
		return fmt.Errorf("在 %s 目录下没有找到有效的配置文件", basePath)
	}

	return nil
}

// GroupConfigsByEnv 按环境分组配置
func GroupConfigsByEnv(configs []ConfigInfo) map[string][]ConfigInfo {
	groups := make(map[string][]ConfigInfo)
	for _, config := range configs {
		groups[config.Env] = append(groups[config.Env], config)
	}
	return groups
}

// GroupConfigsByService 按服务分组配置
func GroupConfigsByService(configs []ConfigInfo) map[string][]ConfigInfo {
	groups := make(map[string][]ConfigInfo)
	for _, config := range configs {
		key := fmt.Sprintf("%s/%s", config.Env, config.Service)
		groups[key] = append(groups[key], config)
	}
	return groups
}

// PrintConfigSummary 打印配置摘要
func PrintConfigSummary(configs []ConfigInfo) {
	if len(configs) == 0 {
		fmt.Println("没有找到配置文件")
		return
	}

	fmt.Printf("📋 找到 %d 个配置文件:\n", len(configs))
	fmt.Println(strings.Repeat("=", 60))

	// 按环境分组显示
	envGroups := GroupConfigsByEnv(configs)
	for env, envConfigs := range envGroups {
		fmt.Printf("\n🌍 环境: %s (%d 个配置)\n", env, len(envConfigs))
		
		// 按服务分组
		serviceGroups := make(map[string][]ConfigInfo)
		for _, config := range envConfigs {
			serviceGroups[config.Service] = append(serviceGroups[config.Service], config)
		}

		for service, serviceConfigs := range serviceGroups {
			fmt.Printf("  📦 服务: %s (%d 个组件)\n", service, len(serviceConfigs))
			for _, config := range serviceConfigs {
				fmt.Printf("    📄 %s -> %s\n", config.Component, config.Key)
			}
		}
	}
	fmt.Println(strings.Repeat("=", 60))
}

// FormatConfigKey 格式化配置键显示
func FormatConfigKey(key string) (env, service, component string) {
	// 移除 /config/ 前缀
	trimmed := strings.TrimPrefix(key, "/config/")
	parts := strings.Split(trimmed, "/")
	
	if len(parts) >= 1 {
		env = parts[0]
	}
	if len(parts) >= 2 {
		service = parts[1]
	}
	if len(parts) >= 3 {
		component = parts[2]
	}
	
	return
}

// BuildConfigKey 构建配置键
func BuildConfigKey(env, service, component string) string {
	return fmt.Sprintf("/config/%s/%s/%s", env, service, component)
}

// GetConfigPrefix 获取配置前缀
func GetConfigPrefix(env, service string) string {
	if env == "" {
		return "/config/"
	}
	if service == "" {
		return fmt.Sprintf("/config/%s/", env)
	}
	return fmt.Sprintf("/config/%s/%s/", env, service)
}
