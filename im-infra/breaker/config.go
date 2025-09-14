package breaker

import (
	"fmt"
	"time"
)

// GetDefaultConfig 返回一个推荐的默认配置
// serviceName 会被用作 Config.ServiceName，并用于构建默认的 PoliciesPath
// 生产环境 ("production") 和开发环境 ("development") 会有不同的熔断策略
func GetDefaultConfig(serviceName, env string) *Config {
	config := &Config{
		ServiceName: serviceName,
	}

	switch env {
	case "production":
		config.PoliciesPath = fmt.Sprintf("/config/prod/%s/breakers/", serviceName)
	default:
		config.PoliciesPath = fmt.Sprintf("/config/dev/%s/breakers/", serviceName)
	}

	return config
}

// GetDefaultPolicy 返回默认的熔断策略
func GetDefaultPolicy() *Policy {
	return &Policy{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		OpenStateTimeout: time.Minute,
	}
}