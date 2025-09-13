package coord

import "time"

// Config 是 coord 组件的配置结构体
type Config struct {
	// Endpoints 是 etcd 集群的地址列表
	Endpoints []string `json:"endpoints"`
	
	// DialTimeout 是连接 etcd 的超时时间
	DialTimeout time.Duration `json:"dialTimeout"`
	
	// KeepAliveTime 是 keepalive 心跳间隔
	KeepAliveTime time.Duration `json:"keepAliveTime"`
	
	// KeepAliveTimeout 是 keepalive 超时时间
	KeepAliveTimeout time.Duration `json:"keepAliveTimeout"`
	
	// Username 是认证用户名，可选
	Username string `json:"username,omitempty"`
	
	// Password 是认证密码，可选
	Password string `json:"password,omitempty"`
	
	// TLS 相关配置，可选
	TLS *TLSConfig `json:"tls,omitempty"`
}

// TLSConfig 定义了 TLS 连接配置
type TLSConfig struct {
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
	CAFile   string `json:"caFile,omitempty"`
}

// GetDefaultConfig 返回默认的 coord 配置
func GetDefaultConfig(env string) *Config {
	switch env {
	case "development":
		return &Config{
			Endpoints:       []string{"localhost:2379"},
			DialTimeout:     5 * time.Second,
			KeepAliveTime:   30 * time.Second,
			KeepAliveTimeout: 10 * time.Second,
		}
	case "production":
		return &Config{
			Endpoints:       []string{"etcd1:2379", "etcd2:2379", "etcd3:2379"},
			DialTimeout:     10 * time.Second,
			KeepAliveTime:   30 * time.Second,
			KeepAliveTimeout: 10 * time.Second,
		}
	default:
		return &Config{
			Endpoints:       []string{"localhost:2379"},
			DialTimeout:     5 * time.Second,
			KeepAliveTime:   30 * time.Second,
			KeepAliveTimeout: 10 * time.Second,
		}
	}
}

// CoordinatorConfig 保持向后兼容的别名
type CoordinatorConfig = Config

// DefaultConfig 保持向后兼容的函数
func DefaultConfig() Config {
	return *GetDefaultConfig("development")
}
