package metrics

import "time"

// Config holds the public-facing configuration for the metrics and tracing system.
// Users of the library will interact with this struct.
type Config struct {
	ServiceName          string
	ExporterType         string
	ExporterEndpoint     string
	PrometheusListenAddr string
	SamplerType          string
	SamplerRatio         float64
	SlowRequestThreshold time.Duration
}

// DefaultConfig returns a new Config with sensible default values.
func DefaultConfig() *Config {
	return &Config{
		ServiceName:          "unknown-service",
		ExporterType:         "stdout", // Use stdout for easy debugging by default
		ExporterEndpoint:     "http://localhost:14268/api/traces",
		PrometheusListenAddr: "", // Disabled by default, enable by providing an address like ":9090"
		SamplerType:          "always_on",
		SamplerRatio:         1.0,
		SlowRequestThreshold: 500 * time.Millisecond,
	}
}
