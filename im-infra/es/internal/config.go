package internal

import "time"

const (
	// EnvDev 是开发环境
	EnvDev = "development"
	// EnvProd 是生产环境
	EnvProd = "production"
)

// Config 定义了 Elasticsearch 客户端的配置
type Config struct {
	// Addresses 是要连接的 Elasticsearch 节点列表
	Addresses []string `json:"addresses" toml:"addresses"`
	// Username 用于认证
	Username string `json:"username" toml:"username"`
	// Password 用于认证
	Password string `json:"password" toml:"password"`
	// CloudID 是 Elastic Cloud 部署的 ID
	// 如果设置了 CloudID，Addresses 应该为空
	CloudID string `json:"cloud_id" toml:"cloud_id"`
	// APIKey 用于认证
	APIKey string `json:"api_key" toml:"api_key"`

	// BulkIndexer 包含批量索引器的参数
	BulkIndexer struct {
		// FlushBytes 是批量索引器应该刷新的字节大小
		FlushBytes int `json:"flush_bytes" toml:"flush_bytes"`
		// FlushInterval 是批量索引器应该刷新的时间间隔
		FlushInterval time.Duration `json:"flush_interval" toml:"flush_interval"`
		// Workers 是批量索引器的并发工作线程数
		Workers int `json:"workers" toml:"workers"`
	} `json:"bulk_indexer" toml:"bulk_indexer"`
}

// GetDefaultConfig 返回给定环境的默认配置
func GetDefaultConfig(env string) *Config {
	cfg := &Config{
		Addresses: []string{"http://localhost:9200"},
	}

	cfg.BulkIndexer.FlushBytes = 1024 * 1024 // 1MB
	cfg.BulkIndexer.FlushInterval = 5 * time.Second
	cfg.BulkIndexer.Workers = 2

	if env == EnvProd {
		cfg.BulkIndexer.Workers = 4
	}

	return cfg
}
