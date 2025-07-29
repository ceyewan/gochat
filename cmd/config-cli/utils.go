package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord/config"
)

// updateWithCAS 使用 Compare-And-Set 进行原子更新
func updateWithCAS(
	ctx context.Context,
	configCenter config.ConfigCenter,
	key string,
	updateData interface{},
	updateFunc func(existing map[string]interface{}, updateData interface{}) (map[string]interface{}, error),
) error {
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		// 获取当前配置和版本
		var currentValue map[string]interface{}
		version, err := configCenter.GetWithVersion(ctx, key, &currentValue)
		if err != nil {
			// 如果配置不存在，使用空配置
			if isNotFoundError(err) {
				currentValue = make(map[string]interface{})
				version = 0
			} else {
				return fmt.Errorf("failed to get current config: %w", err)
			}
		}

		// 应用更新
		updatedValue, err := updateFunc(currentValue, updateData)
		if err != nil {
			return fmt.Errorf("failed to apply update: %w", err)
		}

		// 尝试 CAS 更新
		err = configCenter.CompareAndSet(ctx, key, updatedValue, version)
		if err == nil {
			fmt.Printf("✅ Configuration updated successfully: %s\n", key)
			return nil
		}

		// 如果是版本冲突，重试
		if isConflictError(err) {
			fmt.Printf("⚠️  Version conflict, retrying... (attempt %d/%d)\n", attempt+1, maxRetries)
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			continue
		}

		// 其他错误直接返回
		return fmt.Errorf("failed to update config: %w", err)
	}

	return fmt.Errorf("failed to update config after %d retries due to version conflicts", maxRetries)
}

// deepMerge 深度合并两个 map
func deepMerge(existing map[string]interface{}, updateData interface{}) (map[string]interface{}, error) {
	updateMap, ok := updateData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("update data must be a map[string]interface{}")
	}

	result := make(map[string]interface{})

	// 复制现有配置
	for k, v := range existing {
		result[k] = v
	}

	// 递归合并更新配置
	for k, v := range updateMap {
		if existingValue, exists := result[k]; exists {
			// 如果两个值都是 map，递归合并
			if existingMap, ok := existingValue.(map[string]interface{}); ok {
				if updateSubMap, ok := v.(map[string]interface{}); ok {
					merged, err := deepMerge(existingMap, updateSubMap)
					if err != nil {
						return nil, err
					}
					result[k] = merged
					continue
				}
			}
		}
		// 否则直接覆盖
		result[k] = v
	}

	return result, nil
}

// deleteField 删除指定字段
func deleteField(existing map[string]interface{}, updateData interface{}) (map[string]interface{}, error) {
	fieldPath, ok := updateData.(string)
	if !ok {
		return nil, fmt.Errorf("field path must be a string")
	}

	result := make(map[string]interface{})

	// 深拷贝现有配置
	for k, v := range existing {
		result[k] = deepCopy(v)
	}

	// 解析字段路径并删除
	return deleteFieldByPath(result, fieldPath)
}

// deleteFieldByPath 根据路径删除字段
func deleteFieldByPath(data map[string]interface{}, path string) (map[string]interface{}, error) {
	if path == "" {
		return data, nil
	}

	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		// 直接删除字段
		delete(data, parts[0])
		return data, nil
	}

	// 递归删除嵌套字段
	key := parts[0]
	remainingPath := strings.Join(parts[1:], ".")

	if value, exists := data[key]; exists {
		if subMap, ok := value.(map[string]interface{}); ok {
			updatedSubMap, err := deleteFieldByPath(subMap, remainingPath)
			if err != nil {
				return nil, err
			}
			data[key] = updatedSubMap
		} else {
			return nil, fmt.Errorf("cannot delete field '%s': parent is not an object", remainingPath)
		}
	}

	return data, nil
}

// deepCopy 深拷贝 interface{}
func deepCopy(src interface{}) interface{} {
	switch v := src.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = deepCopy(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = deepCopy(val)
		}
		return result
	default:
		return v
	}
}

// isNotFoundError 检查是否为 NotFound 错误
func isNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "NOT_FOUND") ||
		strings.Contains(err.Error(), "not found")
}

