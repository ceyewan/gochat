package internal

import "time"

// Config holds the configuration for the metrics and tracing system.
// This is an internal config struct.
type Config struct {
	ServiceName          string        `mapstructure:"service_name"`
	ExporterType         string        `mapstructure:"exporter_type"`
	ExporterEndpoint     string        `mapstructure:"exporter_endpoint"`
	PrometheusListenAddr string        `mapstructure:"prometheus_listen_addr"`
	SamplerType          string        `mapstructure:"sampler_type"`
	SamplerRatio         float64       `mapstructure:"sampler_ratio"`
	SlowRequestThreshold time.Duration `mapstructure:"slow_request_threshold"`
}
