package etcd

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigSource 配置源类型
type ConfigSource string

const (
	SourceUserInput ConfigSource = "user_input"
	SourceFile      ConfigSource = "file"
	SourceDefault   ConfigSource = "default"
	SourceMixed     ConfigSource = "mixed"
)

// Config 定义etcd客户端配置
type Config struct {
	Endpoints   []string      `json:"endpoints" yaml:"endpoints"`
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	LogLevel    string        `json:"log_level" yaml:"log_level"`

	// 配置元信息
	Source ConfigSource `json:"-" yaml:"-"`
}

// ConfigJSON 用于JSON解析的结构体
type ConfigJSON struct {
	Endpoints   []string `json:"endpoints"`
	DialTimeout string   `json:"dial_timeout"`
	LogLevel    string   `json:"log_level"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Endpoints:   []string{"localhost:23791", "localhost:23792", "localhost:23793"},
		DialTimeout: 5 * time.Second,
		LogLevel:    "info",
		Source:      SourceDefault,
	}
}

// LoadConfigFromFile 从文件加载配置
func LoadConfigFromFile(configPath string) (*Config, error) {
	if configPath == "" {
		// 尝试默认配置文件路径
		candidates := []string{
			"etcd-config.json",
			"etcd-config.yaml",
			"etcd-config.yml",
			"config/etcd.json",
			"config/etcd.yaml",
			"config/etcd.yml",
		}

		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				configPath = candidate
				break
			}
		}

		if configPath == "" {
			return nil, nil // 没有找到配置文件，返回 nil 而不是错误
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, WrapConfigurationError(err, "配置文件不存在: "+configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, WrapConfigurationError(err, "读取配置文件失败")
	}

	config := &Config{}
	ext := strings.ToLower(filepath.Ext(configPath))

	switch ext {
	case ".json":
		// 先解析到临时结构体
		var jsonConfig ConfigJSON
		if err := json.Unmarshal(data, &jsonConfig); err != nil {
			return nil, WrapConfigurationError(err, "解析JSON配置文件失败")
		}

		// 转换到最终配置
		config.Endpoints = jsonConfig.Endpoints
		config.LogLevel = jsonConfig.LogLevel

		// 解析时间字符串
		if jsonConfig.DialTimeout != "" {
			duration, err := time.ParseDuration(jsonConfig.DialTimeout)
			if err != nil {
				return nil, WrapConfigurationError(err, "解析dial_timeout失败")
			}
			config.DialTimeout = duration
		}

	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, WrapConfigurationError(err, "解析YAML配置文件失败")
		}
	default:
		return nil, WrapConfigurationError(ErrInvalidConfiguration, "不支持的配置文件格式: "+ext)
	}

	config.Source = SourceFile
	return config, nil
}

// MergeConfigs 合并配置，按优先级：用户输入 > 配置文件 > 默认值
func MergeConfigs(userConfig, fileConfig, defaultConfig *Config) *Config {
	if defaultConfig == nil {
		defaultConfig = DefaultConfig()
	}

	result := *defaultConfig // 从默认配置开始
	result.Source = SourceDefault

	// 应用文件配置（覆盖默认值）
	if fileConfig != nil {
		if len(fileConfig.Endpoints) > 0 {
			result.Endpoints = fileConfig.Endpoints
			result.Source = SourceFile
		}
		if fileConfig.DialTimeout > 0 {
			result.DialTimeout = fileConfig.DialTimeout
		}
		if fileConfig.LogLevel != "" {
			result.LogLevel = fileConfig.LogLevel
		}
	}

	// 应用用户输入配置（最高优先级）
	if userConfig != nil {
		hasUserInput := false

		if len(userConfig.Endpoints) > 0 {
			result.Endpoints = userConfig.Endpoints
			hasUserInput = true
		}
		if userConfig.DialTimeout > 0 {
			result.DialTimeout = userConfig.DialTimeout
			hasUserInput = true
		}
		if userConfig.LogLevel != "" {
			result.LogLevel = userConfig.LogLevel
			hasUserInput = true
		}

		if hasUserInput {
			if result.Source == SourceFile {
				result.Source = SourceMixed
			} else {
				result.Source = SourceUserInput
			}
		}
	}

	return &result
}

// LoadConfig 智能加载配置，支持配置优先级
func LoadConfig(userEndpoints []string, configPath string) (*Config, error) {
	// 1. 加载默认配置
	defaultConfig := DefaultConfig()

	// 2. 尝试加载文件配置（如果失败，不返回错误，继续使用默认配置）
	fileConfig, err := LoadConfigFromFile(configPath)
	if err != nil {
		// 检查是否是文件不存在的错误
		if IsConfigurationError(err) && strings.Contains(err.Error(), "配置文件不存在") {
			// 文件不存在，使用默认配置
			fileConfig = nil
		} else {
			// 其他错误（如解析错误）应该返回
			return nil, err
		}
	}

	// 3. 构建用户输入配置
	var userConfig *Config
	if len(userEndpoints) > 0 {
		userConfig = &Config{
			Endpoints: userEndpoints,
			Source:    SourceUserInput,
		}
	}

	// 4. 合并配置
	finalConfig := MergeConfigs(userConfig, fileConfig, defaultConfig)

	return finalConfig, nil
}

// ToManagerOptions 将 Config 转换为 ManagerOptions
func (c *Config) ToManagerOptions() *ManagerOptions {
	return &ManagerOptions{
		Endpoints:   c.Endpoints,
		DialTimeout: c.DialTimeout,
		Logger:      &DefaultLogger{Logger: log.Default()},
		RetryConfig: &RetryConfig{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     3 * time.Second,
			Multiplier:      2.0,
		},
		DefaultTTL:          30,
		ServicePrefix:       "/services",
		LockPrefix:          "/locks",
		MaxIdleConns:        10,
		MaxActiveConns:      100,
		ConnMaxLifetime:     30 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}
}