// isConflictError 检查是否为冲突错误
func isConflictError(err error) bool {
	return strings.Contains(err.Error(), "CONFLICT") ||
		strings.Contains(err.Error(), "version mismatch")
}

// batchWriteConfigs 批量写入配置
func batchWriteConfigs(ctx context.Context, configCenter config.ConfigCenter, configs []ConfigInfo) error {
	fmt.Printf("\n🚀 开始批量写入 %d 个配置...\n", len(configs))
	fmt.Println(strings.Repeat("=", 60))

	successCount := 0
	var errors []string

	for i, cfg := range configs {
		fmt.Printf("[%d/%d] 写入 %s...", i+1, len(configs), cfg.Key)

		// 写入配置
		err := configCenter.Set(ctx, cfg.Key, cfg.Config)
		if err != nil {
			fmt.Printf(" ❌ 失败: %v\n", err)
			errors = append(errors, fmt.Sprintf("%s: %v", cfg.Key, err))
			continue
		}

		// 验证配置是否写入成功
		var retrieved interface{}
		err = configCenter.Get(ctx, cfg.Key, &retrieved)
		if err != nil {
			fmt.Printf(" ❌ 验证失败: %v\n", err)
			errors = append(errors, fmt.Sprintf("%s (验证失败): %v", cfg.Key, err))
			continue
		}

		fmt.Printf(" ✅ 成功\n")
		successCount++
	}

	// 显示结果摘要
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("📊 批量写入完成:\n")
	fmt.Printf("   总计: %d 个配置\n", len(configs))
	fmt.Printf("   成功: %d 个配置\n", successCount)
	fmt.Printf("   失败: %d 个配置\n", len(configs)-successCount)

	if len(errors) > 0 {
		fmt.Printf("\n❌ 失败详情:\n")
		for _, errMsg := range errors {
			fmt.Printf("   - %s\n", errMsg)
		}
		return fmt.Errorf("批量写入过程中有 %d 个配置失败", len(errors))
	}

	fmt.Printf("\n🎉 所有配置写入成功！\n")
	return nil
}

// printKeysAsJSON 以 JSON 格式打印配置键
func printKeysAsJSON(keys []string) error {
	output, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format as JSON: %w", err)
	}
	fmt.Println(string(output))
	return nil
}

// printKeysAsTable 以表格格式打印配置键
func printKeysAsTable(keys []string, detailed bool) error {
	fmt.Printf("📋 找到 %d 个配置:\n", len(keys))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-20s %-20s %-20s %s\n", "环境", "服务", "组件", "配置键")
	fmt.Println(strings.Repeat("-", 80))

	for _, key := range keys {
		env, service, component := FormatConfigKey(key)
		fmt.Printf("%-20s %-20s %-20s %s\n", env, service, component, key)
	}

	fmt.Println(strings.Repeat("=", 80))
	return nil
}

// printKeysAsTree 以树形格式打印配置键
func printKeysAsTree(keys []string, detailed bool, configCenter config.ConfigCenter, ctx context.Context) error {
	if len(keys) == 0 {
		fmt.Println("没有找到配置")
		return nil
	}

	fmt.Printf("📋 找到 %d 个配置:\n", len(keys))
	fmt.Println(strings.Repeat("=", 60))

	// 按环境分组
	envGroups := make(map[string]map[string][]string)
	for _, key := range keys {
		env, service, component := FormatConfigKey(key)
		if envGroups[env] == nil {
			envGroups[env] = make(map[string][]string)
		}
		envGroups[env][service] = append(envGroups[env][service], component)
	}

	// 显示树形结构
	for env, services := range envGroups {
		fmt.Printf("\n🌍 %s\n", env)
		for service, components := range services {
			fmt.Printf("  📦 %s\n", service)
			for _, component := range components {
				key := BuildConfigKey(env, service, component)
				fmt.Printf("    📄 %s", component)

				if detailed {
					// 显示配置详情
					var config interface{}
					if err := configCenter.Get(ctx, key, &config); err == nil {
						configJSON, _ := json.Marshal(config)
						fmt.Printf(" -> %s", string(configJSON))
					}
				}
				fmt.Printf("\n")
			}
		}
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\n💡 提示: 使用 'config-cli get <key>' 查看具体配置内容\n")
	return nil
}
