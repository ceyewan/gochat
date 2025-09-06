package uid

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/uid/internal"
)

type UID interface {
	GenerateInt64() int64
	GenerateString() string
	GenerateUUIDV4() string
	GenerateUUIDV7() string
	ValidateUUID(uuidStr string) bool
	ParseSnowflake(id int64) (timestamp int64, workerID int64, datacenterID int64, sequence int64)
	Close() error
}

type Config struct {
	WorkerID     int64 `json:"workerID" yaml:"workerID"`
	DatacenterID int64 `json:"datacenterID" yaml:"datacenterID"`
	EnableUUID   bool  `json:"enableUUID" yaml:"enableUUID"`
}

func DefaultConfig() Config {
	return Config{
		WorkerID:     1,
		DatacenterID: 1,
		EnableUUID:   true,
	}
}

func (c *Config) Validate() error {
	if c.WorkerID < 0 || c.WorkerID > 31 {
		return fmt.Errorf("workerID must be between 0 and 31, got: %d", c.WorkerID)
	}
	if c.DatacenterID < 0 || c.DatacenterID > 31 {
		return fmt.Errorf("datacenterID must be between 0 and 31, got: %d", c.DatacenterID)
	}
	return nil
}

type Options struct {
	Logger        clog.Logger
	ComponentName string
}

type Option func(*Options)

func WithLogger(logger clog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

func WithComponentName(name string) Option {
	return func(o *Options) {
		o.ComponentName = name
	}
}

func (c Config) GetWorkerID() int64 {
	return c.WorkerID
}

func (c Config) GetDatacenterID() int64 {
	return c.DatacenterID
}

func (c Config) GetEnableUUID() bool {
	return c.EnableUUID
}

func New(ctx context.Context, cfg Config, opts ...Option) (UID, error) {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger := options.Logger
	if logger == nil {
		logger = clog.Module("uid")
	}

	if options.ComponentName != "" {
		logger = logger.With(clog.String("name", options.ComponentName))
	}

	client, err := internal.NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create uid client: %w", err)
	}

	logger.Info("uid component initialized successfully",
		clog.Int64("workerID", cfg.WorkerID),
		clog.Int64("datacenterID", cfg.DatacenterID),
		clog.Bool("enableUUID", cfg.EnableUUID),
	)

	return client, nil
}
