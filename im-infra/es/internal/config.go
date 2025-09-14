package internal

import "time"

const (
	// EnvDev is the development environment.
	EnvDev = "development"
	// EnvProd is the production environment.
	EnvProd = "production"
)

// Config defines the configuration for the Elasticsearch client.
type Config struct {
	// Addresses is a list of Elasticsearch nodes to connect to.
	Addresses []string `json:"addresses" toml:"addresses"`
	// Username for authentication.
	Username string `json:"username" toml:"username"`
	// Password for authentication.
	Password string `json:"password" toml:"password"`
	// CloudID is the ID of the Elastic Cloud deployment.
	// If CloudID is set, Addresses should be empty.
	CloudID string `json:"cloud_id" toml:"cloud_id"`
	// APIKey for authentication.
	APIKey string `json:"api_key" toml:"api_key"`

	// BulkIndexer includes parameters for the bulk indexer.
	BulkIndexer struct {
		// FlushBytes is the size in bytes at which the bulk indexer should flush.
		FlushBytes int `json:"flush_bytes" toml:"flush_bytes"`
		// FlushInterval is the time interval at which the bulk indexer should flush.
		FlushInterval time.Duration `json:"flush_interval" toml:"flush_interval"`
		// Workers is the number of concurrent workers for the bulk indexer.
		Workers int `json:"workers" toml:"workers"`
	} `json:"bulk_indexer" toml:"bulk_indexer"`
}

// GetDefaultConfig returns the default configuration for the given environment.
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
