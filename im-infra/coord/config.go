package coord

import "time"

// CoordinatorConfig 协调器配置配置
type CoordinatorConfig struct {
	// Endpoints etcd 服务器地址列表
	Endpoints []string `json:"endpoints"`

	// Username etcd 用户名（可选）
	Username string `json:"username,omitempty"`

	// Password etcd 密码（可选）
	Password string `json:"password,omitempty"`

	// Timeout 连接超时时间
	Timeout time.Duration `json:"timeout"`

	// RetryConfig 重试配置
	RetryConfig *RetryConfig `json:"retry_config,omitempty"`
}

// RetryConfig 重试机制配置
type RetryConfig struct {
	// MaxAttempts 最大重试次数
	MaxAttempts int `json:"max_attempts"`

	// InitialDelay 初始延迟
	InitialDelay time.Duration `json:"initial_delay"`

	// MaxDelay 最大延迟
	MaxDelay time.Duration `json:"max_delay"`

	// Multiplier 退避倍数
	Multiplier float64 `json:"multiplier"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() CoordinatorConfig {
	return CoordinatorConfig{
		Endpoints: []string{"localhost:2379"},
		Timeout:   5 * time.Second,
		RetryConfig: &RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     2 * time.Second,
			Multiplier:   2.0,
		},
	}
}
